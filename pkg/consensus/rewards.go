package consensus

import (
	"fmt"
	"math"
	"time"

	"github.com/govm-net/shardmatrix/pkg/types"
)

// RewardConfig 奖励配置
type RewardConfig struct {
	BlockReward         uint64  `json:"block_reward"`         // 每个区块的基础奖励
	ValidatorRate       float64 `json:"validator_rate"`       // 验证者奖励比例
	DelegatorRate       float64 `json:"delegator_rate"`       // 委托者奖励比例
	CommunityRate       float64 `json:"community_rate"`       // 社区基金比例
	InflationRate       float64 `json:"inflation_rate"`       // 年通胀率
	MaxInflation        float64 `json:"max_inflation"`        // 最大通胀率
	MinInflation        float64 `json:"min_inflation"`        // 最小通胀率
	InflationAdjustment float64 `json:"inflation_adjustment"` // 通胀调整系数
}

// DefaultRewardConfig 默认奖励配置
func DefaultRewardConfig() *RewardConfig {
	return &RewardConfig{
		BlockReward:         1000000, // 100万代币
		ValidatorRate:       0.5,     // 50%给验证者
		DelegatorRate:       0.4,     // 40%给委托者
		CommunityRate:       0.1,     // 10%给社区基金
		InflationRate:       0.08,    // 8%年通胀率
		MaxInflation:        0.15,    // 最大15%
		MinInflation:        0.02,    // 最小2%
		InflationAdjustment: 0.01,    // 1%调整系数
	}
}

// RewardDistribution 奖励分配
type RewardDistribution struct {
	Epoch            uint64                   `json:"epoch"`
	TotalReward      uint64                   `json:"total_reward"`
	ValidatorRewards map[types.Address]uint64 `json:"validator_rewards"`
	DelegatorRewards map[string]uint64        `json:"delegator_rewards"` // key: delegator:validator
	CommunityReward  uint64                   `json:"community_reward"`
	DistributedAt    time.Time                `json:"distributed_at"`
}

// CalculateBlockReward 计算区块奖励
func (dpos *DPoSConsensus) CalculateBlockReward(block *types.Block) (*RewardDistribution, error) {
	dpos.mutex.RLock()
	defer dpos.mutex.RUnlock()

	// 获取当前轮次
	epoch := dpos.currentEpoch

	// 计算基础奖励
	baseReward := dpos.calculateBaseReward()

	// 创建奖励分配
	distribution := &RewardDistribution{
		Epoch:            epoch,
		TotalReward:      baseReward,
		ValidatorRewards: make(map[types.Address]uint64),
		DelegatorRewards: make(map[string]uint64),
		DistributedAt:    time.Now(),
	}

	// 获取区块生产者
	producer := block.Header.Validator
	validatorInfo, exists := dpos.validators[producer]
	if !exists {
		return nil, fmt.Errorf("block producer %s not found", producer.String())
	}

	// 计算各部分奖励
	validatorReward := uint64(float64(baseReward) * dpos.config.RewardRate)
	delegatorReward := uint64(float64(baseReward) * (1 - dpos.config.RewardRate))
	communityReward := uint64(float64(baseReward) * 0.1) // 固定10%给社区

	// 调整奖励，确保总和不超过基础奖励
	totalDistributed := validatorReward + delegatorReward + communityReward
	if totalDistributed > baseReward {
		ratio := float64(baseReward) / float64(totalDistributed)
		validatorReward = uint64(float64(validatorReward) * ratio)
		delegatorReward = uint64(float64(delegatorReward) * ratio)
		communityReward = uint64(float64(communityReward) * ratio)
	}

	// 分配验证者奖励
	validatorActualReward := dpos.distributeValidatorReward(validatorInfo, validatorReward)
	distribution.ValidatorRewards[producer] = validatorActualReward

	// 分配委托者奖励
	delegatorActualReward := dpos.distributeDelegatorReward(producer, validatorInfo, delegatorReward, distribution)

	// 设置社区奖励
	distribution.CommunityReward = communityReward

	// 更新实际总奖励
	distribution.TotalReward = validatorActualReward + delegatorActualReward + communityReward

	return distribution, nil
}

// calculateBaseReward 计算基础奖励
func (dpos *DPoSConsensus) calculateBaseReward() uint64 {
	// 基于通胀率和总供应量计算
	// 这里简化为固定值，实际应该根据经济模型动态调整
	return 1000000 // 100万代币
}

// distributeValidatorReward 分配验证者奖励
func (dpos *DPoSConsensus) distributeValidatorReward(
	validatorInfo *ValidatorInfo,
	totalReward uint64,
) uint64 {
	// 验证者获得佣金 + 自质押比例的奖励
	commission := uint64(float64(totalReward) * validatorInfo.Commission)

	// 自质押奖励
	var selfStakeReward uint64
	if validatorInfo.TotalStake > 0 {
		selfStakeRatio := float64(validatorInfo.SelfStake) / float64(validatorInfo.TotalStake)
		remainingReward := totalReward - commission
		selfStakeReward = uint64(float64(remainingReward) * selfStakeRatio)
	}

	return commission + selfStakeReward
}

// distributeDelegatorReward 分配委托者奖励
func (dpos *DPoSConsensus) distributeDelegatorReward(
	validator types.Address,
	validatorInfo *ValidatorInfo,
	totalReward uint64,
	distribution *RewardDistribution,
) uint64 {
	if validatorInfo.DelegatedStake == 0 {
		return 0
	}

	totalDistributed := uint64(0)

	// 获取该验证者的所有委托
	for _, delegation := range dpos.delegations {
		if !delegation.Validator.Equal(validator) || delegation.IsUnbonding {
			continue
		}

		// 计算委托者比例
		delegationRatio := float64(delegation.Amount) / float64(validatorInfo.DelegatedStake)

		// 扣除验证者佣金后的奖励
		afterCommission := float64(totalReward) * (1 - validatorInfo.Commission)
		delegatorReward := uint64(afterCommission * delegationRatio)

		// 记录奖励
		key := fmt.Sprintf("%s:%s", delegation.Delegator.String(), validator.String())
		distribution.DelegatorRewards[key] = delegatorReward
		totalDistributed += delegatorReward
	}

	return totalDistributed
}

// DistributeRewards 分发奖励
func (dpos *DPoSConsensus) DistributeRewards(distribution *RewardDistribution) error {
	dpos.mutex.Lock()
	defer dpos.mutex.Unlock()

	// 分发验证者奖励
	for validator, reward := range distribution.ValidatorRewards {
		if dpos.onRewardValidator != nil {
			dpos.onRewardValidator(validator, reward)
		}

		fmt.Printf("Distributed %d tokens to validator %s\n",
			reward, validator.String())
	}

	// 分发委托者奖励
	for key, reward := range distribution.DelegatorRewards {
		// 解析委托者和验证者地址
		// key格式: "delegator:validator"
		// 这里简化处理，实际应该解析地址
		fmt.Printf("Distributed %d tokens to delegator %s\n", reward, key)
	}

	// 处理社区奖励
	if distribution.CommunityReward > 0 {
		fmt.Printf("Allocated %d tokens to community fund\n", distribution.CommunityReward)
	}

	return nil
}

// CalculateStakingReward 计算质押奖励
func (dpos *DPoSConsensus) CalculateStakingReward(
	address types.Address,
	amount uint64,
	duration time.Duration,
) uint64 {
	// 年化收益率计算
	annualRate := dpos.config.RewardRate

	// 时间比例（以年为单位）
	timeRatio := duration.Hours() / (365 * 24)

	// 计算奖励
	reward := float64(amount) * annualRate * timeRatio

	return uint64(reward)
}

// GetRewardRate 获取当前奖励率
func (dpos *DPoSConsensus) GetRewardRate() float64 {
	return dpos.config.RewardRate
}

// UpdateRewardRate 更新奖励率
func (dpos *DPoSConsensus) UpdateRewardRate(newRate float64) error {
	if newRate < 0 || newRate > 1 {
		return fmt.Errorf("reward rate must be between 0 and 1, got %f", newRate)
	}

	dpos.mutex.Lock()
	defer dpos.mutex.Unlock()

	dpos.config.RewardRate = newRate

	fmt.Printf("Updated reward rate to %f\n", newRate)
	return nil
}

// CalculateAPY 计算年化收益率
func (dpos *DPoSConsensus) CalculateAPY(validator types.Address) (float64, error) {
	dpos.mutex.RLock()
	defer dpos.mutex.RUnlock()

	validatorInfo, exists := dpos.validators[validator]
	if !exists {
		return 0, fmt.Errorf("validator %s not found", validator.String())
	}

	// 基础年化收益率
	baseAPY := dpos.config.RewardRate

	// 根据质押比例调整
	if validatorInfo.TotalStake > 0 {
		// 质押越多，收益率可能降低（供需关系）
		stakingRatio := float64(validatorInfo.TotalStake) / float64(dpos.getTotalStakeUnsafe())
		adjustment := 1.0 - (stakingRatio * 0.1) // 最多降低10%
		if adjustment < 0.5 {
			adjustment = 0.5 // 最低50%
		}
		baseAPY *= adjustment
	}

	// 考虑验证者佣金
	delegatorAPY := baseAPY * (1 - validatorInfo.Commission)

	return delegatorAPY, nil
}

// getTotalStakeUnsafe 获取总质押量（不加锁版本）
func (dpos *DPoSConsensus) getTotalStakeUnsafe() uint64 {
	totalStake := uint64(0)
	for _, validator := range dpos.validators {
		totalStake += validator.TotalStake
	}
	return totalStake
}

// GetTotalStake 获取总质押量
func (dpos *DPoSConsensus) GetTotalStake() uint64 {
	dpos.mutex.RLock()
	defer dpos.mutex.RUnlock()
	return dpos.getTotalStakeUnsafe()
}

// CalculateInflationRate 计算通胀率
func (dpos *DPoSConsensus) CalculateInflationRate(totalSupply, totalStaked uint64) float64 {
	// 根据质押比例动态调整通胀率
	stakingRatio := float64(totalStaked) / float64(totalSupply)

	// 目标质押比例（例如67%）
	targetRatio := 0.67

	// 如果质押比例低于目标，提高通胀率激励质押
	if stakingRatio < targetRatio {
		adjustment := (targetRatio - stakingRatio) * dpos.config.InflationAdjustment
		newRate := dpos.config.InflationRate + adjustment
		return math.Min(newRate, dpos.config.MaxInflation)
	} else {
		// 如果质押比例高于目标，降低通胀率
		adjustment := (stakingRatio - targetRatio) * dpos.config.InflationAdjustment
		newRate := dpos.config.InflationRate - adjustment
		return math.Max(newRate, dpos.config.MinInflation)
	}
}

// GetRewardHistory 获取奖励历史
func (dpos *DPoSConsensus) GetRewardHistory(address types.Address, limit int) ([]*RewardRecord, error) {
	// TODO: 实现奖励历史记录
	// 这里应该从存储中获取历史记录
	return make([]*RewardRecord, 0), nil
}

// RewardRecord 奖励记录
type RewardRecord struct {
	Address     types.Address `json:"address"`
	Amount      uint64        `json:"amount"`
	Type        string        `json:"type"` // "validator" or "delegator"
	Epoch       uint64        `json:"epoch"`
	BlockHeight uint64        `json:"block_height"`
	Timestamp   time.Time     `json:"timestamp"`
}

// GetRewardConfig 获取奖励配置
func (dpos *DPoSConsensus) GetRewardConfig() *RewardConfig {
	return &RewardConfig{
		BlockReward:         1000000,
		ValidatorRate:       dpos.config.RewardRate,
		DelegatorRate:       1 - dpos.config.RewardRate,
		CommunityRate:       0.1,
		InflationRate:       0.08,
		MaxInflation:        0.15,
		MinInflation:        0.02,
		InflationAdjustment: 0.01,
	}
}
