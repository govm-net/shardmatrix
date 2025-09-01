package storage

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/govm-net/shardmatrix/pkg/types"
)

// StorageManager 存储管理器
type StorageManager struct {
	BlockStore     BlockStore
	TxStore        TransactionStoreInterface
	AccountStore   AccountStoreInterface
	ValidatorStore ValidatorStoreInterface
	dataDir        string
	useMemory      bool
}

// StorageConfig 存储配置
type StorageConfig struct {
	DataDir   string `json:"data_dir"`
	UseMemory bool   `json:"use_memory"` // 是否使用内存存储（用于测试）
}

// NewStorageManager 创建存储管理器
func NewStorageManager(config *StorageConfig) (*StorageManager, error) {
	if config.UseMemory {
		return NewMemoryStorageManager(), nil
	}

	// 确保数据目录存在
	if err := os.MkdirAll(config.DataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %v", err)
	}

	// 创建各种存储实例
	blockStore, err := NewLevelDBBlockStore(config.DataDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create block store: %v", err)
	}

	txStore, err := NewTransactionStore(filepath.Join(config.DataDir, "transactions"))
	if err != nil {
		return nil, fmt.Errorf("failed to create transaction store: %v", err)
	}

	accountStore, err := NewAccountStore(filepath.Join(config.DataDir, "accounts"))
	if err != nil {
		return nil, fmt.Errorf("failed to create account store: %v", err)
	}

	validatorStore, err := NewValidatorStore(filepath.Join(config.DataDir, "validators"))
	if err != nil {
		return nil, fmt.Errorf("failed to create validator store: %v", err)
	}

	return &StorageManager{
		BlockStore:     blockStore,
		TxStore:        txStore,
		AccountStore:   accountStore,
		ValidatorStore: validatorStore,
		dataDir:        config.DataDir,
		useMemory:      false,
	}, nil
}

// NewMemoryStorageManager 创建内存存储管理器（用于测试）
func NewMemoryStorageManager() *StorageManager {
	return &StorageManager{
		BlockStore:     NewMemoryBlockStore(),
		TxStore:        NewMemoryTransactionStore(),
		AccountStore:   NewMemoryAccountStore(),
		ValidatorStore: NewMemoryValidatorStore(),
		useMemory:      true,
	}
}

// Close 关闭所有存储
func (sm *StorageManager) Close() error {
	var errors []error

	if err := sm.BlockStore.Close(); err != nil {
		errors = append(errors, fmt.Errorf("block store close error: %v", err))
	}

	if err := sm.TxStore.Close(); err != nil {
		errors = append(errors, fmt.Errorf("transaction store close error: %v", err))
	}

	if err := sm.AccountStore.Close(); err != nil {
		errors = append(errors, fmt.Errorf("account store close error: %v", err))
	}

	if err := sm.ValidatorStore.Close(); err != nil {
		errors = append(errors, fmt.Errorf("validator store close error: %v", err))
	}

	if len(errors) > 0 {
		return fmt.Errorf("storage close errors: %v", errors)
	}

	return nil
}

// GetStats 获取存储统计信息
func (sm *StorageManager) GetStats() (*StorageStats, error) {
	stats := &StorageStats{
		UseMemory: sm.useMemory,
		DataDir:   sm.dataDir,
	}

	// 获取最新区块信息
	latestBlock, err := sm.BlockStore.GetLatestBlock()
	if err == nil {
		stats.LatestBlockHeight = latestBlock.Header.Number
		stats.LatestBlockHash = latestBlock.Hash().String()
	}

	// 获取验证者数量
	validators, err := sm.ValidatorStore.GetAllValidators()
	if err == nil {
		stats.ValidatorCount = uint64(len(validators))
	}

	return stats, nil
}

// StorageStats 存储统计信息
type StorageStats struct {
	UseMemory         bool   `json:"use_memory"`
	DataDir           string `json:"data_dir"`
	LatestBlockHeight uint64 `json:"latest_block_height"`
	LatestBlockHash   string `json:"latest_block_hash"`
	ValidatorCount    uint64 `json:"validator_count"`
}

// ValidateIntegrity 验证存储完整性
func (sm *StorageManager) ValidateIntegrity() error {
	// 验证最新区块
	latestBlock, err := sm.BlockStore.GetLatestBlock()
	if err != nil {
		return fmt.Errorf("failed to get latest block: %v", err)
	}

	// 验证区块中的交易是否存在
	for _, txHash := range latestBlock.Transactions {
		if !sm.TxStore.HasTransaction(txHash) {
			return fmt.Errorf("transaction %x not found in transaction store", txHash)
		}
	}

	// 验证验证者
	validators, err := sm.ValidatorStore.GetAllValidators()
	if err != nil {
		return fmt.Errorf("failed to get validators: %v", err)
	}

	// 验证每个验证者对应的账户是否存在
	for _, validator := range validators {
		if !sm.AccountStore.HasAccount(validator.Address) {
			return fmt.Errorf("account for validator %x not found", validator.Address)
		}
	}

	return nil
}

// InitializeGenesis 初始化创世状态
func (sm *StorageManager) InitializeGenesis(genesisValidators []*types.Validator, genesisAccounts []*types.Account) error {
	// 存储创世验证者
	for _, validator := range genesisValidators {
		if err := sm.ValidatorStore.PutValidator(validator); err != nil {
			return fmt.Errorf("failed to store genesis validator: %v", err)
		}
	}

	// 存储创世账户
	for _, account := range genesisAccounts {
		if err := sm.AccountStore.PutAccount(account); err != nil {
			return fmt.Errorf("failed to store genesis account: %v", err)
		}
	}

	return nil
}
