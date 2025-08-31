package storage

import (
	"fmt"

	"github.com/govm-net/shardmatrix/pkg/types"
	"github.com/syndtr/goleveldb/leveldb"
)

// TransactionStore 交易存储接口
type TransactionStoreInterface interface {
	// PutTransaction 存储交易
	PutTransaction(tx *types.Transaction) error

	// GetTransaction 获取交易
	GetTransaction(txHash types.Hash) (*types.Transaction, error)

	// HasTransaction 检查交易是否存在
	HasTransaction(txHash types.Hash) bool

	// DeleteTransaction 删除交易
	DeleteTransaction(txHash types.Hash) error

	// Close 关闭存储
	Close() error
}

// TransactionStore 交易存储实现
type TransactionStore struct {
	db *leveldb.DB
}

// NewTransactionStore 创建交易存储
func NewTransactionStore(dbPath string) (*TransactionStore, error) {
	db, err := leveldb.OpenFile(dbPath, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to open transaction store: %v", err)
	}

	return &TransactionStore{
		db: db,
	}, nil
}

// PutTransaction 存储交易
func (ts *TransactionStore) PutTransaction(tx *types.Transaction) error {
	txHash := tx.Hash()

	// 使用JSON序列化交易
	data, err := tx.Serialize()
	if err != nil {
		return fmt.Errorf("failed to serialize transaction: %v", err)
	}

	// 存储到数据库
	key := fmt.Sprintf("tx:%s", txHash.String())
	if err := ts.db.Put([]byte(key), data, nil); err != nil {
		return fmt.Errorf("failed to store transaction: %v", err)
	}

	return nil
}

// GetTransaction 获取交易
func (ts *TransactionStore) GetTransaction(txHash types.Hash) (*types.Transaction, error) {
	key := fmt.Sprintf("tx:%s", txHash.String())
	data, err := ts.db.Get([]byte(key), nil)
	if err != nil {
		if err == leveldb.ErrNotFound {
			return nil, fmt.Errorf("transaction not found: %x", txHash)
		}
		return nil, fmt.Errorf("failed to get transaction: %v", err)
	}

	// 使用JSON反序列化交易
	tx, err := types.DeserializeTransaction(data)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize transaction: %v", err)
	}

	return tx, nil
}

// HasTransaction 检查交易是否存在
func (ts *TransactionStore) HasTransaction(txHash types.Hash) bool {
	key := fmt.Sprintf("tx:%s", txHash.String())
	_, err := ts.db.Get([]byte(key), nil)
	return err == nil
}

// DeleteTransaction 删除交易
func (ts *TransactionStore) DeleteTransaction(txHash types.Hash) error {
	key := fmt.Sprintf("tx:%s", txHash.String())
	if err := ts.db.Delete([]byte(key), nil); err != nil {
		return fmt.Errorf("failed to delete transaction: %v", err)
	}
	return nil
}

// Close 关闭存储
func (ts *TransactionStore) Close() error {
	return ts.db.Close()
}

// MemoryTransactionStore 内存交易存储（用于测试）
type MemoryTransactionStore struct {
	transactions map[string]*types.Transaction
}

// NewMemoryTransactionStore 创建内存交易存储
func NewMemoryTransactionStore() *MemoryTransactionStore {
	return &MemoryTransactionStore{
		transactions: make(map[string]*types.Transaction),
	}
}

// PutTransaction 存储交易
func (mts *MemoryTransactionStore) PutTransaction(tx *types.Transaction) error {
	txHash := tx.Hash()
	hashStr := txHash.String()
	mts.transactions[hashStr] = tx
	return nil
}

// GetTransaction 获取交易
func (mts *MemoryTransactionStore) GetTransaction(txHash types.Hash) (*types.Transaction, error) {
	hashStr := txHash.String()
	tx, exists := mts.transactions[hashStr]
	if !exists {
		return nil, fmt.Errorf("transaction not found: %x", txHash)
	}
	return tx, nil
}

// HasTransaction 检查交易是否存在
func (mts *MemoryTransactionStore) HasTransaction(txHash types.Hash) bool {
	hashStr := txHash.String()
	_, exists := mts.transactions[hashStr]
	return exists
}

// DeleteTransaction 删除交易
func (mts *MemoryTransactionStore) DeleteTransaction(txHash types.Hash) error {
	hashStr := txHash.String()
	delete(mts.transactions, hashStr)
	return nil
}

// Close 关闭存储
func (mts *MemoryTransactionStore) Close() error {
	return nil
}

// GetAllTransactions 获取所有交易（用于测试）
func (mts *MemoryTransactionStore) GetAllTransactions() []*types.Transaction {
	txs := make([]*types.Transaction, 0, len(mts.transactions))
	for _, tx := range mts.transactions {
		txs = append(txs, tx)
	}
	return txs
}
