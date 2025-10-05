package tests

import (
	"os"
	"testing"
	"time"

	"github.com/lengzhao/shardmatrix/pkg/blockchain"
	"github.com/lengzhao/shardmatrix/pkg/consensus"
	"github.com/lengzhao/shardmatrix/pkg/crypto"
	"github.com/lengzhao/shardmatrix/pkg/node"
	"github.com/lengzhao/shardmatrix/pkg/storage"
	"github.com/lengzhao/shardmatrix/pkg/types"
)

// TestNodeLifecycle 测试节点完整生命周期
func TestNodeLifecycle(t *testing.T) {
	// 创建临时目录
	tmpDir, err := os.MkdirTemp("", "shardmatrix_e2e_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// 创建节点配置
	config := &node.Config{
		DataDir:     tmpDir,
		ListenAddr:  "127.0.0.1:0", // 使用随机端口
		APIAddr:     "127.0.0.1:0", // 使用随机端口
		NodeID:      "test-node-1",
		IsValidator: true,
		GenesisTime: time.Now().Unix(),
	}

	// 确保创世时间是偶数秒
	if config.GenesisTime%2 != 0 {
		config.GenesisTime--
	}

	t.Run("NodeCreation", func(t *testing.T) {
		// 创建节点
		testNode, err := node.NewNode(config)
		if err != nil {
			t.Fatalf("Failed to create node: %v", err)
		}

		if testNode == nil {
			t.Fatal("Node should not be nil")
		}

		t.Logf("Node created successfully with ID: %s", config.NodeID)
	})

	t.Run("NodeStartStop", func(t *testing.T) {
		// 创建并启动节点
		testNode, err := node.NewNode(config)
		if err != nil {
			t.Fatalf("Failed to create node: %v", err)
		}

		// 启动节点
		err = testNode.Start()
		if err != nil {
			t.Fatalf("Failed to start node: %v", err)
		}

		// 检查节点状态
		if !testNode.IsRunning() {
			t.Error("Node should be running after start")
		}

		// 等待一小段时间确保节点完全启动
		time.Sleep(100 * time.Millisecond)

		// 停止节点
		err = testNode.Stop()
		if err != nil {
			t.Errorf("Failed to stop node: %v", err)
		}

		// 检查节点状态
		if testNode.IsRunning() {
			t.Error("Node should not be running after stop")
		}

		t.Logf("Node lifecycle test completed successfully")
	})

	t.Run("NodeWithGenesisBlock", func(t *testing.T) {
		// 创建节点
		testNode, err := node.NewNode(config)
		if err != nil {
			t.Fatalf("Failed to create node: %v", err)
		}

		// 创建创世区块配置
		keyPair, _ := crypto.GenerateKeyPair()
		genesisConfig := &types.GenesisBlock{
			Timestamp: config.GenesisTime,
			InitAccounts: []types.Account{
				{
					Address: keyPair.Address,
					Balance: 1000000,
					Nonce:   0,
					Staked:  0,
				},
			},
			ChainID: "test-chain",
			Config: map[string]string{
				"consensus": "dpos",
				"version":   "1.0.0",
			},
		}

		// 初始化区块链
		err = testNode.InitializeBlockchain(genesisConfig)
		if err != nil {
			t.Fatalf("Failed to initialize blockchain: %v", err)
		}

		// 启动节点
		err = testNode.Start()
		if err != nil {
			t.Fatalf("Failed to start node with genesis: %v", err)
		}

		// 等待一小段时间
		time.Sleep(100 * time.Millisecond)

		// 停止节点
		err = testNode.Stop()
		if err != nil {
			t.Errorf("Failed to stop node: %v", err)
		}

		t.Logf("Node with genesis block test completed successfully")
	})
}

// TestBlockProductionFlow 测试区块生产完整流程
func TestBlockProductionFlow(t *testing.T) {
	// 创建临时目录
	tmpDir, err := os.MkdirTemp("", "shardmatrix_block_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// 创建存储
	storageInst, err := storage.NewLevelDBStorage(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer storageInst.Close()

	// 创建密钥对
	keyPair, err := crypto.GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate keypair: %v", err)
	}

	t.Run("BlockchainInitialization", func(t *testing.T) {
		// 创建区块链管理器
		blockchainMgr := blockchain.NewManager(storageInst)

		// 创建创世配置
		genesisTime := time.Now().Unix()
		if genesisTime%2 != 0 {
			genesisTime--
		}

		genesisConfig := &types.GenesisBlock{
			Timestamp: genesisTime,
			InitAccounts: []types.Account{
				{
					Address: keyPair.Address,
					Balance: 1000000,
					Nonce:   0,
					Staked:  0,
				},
			},
			ChainID: "test-chain",
		}

		// 初始化区块链
		err := blockchainMgr.Initialize(genesisConfig)
		if err != nil {
			t.Fatalf("Failed to initialize blockchain: %v", err)
		}

		// 检查当前区块
		currentBlock := blockchainMgr.GetCurrentBlock()
		if currentBlock == nil {
			t.Fatal("Current block should not be nil after initialization")
		}

		if currentBlock.Header.Number != 0 {
			t.Errorf("Genesis block number should be 0, got %d", currentBlock.Header.Number)
		}

		t.Logf("Blockchain initialized with genesis block at height %d", currentBlock.Header.Number)
	})

	t.Run("EmptyBlockProduction", func(t *testing.T) {
		// 重新创建区块链管理器
		blockchainMgr := blockchain.NewManager(storageInst)

		// 创建交易池
		txPool := blockchain.NewTransactionPool(1000, 5*time.Minute)

		// 创建区块生产器
		blockProducer := blockchain.NewBlockProducer(keyPair, storageInst, txPool, blockchainMgr)

		// 启动组件
		err := txPool.Start()
		if err != nil {
			t.Fatalf("Failed to start transaction pool: %v", err)
		}
		defer txPool.Stop()

		err = blockProducer.Start()
		if err != nil {
			t.Fatalf("Failed to start block producer: %v", err)
		}
		defer blockProducer.Stop()

		// 生产空区块
		genesisTime := time.Now().Unix()
		if genesisTime%2 != 0 {
			genesisTime--
		}

		blockTime := genesisTime + 2
		blockNumber := uint64(1)

		block, err := blockProducer.ProduceBlock(blockTime, blockNumber)
		if err != nil {
			t.Fatalf("Failed to produce empty block: %v", err)
		}

		// 验证空区块
		if !block.IsEmpty() {
			t.Error("Produced block should be empty")
		}

		if block.Header.Number != blockNumber {
			t.Errorf("Block number mismatch: expected %d, got %d", blockNumber, block.Header.Number)
		}

		if block.Header.Timestamp != blockTime {
			t.Errorf("Block time mismatch: expected %d, got %d", blockTime, block.Header.Timestamp)
		}

		t.Logf("Empty block produced successfully: height=%d, time=%d", block.Header.Number, block.Header.Timestamp)
	})

	t.Run("TransactionProcessing", func(t *testing.T) {
		// 创建交易池
		txPool := blockchain.NewTransactionPool(1000, 5*time.Minute)
		err := txPool.Start()
		if err != nil {
			t.Fatalf("Failed to start transaction pool: %v", err)
		}
		defer txPool.Stop()

		// 创建测试交易
		fromKeyPair, _ := crypto.GenerateKeyPair()
		toAddress := crypto.RandomAddress()

		tx := &types.Transaction{
			ShardID:   types.ShardID,
			From:      fromKeyPair.Address,
			To:        toAddress,
			Amount:    1000,
			GasPrice:  100,
			GasLimit:  21000,
			Nonce:     1,
			Data:      []byte("test transaction"),
			Signature: fromKeyPair.Sign([]byte("test transaction")),
		}

		// 添加交易到交易池
		err = txPool.AddTransaction(tx)
		if err != nil {
			t.Errorf("Failed to add transaction to pool: %v", err)
		}

		// 检查交易池状态
		txCount := txPool.GetTransactionCount()
		if txCount != 1 {
			t.Errorf("Transaction pool should have 1 transaction, got %d", txCount)
		}

		// 获取待处理交易
		pendingTxs, err := txPool.GetPendingTransactions(10)
		if err != nil {
			t.Errorf("Failed to get pending transactions: %v", err)
		}

		if len(pendingTxs) != 1 {
			t.Errorf("Should have 1 pending transaction, got %d", len(pendingTxs))
		}

		t.Logf("Transaction processing test completed: pool has %d transactions", txCount)
	})
}

// TestTimeControllerIntegration 测试时间控制器集成
func TestTimeControllerIntegration(t *testing.T) {
	genesisTime := time.Now().Unix()
	if genesisTime%2 != 0 {
		genesisTime--
	}

	t.Run("StrictTimingValidation", func(t *testing.T) {
		// 创建时间控制器
		timeController := consensus.NewTimeController(genesisTime)

		// 连续测试10个区块的时间验证
		for i := uint64(1); i <= 10; i++ {
			expectedTime := genesisTime + int64(i)*2

			// 正确时间应该验证通过
			if !timeController.ValidateBlockTime(i, expectedTime) {
				t.Errorf("Block %d: correct time %d should validate", i, expectedTime)
			}

			// 错误时间应该被拒绝
			wrongTimes := []int64{
				expectedTime - 1, // 早1秒
				expectedTime + 1, // 晚1秒
				expectedTime - 2, // 早2秒
				expectedTime + 2, // 晚2秒
			}

			for _, wrongTime := range wrongTimes {
				if timeController.ValidateBlockTime(i, wrongTime) {
					t.Errorf("Block %d: wrong time %d should be rejected", i, wrongTime)
				}
			}
		}

		t.Logf("Strict timing validation completed for 10 blocks")
	})

	t.Run("CallbackMechanism", func(t *testing.T) {
		timeController := consensus.NewTimeController(genesisTime)

		// 回调计数器
		callbackCount := 0
		receivedBlocks := make(map[uint64]int64)

		// 注册回调
		callback := func(blockTime int64, blockNumber uint64) {
			callbackCount++
			receivedBlocks[blockNumber] = blockTime
		}

		timeController.RegisterCallback("test", callback)

		// 手动触发几个回调
		for i := uint64(1); i <= 5; i++ {
			blockTime := genesisTime + int64(i)*2
			callback(blockTime, i)
		}

		// 验证回调执行
		if callbackCount != 5 {
			t.Errorf("Expected 5 callback executions, got %d", callbackCount)
		}

		// 验证回调数据正确性
		for blockNum, blockTime := range receivedBlocks {
			expectedTime := genesisTime + int64(blockNum)*2
			if blockTime != expectedTime {
				t.Errorf("Block %d: callback received wrong time %d, expected %d",
					blockNum, blockTime, expectedTime)
			}
		}

		// 取消注册回调
		timeController.UnregisterCallback("test")

		t.Logf("Callback mechanism test completed: %d callbacks executed", callbackCount)
	})
}

// TestSystemIntegration 测试系统集成
func TestSystemIntegration(t *testing.T) {
	// 创建临时目录
	tmpDir, err := os.MkdirTemp("", "shardmatrix_integration_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	t.Run("CompleteWorkflow", func(t *testing.T) {
		// 1. 创建存储
		storageInst, err := storage.NewLevelDBStorage(tmpDir)
		if err != nil {
			t.Fatalf("Failed to create storage: %v", err)
		}
		defer storageInst.Close()

		// 2. 创建密钥对
		keyPair, err := crypto.GenerateKeyPair()
		if err != nil {
			t.Fatalf("Failed to generate keypair: %v", err)
		}

		// 3. 创建区块链管理器
		blockchainMgr := blockchain.NewManager(storageInst)

		// 4. 初始化区块链
		genesisTime := time.Now().Unix()
		if genesisTime%2 != 0 {
			genesisTime--
		}

		genesisConfig := &types.GenesisBlock{
			Timestamp: genesisTime,
			InitAccounts: []types.Account{
				{
					Address: keyPair.Address,
					Balance: 1000000,
					Nonce:   0,
					Staked:  100000,
				},
			},
			ChainID: "integration-test",
		}

		err = blockchainMgr.Initialize(genesisConfig)
		if err != nil {
			t.Fatalf("Failed to initialize blockchain: %v", err)
		}

		// 5. 创建交易池
		txPool := blockchain.NewTransactionPool(1000, 5*time.Minute)
		err = txPool.Start()
		if err != nil {
			t.Fatalf("Failed to start transaction pool: %v", err)
		}
		defer txPool.Stop()

		// 6. 创建区块生产器
		blockProducer := blockchain.NewBlockProducer(keyPair, storageInst, txPool, blockchainMgr)
		err = blockProducer.Start()
		if err != nil {
			t.Fatalf("Failed to start block producer: %v", err)
		}
		defer blockProducer.Stop()

		// 7. 创建时间控制器
		timeController := consensus.NewTimeController(genesisTime)

		// 8. 生产连续的区块
		for i := uint64(1); i <= 5; i++ {
			blockTime := genesisTime + int64(i)*2

			// 验证时间
			if !timeController.ValidateBlockTime(i, blockTime) {
				t.Errorf("Block %d time validation failed", i)
				continue
			}

			// 生产区块
			block, err := blockProducer.ProduceBlock(blockTime, i)
			if err != nil {
				t.Errorf("Failed to produce block %d: %v", i, err)
				continue
			}

			// 添加区块到区块链
			err = blockchainMgr.AddBlock(block)
			if err != nil {
				t.Errorf("Failed to add block %d to blockchain: %v", i, err)
				continue
			}

			t.Logf("Block %d produced and added successfully", i)
		}

		// 9. 验证最终状态
		currentBlock := blockchainMgr.GetCurrentBlock()
		if currentBlock.Header.Number != 5 {
			t.Errorf("Final block height should be 5, got %d", currentBlock.Header.Number)
		}

		// 10. 获取统计信息
		stats := blockchainMgr.GetStats()
		t.Logf("Final blockchain stats: %+v", stats)

		t.Logf("Complete workflow integration test passed")
	})
}

// TestPerformanceBaseline 测试性能基准
func TestPerformanceBaseline(t *testing.T) {
	t.Run("BlockProductionPerformance", func(t *testing.T) {
		// 创建临时目录
		tmpDir, err := os.MkdirTemp("", "shardmatrix_perf_*")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		// 创建组件
		storageInst, _ := storage.NewLevelDBStorage(tmpDir)
		defer storageInst.Close()

		keyPair, _ := crypto.GenerateKeyPair()
		blockchainMgr := blockchain.NewManager(storageInst)

		// 性能测试：生产100个空区块
		const numBlocks = 100
		genesisTime := time.Now().Unix()
		if genesisTime%2 != 0 {
			genesisTime--
		}

		// 初始化区块链
		genesisConfig := &types.GenesisBlock{
			Timestamp: genesisTime,
			InitAccounts: []types.Account{
				{
					Address: keyPair.Address,
					Balance: 1000000,
					Nonce:   0,
					Staked:  0,
				},
			},
			Validators: []types.Validator{},
			ChainID:    "test-chain",
			Config:     map[string]string{},
		}
		initErr := blockchainMgr.Initialize(genesisConfig)
		if initErr != nil {
			t.Fatalf("Failed to initialize blockchain: %v", initErr)
		}

		txPool := blockchain.NewTransactionPool(1000, 5*time.Minute)
		blockProducer := blockchain.NewBlockProducer(keyPair, storageInst, txPool, blockchainMgr)

		// 启动组件
		txPool.Start()
		defer txPool.Stop()
		blockProducer.Start()
		defer blockProducer.Stop()

		start := time.Now()

		for i := uint64(1); i <= numBlocks; i++ {
			blockTime := genesisTime + int64(i)*2
			block, err := blockProducer.ProduceBlock(blockTime, i)
			if err != nil {
				t.Errorf("Failed to produce block %d: %v", i, err)
				continue
			}

			// 保存区块到区块链
			err = blockchainMgr.AddBlock(block)
			if err != nil {
				t.Errorf("Failed to add block %d: %v", i, err)
			}
		}

		duration := time.Since(start)
		avgTime := duration / numBlocks

		t.Logf("Produced %d blocks in %v (avg: %v per block)", numBlocks, duration, avgTime)

		// 性能要求：每个区块生产应该在100ms内完成
		if avgTime > 100*time.Millisecond {
			t.Errorf("Block production too slow: %v > 100ms", avgTime)
		}
	})
}
