package blockchain

import (
	"container/heap"
	"fmt"
	"sync"
	"time"

	"github.com/lengzhao/shardmatrix/pkg/types"
)

// TransactionPool 交易池实现
type TransactionPool struct {
	mu           sync.RWMutex
	transactions map[types.Hash]*types.Transaction // 所有交易
	pending      *TxPriorityQueue                  // 待处理交易优先队列
	queued       map[types.Address][]*types.Transaction // 按地址排队的交易
	maxTxs       int                               // 最大交易数
	maxLifetime  time.Duration                     // 交易最大生存时间
	stopCh       chan struct{}                     // 停止信号
	isRunning    bool                              // 是否运行中
}

// TxPriorityItem 交易优先队列项
type TxPriorityItem struct {
	Tx       *types.Transaction
	Priority uint64    // Gas价格作为优先级
	AddTime  time.Time // 添加时间
	Index    int       // 在堆中的索引
}

// TxPriorityQueue 交易优先队列
type TxPriorityQueue []*TxPriorityItem

func (pq TxPriorityQueue) Len() int { return len(pq) }

func (pq TxPriorityQueue) Less(i, j int) bool {
	// 优先级高的在前（Gas价格高的先处理）
	if pq[i].Priority != pq[j].Priority {
		return pq[i].Priority > pq[j].Priority
	}
	// 同等优先级，时间早的在前
	return pq[i].AddTime.Before(pq[j].AddTime)
}

func (pq TxPriorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].Index = i
	pq[j].Index = j
}

func (pq *TxPriorityQueue) Push(x interface{}) {
	n := len(*pq)
	item := x.(*TxPriorityItem)
	item.Index = n
	*pq = append(*pq, item)
}

func (pq *TxPriorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	item := old[n-1]
	old[n-1] = nil  // 避免内存泄漏
	item.Index = -1 // 安全起见
	*pq = old[0 : n-1]
	return item
}

// NewTransactionPool 创建新的交易池
func NewTransactionPool(maxTxs int, maxLifetime time.Duration) *TransactionPool {
	return &TransactionPool{
		transactions: make(map[types.Hash]*types.Transaction),
		pending:      &TxPriorityQueue{},
		queued:       make(map[types.Address][]*types.Transaction),
		maxTxs:       maxTxs,
		maxLifetime:  maxLifetime,
		stopCh:       make(chan struct{}),
	}
}

// Start 启动交易池
func (tp *TransactionPool) Start() error {
	tp.mu.Lock()
	defer tp.mu.Unlock()
	
	if tp.isRunning {
		return fmt.Errorf("transaction pool already running")
	}
	
	tp.isRunning = true
	tp.stopCh = make(chan struct{})
	
	// 启动清理协程
	go tp.cleanupLoop()
	
	return nil
}

// Stop 停止交易池
func (tp *TransactionPool) Stop() {
	tp.mu.Lock()
	defer tp.mu.Unlock()
	
	if !tp.isRunning {
		return
	}
	
	tp.isRunning = false
	close(tp.stopCh)
}

// AddTransaction 添加交易到池中
func (tp *TransactionPool) AddTransaction(tx *types.Transaction) error {
	tp.mu.Lock()
	defer tp.mu.Unlock()
	
	if !tp.isRunning {
		return fmt.Errorf("transaction pool not running")
	}
	
	hash := tx.Hash()
	
	// 检查交易是否已存在
	if _, exists := tp.transactions[hash]; exists {
		return fmt.Errorf("transaction already exists: %s", hash.String())
	}
	
	// 检查交易池是否已满
	if len(tp.transactions) >= tp.maxTxs {
		// 移除优先级最低的交易
		if err := tp.evictLowestPriority(); err != nil {
			return fmt.Errorf("failed to evict transaction: %w", err)
		}
	}
	
	// 验证交易
	if err := tp.validateTransaction(tx); err != nil {
		return fmt.Errorf("transaction validation failed: %w", err)
	}
	
	// 添加到交易池
	tp.transactions[hash] = tx
	
	// 添加到优先队列
	item := &TxPriorityItem{
		Tx:       tx,
		Priority: tx.GasPrice,
		AddTime:  time.Now(),
	}
	heap.Push(tp.pending, item)
	
	return nil
}

// GetTransaction 获取交易
func (tp *TransactionPool) GetTransaction(hash types.Hash) (*types.Transaction, error) {
	tp.mu.RLock()
	defer tp.mu.RUnlock()
	
	tx, exists := tp.transactions[hash]
	if !exists {
		return nil, fmt.Errorf("transaction not found: %s", hash.String())
	}
	
	return tx, nil
}

// RemoveTransaction 移除交易
func (tp *TransactionPool) RemoveTransaction(hash types.Hash) error {
	tp.mu.Lock()
	defer tp.mu.Unlock()
	
	return tp.removeTransaction(hash)
}

// removeTransaction 内部移除交易（需要持有锁）
func (tp *TransactionPool) removeTransaction(hash types.Hash) error {
	tx, exists := tp.transactions[hash]
	if !exists {
		return fmt.Errorf("transaction not found: %s", hash.String())
	}
	
	delete(tp.transactions, hash)
	
	// 从优先队列中移除（简化实现，重建队列）
	tp.rebuildPendingQueue()
	
	// 从排队列表中移除
	tp.removeFromQueued(tx.From, hash)
	
	return nil
}

// RemoveTransactions 批量移除交易
func (tp *TransactionPool) RemoveTransactions(hashes []types.Hash) error {
	tp.mu.Lock()
	defer tp.mu.Unlock()
	
	for _, hash := range hashes {
		tp.removeTransaction(hash)
	}
	
	return nil
}

// GetPendingTransactions 获取待处理的交易
func (tp *TransactionPool) GetPendingTransactions(maxCount int) ([]*types.Transaction, error) {
	tp.mu.RLock()
	defer tp.mu.RUnlock()
	
	var transactions []*types.Transaction
	count := 0
	
	// 从优先队列获取高优先级交易
	tempQueue := make(TxPriorityQueue, len(*tp.pending))
	copy(tempQueue, *tp.pending)
	
	for len(tempQueue) > 0 && count < maxCount {
		item := heap.Pop(&tempQueue).(*TxPriorityItem)
		transactions = append(transactions, item.Tx)
		count++
	}
	
	return transactions, nil
}

// GetTransactionCount 获取交易总数
func (tp *TransactionPool) GetTransactionCount() int {
	tp.mu.RLock()
	defer tp.mu.RUnlock()
	return len(tp.transactions)
}

// GetPendingCount 获取待处理交易数量
func (tp *TransactionPool) GetPendingCount() int {
	tp.mu.RLock()
	defer tp.mu.RUnlock()
	return tp.pending.Len()
}

// validateTransaction 验证交易
func (tp *TransactionPool) validateTransaction(tx *types.Transaction) error {
	// 基本字段验证
	if tx.From.IsEmpty() || tx.To.IsEmpty() {
		return fmt.Errorf("invalid from or to address")
	}
	
	if tx.Amount == 0 {
		return fmt.Errorf("invalid amount")
	}
	
	if tx.GasPrice == 0 || tx.GasLimit == 0 {
		return fmt.Errorf("invalid gas price or limit")
	}
	
	if tx.Signature.IsEmpty() {
		return fmt.Errorf("missing signature")
	}
	
	// 检查分片ID
	if tx.ShardID != types.ShardID {
		return fmt.Errorf("invalid shard ID")
	}
	
	return nil
}

// evictLowestPriority 驱逐最低优先级的交易
func (tp *TransactionPool) evictLowestPriority() error {
	if tp.pending.Len() == 0 {
		return fmt.Errorf("no transactions to evict")
	}
	
	// 找到优先级最低的交易
	lowest := (*tp.pending)[tp.pending.Len()-1]
	hash := lowest.Tx.Hash()
	
	return tp.removeTransaction(hash)
}

// rebuildPendingQueue 重建优先队列
func (tp *TransactionPool) rebuildPendingQueue() {
	tp.pending = &TxPriorityQueue{}
	heap.Init(tp.pending)
	
	for _, tx := range tp.transactions {
		item := &TxPriorityItem{
			Tx:       tx,
			Priority: tx.GasPrice,
			AddTime:  time.Now(), // 简化实现，使用当前时间
		}
		heap.Push(tp.pending, item)
	}
}

// removeFromQueued 从排队列表中移除
func (tp *TransactionPool) removeFromQueued(address types.Address, hash types.Hash) {
	txs := tp.queued[address]
	for i, tx := range txs {
		if tx.Hash() == hash {
			// 移除交易
			tp.queued[address] = append(txs[:i], txs[i+1:]...)
			if len(tp.queued[address]) == 0 {
				delete(tp.queued, address)
			}
			break
		}
	}
}

// cleanupLoop 清理过期交易的循环
func (tp *TransactionPool) cleanupLoop() {
	ticker := time.NewTicker(30 * time.Second) // 每30秒清理一次
	defer ticker.Stop()
	
	for {
		select {
		case <-tp.stopCh:
			return
		case <-ticker.C:
			tp.cleanupExpiredTransactions()
		}
	}
}

// cleanupExpiredTransactions 清理过期交易
func (tp *TransactionPool) cleanupExpiredTransactions() {
	tp.mu.Lock()
	defer tp.mu.Unlock()
	
	now := time.Now()
	var expiredHashes []types.Hash
	
	// 检查所有交易，找出过期的
	for hash, tx := range tp.transactions {
		// 简化实现：使用交易添加时间（实际应存储添加时间戳）
		// 这里假设交易在池中存在超过最大生存时间就过期
		_ = tx // 避免编译器警告
		
		// 实际实现中应该检查交易的添加时间
		// 这里简化为检查优先队列中的时间戳
		for i := 0; i < tp.pending.Len(); i++ {
			item := (*tp.pending)[i]
			if item.Tx.Hash() == hash && now.Sub(item.AddTime) > tp.maxLifetime {
				expiredHashes = append(expiredHashes, hash)
				break
			}
		}
	}
	
	// 移除过期交易
	for _, hash := range expiredHashes {
		tp.removeTransaction(hash)
	}
}

// GetStats 获取交易池统计信息
func (tp *TransactionPool) GetStats() map[string]interface{} {
	tp.mu.RLock()
	defer tp.mu.RUnlock()
	
	return map[string]interface{}{
		"total_transactions":   len(tp.transactions),
		"pending_transactions": tp.pending.Len(),
		"queued_accounts":      len(tp.queued),
		"max_transactions":     tp.maxTxs,
		"max_lifetime_seconds": tp.maxLifetime.Seconds(),
		"is_running":           tp.isRunning,
	}
}

// Clear 清空交易池
func (tp *TransactionPool) Clear() {
	tp.mu.Lock()
	defer tp.mu.Unlock()
	
	tp.transactions = make(map[types.Hash]*types.Transaction)
	tp.pending = &TxPriorityQueue{}
	tp.queued = make(map[types.Address][]*types.Transaction)
	heap.Init(tp.pending)
}

// HasTransaction 检查交易是否存在
func (tp *TransactionPool) HasTransaction(hash types.Hash) bool {
	tp.mu.RLock()
	defer tp.mu.RUnlock()
	
	_, exists := tp.transactions[hash]
	return exists
}

// GetTransactionsByAddress 获取某个地址的所有交易
func (tp *TransactionPool) GetTransactionsByAddress(address types.Address) []*types.Transaction {
	tp.mu.RLock()
	defer tp.mu.RUnlock()
	
	var transactions []*types.Transaction
	
	for _, tx := range tp.transactions {
		if tx.From == address || tx.To == address {
			transactions = append(transactions, tx)
		}
	}
	
	return transactions
}