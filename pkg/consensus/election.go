package consensus

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"sort"
	"time"

	"github.com/govm-net/shardmatrix/pkg/types"
)

// ElectionConfig 选举配置
type ElectionConfig struct {
	SlotDuration   time.Duration `json:"slot_duration"`    // 槽位持续时间
	SlotsPerEpoch  uint64        `json:"slots_per_epoch"`  // 每轮槽位数量
	RandomSeed     []byte        `json:"random_seed"`      // 随机种子
	MaxMissedSlots uint64        `json:"max_missed_slots"` // 最大错过槽位数
}

// ProducerSchedule 生产者调度表
type ProducerSchedule struct {
	Epoch      uint64                   `json:"epoch"`
	Validators []*ValidatorInfo         `json:"validators"`
	Schedule   map[uint64]types.Address `json:"schedule"` // slot -> validator
	Generated  time.Time                `json:"generated"`
}

// GetProducerForSlot 获取指定槽位的出块者
func (dpos *DPoSConsensus) GetProducerForSlot(slot uint64) (types.Address, error) {
	dpos.mutex.RLock()
	defer dpos.mutex.RUnlock()

	if len(dpos.activeValidators) == 0 {
		return types.Address{}, fmt.Errorf("no active validators")
	}

	// 计算当前轮次
	slotsPerEpoch := uint64(dpos.config.EpochDuration / dpos.config.BlockInterval)
	epoch := slot / slotsPerEpoch
	slotInEpoch := slot % slotsPerEpoch

	// 如果轮次发生变化，重新生成调度表
	if epoch != dpos.currentEpoch {
		if err := dpos.generateProducerScheduleUnsafe(epoch); err != nil {
			return types.Address{}, fmt.Errorf("failed to generate schedule: %v", err)
		}
	}

	// 从当前活跃验证者中选择
	validatorIndex := slotInEpoch % uint64(len(dpos.activeValidators))
	return dpos.activeValidators[validatorIndex].Validator.Address, nil
}

// GetCurrentProducer 获取当前槽位的出块者
func (dpos *DPoSConsensus) GetCurrentProducerForTime(timestamp time.Time) (types.Address, uint64, error) {
	// 计算当前槽位
	slot := uint64(timestamp.Unix()) / uint64(dpos.config.BlockInterval.Seconds())

	producer, err := dpos.GetProducerForSlot(slot)
	if err != nil {
		return types.Address{}, 0, err
	}

	return producer, slot, nil
}

// IsProducerForSlot 检查验证者是否为指定槽位的出块者
func (dpos *DPoSConsensus) IsProducerForSlot(validator types.Address, slot uint64) bool {
	producer, err := dpos.GetProducerForSlot(slot)
	if err != nil {
		return false
	}

	return producer.Equal(validator)
}

// CanProduceBlock 检查验证者是否可以在当前时间出块
func (dpos *DPoSConsensus) CanProduceBlock(validator types.Address, timestamp time.Time) bool {
	dpos.mutex.RLock()
	defer dpos.mutex.RUnlock()

	// 检查验证者是否存在且活跃
	validatorInfo, exists := dpos.validators[validator]
	if !exists || !validatorInfo.IsActive {
		return false
	}

	// 计算当前槽位
	slot := uint64(timestamp.Unix()) / uint64(dpos.config.BlockInterval.Seconds())

	// 检查是否为当前槽位的出块者
	producer, err := dpos.GetProducerForSlot(slot)
	if err != nil {
		return false
	}

	return producer.Equal(validator)
}

// generateProducerScheduleUnsafe 生成生产者调度表（不加锁版本）
func (dpos *DPoSConsensus) generateProducerScheduleUnsafe(epoch uint64) error {
	if len(dpos.activeValidators) == 0 {
		return fmt.Errorf("no active validators")
	}

	// 更新当前轮次
	dpos.currentEpoch = epoch

	// 按权重排序验证者（权重越高，出块机会越多）
	weightedValidators := dpos.calculateValidatorWeights()

	// 生成随机种子
	seed := dpos.generateRandomSeed(epoch)

	// 根据权重和随机种子重新排列验证者
	shuffledValidators := dpos.shuffleValidatorsByWeight(weightedValidators, seed)

	// 更新活跃验证者列表
	dpos.activeValidators = shuffledValidators

	// 触发回调
	if dpos.onValidatorSet != nil {
		dpos.onValidatorSet(dpos.activeValidators)
	}

	if dpos.onEpochChange != nil {
		dpos.onEpochChange(epoch)
	}

	fmt.Printf("Generated producer schedule for epoch %d with %d validators\n",
		epoch, len(dpos.activeValidators))

	return nil
}

// calculateValidatorWeights 计算验证者权重
func (dpos *DPoSConsensus) calculateValidatorWeights() []*ValidatorInfo {
	validators := make([]*ValidatorInfo, 0, len(dpos.activeValidators))

	for _, validator := range dpos.activeValidators {
		if validator.IsActive && validator.TotalStake >= dpos.config.MinStake {
			validators = append(validators, validator)
		}
	}

	// 按总质押量排序（降序）
	sort.Slice(validators, func(i, j int) bool {
		return validators[i].TotalStake > validators[j].TotalStake
	})

	return validators
}

// generateRandomSeed 生成随机种子
func (dpos *DPoSConsensus) generateRandomSeed(epoch uint64) []byte {
	// 使用轮次号和固定种子生成随机数
	h := sha256.New()

	// 添加轮次号
	epochBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(epochBytes, epoch)
	h.Write(epochBytes)

	// 添加验证者集合的哈希
	for _, validator := range dpos.activeValidators {
		h.Write(validator.Validator.Address.Bytes())
		stakeBytes := make([]byte, 8)
		binary.BigEndian.PutUint64(stakeBytes, validator.TotalStake)
		h.Write(stakeBytes)
	}

	return h.Sum(nil)
}

// shuffleValidatorsByWeight 根据权重和随机种子重新排列验证者
func (dpos *DPoSConsensus) shuffleValidatorsByWeight(validators []*ValidatorInfo, seed []byte) []*ValidatorInfo {
	if len(validators) <= 1 {
		return validators
	}

	// 创建副本
	result := make([]*ValidatorInfo, len(validators))
	copy(result, validators)

	// 使用种子作为随机源进行Fisher-Yates洗牌
	for i := len(result) - 1; i > 0; i-- {
		// 使用种子生成随机索引
		randomBytes := sha256.Sum256(append(seed, byte(i)))
		randomIndex := int(binary.BigEndian.Uint64(randomBytes[:8])) % (i + 1)

		// 确保索引在有效范围内
		if randomIndex < 0 {
			randomIndex = 0
		}
		if randomIndex > i {
			randomIndex = i
		}

		// 交换元素，但考虑权重
		// 权重高的验证者有更高概率排在前面
		if dpos.shouldSwapByWeight(result[i], result[randomIndex]) {
			result[i], result[randomIndex] = result[randomIndex], result[i]
		}
	}

	return result
}

// shouldSwapByWeight 根据权重决定是否交换
func (dpos *DPoSConsensus) shouldSwapByWeight(a, b *ValidatorInfo) bool {
	// 如果权重相同，随机决定
	if a.TotalStake == b.TotalStake {
		return true
	}

	// 权重高的验证者有更高概率保持位置
	ratio := float64(a.TotalStake) / float64(a.TotalStake+b.TotalStake)
	return ratio < 0.5 // 简化的权重比较
}

// AdvanceSlot 推进到下一个槽位
func (dpos *DPoSConsensus) AdvanceSlot() error {
	dpos.mutex.Lock()
	defer dpos.mutex.Unlock()

	// 计算新的槽位和时间
	dpos.currentSlot++
	dpos.calculateNextSlotTime()

	// 检查是否需要切换轮次
	slotsPerEpoch := uint64(dpos.config.EpochDuration / dpos.config.BlockInterval)
	newEpoch := dpos.currentSlot / slotsPerEpoch

	if newEpoch != dpos.currentEpoch {
		if err := dpos.generateProducerScheduleUnsafe(newEpoch); err != nil {
			return fmt.Errorf("failed to generate new schedule: %v", err)
		}
	}

	// 触发槽位变化回调
	if dpos.onSlotChange != nil {
		dpos.onSlotChange(dpos.currentProducer, dpos.currentSlot)
	}

	return nil
}

// GetTimeToNextSlot 获取到下一个槽位的时间
func (dpos *DPoSConsensus) GetTimeToNextSlot() time.Duration {
	dpos.mutex.RLock()
	defer dpos.mutex.RUnlock()

	now := time.Now()
	if dpos.nextSlotTime.After(now) {
		return dpos.nextSlotTime.Sub(now)
	}
	return 0
}

// IsSlotTime 检查是否到了出块时间
func (dpos *DPoSConsensus) IsSlotTime(timestamp time.Time) bool {
	slotStart := dpos.GetSlotStartTime(timestamp)
	slotEnd := slotStart.Add(dpos.config.BlockInterval)

	return timestamp.After(slotStart) && timestamp.Before(slotEnd)
}

// GetSlotStartTime 获取槽位开始时间
func (dpos *DPoSConsensus) GetSlotStartTime(timestamp time.Time) time.Time {
	slot := uint64(timestamp.Unix()) / uint64(dpos.config.BlockInterval.Seconds())
	return time.Unix(int64(slot*uint64(dpos.config.BlockInterval.Seconds())), 0)
}

// ValidateBlockTiming 验证区块时间
func (dpos *DPoSConsensus) ValidateBlockTiming(block *types.Block) error {
	blockTime := time.Unix(block.Header.Timestamp, 0)

	// 检查是否在有效的槽位时间范围内
	if !dpos.IsSlotTime(blockTime) {
		return fmt.Errorf("block timestamp %v is not in valid slot time", blockTime)
	}

	// 检查出块者是否正确
	slot := uint64(blockTime.Unix()) / uint64(dpos.config.BlockInterval.Seconds())
	expectedProducer, err := dpos.GetProducerForSlot(slot)
	if err != nil {
		return fmt.Errorf("failed to get producer for slot %d: %v", slot, err)
	}

	if !expectedProducer.Equal(block.Header.Validator) {
		return fmt.Errorf("invalid block producer: expected %s, got %s",
			expectedProducer.String(), block.Header.Validator.String())
	}

	return nil
}

// GetProducerStats 获取生产者统计
func (dpos *DPoSConsensus) GetProducerStats(validator types.Address) (*ProducerStats, error) {
	dpos.mutex.RLock()
	defer dpos.mutex.RUnlock()

	validatorInfo, exists := dpos.validators[validator]
	if !exists {
		return nil, fmt.Errorf("validator %s not found", validator.String())
	}

	stats := &ProducerStats{
		Validator:      validator,
		ProducedBlocks: validatorInfo.ProducedBlocks,
		MissedBlocks:   validatorInfo.MissedBlocks,
		LastBlockSlot:  validatorInfo.LastBlockSlot,
		TotalStake:     validatorInfo.TotalStake,
		IsActive:       validatorInfo.IsActive,
	}

	// 计算生产率
	totalSlots := validatorInfo.ProducedBlocks + validatorInfo.MissedBlocks
	if totalSlots > 0 {
		stats.ProductionRate = float64(validatorInfo.ProducedBlocks) / float64(totalSlots)
	}

	return stats, nil
}

// ProducerStats 生产者统计信息
type ProducerStats struct {
	Validator      types.Address `json:"validator"`
	ProducedBlocks uint64        `json:"produced_blocks"`
	MissedBlocks   uint64        `json:"missed_blocks"`
	LastBlockSlot  uint64        `json:"last_block_slot"`
	TotalStake     uint64        `json:"total_stake"`
	ProductionRate float64       `json:"production_rate"`
	IsActive       bool          `json:"is_active"`
}

// GetAllProducerStats 获取所有生产者统计
func (dpos *DPoSConsensus) GetAllProducerStats() ([]*ProducerStats, error) {
	dpos.mutex.RLock()
	defer dpos.mutex.RUnlock()

	stats := make([]*ProducerStats, 0, len(dpos.validators))

	for address := range dpos.validators {
		stat, err := dpos.GetProducerStats(address)
		if err != nil {
			continue // 跳过错误的验证者
		}
		stats = append(stats, stat)
	}

	// 按生产区块数排序
	sort.Slice(stats, func(i, j int) bool {
		return stats[i].ProducedBlocks > stats[j].ProducedBlocks
	})

	return stats, nil
}
