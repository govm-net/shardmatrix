package state

import (
	"context"
	"encoding/json"
)

// Transaction represents a blockchain transaction
type Transaction struct {
	Type      byte            `json:"type"`       // 交易类型，支持255种
	Data      json.RawMessage `json:"data"`       // 交易数据
	Signature []byte          `json:"signature"`  // 交易签名
	PublicKey []byte          `json:"public_key"` // 公钥
}

// TxHandler 定义交易处理函数
type TxHandler func(ctx context.Context, state *State, tx *Transaction) error

// TxValidator 定义交易验证函数
type TxValidator func(ctx context.Context, tx *Transaction) error

// TxProcess 定义交易处理接口
type TxProcess interface {
	// Type 返回交易类型
	Type() byte
	// Validate 验证交易
	Validate(ctx context.Context, state *State, tx *Transaction) error
	// Execute 执行交易
	Execute(ctx context.Context, state *State, tx *Transaction) error
}

// TxRegistry 管理交易处理器
type TxRegistry struct {
	processes map[byte]TxProcess
}

var DefaultTxRegistry *TxRegistry

func init() {
	DefaultTxRegistry = NewTxRegistry()
}

// NewTxRegistry 创建交易注册表
func NewTxRegistry() *TxRegistry {
	return &TxRegistry{
		processes: make(map[byte]TxProcess),
	}
}

// Register 注册交易处理器
func (r *TxRegistry) Register(process TxProcess) {
	r.processes[process.Type()] = process
}

// GetProcess 获取交易处理器
func (r *TxRegistry) GetProcess(txType byte) (TxProcess, bool) {
	process, exists := r.processes[txType]
	return process, exists
}

// 交易类型
const (
	TxTypeTransfer = iota + 1 // 转账交易
	TxTypeContract            // 合约交易
)
