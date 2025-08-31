package node

import (
	"context"
	"fmt"
	"time"

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

	// 状态管理
	isRunning bool
	startTime time.Time
}

// New creates a new blockchain node
func New(cfg *config.Config) (*Node, error) {
	// 创建存储层
	blockStore := storage.NewMemoryBlockStore()
	txStore := storage.NewMemoryTransactionStore()
	accountStore := storage.NewMemoryAccountStore()
	validatorStore := storage.NewMemoryValidatorStore()

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

	// 创建网络管理器
	networkConfig := &network.NetworkConfig{
		Port:           cfg.Network.Port,
		Host:           cfg.Network.Host,
		MaxPeers:       50, // 默认值
		BootstrapPeers: cfg.Network.BootstrapPeers,
		PrivateKeyPath: "", // 使用默认生成
	}
	networkManager, err := network.New(networkConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create network manager: %v", err)
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
	}

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

// Stop stops the node
func (n *Node) Stop() error {
	if !n.isRunning {
		return nil
	}

	fmt.Println("Stopping ShardMatrix node...")

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

// generateNodeID 生成节点ID
func generateNodeID() string {
	return fmt.Sprintf("node_%d", time.Now().UnixNano())
}

// 网络消息处理函数

// onBlockMessage 接收区块消息处理
func (n *Node) onBlockMessage(peerID string, msg network.NetMessage) error {
	fmt.Printf("Received block message from peer %s\n", peerID)

	// TODO: 反序列化区块数据并处理
	// 这里需要实现具体的区块反序列化和验证逻辑

	return nil
}

// onTransactionMessage 接收交易消息处理
func (n *Node) onTransactionMessage(peerID string, msg network.NetMessage) error {
	fmt.Printf("Received transaction message from peer %s\n", peerID)

	// TODO: 反序列化交易数据并处理
	// 这里需要实现具体的交易反序列化和验证逻辑

	return nil
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
	// TODO: 实现具体的区块序列化
	blockData := []byte(block.Hash().String())

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

// StartSync 开始同步
func (n *Node) StartSync(peerID string, targetHeight uint64) error {
	return n.blockchain.StartSync(peerID, targetHeight)
}

// StopSync 停止同步
func (n *Node) StopSync() {
	n.blockchain.StopSync()
}
