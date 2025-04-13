package state

import (
	"context"
	"encoding/json"
	"fmt"
)

// Transaction represents a blockchain transaction
type Transaction struct {
	Type      byte            `json:"type"`       // 交易类型，支持255种
	Data      json.RawMessage `json:"data"`       // 交易数据
	Signature []byte          `json:"signature"`  // 交易签名
	PublicKey []byte          `json:"public_key"` // 公钥
}

// TxHandler 交易处理函数类型
type TxHandler func(ctx context.Context, state *State, tx *Transaction) error

// TxRegistry 交易注册器
type TxRegistry struct {
	handlers map[byte]TxHandler
}

var DefaultTxRegistry *TxRegistry

func init() {
	DefaultTxRegistry = NewTxRegistry()
}

// NewTxRegistry 创建新的交易注册器
func NewTxRegistry() *TxRegistry {
	return &TxRegistry{
		handlers: make(map[byte]TxHandler),
	}
}

// Register 注册交易处理函数
func (r *TxRegistry) Register(txType byte, handler TxHandler) error {
	if _, exists := r.handlers[txType]; exists {
		return fmt.Errorf("transaction type %d already registered", txType)
	}
	r.handlers[txType] = handler
	return nil
}

func (r *TxRegistry) Cover(txType byte, handler TxHandler) {
	r.handlers[txType] = handler
}

// GetHandler 获取交易处理函数
func (r *TxRegistry) GetHandler(txType byte) (TxHandler, bool) {
	handler, exists := r.handlers[txType]
	return handler, exists
}
