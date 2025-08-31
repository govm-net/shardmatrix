// Package txpool 提供交易池管理功能
package txpool

import (
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/govm-net/shardmatrix/pkg/types"
)

// TxPool 交易池接口
type TxPool interface {
	// AddTransaction 添加交易到池中
	AddTransaction(tx *types.Transaction) error

	// RemoveTransaction 从池中移除交易
	RemoveTransaction(txHash types.Hash) error

	// GetTransaction 获取交易
	GetTransaction(txHash types.Hash) (*types.Transaction, bool)

	// GetPendingTransactions 获取待处理交易
	GetPendingTransactions() []*types.Transaction

	// GetTransactionsByAddress 获取指定地址的交易
	GetTransactionsByAddress(address types.Address) []*types.Transaction

	// Clear 清空交易池
	Clear()

	// Size 获取交易池大小
	Size() int

	// GetStats 获取交易池统计信息
	GetStats() *TxPoolStats
}

// TxPoolConfig 交易池配置
type TxPoolConfig struct {
	MaxSize       int           `json:"max_size"`        // 最大交易数量
	MaxTxSize     int           `json:"max_tx_size"`     // 单个交易最大大小
	MinFee        uint64        `json:"min_fee"`         // 最小手续费
	TxTimeout     time.Duration `json:"tx_timeout"`      // 交易超时时间
	CleanInterval time.Duration `json:"clean_interval"`  // 清理间隔
	MaxAccountTxs int           `json:"max_account_txs"` // 每个账户最大交易数
	EnableSigning bool          `json:"enable_signing"`  // 是否启用签名验证
}

// DefaultTxPoolConfig 默认交易池配置
func DefaultTxPoolConfig() *TxPoolConfig {
	return &TxPoolConfig{
		MaxSize:       10000,
		MaxTxSize:     1024 * 1024, // 1MB
		MinFee:        1,
		TxTimeout:     30 * time.Minute,
		CleanInterval: 5 * time.Minute,
		MaxAccountTxs: 100,
		EnableSigning: true,
	}
}

// TxPoolStats 交易池统计信息
type TxPoolStats struct {
	PendingCount int    `json:"pending_count"`  // 待处理交易数
	TotalSize    int    `json:"total_size"`     // 总大小（字节）
	TotalFee     uint64 `json:"total_fee"`      // 总手续费
	AverageFee   uint64 `json:"average_fee"`    // 平均手续费
	HighestFee   uint64 `json:"highest_fee"`    // 最高手续费
	LowestFee    uint64 `json:"lowest_fee"`     // 最低手续费
	OldestTxTime int64  `json:"oldest_tx_time"` // 最旧交易时间
	AccountCount int    `json:"account_count"`  // 涉及账户数
}

// TxEntry 交易池条目
type TxEntry struct {
	Tx        *types.Transaction `json:"transaction"`
	AddedAt   time.Time          `json:"added_at"`
	Validated bool               `json:"validated"`
}

// MemoryTxPool 内存交易池实现
type MemoryTxPool struct {
	config *TxPoolConfig

	// 交易存储
	transactions map[types.Hash]*TxEntry      // 按哈希索引的交易
	byAddress    map[types.Address][]*TxEntry // 按地址索引的交易

	// 排序队列
	pendingQueue []*TxEntry // 按手续费排序的待处理队列

	// 同步
	mutex sync.RWMutex

	// 清理器
	cleanTicker *time.Ticker
	stopChan    chan struct{}
}

// NewMemoryTxPool 创建内存交易池
func NewMemoryTxPool(config *TxPoolConfig) *MemoryTxPool {
	if config == nil {
		config = DefaultTxPoolConfig()
	}

	pool := &MemoryTxPool{
		config:       config,
		transactions: make(map[types.Hash]*TxEntry),
		byAddress:    make(map[types.Address][]*TxEntry),
		pendingQueue: make([]*TxEntry, 0),
		stopChan:     make(chan struct{}),
	}

	// 启动清理器
	pool.startCleaner()

	return pool
}

// AddTransaction 添加交易到池中
func (mp *MemoryTxPool) AddTransaction(tx *types.Transaction) error {
	mp.mutex.Lock()
	defer mp.mutex.Unlock()

	// 验证交易
	if err := mp.validateTransaction(tx); err != nil {
		return fmt.Errorf("transaction validation failed: %v", err)
	}

	// 检查是否已存在
	txHash := tx.Hash()
	if _, exists := mp.transactions[txHash]; exists {
		return errors.New("transaction already exists in pool")
	}

	// 检查池容量
	if len(mp.transactions) >= mp.config.MaxSize {
		// 尝试移除最低手续费的交易
		if err := mp.evictLowestFeeTransaction(); err != nil {
			return fmt.Errorf("pool is full and cannot evict: %v", err)
		}
	}

	// 检查账户交易数限制
	if len(mp.byAddress[tx.From]) >= mp.config.MaxAccountTxs {
		return fmt.Errorf("account %x has too many pending transactions", tx.From)
	}

	// 创建交易条目
	entry := &TxEntry{
		Tx:        tx,
		AddedAt:   time.Now(),
		Validated: true,
	}

	// 添加到存储
	mp.transactions[txHash] = entry
	mp.byAddress[tx.From] = append(mp.byAddress[tx.From], entry)

	// 添加到排序队列
	mp.insertToPendingQueue(entry)

	return nil
}

// RemoveTransaction 从池中移除交易
func (mp *MemoryTxPool) RemoveTransaction(txHash types.Hash) error {
	mp.mutex.Lock()
	defer mp.mutex.Unlock()

	return mp.removeTransactionUnsafe(txHash)
}

// removeTransactionUnsafe 不加锁的移除交易（内部使用）
func (mp *MemoryTxPool) removeTransactionUnsafe(txHash types.Hash) error {
	entry, exists := mp.transactions[txHash]
	if !exists {
		return errors.New("transaction not found in pool")
	}

	// 从主存储移除
	delete(mp.transactions, txHash)

	// 从地址索引移除
	fromTxs := mp.byAddress[entry.Tx.From]
	for i, txEntry := range fromTxs {
		if txEntry == entry {
			mp.byAddress[entry.Tx.From] = append(fromTxs[:i], fromTxs[i+1:]...)
			break
		}
	}
	if len(mp.byAddress[entry.Tx.From]) == 0 {
		delete(mp.byAddress, entry.Tx.From)
	}

	// 从待处理队列移除
	mp.removeFromPendingQueue(entry)

	return nil
}

// GetTransaction 获取交易
func (mp *MemoryTxPool) GetTransaction(txHash types.Hash) (*types.Transaction, bool) {
	mp.mutex.RLock()
	defer mp.mutex.RUnlock()

	entry, exists := mp.transactions[txHash]
	if !exists {
		return nil, false
	}
	return entry.Tx, true
}

// GetPendingTransactions 获取待处理交易
func (mp *MemoryTxPool) GetPendingTransactions() []*types.Transaction {
	mp.mutex.RLock()
	defer mp.mutex.RUnlock()

	transactions := make([]*types.Transaction, len(mp.pendingQueue))
	for i, entry := range mp.pendingQueue {
		transactions[i] = entry.Tx
	}
	return transactions
}

// GetTransactionsByAddress 获取指定地址的交易
func (mp *MemoryTxPool) GetTransactionsByAddress(address types.Address) []*types.Transaction {
	mp.mutex.RLock()
	defer mp.mutex.RUnlock()

	entries := mp.byAddress[address]
	transactions := make([]*types.Transaction, len(entries))
	for i, entry := range entries {
		transactions[i] = entry.Tx
	}
	return transactions
}

// Clear 清空交易池
func (mp *MemoryTxPool) Clear() {
	mp.mutex.Lock()
	defer mp.mutex.Unlock()

	mp.transactions = make(map[types.Hash]*TxEntry)
	mp.byAddress = make(map[types.Address][]*TxEntry)
	mp.pendingQueue = make([]*TxEntry, 0)
}

// Size 获取交易池大小
func (mp *MemoryTxPool) Size() int {
	mp.mutex.RLock()
	defer mp.mutex.RUnlock()

	return len(mp.transactions)
}

// GetStats 获取交易池统计信息
func (mp *MemoryTxPool) GetStats() *TxPoolStats {
	mp.mutex.RLock()
	defer mp.mutex.RUnlock()

	stats := &TxPoolStats{
		PendingCount: len(mp.transactions),
		AccountCount: len(mp.byAddress),
	}

	if len(mp.transactions) == 0 {
		return stats
	}

	var totalFee uint64
	var totalSize int
	var oldestTime int64 = time.Now().Unix()
	var highestFee uint64 = 0
	var lowestFee uint64 = ^uint64(0) // 最大uint64值

	for _, entry := range mp.transactions {
		tx := entry.Tx
		totalFee += tx.Fee
		totalSize += tx.Size()

		if tx.Fee > highestFee {
			highestFee = tx.Fee
		}
		if tx.Fee < lowestFee {
			lowestFee = tx.Fee
		}

		entryTime := entry.AddedAt.Unix()
		if entryTime < oldestTime {
			oldestTime = entryTime
		}
	}

	stats.TotalFee = totalFee
	stats.TotalSize = totalSize
	stats.AverageFee = totalFee / uint64(len(mp.transactions))
	stats.HighestFee = highestFee
	stats.LowestFee = lowestFee
	stats.OldestTxTime = oldestTime

	return stats
}

// Stop 停止交易池
func (mp *MemoryTxPool) Stop() {
	if mp.cleanTicker != nil {
		mp.cleanTicker.Stop()
	}
	close(mp.stopChan)
}

// validateTransaction 验证交易
func (mp *MemoryTxPool) validateTransaction(tx *types.Transaction) error {
	// 基本验证
	if !tx.IsValid() {
		return errors.New("transaction is invalid")
	}

	// 检查交易大小
	if tx.Size() > mp.config.MaxTxSize {
		return fmt.Errorf("transaction size %d exceeds maximum %d", tx.Size(), mp.config.MaxTxSize)
	}

	// 检查最小手续费
	if tx.Fee < mp.config.MinFee {
		return fmt.Errorf("transaction fee %d is below minimum %d", tx.Fee, mp.config.MinFee)
	}

	// 检查签名（如果启用）
	if mp.config.EnableSigning && len(tx.Signature) == 0 {
		return errors.New("transaction must be signed")
	}

	return nil
}

// evictLowestFeeTransaction 移除最低手续费的交易
func (mp *MemoryTxPool) evictLowestFeeTransaction() error {
	if len(mp.pendingQueue) == 0 {
		return errors.New("no transactions to evict")
	}

	// 最后一个是手续费最低的
	lastEntry := mp.pendingQueue[len(mp.pendingQueue)-1]
	return mp.removeTransactionUnsafe(lastEntry.Tx.Hash())
}

// insertToPendingQueue 插入到待处理队列（按手续费降序排列）
func (mp *MemoryTxPool) insertToPendingQueue(entry *TxEntry) {
	// 使用二分查找找到插入位置
	index := sort.Search(len(mp.pendingQueue), func(i int) bool {
		return mp.pendingQueue[i].Tx.Fee < entry.Tx.Fee
	})

	// 插入到指定位置
	mp.pendingQueue = append(mp.pendingQueue, nil)
	copy(mp.pendingQueue[index+1:], mp.pendingQueue[index:])
	mp.pendingQueue[index] = entry
}

// removeFromPendingQueue 从待处理队列移除
func (mp *MemoryTxPool) removeFromPendingQueue(entry *TxEntry) {
	for i, queueEntry := range mp.pendingQueue {
		if queueEntry == entry {
			mp.pendingQueue = append(mp.pendingQueue[:i], mp.pendingQueue[i+1:]...)
			break
		}
	}
}

// startCleaner 启动清理器
func (mp *MemoryTxPool) startCleaner() {
	mp.cleanTicker = time.NewTicker(mp.config.CleanInterval)
	go func() {
		for {
			select {
			case <-mp.cleanTicker.C:
				mp.cleanExpiredTransactions()
			case <-mp.stopChan:
				return
			}
		}
	}()
}

// cleanExpiredTransactions 清理过期交易
func (mp *MemoryTxPool) cleanExpiredTransactions() {
	mp.mutex.Lock()
	defer mp.mutex.Unlock()

	now := time.Now()
	expiredTxs := make([]types.Hash, 0)

	for txHash, entry := range mp.transactions {
		if now.Sub(entry.AddedAt) > mp.config.TxTimeout {
			expiredTxs = append(expiredTxs, txHash)
		}
	}

	// 移除过期交易
	for _, txHash := range expiredTxs {
		mp.removeTransactionUnsafe(txHash)
	}
}
