package storage

import (
	"os"
	"testing"

	"github.com/govm-net/shardmatrix/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	validator := types.NewValidator(validatorAddr, nil, 1000)
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

// TestStorageIntegration 测试存储集成
func TestStorageIntegration(t *testing.T) {
	// 创建临时目录
	tempDir, err := os.MkdirTemp("", "storage_integration_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// 创建存储管理器
	config := &StorageConfig{
		DataDir:   tempDir,
		UseMemory: false,
	}

	manager, err := NewStorageManager(config)
	require.NoError(t, err)
	defer manager.Close()

	// 测试区块存储
	block := createTestBlock(1)
	err = manager.BlockStore.PutBlock(block)
	require.NoError(t, err)

	retrievedBlock, err := manager.BlockStore.GetBlock(block.Hash())
	require.NoError(t, err)
	assert.Equal(t, block.Hash(), retrievedBlock.Hash())

	// 测试交易存储
	tx := createTestTransaction()
	err = manager.TxStore.PutTransaction(tx)
	require.NoError(t, err)

	retrievedTx, err := manager.TxStore.GetTransaction(tx.Hash())
	require.NoError(t, err)
	assert.Equal(t, tx.Hash(), retrievedTx.Hash())

	// 测试账户存储
	account := types.NewAccountWithBalance(
		types.Address{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20},
		1000,
	)
	err = manager.AccountStore.PutAccount(account)
	require.NoError(t, err)

	retrievedAccount, err := manager.AccountStore.GetAccount(account.Address)
	require.NoError(t, err)
	assert.Equal(t, account.Address, retrievedAccount.Address)
	assert.Equal(t, account.Balance, retrievedAccount.Balance)

	// 测试验证者存储
	validator := types.NewValidator(
		types.Address{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20},
		nil,
		1000,
	)
	err = manager.ValidatorStore.PutValidator(validator)
	require.NoError(t, err)

	retrievedValidator, err := manager.ValidatorStore.GetValidator(validator.Address)
	require.NoError(t, err)
	assert.Equal(t, validator.Address, retrievedValidator.Address)
	assert.Equal(t, validator.Stake, retrievedValidator.Stake)
}

// createTestBlock 创建测试区块
func createTestBlock(height uint64) *types.Block {
	validator := types.Address{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20}
	prevHash := types.EmptyHash()

	block := types.NewBlock(height, prevHash, validator)
	block.Header.Signature = []byte("test_signature")
	return block
}

// createTestTransaction 创建测试交易
func createTestTransaction() *types.Transaction {
	from := types.Address{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20}
	to := types.Address{21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33, 34, 35, 36, 37, 38, 39, 40}

	tx := types.NewTransaction(from, to, 100, 10, 1, []byte("test_data"))
	tx.Signature = []byte("test_signature")
	return tx
}
