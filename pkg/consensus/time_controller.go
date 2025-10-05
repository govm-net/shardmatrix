package consensus

import (
	"fmt"
	"sync"
	"time"

	"github.com/lengzhao/shardmatrix/pkg/types"
)

// TimeController 严格时间控制器
type TimeController struct {
	mu          sync.RWMutex
	genesisTime int64  // 创世时间戳
	blockTime   int64  // 区块间隔（秒）
	blockOffset uint64 // 区块偏移量（用于从现有区块链继续）
	started     bool   // 是否已启动
	stopCh      chan struct{}
	callbacks   map[string]BlockTimeCallback
}

// BlockTimeCallback 区块时间回调函数
type BlockTimeCallback func(blockTime int64, blockNumber uint64)

// NewTimeController 创建新的时间控制器
func NewTimeController(genesisTime int64) *TimeController {
	return &TimeController{
		genesisTime: genesisTime,
		blockTime:   int64(types.BlockTime.Seconds()), // 严格2秒
		callbacks:   make(map[string]BlockTimeCallback),
		stopCh:      make(chan struct{}),
	}
}

// SetGenesisTime 设置创世时间（必须在启动前调用）
func (tc *TimeController) SetGenesisTime(timestamp int64) error {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	if tc.started {
		return fmt.Errorf("cannot set genesis time after controller started")
	}

	// 确保创世时间戳是偶数秒（便于严格控制）
	if timestamp%2 != 0 {
		timestamp = timestamp - 1 // 向下取整到偶数秒
	}

	tc.genesisTime = timestamp
	return nil
}

// SetBlockOffset 设置区块偏移量（用于从现有区块链继续）
func (tc *TimeController) SetBlockOffset(offset uint64) error {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	if tc.started {
		return fmt.Errorf("cannot set block offset after controller started")
	}

	tc.blockOffset = offset
	return nil
}

// GetGenesisTime 获取创世时间
func (tc *TimeController) GetGenesisTime() int64 {
	tc.mu.RLock()
	defer tc.mu.RUnlock()
	return tc.genesisTime
}

// Start 启动时间控制器
func (tc *TimeController) Start() error {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	if tc.started {
		return fmt.Errorf("time controller already started")
	}

	if tc.genesisTime == 0 {
		return fmt.Errorf("genesis time not set")
	}

	tc.started = true
	tc.stopCh = make(chan struct{})

	go tc.timeControlLoop()

	return nil
}

// Stop 停止时间控制器
func (tc *TimeController) Stop() {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	if !tc.started {
		return
	}

	tc.started = false
	close(tc.stopCh)
}

// RegisterCallback 注册时间回调
func (tc *TimeController) RegisterCallback(name string, callback BlockTimeCallback) {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	tc.callbacks[name] = callback
}

// UnregisterCallback 取消注册时间回调
func (tc *TimeController) UnregisterCallback(name string) {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	delete(tc.callbacks, name)
}

// timeControlLoop 时间控制循环
func (tc *TimeController) timeControlLoop() {
	ticker := time.NewTicker(100 * time.Millisecond) // 100ms检查一次时间
	defer ticker.Stop()

	// 初始化lastTriggeredBlock为当前区块偏移量，这样就从下一个区块开始
	var lastTriggeredBlock uint64 = tc.blockOffset

	for {
		select {
		case <-tc.stopCh:
			return
		case now := <-ticker.C:
			// 计算当前应该是第几个区块（考虑偏移量）
			currentBlockNumber := tc.calculateCurrentBlockNumber(now.Unix())

			// 如果是新的区块时间点，触发回调
			if currentBlockNumber > lastTriggeredBlock {
				blockTime := tc.calculateBlockTime(currentBlockNumber)

				// 严格验证时间（必须精确到秒）
				if now.Unix() >= blockTime {
					tc.triggerCallbacks(blockTime, currentBlockNumber)
					lastTriggeredBlock = currentBlockNumber
				}
			}
		}
	}
}

// calculateCurrentBlockNumber 计算当前时间对应的区块号（考虑偏移量）
func (tc *TimeController) calculateCurrentBlockNumber(currentTime int64) uint64 {
	if currentTime < tc.genesisTime {
		return tc.blockOffset
	}

	elapsed := currentTime - tc.genesisTime
	relativeBlockNumber := uint64(elapsed / tc.blockTime)
	return tc.blockOffset + relativeBlockNumber
}

// calculateBlockTime 计算指定区块号的精确时间戳（考虑偏移量）
func (tc *TimeController) calculateBlockTime(blockNumber uint64) int64 {
	if blockNumber < tc.blockOffset {
		return tc.genesisTime // 不应该发生，但返回创世时间作为默认值
	}
	relativeBlockNumber := blockNumber - tc.blockOffset
	return tc.genesisTime + int64(relativeBlockNumber)*tc.blockTime
}

// triggerCallbacks 触发所有注册的回调
func (tc *TimeController) triggerCallbacks(blockTime int64, blockNumber uint64) {
	tc.mu.RLock()
	callbacks := make(map[string]BlockTimeCallback)
	for name, callback := range tc.callbacks {
		callbacks[name] = callback
	}
	tc.mu.RUnlock()

	// 异步触发回调以避免阻塞
	for name, callback := range callbacks {
		go func(n string, cb BlockTimeCallback) {
			defer func() {
				if r := recover(); r != nil {
					fmt.Printf("Time callback %s panicked: %v\n", n, r)
				}
			}()
			cb(blockTime, blockNumber)
		}(name, callback)
	}
}

// GetCurrentBlockNumber 获取当前时间对应的区块号
func (tc *TimeController) GetCurrentBlockNumber() uint64 {
	tc.mu.RLock()
	defer tc.mu.RUnlock()

	return tc.calculateCurrentBlockNumber(time.Now().Unix())
}

// GetBlockTime 获取指定区块号的时间戳
func (tc *TimeController) GetBlockTime(blockNumber uint64) int64 {
	tc.mu.RLock()
	defer tc.mu.RUnlock()

	return tc.calculateBlockTime(blockNumber)
}

// ValidateBlockTime 验证区块时间戳是否严格正确
func (tc *TimeController) ValidateBlockTime(blockNumber uint64, timestamp int64) bool {
	expectedTime := tc.GetBlockTime(blockNumber)
	return timestamp == expectedTime
}

// GetNextBlockTime 获取下一个区块的时间戳
func (tc *TimeController) GetNextBlockTime(currentBlockNumber uint64) int64 {
	return tc.GetBlockTime(currentBlockNumber + 1)
}

// TimeToNextBlock 计算距离下一个区块还有多长时间
func (tc *TimeController) TimeToNextBlock(currentBlockNumber uint64) time.Duration {
	nextBlockTime := tc.GetNextBlockTime(currentBlockNumber)
	now := time.Now().Unix()

	if nextBlockTime <= now {
		return 0
	}

	return time.Duration(nextBlockTime-now) * time.Second
}

// IsBlockTime 检查当前时间是否是区块时间点
func (tc *TimeController) IsBlockTime(blockNumber uint64) bool {
	expectedTime := tc.GetBlockTime(blockNumber)
	now := time.Now().Unix()

	// 允许1秒的误差范围
	return now >= expectedTime && now < expectedTime+1
}

// WaitForBlockTime 等待指定区块的时间点
func (tc *TimeController) WaitForBlockTime(blockNumber uint64) error {
	targetTime := tc.GetBlockTime(blockNumber)
	now := time.Now().Unix()

	if now >= targetTime {
		return nil // 已经到时间了
	}

	waitDuration := time.Duration(targetTime-now) * time.Second

	select {
	case <-time.After(waitDuration):
		return nil
	case <-tc.stopCh:
		return fmt.Errorf("time controller stopped while waiting")
	}
}

// GetBlockInterval 获取区块间隔
func (tc *TimeController) GetBlockInterval() time.Duration {
	return time.Duration(tc.blockTime) * time.Second
}

// IsValidBlockTimestamp 验证区块时间戳是否有效（用于网络接收的区块）
func (tc *TimeController) IsValidBlockTimestamp(blockNumber uint64, timestamp int64, tolerance time.Duration) bool {
	expectedTime := tc.GetBlockTime(blockNumber)
	diff := timestamp - expectedTime

	if diff < 0 {
		diff = -diff
	}

	return time.Duration(diff)*time.Second <= tolerance
}

// GetTimeStats 获取时间统计信息
func (tc *TimeController) GetTimeStats() map[string]interface{} {
	tc.mu.RLock()
	defer tc.mu.RUnlock()

	now := time.Now().Unix()
	currentBlock := tc.calculateCurrentBlockNumber(now)
	nextBlockTime := tc.calculateBlockTime(currentBlock + 1)

	return map[string]interface{}{
		"genesis_time":       tc.genesisTime,
		"block_interval":     tc.blockTime,
		"current_time":       now,
		"current_block":      currentBlock,
		"next_block_time":    nextBlockTime,
		"time_to_next_block": nextBlockTime - now,
		"is_started":         tc.started,
		"callback_count":     len(tc.callbacks),
	}
}

// SyncWithNetwork 与网络时间同步（未来可扩展为NTP同步）
func (tc *TimeController) SyncWithNetwork() error {
	// 当前是简单实现，未来可以集成NTP同步
	// 确保本地时间与网络时间同步

	// 这里可以添加NTP同步逻辑
	// 暂时返回nil表示同步成功
	return nil
}

// AdjustTime 调整时间（仅用于测试，生产环境禁用）
func (tc *TimeController) AdjustTime(offset time.Duration) error {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	if tc.started {
		return fmt.Errorf("cannot adjust time while controller is running")
	}

	tc.genesisTime += int64(offset.Seconds())
	return nil
}

// 严格时间验证工具函数

// StrictTimeValidator 严格时间验证器
type StrictTimeValidator struct {
	controller *TimeController
}

// NewStrictTimeValidator 创建严格时间验证器
func NewStrictTimeValidator(controller *TimeController) *StrictTimeValidator {
	return &StrictTimeValidator{
		controller: controller,
	}
}

// ValidateBlock 验证区块时间戳
func (v *StrictTimeValidator) ValidateBlock(block *types.Block) error {
	blockNumber := block.Header.Number
	timestamp := block.Header.Timestamp

	if !v.controller.ValidateBlockTime(blockNumber, timestamp) {
		expectedTime := v.controller.GetBlockTime(blockNumber)
		return fmt.Errorf("invalid block timestamp: expected %d, got %d (block %d)",
			expectedTime, timestamp, blockNumber)
	}

	return nil
}

// ValidateSequence 验证区块序列的时间连续性
func (v *StrictTimeValidator) ValidateSequence(blocks []*types.Block) error {
	for i, block := range blocks {
		if err := v.ValidateBlock(block); err != nil {
			return fmt.Errorf("block %d failed time validation: %w", i, err)
		}

		// 验证与前一个区块的时间间隔
		if i > 0 {
			prevBlock := blocks[i-1]
			expectedInterval := int64(types.BlockTime.Seconds())
			actualInterval := block.Header.Timestamp - prevBlock.Header.Timestamp

			if actualInterval != expectedInterval {
				return fmt.Errorf("invalid time interval between blocks %d and %d: expected %d, got %d",
					i-1, i, expectedInterval, actualInterval)
			}
		}
	}

	return nil
}
