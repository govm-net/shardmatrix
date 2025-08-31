package blockchain

import (
	"fmt"
	"time"

	"github.com/govm-net/shardmatrix/pkg/types"
)

// ChainStats 区块链统计信息
type ChainStats struct {
	// 基本信息
	ChainID       uint64 `json:"chain_id"`        // 链ID
	GenesisHash   string `json:"genesis_hash"`    // 创世区块哈希
	BestBlockHash string `json:"best_block_hash"` // 最佳区块哈希
	Height        uint64 `json:"height"`          // 当前高度
	TotalWork     uint64 `json:"total_work"`      // 总工作量

	// 区块统计
	TotalBlocks       uint64  `json:"total_blocks"`       // 总区块数
	TotalTransactions uint64  `json:"total_transactions"` // 总交易数
	AverageBlockSize  float64 `json:"average_block_size"` // 平均区块大小
	AverageBlockTime  float64 `json:"average_block_time"` // 平均出块时间

	// 分叉信息
	ActiveForks       int `json:"active_forks"`        // 活跃分叉数
	LongestForkLength int `json:"longest_fork_length"` // 最长分叉长度

	// 性能指标
	BlocksPerSecond       float64 `json:"blocks_per_second"`       // 每秒区块数
	TransactionsPerSecond float64 `json:"transactions_per_second"` // 每秒交易数

	// 时间信息
	LastBlockTime int64 `json:"last_block_time"` // 最后区块时间
	LastUpdated   int64 `json:"last_updated"`    // 最后更新时间
}

// GetChainStats 获取区块链统计信息
func (bc *Blockchain) GetChainStats() (*ChainStats, error) {
	bc.mutex.RLock()
	defer bc.mutex.RUnlock()

	stats := &ChainStats{
		ChainID:       bc.config.ChainID,
		BestBlockHash: bc.chainState.BestBlockHash.String(),
		Height:        bc.chainState.Height,
		TotalWork:     bc.chainState.TotalWork,
		TotalBlocks:   bc.chainState.Height + 1, // 包括创世区块
		ActiveForks:   len(bc.forks),
		LastUpdated:   time.Now().Unix(),
	}

	// 获取创世区块哈希
	if genesisBlock, err := bc.blockStore.GetBlockByHeight(0); err == nil {
		stats.GenesisHash = genesisBlock.Hash().String()
	}

	// 获取最后区块时间
	if lastBlock, err := bc.GetBestBlock(); err == nil {
		stats.LastBlockTime = lastBlock.Header.Timestamp
	}

	// 计算详细统计信息
	if err := bc.calculateDetailedStats(stats); err == nil {
		// 统计计算成功
	}

	// 计算分叉信息
	bc.calculateForkStats(stats)

	return stats, nil
}

// calculateDetailedStats 计算详细统计信息
func (bc *Blockchain) calculateDetailedStats(stats *ChainStats) error {
	if stats.Height == 0 {
		return nil
	}

	// 采样最近的区块来计算统计信息
	sampleSize := 100
	if stats.Height < uint64(sampleSize) {
		sampleSize = int(stats.Height + 1)
	}

	fromHeight := stats.Height - uint64(sampleSize-1)
	// if fromHeight < 0 {
	// 	fromHeight = 0
	// }

	recentBlocks, err := bc.GetBlockRange(fromHeight, stats.Height)
	if err != nil {
		return err
	}

	if len(recentBlocks) == 0 {
		return nil
	}

	// 计算总交易数和区块大小
	var totalTransactions uint64
	var totalSize int
	var totalTime int64

	for i, block := range recentBlocks {
		totalTransactions += uint64(block.GetTransactionCount())
		totalSize += block.Size()

		// 计算区块间隔时间
		if i > 0 {
			timeDiff := block.Header.Timestamp - recentBlocks[i-1].Header.Timestamp
			totalTime += timeDiff
		}
	}

	// 计算平均值
	stats.TotalTransactions = totalTransactions
	stats.AverageBlockSize = float64(totalSize) / float64(len(recentBlocks))

	if len(recentBlocks) > 1 {
		stats.AverageBlockTime = float64(totalTime) / float64(len(recentBlocks)-1)

		// 计算性能指标
		timeRange := float64(recentBlocks[len(recentBlocks)-1].Header.Timestamp - recentBlocks[0].Header.Timestamp)
		if timeRange > 0 {
			stats.BlocksPerSecond = float64(len(recentBlocks)-1) / timeRange
			stats.TransactionsPerSecond = float64(totalTransactions) / timeRange
		}
	}

	return nil
}

// calculateForkStats 计算分叉统计信息
func (bc *Blockchain) calculateForkStats(stats *ChainStats) {
	longestForkLength := 0

	for _, fork := range bc.forks {
		if len(fork.Blocks) > longestForkLength {
			longestForkLength = len(fork.Blocks)
		}
	}

	stats.LongestForkLength = longestForkLength
}

// ChainHealth 区块链健康状况
type ChainHealth struct {
	Status           string   `json:"status"`            // 健康状态
	IsSyncing        bool     `json:"is_syncing"`        // 是否同步中
	LastBlockAge     int64    `json:"last_block_age"`    // 最后区块年龄(秒)
	IsStale          bool     `json:"is_stale"`          // 是否过时
	ForkCount        int      `json:"fork_count"`        // 分叉数量
	ValidationErrors []string `json:"validation_errors"` // 验证错误
}

// GetChainHealth 获取区块链健康状况
func (bc *Blockchain) GetChainHealth() *ChainHealth {
	bc.mutex.RLock()
	defer bc.mutex.RUnlock()

	health := &ChainHealth{
		Status:           "healthy",
		IsSyncing:        bc.isSyncing,
		ForkCount:        len(bc.forks),
		ValidationErrors: []string{},
	}

	// 检查最后区块年龄
	if lastBlock, err := bc.GetBestBlock(); err == nil {
		now := time.Now().Unix()
		health.LastBlockAge = now - lastBlock.Header.Timestamp

		// 如果最后区块超过5分钟，认为是过时的
		health.IsStale = health.LastBlockAge > 300

		if health.IsStale {
			health.Status = "stale"
		}
	} else {
		health.Status = "error"
		health.ValidationErrors = append(health.ValidationErrors, "Cannot get best block")
	}

	// 检查分叉情况
	if health.ForkCount > 3 {
		health.Status = "warning"
	}

	// 快速验证最近几个区块
	if err := bc.validateRecentBlocks(); err != nil {
		health.Status = "error"
		health.ValidationErrors = append(health.ValidationErrors, err.Error())
	}

	return health
}

// validateRecentBlocks 验证最近的区块
func (bc *Blockchain) validateRecentBlocks() error {
	// 验证最近5个区块
	checkCount := 5
	currentHeight := bc.chainState.Height

	if currentHeight < uint64(checkCount) {
		checkCount = int(currentHeight + 1)
	}

	fromHeight := currentHeight - uint64(checkCount-1)

	for height := fromHeight; height <= currentHeight; height++ {
		block, err := bc.blockStore.GetBlockByHeight(height)
		if err != nil {
			return err
		}

		// 简单验证区块结构
		if !block.IsValid() {
			return fmt.Errorf("invalid block at height %d", height)
		}

		// 验证区块连接
		if height > 0 {
			prevBlock, err := bc.blockStore.GetBlockByHeight(height - 1)
			if err != nil {
				return err
			}

			if !block.Header.PrevHash.Equal(prevBlock.Hash()) {
				return fmt.Errorf("block %d has invalid previous hash", height)
			}
		}
	}

	return nil
}

// PerformanceMetrics 性能指标
type PerformanceMetrics struct {
	// 吞吐量指标
	BlocksPerMinute       float64 `json:"blocks_per_minute"`
	TransactionsPerMinute float64 `json:"transactions_per_minute"`

	// 延迟指标
	AverageBlockTime  float64 `json:"average_block_time"`  // 平均出块时间
	BlockTimeVariance float64 `json:"block_time_variance"` // 出块时间方差

	// 大小指标
	AverageBlockSize  float64 `json:"average_block_size"`   // 平均区块大小
	AverageTxPerBlock float64 `json:"average_tx_per_block"` // 平均每块交易数

	// 网络指标
	NetworkHeight uint64  `json:"network_height"` // 网络高度
	SyncProgress  float64 `json:"sync_progress"`  // 同步进度

	// 统计时间范围
	StartTime  int64 `json:"start_time"`  // 开始时间
	EndTime    int64 `json:"end_time"`    // 结束时间
	SampleSize int   `json:"sample_size"` // 样本大小
}

// GetPerformanceMetrics 获取性能指标
func (bc *Blockchain) GetPerformanceMetrics(minutes int) (*PerformanceMetrics, error) {
	bc.mutex.RLock()
	defer bc.mutex.RUnlock()

	if minutes <= 0 {
		minutes = 60 // 默认1小时
	}

	now := time.Now().Unix()
	startTime := now - int64(minutes*60)

	// 获取时间范围内的区块
	blocks, err := bc.getBlocksInTimeRange(startTime, now)
	if err != nil {
		return nil, err
	}

	if len(blocks) == 0 {
		return &PerformanceMetrics{
			StartTime:  startTime,
			EndTime:    now,
			SampleSize: 0,
		}, nil
	}

	metrics := &PerformanceMetrics{
		StartTime:  startTime,
		EndTime:    now,
		SampleSize: len(blocks),
	}

	// 计算基本指标
	bc.calculateThroughputMetrics(blocks, metrics)
	bc.calculateLatencyMetrics(blocks, metrics)
	bc.calculateSizeMetrics(blocks, metrics)

	// 网络相关指标
	metrics.NetworkHeight = bc.chainState.Height
	if bc.isSyncing {
		// 这里需要从同步状态获取进度
		metrics.SyncProgress = 0.0 // 简化实现
	} else {
		metrics.SyncProgress = 1.0
	}

	return metrics, nil
}

// getBlocksInTimeRange 获取时间范围内的区块
func (bc *Blockchain) getBlocksInTimeRange(startTime, endTime int64) ([]*types.Block, error) {
	var blocks []*types.Block

	// 从最新区块开始向前查找
	currentHeight := bc.chainState.Height

	for height := currentHeight; height > 0; height-- {
		block, err := bc.blockStore.GetBlockByHeight(height)
		if err != nil {
			continue
		}

		if block.Header.Timestamp < startTime {
			break // 超出时间范围
		}

		if block.Header.Timestamp <= endTime {
			blocks = append(blocks, block)
		}
	}

	// 反转数组，使其按时间顺序排列
	for i, j := 0, len(blocks)-1; i < j; i, j = i+1, j-1 {
		blocks[i], blocks[j] = blocks[j], blocks[i]
	}

	return blocks, nil
}

// calculateThroughputMetrics 计算吞吐量指标
func (bc *Blockchain) calculateThroughputMetrics(blocks []*types.Block, metrics *PerformanceMetrics) {
	if len(blocks) == 0 {
		return
	}

	timeRange := float64(metrics.EndTime - metrics.StartTime)
	if timeRange <= 0 {
		return
	}

	// 计算每分钟区块数
	metrics.BlocksPerMinute = float64(len(blocks)) / (timeRange / 60.0)

	// 计算总交易数
	var totalTransactions int
	for _, block := range blocks {
		totalTransactions += block.GetTransactionCount()
	}

	// 计算每分钟交易数
	metrics.TransactionsPerMinute = float64(totalTransactions) / (timeRange / 60.0)
}

// calculateLatencyMetrics 计算延迟指标
func (bc *Blockchain) calculateLatencyMetrics(blocks []*types.Block, metrics *PerformanceMetrics) {
	if len(blocks) < 2 {
		return
	}

	var blockTimes []float64
	var totalTime float64

	for i := 1; i < len(blocks); i++ {
		timeDiff := float64(blocks[i].Header.Timestamp - blocks[i-1].Header.Timestamp)
		blockTimes = append(blockTimes, timeDiff)
		totalTime += timeDiff
	}

	// 计算平均出块时间
	if len(blockTimes) > 0 {
		metrics.AverageBlockTime = totalTime / float64(len(blockTimes))

		// 计算方差
		var varianceSum float64
		for _, time := range blockTimes {
			diff := time - metrics.AverageBlockTime
			varianceSum += diff * diff
		}
		metrics.BlockTimeVariance = varianceSum / float64(len(blockTimes))
	}
}

// calculateSizeMetrics 计算大小指标
func (bc *Blockchain) calculateSizeMetrics(blocks []*types.Block, metrics *PerformanceMetrics) {
	if len(blocks) == 0 {
		return
	}

	var totalSize int
	var totalTransactions int

	for _, block := range blocks {
		totalSize += block.Size()
		totalTransactions += block.GetTransactionCount()
	}

	metrics.AverageBlockSize = float64(totalSize) / float64(len(blocks))

	if len(blocks) > 0 {
		metrics.AverageTxPerBlock = float64(totalTransactions) / float64(len(blocks))
	}
}
