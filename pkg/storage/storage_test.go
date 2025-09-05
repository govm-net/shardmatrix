package storage

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/govm-net/shardmatrix/pkg/types"
)

func TestMemoryTransactionStore(t *testing.T) {
	store := NewMemoryTransactionStore()
	defer store.Close()

	// 创建测试交易
	from := types.AddressFromPublicKey([]byte("from_public_key"))
	to := types.AddressFromPublicKey([]byte("to_public_key"))
	tx := types.NewTransaction(from, to, 100, 10, 1, []byte("test data"))

	// 测试存储交易
	err := store.PutTransaction(tx)
	if err != nil {
		t.Fatalf("Failed to store transaction: %v", err)
	}

	// 测试获取交易
	txHash := tx.Hash()
	retrievedTx, err := store.GetTransaction(txHash)
	if err != nil {
		t.Fatalf("Failed to get transaction: %v", err)
	}

	// 验证交易内容
	if retrievedTx.From != tx.From {
		t.Errorf("Expected from %x, got %x", tx.From, retrievedTx.From)
	}
	if retrievedTx.To != tx.To {
		t.Errorf("Expected to %x, got %x", tx.To, retrievedTx.To)
	}
	if retrievedTx.Amount != tx.Amount {
		t.Errorf("Expected amount %d, got %d", tx.Amount, retrievedTx.Amount)
	}

	// 测试HasTransaction
	if !store.HasTransaction(txHash) {
		t.Error("Transaction should exist")
	}

	// 测试删除交易
	err = store.DeleteTransaction(txHash)
	if err != nil {
		t.Fatalf("Failed to delete transaction: %v", err)
	}

	// 验证交易已删除
	if store.HasTransaction(txHash) {
		t.Error("Transaction should not exist after deletion")
	}
}

func TestMemoryAccountStore(t *testing.T) {
	store := NewMemoryAccountStore()
	defer store.Close()

	// 创建测试账户
	address := types.AddressFromPublicKey([]byte("test_public_key"))
	account := types.NewAccountWithBalance(address, 1000)

	// 测试存储账户
	err := store.PutAccount(account)
	if err != nil {
		t.Fatalf("Failed to store account: %v", err)
	}

	// 测试获取账户
	retrievedAccount, err := store.GetAccount(address)
	if err != nil {
		t.Fatalf("Failed to get account: %v", err)
	}

	// 验证账户内容
	if retrievedAccount.Address != account.Address {
		t.Errorf("Expected address %x, got %x", account.Address, retrievedAccount.Address)
	}
	if retrievedAccount.Balance != account.Balance {
		t.Errorf("Expected balance %d, got %d", account.Balance, retrievedAccount.Balance)
	}

	// 测试HasAccount
	if !store.HasAccount(address) {
		t.Error("Account should exist")
	}

	// 测试更新余额
	newBalance := uint64(2000)
	err = store.UpdateBalance(address, newBalance)
	if err != nil {
		t.Fatalf("Failed to update balance: %v", err)
	}

	// 验证余额更新
	balance, err := store.GetBalance(address)
	if err != nil {
		t.Fatalf("Failed to get balance: %v", err)
	}
	if balance != newBalance {
		t.Errorf("Expected balance %d, got %d", newBalance, balance)
	}

	// 测试删除账户
	err = store.DeleteAccount(address)
	if err != nil {
		t.Fatalf("Failed to delete account: %v", err)
	}

	// 验证账户已删除
	if store.HasAccount(address) {
		t.Error("Account should not exist after deletion")
	}
}

// TestValidatorStore 测试验证者存储
func TestValidatorStore(t *testing.T) {
	// 创建临时目录
	tempDir, err := os.MkdirTemp("", "validator_store_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// 创建存储
	store, err := NewValidatorStore(tempDir)
	require.NoError(t, err)
	defer store.Close()

	// 创建验证者
	address := types.Address{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20}
	validator := types.NewValidator(address, nil, 1000)

	// 存储验证者
	err = store.PutValidator(validator)
	require.NoError(t, err)

	// 获取验证者
	retrieved, err := store.GetValidator(address)
	require.NoError(t, err)
	assert.Equal(t, validator.Address, retrieved.Address)
	assert.Equal(t, validator.Stake, retrieved.Stake)

	// 检查验证者是否存在
	exists := store.HasValidator(address)
	assert.True(t, exists)

	// 删除验证者
	err = store.DeleteValidator(address)
	require.NoError(t, err)

	// 检查验证者是否已删除
	exists = store.HasValidator(address)
	assert.False(t, exists)
}

// TestMemoryValidatorStore 测试内存验证者存储
func TestMemoryValidatorStore(t *testing.T) {
	// 创建存储
	store := NewMemoryValidatorStore()

	// 创建验证者
	address := types.Address{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20}
	validator := types.NewValidator(address, nil, 1000)

	// 存储验证者
	err := store.PutValidator(validator)
	require.NoError(t, err)

	// 获取验证者
	retrieved, err := store.GetValidator(address)
	require.NoError(t, err)
	assert.Equal(t, validator.Address, retrieved.Address)
	assert.Equal(t, validator.Stake, retrieved.Stake)

	// 检查验证者是否存在
	exists := store.HasValidator(address)
	assert.True(t, exists)

	// 删除验证者
	err = store.DeleteValidator(address)
	require.NoError(t, err)

	// 检查验证者是否已删除
	exists = store.HasValidator(address)
	assert.False(t, exists)
}

func TestMemoryBlockStore(t *testing.T) {
	store := NewMemoryBlockStore()
	defer store.Close()

	// 创建测试数据
	validator := types.AddressFromPublicKey([]byte("validator_public_key"))
	prevHash := types.NewHash([]byte("prev_hash"))
	block := types.NewBlock(1, prevHash, validator)

	// 测试存储区块
	err := store.PutBlock(block)
	if err != nil {
		t.Fatalf("Failed to store block: %v", err)
	}

	// 测试按哈希获取区块
	blockHash := block.Hash()
	retrievedBlock, err := store.GetBlock(blockHash)
	if err != nil {
		t.Fatalf("Failed to get block: %v", err)
	}

	// 验证区块内容
	if retrievedBlock.Header.Number != block.Header.Number {
		t.Errorf("Expected block number %d, got %d", block.Header.Number, retrievedBlock.Header.Number)
	}
	if retrievedBlock.Header.Validator != block.Header.Validator {
		t.Errorf("Expected validator %x, got %x", block.Header.Validator, retrievedBlock.Header.Validator)
	}

	// 测试按高度获取区块
	blockByHeight, err := store.GetBlockByHeight(1)
	if err != nil {
		t.Fatalf("Failed to get block by height: %v", err)
	}
	if blockByHeight.Header.Number != block.Header.Number {
		t.Errorf("Expected block number %d, got %d", block.Header.Number, blockByHeight.Header.Number)
	}

	// 测试HasBlock
	if !store.HasBlock(blockHash) {
		t.Error("Block should exist")
	}

	// 测试获取最新区块
	latestBlock, err := store.GetLatestBlock()
	if err != nil {
		t.Fatalf("Failed to get latest block: %v", err)
	}
	if latestBlock.Header.Number != block.Header.Number {
		t.Errorf("Expected latest block number %d, got %d", block.Header.Number, latestBlock.Header.Number)
	}
}

func TestStorageManager(t *testing.T) {
	// 创建内存存储管理器
	sm := NewMemoryStorageManager()
	defer sm.Close()

	// 创建测试数据
	validatorAddr := types.AddressFromPublicKey([]byte("validator_public_key"))
	validator := types.NewValidator(validatorAddr, nil, 1000)
	account := types.NewAccountWithBalance(validatorAddr, 5000)

	// 初始化创世状态
	err := sm.InitializeGenesis([]*types.Validator{validator}, []*types.Account{account})
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
}
