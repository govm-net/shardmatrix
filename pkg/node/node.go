package node

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/govm-net/shardmatrix/pkg/api"
	"github.com/govm-net/shardmatrix/pkg/blockchain"
	"github.com/govm-net/shardmatrix/pkg/config"
	"github.com/govm-net/shardmatrix/pkg/network"
	"github.com/govm-net/shardmatrix/pkg/storage"
	"github.com/govm-net/shardmatrix/pkg/txpool"
	"github.com/govm-net/shardmatrix/pkg/types"
	"github.com/govm-net/shardmatrix/pkg/validator"
)

// Node represents a blockchain node
type Node struct {
	config         *config.Config
	network        *network.Network
	blockchain     *blockchain.Blockchain
	blockStore     storage.BlockStore
	txStore        storage.TransactionStoreInterface
	accountStore   storage.AccountStoreInterface
	validatorStore storage.ValidatorStoreInterface
	txPool         *txpool.MemoryTxPool
	validator      *validator.Validator
	apiServer      *api.APIServer

	// 状态管理
	isRunning bool
	startTime time.Time

	// 网络状态监控
	networkMonitor *NetworkMonitor
}

// NetworkStatus 网络状态
type NetworkStatus struct {
	ConnectedPeers int       `json:"connected_peers"`
	LastUpdateTime time.Time `json:"last_update_time"`
	IsPartitioned  bool      `json:"is_partitioned"`
	PartitionSince time.Time `json:"partition_since,omitempty"`
	ReconnectCount int       `json:"reconnect_count"`
}

// NetworkMonitor 网络监控器
type NetworkMonitor struct {
	node           *Node
	status         NetworkStatus
	lastPeerCount  int
	partitionTimer *time.Timer
	mutex          sync.RWMutex
}

// BlockSyncRequest 区块同步请求
type BlockSyncRequest struct {
	FromHeight uint64     `json:"from_height"` // 起始高度
	ToHeight   uint64     `json:"to_height"`   // 结束高度
	MaxBlocks  int        `json:"max_blocks"`  // 最大区块数
	Height     uint64     `json:"height,omitempty"`
	Hash       types.Hash `json:"hash,omitempty"`
}

// New creates a new blockchain node
func New(cfg *config.Config) (*Node, error) {
	// 创建存储层 - 根据配置选择存储类型
	var (
		blockStore     storage.BlockStore
		txStore        storage.TransactionStoreInterface
		accountStore   storage.AccountStoreInterface
		validatorStore storage.ValidatorStoreInterface
	)

	// 检查是否配置了使用LevelDB存储
	if cfg.Storage.DBType == "leveldb" {
		// 创建LevelDB存储管理器
		storageConfig := &storage.StorageConfig{
			DataDir:   cfg.Storage.DataDir,
			UseMemory: false,
		}

		storageManager, err := storage.NewStorageManager(storageConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create storage manager: %v", err)
		}

		// 使用LevelDB存储
		blockStore = storageManager.BlockStore
		txStore = storageManager.TxStore
		accountStore = storageManager.AccountStore
		validatorStore = storageManager.ValidatorStore

		fmt.Printf("Using LevelDB storage at: %s\n", cfg.Storage.DataDir)
	} else {
		// 使用内存存储（默认）
		blockStore = storage.NewMemoryBlockStore()
		txStore = storage.NewMemoryTransactionStore()
		accountStore = storage.NewMemoryAccountStore()
		validatorStore = storage.NewMemoryValidatorStore()

		fmt.Println("Using memory storage (for testing)")
	}

	// 创建交易池
	txPoolConfig := txpool.DefaultTxPoolConfig()
	txPool := txpool.NewMemoryTxPool(txPoolConfig)

	// 创建验证器
	validatorConfig := validator.DefaultValidationConfig()
	blockValidator := validator.NewValidator(
		validatorConfig,
		blockStore,
		txStore,
		validatorStore,
		accountStore,
	)

	// 创建区块链管理器
	blockchainConfig := blockchain.DefaultBlockchainConfig()
	blockchainConfig.ChainID = cfg.Blockchain.ChainID
	blockchainConfig.BlockInterval = time.Duration(cfg.Blockchain.BlockInterval) * time.Second
	blockchainConfig.MaxBlockSize = cfg.Blockchain.MaxBlockSize

	blockchainManager, err := blockchain.NewBlockchain(blockchainConfig, blockStore, blockValidator)
	if err != nil {
		return nil, fmt.Errorf("failed to create blockchain manager: %v", err)
	}

	// 创建网络管理器配置
	networkConfig := &network.NetworkConfig{
		Port:           cfg.Network.Port,
		Host:           cfg.Network.Host,
		MaxPeers:       cfg.Network.Protection.MaxConnections,
		BootstrapPeers: cfg.Network.BootstrapPeers,
		PrivateKeyPath: "", // 使用默认生成
	}

	networkManager, err := network.New(networkConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create network manager: %v", err)
	}

	// 创建网络监控器
	networkMonitor := &NetworkMonitor{
		status: NetworkStatus{
			LastUpdateTime: time.Now(),
		},
		lastPeerCount: 0,
	}

	// 创建节点
	node := &Node{
		config:         cfg,
		network:        networkManager,
		blockchain:     blockchainManager,
		blockStore:     blockStore,
		txStore:        txStore,
		accountStore:   accountStore,
		validatorStore: validatorStore,
		txPool:         txPool,
		validator:      blockValidator,
		isRunning:      false,
		networkMonitor: networkMonitor,
	}

	// 设置网络监控器的节点引用
	networkMonitor.node = node

	// 创建API服务器
	apiServer := api.NewAPIServer(cfg, blockchainManager, networkManager)
	node.apiServer = apiServer

	// 设置区块链回调
	blockchainManager.SetCallbacks(
		node.onNewBlockFromBlockchain,
		node.onChainReorganization,
		node.onSyncComplete,
	)

	// 设置网络回调
	// 注册消息处理器
	networkManager.RegisterMessageHandler("blocks", node.onBlockMessage)
	networkManager.RegisterMessageHandler("transactions", node.onTransactionMessage)

	// 注册区块同步请求处理器
	networkManager.RegisterRequestHandler("block_request", node.onBlockRequest)

	return node, nil
}

// Start starts the node
func (n *Node) Start() error {
	if n.isRunning {
		return fmt.Errorf("node is already running")
	}

	fmt.Printf("Starting ShardMatrix node on libp2p network\n")

	// 启动网络层需要使用 Run 方法
	go func() {
		if err := n.network.Run(context.Background()); err != nil {
			fmt.Printf("Network run error: %v\n", err)
		}
	}()

	// 启动网络监控循环
	go n.networkMonitorLoop()

	// 启动区块生产循环（如果有共识）
	if n.blockchain.GetConsensus() != nil && n.blockchain.GetConsensus().IsConsensusEnabled() {
		go n.blockProductionLoop()
	}

	// 启动API服务器
	if n.apiServer != nil {
		if err := n.apiServer.Start(); err != nil {
			fmt.Printf("Failed to start API server: %v\n", err)
		} else {
			fmt.Printf("API server started on %s:%d\n", n.config.API.Host, n.config.API.Port)
		}
	}

	// 区块链管理器已在创建时自动初始化创世状态
	// 无需额外的初始化步骤

	n.isRunning = true
	n.startTime = time.Now()

	fmt.Printf("ShardMatrix node started successfully\n")
	fmt.Printf("Node ID: %s\n", n.network.GetLocalPeerID())
	fmt.Printf("Listen Addresses: %v\n", n.network.GetLocalAddresses())

	// 显示区块链状态
	chainState := n.blockchain.GetChainState()
	fmt.Printf("Blockchain height: %d\n", chainState.Height)
	fmt.Printf("Best block: %s\n", chainState.BestBlockHash.String())

	return nil
}

// networkMonitorLoop 网络监控循环
func (n *Node) networkMonitorLoop() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for n.isRunning {
		<-ticker.C
		n.networkMonitor.checkNetworkStatus()

	}
}

// checkNetworkStatus 检查网络状态
func (nm *NetworkMonitor) checkNetworkStatus() {
	nm.mutex.Lock()
	defer nm.mutex.Unlock()

	// 获取当前连接的节点数量
	currentPeers := len(nm.node.network.GetPeers())
	nm.status.ConnectedPeers = currentPeers
	nm.status.LastUpdateTime = time.Now()

	// 检查是否有网络分区
	if nm.lastPeerCount > 0 && currentPeers == 0 {
		// 检测到可能的网络分区
		if !nm.status.IsPartitioned {
			nm.status.IsPartitioned = true
			nm.status.PartitionSince = time.Now()
			fmt.Printf("⚠️  Network partition detected at %s\n", nm.status.PartitionSince.Format("2006-01-02 15:04:05"))

			// 触发分区处理
			nm.handleNetworkPartition()
		}
	} else if nm.lastPeerCount == 0 && currentPeers > 0 {
		// 网络连接恢复
		if nm.status.IsPartitioned {
			nm.status.IsPartitioned = false
			partitionDuration := time.Since(nm.status.PartitionSince)
			nm.status.ReconnectCount++
			fmt.Printf("✅ Network reconnected after partition (duration: %v)\n", partitionDuration)

			// 触发分区恢复处理
			nm.handleNetworkRecovery()
		}
	}

	// 更新最后的节点数量
	nm.lastPeerCount = currentPeers
}

// handleNetworkPartition 处理网络分区
func (nm *NetworkMonitor) handleNetworkPartition() {
	fmt.Printf("🔄 Handling network partition...\n")

	// 在分区期间，节点继续正常工作但不广播新区块
	// 可以记录分区事件到日志
	chainState := nm.node.blockchain.GetChainState()
	fmt.Printf("  Current chain height: %d\n", chainState.Height)
	fmt.Printf("  Best block: %s\n", chainState.BestBlockHash.String())

	// 可以暂停某些网络相关的操作
	// 但保持区块生产和交易处理
}

// handleNetworkRecovery 处理网络恢复
func (nm *NetworkMonitor) handleNetworkRecovery() {
	fmt.Printf("🔄 Handling network recovery...\n")

	// 网络恢复后，触发区块同步
	go func() {
		// 等待一段时间让连接稳定
		time.Sleep(5 * time.Second)

		// 触发区块同步
		fmt.Printf("🔄 Initiating blockchain sync after network recovery...\n")
		// 这里可以调用同步机制
	}()
}

// GetNetworkStatus 获取网络状态
func (nm *NetworkMonitor) GetNetworkStatus() NetworkStatus {
	nm.mutex.RLock()
	defer nm.mutex.RUnlock()

	return nm.status
}

// Stop stops the node
func (n *Node) Stop() error {
	if !n.isRunning {
		return nil
	}

	fmt.Println("Stopping ShardMatrix node...")

	// 停止API服务器
	if n.apiServer != nil {
		if err := n.apiServer.Stop(); err != nil {
			fmt.Printf("Failed to stop API server: %v\n", err)
		} else {
			fmt.Println("API server stopped")
		}
	}

	// 停止网络层通过关闭主机实现
	if n.network.GetHost() != nil {
		n.network.GetHost().Close()
	}

	// 关闭存储层
	if n.blockStore != nil {
		n.blockStore.Close()
	}
	if n.txStore != nil {
		n.txStore.Close()
	}
	if n.accountStore != nil {
		n.accountStore.Close()
	}
	if n.validatorStore != nil {
		n.validatorStore.Close()
	}

	n.isRunning = false

	fmt.Println("ShardMatrix node stopped")
	return nil
}

// 网络消息处理函数

// onBlockMessage 接收区块消息处理
func (n *Node) onBlockMessage(peerID string, msg network.NetMessage) error {
	fmt.Printf("Received block message from peer %s\n", peerID)

	// 反序列化区块数据
	var block types.Block
	if err := json.Unmarshal(msg.Data, &block); err != nil {
		fmt.Printf("Failed to deserialize block from peer %s: %v\n", peerID, err)
		return err
	}

	fmt.Printf("Received block %s at height %d from peer %s\n",
		block.Hash().String(), block.Header.Number, peerID)

	// 验证区块
	if err := n.validator.ValidateNewBlock(&block, nil); err != nil {
		// 检查是否是前置区块缺失错误
		if n.isPrevBlockNotFoundError(err) {
			fmt.Printf("⛓️  Missing previous block for block %s (height: %d), initiating sync...\n",
				block.Hash().String(), block.Header.Number)

			// 主动请求缺失的前置区块
			if err := n.requestMissingBlock(peerID, block.Header.PrevHash); err != nil {
				fmt.Printf("❌ Failed to request missing block: %v\n", err)
			} else {
				fmt.Printf("🔄 Successfully requested missing block %s\n", block.Header.PrevHash.String())
			}
		} else {
			fmt.Printf("Block validation failed: %v\n", err)
		}
		return err
	}

	// 检查是否应该接受这个区块
	if n.shouldAcceptBlock(&block) {
		// 尝试添加区块到链中
		if err := n.blockchain.AddBlock(&block); err != nil {
			fmt.Printf("Failed to add block to blockchain: %v\n", err)
			return err
		}

		fmt.Printf("✅ Block %s added to blockchain (height: %d)\n",
			block.Hash().String(), block.Header.Number)

		// 从交易池中移除已确认的交易
		for _, txHash := range block.Transactions {
			n.txPool.RemoveTransaction(txHash)
		}
	} else {
		fmt.Printf("⚠️  Block %s rejected (height: %d, current: %d)\n",
			block.Hash().String(), block.Header.Number, n.blockchain.GetChainState().Height)
	}

	return nil
}

// shouldAcceptBlock 判断是否应该接受区块
func (n *Node) shouldAcceptBlock(block *types.Block) bool {
	chainState := n.blockchain.GetChainState()

	// 只接受高度比当前链高的区块（最长链规则）
	if block.Header.Number <= chainState.Height {
		// 如果是相同高度但不同哈希，可能是分叉
		if block.Header.Number == chainState.Height {
			// 只有当区块哈希与当前最佳区块不同时才考虑作为分叉
			return !block.Hash().Equal(chainState.BestBlockHash)
		}
		return false
	}

	// 接受高度更高的区块
	return true
}

// isPrevBlockNotFoundError 检查是否为前置区块未找到错误
func (n *Node) isPrevBlockNotFoundError(err error) bool {
	if err == nil {
		return false
	}

	// 检查错误消息中是否包含 PREV_BLOCK_NOT_FOUND
	return strings.Contains(err.Error(), "PREV_BLOCK_NOT_FOUND") ||
		strings.Contains(err.Error(), "previous block not found")
}

// requestMissingBlock 请求缺失的区块
func (n *Node) requestMissingBlock(peerID string, blockHash types.Hash) error {
	// 构造区块请求
	blockReq := BlockSyncRequest{
		Hash: blockHash,
	}

	reqData, err := json.Marshal(blockReq)
	if err != nil {
		return fmt.Errorf("failed to serialize block request: %v", err)
	}

	// 向对等节点发送请求
	respData, err := n.network.SendRequest(peerID, "block_request", reqData)
	if err != nil {
		return fmt.Errorf("failed to send block request to peer %s: %v", peerID, err)
	}

	// 反序列化响应数据
	var receivedBlock types.Block
	if err := json.Unmarshal(respData, &receivedBlock); err != nil {
		return fmt.Errorf("failed to deserialize received block: %v", err)
	}

	fmt.Printf("📥 Received missing block %s (height: %d) from peer %s\n",
		receivedBlock.Hash().String(), receivedBlock.Header.Number, peerID)

	// 验证并添加区块
	if err := n.validator.ValidateNewBlock(&receivedBlock, nil); err != nil {
		return fmt.Errorf("received block validation failed: %v", err)
	}

	// 尝试添加到区块链
	if err := n.blockchain.AddBlock(&receivedBlock); err != nil {
		return fmt.Errorf("failed to add received block to blockchain: %v", err)
	}

	fmt.Printf("✅ Successfully added missing block %s to blockchain\n",
		receivedBlock.Hash().String())

	return nil
}

// onTransactionMessage 接收交易消息处理
func (n *Node) onTransactionMessage(peerID string, msg network.NetMessage) error {
	fmt.Printf("Received transaction message from peer %s\n", peerID)

	// TODO: 反序列化交易数据并处理
	// 这里需要实现具体的交易反序列化和验证逻辑

	return nil
}

// onBlockRequest 处理区块请求
func (n *Node) onBlockRequest(peerID string, req network.Request) ([]byte, error) {
	fmt.Printf("📩 Received block request from peer %s\n", peerID)

	// 解析请求参数
	var blockReq BlockSyncRequest
	if err := json.Unmarshal(req.Data, &blockReq); err != nil {
		return nil, fmt.Errorf("failed to parse block request: %v", err)
	}

	fmt.Printf("🔍 Block request: height=%d, hash=%s\n", blockReq.Height, blockReq.Hash)

	// 按高度查找区块
	if blockReq.Height > 0 {
		block, err := n.blockStore.GetBlockByHeight(blockReq.Height)
		if err != nil {
			return nil, fmt.Errorf("block at height %d not found: %v", blockReq.Height, err)
		}

		// 序列化区块返回
		blockData, err := json.Marshal(block)
		if err != nil {
			return nil, fmt.Errorf("failed to serialize block: %v", err)
		}

		fmt.Printf("✅ Sending block %s (height: %d) to peer %s\n",
			block.Hash().String(), block.Header.Number, peerID)

		return blockData, nil
	}

	// 按哈希查找区块
	if !blockReq.Hash.IsZero() {
		block, err := n.blockStore.GetBlock(blockReq.Hash)
		if err != nil {
			return nil, fmt.Errorf("block with hash %s not found: %v", blockReq.Hash.String(), err)
		}

		// 序列化区块返回
		blockData, err := json.Marshal(block)
		if err != nil {
			return nil, fmt.Errorf("failed to serialize block: %v", err)
		}

		fmt.Printf("✅ Sending block %s (height: %d) to peer %s\n",
			block.Hash().String(), block.Header.Number, peerID)

		return blockData, nil
	}

	// 按范围查找区块
	if blockReq.FromHeight > 0 {
		toHeight := blockReq.ToHeight
		if toHeight == 0 {
			toHeight = blockReq.FromHeight
		}

		blocks, err := n.blockchain.GetBlocksForSync(blockReq.FromHeight, toHeight, blockReq.MaxBlocks)
		if err != nil {
			return nil, fmt.Errorf("failed to get blocks for sync: %v", err)
		}

		// 如果没有找到区块，返回错误
		if len(blocks) == 0 {
			return nil, fmt.Errorf("no blocks found in range %d-%d", blockReq.FromHeight, toHeight)
		}

		// 序列化第一个区块返回（为了保持向后兼容）
		blockData, err := json.Marshal(blocks[0])
		if err != nil {
			return nil, fmt.Errorf("failed to serialize block: %v", err)
		}

		fmt.Printf("✅ Sending block %s (height: %d) to peer %s\n",
			blocks[0].Hash().String(), blocks[0].Header.Number, peerID)

		return blockData, nil
	}

	return nil, fmt.Errorf("invalid block request: no valid parameters specified")
}

// 区块链回调函数

// onNewBlockFromBlockchain 区块链新区块回调
func (n *Node) onNewBlockFromBlockchain(block *types.Block) {
	fmt.Printf("New block added to main chain: %s (height: %d)\n", block.Hash().String(), block.Header.Number)

	// 从交易池中移除已确认的交易
	for _, txHash := range block.Transactions {
		n.txPool.RemoveTransaction(txHash)
	}
}

// onChainReorganization 链重组回调
func (n *Node) onChainReorganization(oldBlocks []*types.Block, newBlocks []*types.Block) {
	fmt.Printf("Chain reorganization: reverting %d blocks, adding %d blocks\n", len(oldBlocks), len(newBlocks))

	// 将旧链的交易重新添加到交易池
	for _, block := range oldBlocks {
		for _, txHash := range block.Transactions {
			if tx, err := n.txStore.GetTransaction(txHash); err == nil {
				n.txPool.AddTransaction(tx)
			}
		}
	}

	// 从交易池中移除新链的交易
	for _, block := range newBlocks {
		for _, txHash := range block.Transactions {
			n.txPool.RemoveTransaction(txHash)
		}
	}
}

// onSyncComplete 同步完成回调
func (n *Node) onSyncComplete() {
	fmt.Printf("Blockchain synchronization completed\n")
	chainState := n.blockchain.GetChainState()
	fmt.Printf("Current height: %d, Best block: %s\n", chainState.Height, chainState.BestBlockHash.String())
}

// 公开接口

// GetBlockchain 获取区块链管理器
func (n *Node) GetBlockchain() *blockchain.Blockchain {
	return n.blockchain
}

// GetNetworkManager 获取网络管理器
func (n *Node) GetNetworkManager() *network.Network {
	return n.network
}

// GetBlockStore 获取区块存储
func (n *Node) GetBlockStore() storage.BlockStore {
	return n.blockStore
}

// GetAPIServer 获取API服务器
func (n *Node) GetAPIServer() *api.APIServer {
	return n.apiServer
}

// GetTxPool 获取交易池
func (n *Node) GetTxPool() *txpool.MemoryTxPool {
	return n.txPool
}

// GetValidator 获取验证器
func (n *Node) GetValidator() *validator.Validator {
	return n.validator
}

// IsRunning 检查节点是否运行中
func (n *Node) IsRunning() bool {
	return n.isRunning
}

// GetUptime 获取运行时间
func (n *Node) GetUptime() time.Duration {
	if !n.isRunning {
		return 0
	}
	return time.Since(n.startTime)
}

// GetNodeInfo 获取节点信息
func (n *Node) GetNodeInfo() map[string]interface{} {
	chainState := n.blockchain.GetChainState()
	chainHealth := n.blockchain.GetChainHealth()
	networkStatus := n.networkMonitor.GetNetworkStatus()

	return map[string]interface{}{
		"node_id":      n.network.GetLocalPeerID(),
		"version":      "1.0.0",
		"is_running":   n.isRunning,
		"uptime":       n.GetUptime().String(),
		"peer_count":   len(n.network.GetPeers()),
		"active_peers": len(n.network.GetPeers()),
		"listen_addr":  n.network.GetLocalAddresses(),
		"chain_height": chainState.Height,
		"best_block":   chainState.BestBlockHash.String(),
		"total_work":   chainState.TotalWork,
		"chain_health": chainHealth.Status,
		"is_syncing":   chainHealth.IsSyncing,
		"fork_count":   chainHealth.ForkCount,
		"network_status": map[string]interface{}{
			"connected_peers": networkStatus.ConnectedPeers,
			"is_partitioned":  networkStatus.IsPartitioned,
			"partition_since": networkStatus.PartitionSince,
			"reconnect_count": networkStatus.ReconnectCount,
			"last_update":     networkStatus.LastUpdateTime,
		},
	}
}

// BroadcastTransaction 广播交易
func (n *Node) BroadcastTransaction(tx *types.Transaction) error {
	// 验证交易
	if err := n.validator.ValidateTransaction(tx); err != nil {
		return fmt.Errorf("transaction validation failed: %v", err)
	}

	// 添加到交易池
	if err := n.txPool.AddTransaction(tx); err != nil {
		return fmt.Errorf("failed to add transaction to pool: %v", err)
	}

	// 序列化交易数据用于广播
	// TODO: 实现具体的交易序列化
	txData := []byte(tx.Hash().String())

	// 广播给其他节点
	return n.network.BroadcastMessage("transactions", txData)
}

// BroadcastBlock 广播区块
func (n *Node) BroadcastBlock(block *types.Block) error {
	// 通过区块链管理器添加区块
	if err := n.blockchain.AddBlock(block); err != nil {
		return fmt.Errorf("failed to add block to blockchain: %v", err)
	}

	// 序列化区块数据用于广播
	blockData, err := json.Marshal(block)
	if err != nil {
		return fmt.Errorf("failed to serialize block: %v", err)
	}

	fmt.Printf("📶 Broadcasting block %s (height: %d, size: %d bytes)\n",
		block.Hash().String(), block.Header.Number, len(blockData))

	// 广播给其他节点
	return n.network.BroadcastMessage("blocks", blockData)
}

// CreateBlock 创建新区块
func (n *Node) CreateBlock(validator types.Address) (*types.Block, error) {
	// 获取最佳区块
	bestBlock, err := n.blockchain.GetBestBlock()
	if err != nil {
		return nil, fmt.Errorf("failed to get best block: %v", err)
	}

	// 创建新区块
	newBlock := types.NewBlock(
		bestBlock.Header.Number+1,
		bestBlock.Hash(),
		validator,
	)

	// 添加交易池中的交易
	transactions := n.txPool.GetPendingTransactions()
	for _, tx := range transactions {
		newBlock.AddTransaction(tx.Hash())
	}

	return newBlock, nil
}

// GetChainStats 获取区块链统计信息
func (n *Node) GetChainStats() (*blockchain.ChainStats, error) {
	return n.blockchain.GetChainStats()
}

// GetChainHealth 获取区块链健康状况
func (n *Node) GetChainHealth() *blockchain.ChainHealth {
	return n.blockchain.GetChainHealth()
}

// GetChainState 获取区块链状态
func (n *Node) GetChainState() *blockchain.ChainState {
	return n.blockchain.GetChainState()
}

// StartSync 开始同步
func (n *Node) StartSync(peerID string, targetHeight uint64) error {
	return n.blockchain.StartSync(peerID, targetHeight)
}

// StopSync 停止同步
func (n *Node) StopSync() {
	n.blockchain.StopSync()
}

// blockProductionLoop 区块生产循环
func (n *Node) blockProductionLoop() {
	fmt.Printf("💰 Starting block production loop...\n")

	for n.isRunning {
		if !n.blockchain.GetConsensus().IsConsensusEnabled() {
			time.Sleep(time.Second)
			continue
		}

		// 获取当前时间的出块者
		now := time.Now()
		producer, slot, err := n.blockchain.GetConsensus().GetDPoS().GetCurrentProducerForTime(now)
		if err != nil {
			time.Sleep(time.Millisecond * 100)
			continue
		}

		// 检查是否该轮到这个验证者出块
		if n.blockchain.GetConsensus().IsMyTurnToProduce(producer) {
			// 获取交易池中的交易
			pendingTxs := n.txPool.GetPendingTransactions()
			txHashes := make([]types.Hash, 0, len(pendingTxs))
			for _, tx := range pendingTxs {
				txHashes = append(txHashes, tx.Hash())
			}

			// 生产区块
			block, err := n.blockchain.GetConsensus().ProduceBlock(producer, txHashes)
			if err != nil {
				fmt.Printf("⚠️  Failed to produce block: %v\n", err)
				time.Sleep(time.Millisecond * 100)
				continue
			}

			// 将区块添加到区块链
			// if err := n.blockchain.AddBlock(block); err != nil {
			// 	fmt.Printf("⚠️  Failed to add block to chain: %v\n", err)
			// } else {
			fmt.Printf("✅ Block produced successfully: %s (height: %d, slot: %d, txs: %d)\n",
				block.Hash().String()[:16], block.Header.Number, slot, len(block.Transactions))

			// 广播区块
			if err := n.BroadcastBlock(block); err != nil {
				fmt.Printf("⚠️  Failed to broadcast block: %v\n", err)
			}
			// }
		}

		// 等待一小段时间再检查
		time.Sleep(time.Millisecond * 500)
	}

	fmt.Printf("🛑 Block production loop stopped\n")
}
