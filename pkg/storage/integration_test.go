package storage

import (
	"os"
	"testing"

	"github.com/govm-net/shardmatrix/pkg/types"
)

func TestStorageManagerWithLevelDB(t *testing.T) {
	// 创建临时目录用于测试
	tempDir, err := os.MkdirTemp("", "storage_manager_leveldb_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 创建使用LevelDB的存储管理器
	config := &StorageConfig{
		DataDir:   tempDir,
		UseMemory: false,
	}

	sm, err := NewStorageManager(config)
	if err != nil {
		t.Fatalf("Failed to create storage manager: %v", err)
	}
	defer sm.Close()

	// 创建测试数据
	validatorAddr := types.AddressFromPublicKey([]byte("validator_public_key"))
	validator := types.NewValidator(validatorAddr, 1000)
	account := types.NewAccountWithBalance(validatorAddr, 5000)

	// 初始化创世状态
	err = sm.InitializeGenesis([]*types.Validator{validator}, []*types.Account{account})
	if err != nil {
		t.Fatalf("Failed to initialize genesis: %v", err)
	}

	// 验证验证者存在
	if !sm.ValidatorStore.HasValidator(validatorAddr) {
		t.Error("Validator should exist after genesis initialization")
	}

	// 验证账户存在
	if !sm.AccountStore.HasAccount(validatorAddr) {
		t.Error("Account should exist after genesis initialization")
	}

	// 创建初始区块
	genesisBlock := types.NewGenesisBlock(validatorAddr)
	err = sm.BlockStore.PutBlock(genesisBlock)
	if err != nil {
		t.Fatalf("Failed to store genesis block: %v", err)
	}

	// 验证区块存储
	storedGenesis, err := sm.BlockStore.GetBlock(genesisBlock.Hash())
	if err != nil {
		t.Fatalf("Failed to get stored genesis block: %v", err)
	}

	if storedGenesis.Header.Number != 0 {
		t.Errorf("Expected genesis block number 0, got %d", storedGenesis.Header.Number)
	}

	// 获取统计信息
	stats, err := sm.GetStats()
	if err != nil {
		t.Fatalf("Failed to get storage stats: %v", err)
	}

	if stats.ValidatorCount != 1 {
		t.Errorf("Expected 1 validator, got %d", stats.ValidatorCount)
	}

	if stats.LatestBlockHeight != 0 {
		t.Errorf("Expected genesis block height 0, got %d", stats.LatestBlockHeight)
	}

	// 验证存储完整性
	err = sm.ValidateIntegrity()
	if err != nil {
		t.Fatalf("Storage integrity validation failed: %v", err)
	}

	// 创建并存储一个新区块
	block1 := types.NewBlock(1, genesisBlock.Hash(), validatorAddr)
	err = sm.BlockStore.PutBlock(block1)
	if err != nil {
		t.Fatalf("Failed to store block1: %v", err)
	}

	// 验证最新区块已更新
	stats, err = sm.GetStats()
	if err != nil {
		t.Fatalf("Failed to get storage stats: %v", err)
	}

	if stats.LatestBlockHeight != 1 {
		t.Errorf("Expected latest block height 1, got %d", stats.LatestBlockHeight)
	}
}