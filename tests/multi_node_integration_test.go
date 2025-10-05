package tests

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/lengzhao/shardmatrix/pkg/blockchain"
	"github.com/lengzhao/shardmatrix/pkg/consensus"
	"github.com/lengzhao/shardmatrix/pkg/crypto"
	"github.com/lengzhao/shardmatrix/pkg/monitoring"
	"github.com/lengzhao/shardmatrix/pkg/network"
	"github.com/lengzhao/shardmatrix/pkg/node"
	"github.com/lengzhao/shardmatrix/pkg/storage"
	"github.com/lengzhao/shardmatrix/pkg/types"
)

// MultiNodeTestNetwork 多节点测试网络
type MultiNodeTestNetwork struct {
	t           *testing.T
	testDir     string
	nodes       []*TestNode
	validators  []types.Validator
	genesisTime int64
	monitoring  *monitoring.MonitoringSystem
	isRunning   bool
	stopCh      chan struct{}
}

// TestNode 测试节点
type TestNode struct {
	ID            string
	Config        *node.Config
	Node          *node.Node
	KeyPair       *crypto.KeyPair
	Storage       storage.Storage
	Blockchain    *blockchain.Manager
	Consensus     *consensus.DPoSConsensus
	BlockProducer *blockchain.BlockProducer
	Validator     *blockchain.ThreeLayerValidator
	Synchronizer  *blockchain.BlockSynchronizer
	HealthMonitor *node.HealthMonitor
	Network       *network.Network
	Address       string
	IsValidator   bool
	IsRunning     bool
}

// NewMultiNodeTestNetwork 创建多节点测试网络
func NewMultiNodeTestNetwork(t *testing.T, nodeCount int) *MultiNodeTestNetwork {
	testDir := filepath.Join(os.TempDir(), fmt.Sprintf("shardmatrix_multinode_test_%d", time.Now().UnixNano()))
	err := os.MkdirAll(testDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	return &MultiNodeTestNetwork{
		t:           t,
		testDir:     testDir,
		nodes:       make([]*TestNode, 0, nodeCount),
		validators:  make([]types.Validator, 0, nodeCount),
		genesisTime: time.Now().Unix(),
		isRunning:   false,
		stopCh:      make(chan struct{}),
	}
}

// SetupNetwork 设置多节点网络
func (testNet *MultiNodeTestNetwork) SetupNetwork(nodeCount int, validatorCount int) error {
	if validatorCount > nodeCount {
		return fmt.Errorf("validator count (%d) cannot exceed node count (%d)", validatorCount, nodeCount)
	}

	testNet.t.Logf("🏗️ Setting up multi-node network with %d nodes (%d validators)", nodeCount, validatorCount)

	// 1. 创建监控系统
	testNet.monitoring = monitoring.NewMonitoringSystem()

	// 注册基本的指标收集器
	systemCollector := monitoring.NewSystemMetricsCollector()
	testNet.monitoring.RegisterCollector(systemCollector)

	err := testNet.monitoring.Start()
	if err != nil {
		return fmt.Errorf("failed to start monitoring system: %w", err)
	}

	// 2. 创建节点
	for i := 0; i < nodeCount; i++ {
		isValidator := i < validatorCount
		testNode, err := testNet.createTestNode(i, isValidator)
		if err != nil {
			return fmt.Errorf("failed to create test node %d: %w", i, err)
		}
		testNet.nodes = append(testNet.nodes, testNode)

		if isValidator {
			validator := types.Validator{
				Address:      testNode.KeyPair.Address,
				PublicKey:    testNode.KeyPair.PublicKey,
				StakeAmount:  1000000,
				DelegatedAmt: 0,
				VotePower:    1000000,
				IsActive:     true,
				SlashCount:   0,
			}
			testNet.validators = append(testNet.validators, validator)
		}
	}

	// 3. 初始化创世区块
	err = testNet.initializeGenesis()
	if err != nil {
		return fmt.Errorf("failed to initialize genesis: %w", err)
	}

	testNet.t.Logf("✓ Multi-node network setup completed with %d nodes", len(testNet.nodes))
	return nil
}

// createTestNode 创建测试节点
func (testNet *MultiNodeTestNetwork) createTestNode(index int, isValidator bool) (*TestNode, error) {
	nodeID := fmt.Sprintf("test-node-%d", index)

	// 生成密钥对
	keyPair, err := crypto.GenerateKeyPair()
	if err != nil {
		return nil, fmt.Errorf("failed to generate keypair: %w", err)
	}

	// 创建节点目录
	nodeDir := filepath.Join(testNet.testDir, nodeID)
	err = os.MkdirAll(nodeDir, 0755)
	if err != nil {
		return nil, fmt.Errorf("failed to create node directory: %w", err)
	}

	// 创建存储
	storageDir := filepath.Join(nodeDir, "storage")
	stor, err := storage.NewLevelDBStorage(storageDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create storage: %w", err)
	}

	// 创建区块链管理器
	blockchainMgr := blockchain.NewManager(stor)

	// 创建时间控制器
	timeController := consensus.NewTimeController(testNet.genesisTime)

	// 创建DPoS共识引擎
	consensusEngine := consensus.NewDPoSConsensus(stor, timeController, keyPair)

	// 创建三层验证器
	validator := blockchain.NewThreeLayerValidator(consensusEngine, blockchainMgr, timeController)

	// 创建区块同步器
	synchronizer := blockchain.NewBlockSynchronizer(stor, blockchainMgr, validator)

	// 创建健康监控器
	healthMonitor := node.NewHealthMonitor(len(testNet.nodes) + 1)

	// 创建网络层
	net := network.NewNetwork(nodeID, fmt.Sprintf("127.0.0.1:%d", 30000+index))

	// 创建交易池
	txPool := blockchain.NewTransactionPool(10000, 30*time.Minute)

	// 创建区块生产器
	blockProducer := blockchain.NewBlockProducer(keyPair, stor, txPool, blockchainMgr)
	blockProducer.SetNetworkChecker(consensusEngine)

	// 创建节点配置
	nodeConfig := &node.Config{
		DataDir:     nodeDir,
		ListenAddr:  fmt.Sprintf("127.0.0.1:%d", 40000+index),
		APIAddr:     fmt.Sprintf("127.0.0.1:%d", 50000+index),
		NodeID:      nodeID,
		IsValidator: isValidator,
		GenesisTime: testNet.genesisTime,
	}

	// 创建节点
	n, err := node.NewNode(nodeConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create node: %w", err)
	}

	testNode := &TestNode{
		ID:            nodeID,
		Config:        nodeConfig,
		Node:          n,
		KeyPair:       keyPair,
		Storage:       stor,
		Blockchain:    blockchainMgr,
		Consensus:     consensusEngine,
		BlockProducer: blockProducer,
		Validator:     validator,
		Synchronizer:  synchronizer,
		HealthMonitor: healthMonitor,
		Network:       net,
		Address:       nodeConfig.ListenAddr,
		IsValidator:   isValidator,
		IsRunning:     false,
	}

	return testNode, nil
}

// initializeGenesis 初始化创世区块
func (testNet *MultiNodeTestNetwork) initializeGenesis() error {
	// 创建初始账户
	initAccounts := make([]types.Account, len(testNet.nodes))
	for i, node := range testNet.nodes {
		initAccounts[i] = types.Account{
			Address: node.KeyPair.Address,
			Balance: 1000000000,
			Nonce:   0,
			Staked:  1000000,
		}
	}

	// 创建创世区块配置
	genesisConfig := &types.GenesisBlock{
		Timestamp:    testNet.genesisTime,
		InitAccounts: initAccounts,
		Validators:   testNet.validators,
		ChainID:      "shardmatrix-test-multinode",
		Config: map[string]string{
			"consensus":       "dpos",
			"version":         "1.0.0",
			"validator_count": fmt.Sprintf("%d", len(testNet.validators)),
			"node_count":      fmt.Sprintf("%d", len(testNet.nodes)),
		},
	}

	// 为所有节点初始化相同的创世区块
	for i, node := range testNet.nodes {
		err := node.Blockchain.Initialize(genesisConfig)
		if err != nil {
			return fmt.Errorf("failed to initialize genesis for node %d: %w", i, err)
		}
	}

	return nil
}

// StartNetwork 启动网络
func (testNet *MultiNodeTestNetwork) StartNetwork() error {
	testNet.t.Log("🚀 Starting multi-node network...")

	// 启动所有节点
	for i, node := range testNet.nodes {
		err := testNet.startNode(i, node)
		if err != nil {
			return fmt.Errorf("failed to start node %d: %w", i, err)
		}
	}

	testNet.isRunning = true
	testNet.t.Logf("✓ All %d nodes started successfully", len(testNet.nodes))

	// 等待网络稳定
	time.Sleep(2 * time.Second)
	return nil
}

// startNode 启动单个节点
func (testNet *MultiNodeTestNetwork) startNode(index int, node *TestNode) error {
	// 1. 启动网络层
	err := node.Network.Start()
	if err != nil {
		return fmt.Errorf("failed to start network: %w", err)
	}

	// 2. 启动共识引擎
	err = node.Consensus.Start()
	if err != nil {
		return fmt.Errorf("failed to start consensus: %w", err)
	}

	// 3. 启动区块生产器（仅验证者节点）
	if node.IsValidator {
		err = node.BlockProducer.Start()
		if err != nil {
			return fmt.Errorf("failed to start block producer: %w", err)
		}
	}

	// 4. 启动节点
	err = node.Node.Start()
	if err != nil {
		return fmt.Errorf("failed to start node: %w", err)
	}

	node.IsRunning = true
	return nil
}

// StopNetwork 停止网络
func (testNet *MultiNodeTestNetwork) StopNetwork() {
	if !testNet.isRunning {
		return
	}

	testNet.t.Log("🛑 Stopping multi-node network...")

	// 发送停止信号
	close(testNet.stopCh)
	testNet.isRunning = false

	// 停止所有节点
	for i, node := range testNet.nodes {
		err := testNet.stopNode(i, node)
		if err != nil {
			testNet.t.Logf("Warning: Failed to stop node %d: %v", i, err)
		}
	}

	// 停止监控系统
	if testNet.monitoring != nil {
		testNet.monitoring.Stop()
	}

	testNet.t.Logf("✓ Multi-node network stopped")
}

// stopNode 停止单个节点
func (testNet *MultiNodeTestNetwork) stopNode(index int, node *TestNode) error {
	if !node.IsRunning {
		return nil
	}

	// 停止节点
	if node.Node != nil {
		node.Node.Stop()
	}

	// 停止区块生产器
	if node.BlockProducer != nil && node.IsValidator {
		node.BlockProducer.Stop()
	}

	// 停止共识引擎
	if node.Consensus != nil {
		node.Consensus.Stop()
	}

	// 停止网络层
	if node.Network != nil {
		node.Network.Stop()
	}

	// 关闭存储
	if node.Storage != nil {
		node.Storage.Close()
	}

	node.IsRunning = false
	return nil
}

// Cleanup 清理测试环境
func (testNet *MultiNodeTestNetwork) Cleanup() {
	testNet.StopNetwork()

	if testNet.testDir != "" {
		os.RemoveAll(testNet.testDir)
	}

	testNet.t.Log("✓ Test environment cleaned up")
}

// GetValidatorNodes 获取验证者节点
func (testNet *MultiNodeTestNetwork) GetValidatorNodes() []*TestNode {
	var validators []*TestNode
	for _, node := range testNet.nodes {
		if node.IsValidator {
			validators = append(validators, node)
		}
	}
	return validators
}

// =============== 多节点集成测试用例 ===============

// TestMultiNodeConsensus 测试多节点共识
func TestMultiNodeConsensus(t *testing.T) {
	// 创建3个节点的网络，所有都是验证者
	testNet := NewMultiNodeTestNetwork(t, 3)
	defer testNet.Cleanup()

	// 设置网络
	err := testNet.SetupNetwork(3, 3)
	if err != nil {
		t.Fatalf("Failed to setup network: %v", err)
	}

	// 启动网络
	err = testNet.StartNetwork()
	if err != nil {
		t.Fatalf("Failed to start network: %v", err)
	}

	// 测试共识过程
	err = testConsensusProcess(testNet, t)
	if err != nil {
		t.Fatalf("Consensus test failed: %v", err)
	}

	t.Log("✓ Multi-node consensus test completed successfully")
}

// TestMultiNodeMonitoring 测试多节点监控
func TestMultiNodeMonitoring(t *testing.T) {
	// 创建2个节点的网络
	testNet := NewMultiNodeTestNetwork(t, 2)
	defer testNet.Cleanup()

	// 设置网络
	err := testNet.SetupNetwork(2, 2)
	if err != nil {
		t.Fatalf("Failed to setup network: %v", err)
	}

	// 启动网络
	err = testNet.StartNetwork()
	if err != nil {
		t.Fatalf("Failed to start network: %v", err)
	}

	// 测试监控系统
	err = testMonitoringSystem(testNet, t)
	if err != nil {
		t.Fatalf("Monitoring system test failed: %v", err)
	}

	t.Log("✓ Multi-node monitoring test completed successfully")
}

// testConsensusProcess 测试共识过程
func testConsensusProcess(testNet *MultiNodeTestNetwork, t *testing.T) error {
	t.Log("🔄 Testing consensus process...")

	validators := testNet.GetValidatorNodes()
	if len(validators) == 0 {
		return fmt.Errorf("no validator nodes found")
	}

	// 等待初始同步
	time.Sleep(5 * time.Second)

	// 获取初始高度
	initialHeight := uint64(0)

	// 等待区块生产
	t.Log("⏳ Waiting for block production...")
	timeout := time.After(60 * time.Second)   // 增加超时时间至60秒
	ticker := time.NewTicker(3 * time.Second) // 增加检查间隔至3秒
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			return fmt.Errorf("timeout waiting for blocks to be produced")
		case <-ticker.C:
			// 检查所有验证者节点的区块高度
			heights := make([]uint64, len(validators))
			var maxHeight uint64

			for i, validator := range validators {
				height, err := validator.Storage.GetLatestBlockHeight()
				if err != nil {
					t.Logf("Warning: Failed to get height from validator %d: %v", i, err)
					continue
				}
				heights[i] = height
				if height > maxHeight {
					maxHeight = height
				}
			}

			// 检查同步状态（允许高度差据在可接受范围内）
			syncedCount := 0
			for i, height := range heights {
				if maxHeight == 0 || maxHeight-height <= 2 { // 允许最多2个区块的差距
					syncedCount++
				} else {
					t.Logf("Validator %d at height %d, max height %d", i, height, maxHeight)
				}
			}

			// 如果有进展且大多数节点同步，认为成功
			if maxHeight > initialHeight && syncedCount >= len(validators)*2/3 {
				t.Logf("✓ Majority validators synchronized at height around %d (synced: %d/%d)",
					maxHeight, syncedCount, len(validators))
				return nil
			}

			if maxHeight > initialHeight {
				t.Logf("📈 Block height progressed to %d, synced nodes: %d/%d",
					maxHeight, syncedCount, len(validators))
			}
		}
	}
}

// testMonitoringSystem 测试监控系统
func testMonitoringSystem(testNet *MultiNodeTestNetwork, t *testing.T) error {
	t.Log("📊 Testing monitoring system...")

	if testNet.monitoring == nil {
		return fmt.Errorf("monitoring system not initialized")
	}

	// 等待监控系统收集数据
	time.Sleep(15 * time.Second) // 增加等待时间让监控系统有足够时间收集数据

	// 获取系统诊断
	diagnostics := testNet.monitoring.GetSystemDiagnostics()
	if diagnostics == nil {
		return fmt.Errorf("failed to get system diagnostics")
	}

	t.Logf("📈 System diagnostics:")
	t.Logf("  Overall health: %v", diagnostics.OverallHealth)
	t.Logf("  Active alerts: %d", len(diagnostics.ActiveAlerts))
	t.Logf("  Monitored components: %d", len(diagnostics.Components))
	t.Logf("  Collected metrics: %d", len(diagnostics.Metrics))

	// 检查组件健康状态
	for name, health := range diagnostics.Components {
		t.Logf("  Component %s: %v (checks: %d, fails: %d)",
			name, health.Status, health.CheckCount, health.FailCount)
	}

	// 验证至少有一些指标被收集
	if len(diagnostics.Metrics) == 0 {
		return fmt.Errorf("no metrics collected by monitoring system")
	}

	t.Log("✓ Monitoring system test completed")
	return nil
}
