// Package storage provides storage interfaces for the blockchain
package storage

import "github.com/lengzhao/shardmatrix/pkg/types"

// Storage 存储接口定义
type Storage interface {
	// 区块相关
	SaveBlock(block *types.Block) error
	GetBlock(height uint64) (*types.Block, error)
	GetBlockByHash(hash types.Hash) (*types.Block, error)
	HasBlock(hash types.Hash) bool
	
	// 交易相关
	SaveTransaction(tx *types.Transaction) error
	GetTransaction(hash types.Hash) (*types.Transaction, error)
	HasTransaction(hash types.Hash) bool
	
	// 账户相关
	SaveAccount(account *types.Account) error
	GetAccount(address types.Address) (*types.Account, error)
	HasAccount(address types.Address) bool
	
	// 验证者相关
	SaveValidator(validator *types.Validator) error
	GetValidator(address types.Address) (*types.Validator, error)
	GetAllValidators() ([]*types.Validator, error)
	
	// 链状态相关
	GetLatestBlockHeight() (uint64, error)
	UpdateLatestBlockHeight(height uint64) error
	SaveGenesisBlock(genesis *types.GenesisBlock) error
	GetGenesisBlock() (*types.GenesisBlock, error)
	
	// 通用操作
	Close() error
	GetStats() map[string]interface{} // 获取存储统计信息
}

// MockStorage 模拟存储实现（用于测试）
type MockStorage struct{}

// NewMockStorage 创建模拟存储
func NewMockStorage() *MockStorage {
	return &MockStorage{}
}

// 实现所有接口方法（简化版本）
func (m *MockStorage) SaveBlock(block *types.Block) error                    { return nil }
func (m *MockStorage) GetBlock(height uint64) (*types.Block, error)         { return nil, nil }
func (m *MockStorage) GetBlockByHash(hash types.Hash) (*types.Block, error) { return nil, nil }
func (m *MockStorage) HasBlock(hash types.Hash) bool                        { return false }
func (m *MockStorage) SaveTransaction(tx *types.Transaction) error          { return nil }
func (m *MockStorage) GetTransaction(hash types.Hash) (*types.Transaction, error) { return nil, nil }
func (m *MockStorage) HasTransaction(hash types.Hash) bool                  { return false }
func (m *MockStorage) SaveAccount(account *types.Account) error             { return nil }
func (m *MockStorage) GetAccount(address types.Address) (*types.Account, error) {
	return &types.Account{Address: address}, nil
}
func (m *MockStorage) HasAccount(address types.Address) bool                    { return false }
func (m *MockStorage) SaveValidator(validator *types.Validator) error          { return nil }
func (m *MockStorage) GetValidator(address types.Address) (*types.Validator, error) { return nil, nil }
func (m *MockStorage) GetAllValidators() ([]*types.Validator, error)           { return nil, nil }
func (m *MockStorage) GetLatestBlockHeight() (uint64, error)                   { return 0, nil }
func (m *MockStorage) UpdateLatestBlockHeight(height uint64) error             { return nil }
func (m *MockStorage) SaveGenesisBlock(genesis *types.GenesisBlock) error      { return nil }
func (m *MockStorage) GetGenesisBlock() (*types.GenesisBlock, error)           { return nil, nil }
func (m *MockStorage) Close() error                                            { return nil }
func (m *MockStorage) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"reads_total":      int64(0),
		"writes_total":     int64(0),
		"bytes_read":       int64(0),
		"bytes_written":    int64(0),
		"batches_total":    int64(0),
		"compactions_total": int64(0),
	}
}