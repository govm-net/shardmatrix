package storage

import (
	"bytes"
	"encoding/gob"
	"fmt"

	"github.com/govm-net/shardmatrix/pkg/types"
	"github.com/syndtr/goleveldb/leveldb"
)

// TransactionStore 交易存储接口
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

	// 序列化交易
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(tx); err != nil {
		return fmt.Errorf("failed to encode transaction: %v", err)
	}

	// 存储到数据库
	key := fmt.Sprintf("tx:%x", txHash)
	if err := ts.db.Put([]byte(key), buf.Bytes(), nil); err != nil {
		return fmt.Errorf("failed to store transaction: %v", err)
	}

	return nil
}

// GetTransaction 获取交易
func (ts *TransactionStore) GetTransaction(txHash []byte) (*types.Transaction, error) {
	key := fmt.Sprintf("tx:%x", txHash)
	data, err := ts.db.Get([]byte(key), nil)
	if err != nil {
		if err == leveldb.ErrNotFound {
			return nil, fmt.Errorf("transaction not found: %x", txHash)
		}
		return nil, fmt.Errorf("failed to get transaction: %v", err)
	}

	// 反序列化交易
	var tx types.Transaction
	var buf bytes.Buffer
	buf.Write(data)
	dec := gob.NewDecoder(&buf)
	if err := dec.Decode(&tx); err != nil {
		return nil, fmt.Errorf("failed to decode transaction: %v", err)
	}

	return &tx, nil
}

// HasTransaction 检查交易是否存在
func (ts *TransactionStore) HasTransaction(txHash []byte) bool {
	key := fmt.Sprintf("tx:%x", txHash)
	_, err := ts.db.Get([]byte(key), nil)
	return err == nil
}

// DeleteTransaction 删除交易
func (ts *TransactionStore) DeleteTransaction(txHash []byte) error {
	key := fmt.Sprintf("tx:%x", txHash)
	if err := ts.db.Delete([]byte(key), nil); err != nil {
		return fmt.Errorf("failed to delete transaction: %v", err)
	}
	return nil
}

// Close 关闭存储
func (ts *TransactionStore) Close() error {
	return ts.db.Close()
}
