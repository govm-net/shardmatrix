package tx

import (
	"sync"
)

// Registry 交易处理器注册表
type Registry struct {
	mu         sync.RWMutex
	processors map[byte]TxProcess
}

// NewRegistry 创建新的交易处理器注册表
func NewRegistry() *Registry {
	return &Registry{
		processors: make(map[byte]TxProcess),
	}
}

// Register 注册交易处理器
func (r *Registry) Register(p TxProcess) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.processors[p.Type()] = p
}

// GetProcessor 获取交易处理器
func (r *Registry) GetProcessor(txType byte) (TxProcess, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	process, exists := r.processors[txType]
	return process, exists
}

// DefaultRegistry 默认交易处理器注册表
var DefaultRegistry = NewRegistry()

func init() {
	// 注册默认交易处理器
	DefaultRegistry.Register(&TransferProcess{})
	DefaultRegistry.Register(&ContractProcess{})
}
