package node

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/lengzhao/shardmatrix/pkg/api"
	"github.com/lengzhao/shardmatrix/pkg/blockchain"
	"github.com/lengzhao/shardmatrix/pkg/consensus"
	"github.com/lengzhao/shardmatrix/pkg/crypto"
	"github.com/lengzhao/shardmatrix/pkg/network"
	"github.com/lengzhao/shardmatrix/pkg/storage"
	"github.com/lengzhao/shardmatrix/pkg/types"
)

// Config 节点配置
type Config struct {
	DataDir     string `json:"data_dir"`
	ListenAddr  string `json:"listen_addr"`
	APIAddr     string `json:"api_addr"`
	NodeID      string `json:"node_id"`
	IsValidator bool   `json:"is_validator"`
	GenesisTime int64  `json:"genesis_time"`
}

// Node 区块链节点
type Node struct {
	mu             sync.RWMutex
	config         *Config
	keyPair        *crypto.KeyPair
	
	// 核心组件
	timeController *consensus.TimeController
	blockProducer  *blockchain.BlockProducer
	txPool         *blockchain.TransactionPool
	blockchainMgr  *blockchain.Manager
	consensus      *consensus.DPoSConsensus
	network        *network.Network
	apiServer      *api.Server
	
	// 运行状态
	isRunning      bool
	isValidator    bool
	stopCh         chan struct{}
}

// NewNode 创建新节点
func NewNode(config *Config) (*Node, error) {
	node := &Node{
		config:  config,
		stopCh:  make(chan struct{}),
	}
	
	// 加载或生成密钥对
	err := node.loadOrGenerateKeyPair()
	if err != nil {
		return nil, fmt.Errorf("failed to load keypair: %w", err)
	}
	
	// 初始化组件
	err = node.initializeComponents()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize components: %w", err)
	}
	
	return node, nil
}

// loadOrGenerateKeyPair 加载或生成密钥对
func (n *Node) loadOrGenerateKeyPair() error {
	keyFile := filepath.Join(n.config.DataDir, "validator.key")
	
	// 检查密钥文件是否存在
	if _, err := os.Stat(keyFile); os.IsNotExist(err) {
		// 生成新密钥对
		keyPair, err := crypto.GenerateKeyPair()
		if err != nil {
			return fmt.Errorf("failed to generate keypair: %w", err)
		}
		
		// 保存密钥到文件
		err = os.MkdirAll(n.config.DataDir, 0755)
		if err != nil {
			return fmt.Errorf("failed to create data directory: %w", err)
		}
		
		err = os.WriteFile(keyFile, []byte(keyPair.PrivateKeyHex()), 0600)
		if err != nil {
			return fmt.Errorf("failed to save keypair: %w", err)
		}
		
		n.keyPair = keyPair
	} else {
		// 加载现有密钥
		keyData, err := os.ReadFile(keyFile)
		if err != nil {
			return fmt.Errorf("failed to read keypair: %w", err)
		}
		
		keyPair, err := crypto.NewKeyPairFromHex(string(keyData))
		if err != nil {
			return fmt.Errorf("failed to parse keypair: %w", err)
		}
		
		n.keyPair = keyPair
	}
	
	return nil
}

// initializeComponents 初始化所有组件
func (n *Node) initializeComponents() error {
	// 创建存储（简化实现，使用模拟存储）
	mockStorage := storage.NewMockStorage()
	
	// 创建区块链管理器
	n.blockchainMgr = blockchain.NewManager(mockStorage)
	
	// 创建时间控制器
	n.timeController = consensus.NewTimeController(n.config.GenesisTime)
	
	// 创建交易池
	n.txPool = blockchain.NewTransactionPool(10000, 5*time.Minute)
	
	// 创建网络管理器
	n.network = network.NewNetwork(n.config.NodeID, n.config.ListenAddr)
	
	// 注册网络消息处理器
	n.setupNetworkHandlers()
	
	// 注册时间控制器回调
	n.setupTimeControllerCallbacks()
	
	fmt.Printf("Node initialized successfully\n")
	fmt.Printf("Node ID: %s\n", n.config.NodeID)
	fmt.Printf("Validator Address: %s\n", n.keyPair.Address.String())
	
	return nil
}

// setupNetworkHandlers 设置网络消息处理器
func (n *Node) setupNetworkHandlers() {
	n.network.RegisterHandler(network.MsgNewBlock, n.handleNewBlock)
	n.network.RegisterHandler(network.MsgNewTransaction, n.handleNewTransaction)
	n.network.RegisterHandler(network.MsgHeartbeat, n.handleHeartbeat)
}

// setupTimeControllerCallbacks 设置时间控制器回调
func (n *Node) setupTimeControllerCallbacks() {
	n.timeController.RegisterCallback("block_producer", n.onBlockTime)
}

// Start 启动节点
func (n *Node) Start() error {
	n.mu.Lock()
	defer n.mu.Unlock()
	
	if n.isRunning {
		return fmt.Errorf("node already running")
	}
	
	fmt.Println("Starting ShardMatrix node...")
	
	// 启动交易池
	err := n.txPool.Start()
	if err != nil {
		return fmt.Errorf("failed to start transaction pool: %w", err)
	}
	fmt.Println("✓ Transaction pool started")
	
	// 启动网络服务
	err = n.network.Start()
	if err != nil {
		return fmt.Errorf("failed to start network: %w", err)
	}
	fmt.Printf("✓ Network started on %s\n", n.config.ListenAddr)
	
	// 启动时间控制器
	err = n.timeController.Start()
	if err != nil {
		return fmt.Errorf("failed to start time controller: %w", err)
	}
	fmt.Println("✓ Time controller started")
	
	// 如果是验证者，启动区块生产
	if n.config.IsValidator {
		n.isValidator = true
		fmt.Println("✓ Validator mode enabled")
	}
	
	// 启动API服务器
	if n.apiServer != nil {
		err = n.apiServer.Start(n.config.APIAddr)
		if err != nil {
			return fmt.Errorf("failed to start API server: %w", err)
		}
		fmt.Printf("✓ API server started on %s\n", n.config.APIAddr)
	}
	
	n.isRunning = true
	n.stopCh = make(chan struct{})
	
	// 启动主循环
	go n.mainLoop()
	
	fmt.Println("🚀 ShardMatrix node started successfully!")
	return nil
}

// Stop 停止节点
func (n *Node) Stop() error {
	n.mu.Lock()
	defer n.mu.Unlock()
	
	if !n.isRunning {
		return nil
	}
	
	fmt.Println("Stopping ShardMatrix node...")
	
	n.isRunning = false
	close(n.stopCh)
	
	// 停止各个组件
	if n.apiServer != nil {
		n.apiServer.Stop()
	}
	
	if n.timeController != nil {
		n.timeController.Stop()
	}
	
	if n.network != nil {
		n.network.Stop()
	}
	
	if n.txPool != nil {
		n.txPool.Stop()
	}
	
	fmt.Println("✓ ShardMatrix node stopped")
	return nil
}

// mainLoop 主运行循环
func (n *Node) mainLoop() {
	heartbeatTicker := time.NewTicker(30 * time.Second)
	defer heartbeatTicker.Stop()
	
	statusTicker := time.NewTicker(10 * time.Second)
	defer statusTicker.Stop()
	
	for {
		select {
		case <-n.stopCh:
			return
		case <-heartbeatTicker.C:
			n.sendHeartbeat()
		case <-statusTicker.C:
			n.logStatus()
		}
	}
}

// onBlockTime 区块时间回调
func (n *Node) onBlockTime(blockTime int64, blockNumber uint64) {
	if !n.isValidator {
		return
	}
	
	fmt.Printf("⏰ Block time reached: %d (block %d)\n", blockTime, blockNumber)
	
	// 生成空区块（简化实现）
	n.produceEmptyBlock(blockTime, blockNumber)
}

// produceEmptyBlock 生产空区块
func (n *Node) produceEmptyBlock(blockTime int64, blockNumber uint64) {
	// 创建空区块
	block := &types.Block{
		Header: types.BlockHeader{
			Number:         blockNumber,
			Timestamp:      blockTime,
			PrevHash:       crypto.RandomHash(), // 简化实现
			TxRoot:         types.EmptyTxRoot(),
			StateRoot:      crypto.RandomHash(), // 简化实现
			Validator:      n.keyPair.Address,
			ShardID:        types.ShardID,
			AdjacentHashes: [3]types.Hash{},
		},
		Transactions: []types.Hash{},
	}
	
	// 签名区块
	err := n.keyPair.SignBlock(&block.Header)
	if err != nil {
		fmt.Printf("Failed to sign block: %v\n", err)
		return
	}
	
	fmt.Printf("📦 Produced empty block %d at %s\n", 
		blockNumber, time.Unix(blockTime, 0).Format("15:04:05"))
	
	// 广播区块
	if n.network != nil {
		n.network.BroadcastBlock(block)
	}
}

// 网络消息处理器

func (n *Node) handleNewBlock(msg *network.Message, peer *network.Peer) error {
	fmt.Printf("📨 Received new block from %s\n", peer.ID)
	// 这里应该验证和处理区块
	return nil
}

func (n *Node) handleNewTransaction(msg *network.Message, peer *network.Peer) error {
	fmt.Printf("📨 Received new transaction from %s\n", peer.ID)
	// 这里应该验证和添加交易到池中
	return nil
}

func (n *Node) handleHeartbeat(msg *network.Message, peer *network.Peer) error {
	// 处理心跳消息
	return nil
}

// sendHeartbeat 发送心跳
func (n *Node) sendHeartbeat() {
	if n.network != nil {
		n.network.SendHeartbeat()
	}
}

// logStatus 记录状态
func (n *Node) logStatus() {
	stats := n.GetStats()
	fmt.Printf("📊 Node Status - Block: %v, Peers: %v, TxPool: %v\n",
		stats["current_block"], stats["peer_count"], stats["tx_pool_size"])
}

// GetStats 获取节点统计信息
func (n *Node) GetStats() map[string]interface{} {
	n.mu.RLock()
	defer n.mu.RUnlock()
	
	stats := map[string]interface{}{
		"node_id":       n.config.NodeID,
		"is_running":    n.isRunning,
		"is_validator":  n.isValidator,
		"validator_addr": n.keyPair.Address.String(),
		"data_dir":      n.config.DataDir,
		"listen_addr":   n.config.ListenAddr,
		"api_addr":      n.config.APIAddr,
	}
	
	if n.timeController != nil {
		timeStats := n.timeController.GetTimeStats()
		for k, v := range timeStats {
			stats["time_"+k] = v
		}
	}
	
	if n.network != nil {
		stats["peer_count"] = n.network.GetPeerCount()
	} else {
		stats["peer_count"] = 0
	}
	
	if n.txPool != nil {
		stats["tx_pool_size"] = n.txPool.GetTransactionCount()
	} else {
		stats["tx_pool_size"] = 0
	}
	
	stats["current_block"] = "genesis" // 简化实现
	
	return stats
}

// IsRunning 检查节点是否正在运行
func (n *Node) IsRunning() bool {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.isRunning
}

// GetAddress 获取验证者地址
func (n *Node) GetAddress() types.Address {
	return n.keyPair.Address
}

// ConnectToPeer 连接到其他节点
func (n *Node) ConnectToPeer(address string) error {
	if n.network == nil {
		return fmt.Errorf("network not initialized")
	}
	
	return n.network.ConnectToPeer(address)
}

// SubmitTransaction 提交交易
func (n *Node) SubmitTransaction(tx *types.Transaction) error {
	if n.txPool == nil {
		return fmt.Errorf("transaction pool not initialized")
	}
	
	// 添加到交易池
	err := n.txPool.AddTransaction(tx)
	if err != nil {
		return fmt.Errorf("failed to add transaction: %w", err)
	}
	
	// 广播交易
	if n.network != nil {
		n.network.BroadcastTransaction(tx)
	}
	
	return nil
}

// SetAPIServer 设置API服务器
func (n *Node) SetAPIServer(server *api.Server) {
	n.apiServer = server
}

// InitializeBlockchain 初始化区块链
func (n *Node) InitializeBlockchain(genesisConfig *types.GenesisBlock) error {
	n.mu.Lock()
	defer n.mu.Unlock()
	
	if n.blockchainMgr == nil {
		return fmt.Errorf("blockchain manager not initialized")
	}
	
	err := n.blockchainMgr.Initialize(genesisConfig)
	if err != nil {
		return fmt.Errorf("failed to initialize blockchain: %w", err)
	}
	
	// 创建区块生产器（如果是验证者）
	if n.config.IsValidator && n.blockProducer == nil {
		n.blockProducer = blockchain.NewBlockProducer(n.keyPair, nil, n.txPool, n.blockchainMgr)
	}
	
	return nil
}

// GetBlockchainManager 获取区块链管理器
func (n *Node) GetBlockchainManager() *blockchain.Manager {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.blockchainMgr
}

// GetTransactionPool 获取交易池
func (n *Node) GetTransactionPool() *blockchain.TransactionPool {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.txPool
}

// GetValidatorAddress 获取验证者地址
func (n *Node) GetValidatorAddress() types.Address {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.keyPair.Address
}