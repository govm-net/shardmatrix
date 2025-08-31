package consensus

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/govm-net/shardmatrix/pkg/types"
)

// DPoSConfig DPoS共识配置
type DPoSConfig struct {
	ValidatorCount      int           `json:"validator_count"`      // 验证者数量
	BlockInterval       time.Duration `json:"block_interval"`       // 出块间隔
	EpochDuration       time.Duration `json:"epoch_duration"`       // 轮次持续时间
	MinStake            uint64        `json:"min_stake"`            // 最小质押量
	SlashingRate        float64       `json:"slashing_rate"`        // 惩罚比例
	RewardRate          float64       `json:"reward_rate"`          // 奖励比例
	MaxDelegators       int           `json:"max_delegators"`       // 最大委托人数量
	UnbondingPeriod     time.Duration `json:"unbonding_period"`     // 解绑期
	InflationRate       float64       `json:"inflation_rate"`       // 年通胀率
	MaxInflation        float64       `json:"max_inflation"`        // 最大通胀率
	MinInflation        float64       `json:"min_inflation"`        // 最小通胀率
	InflationAdjustment float64       `json:"inflation_adjustment"` // 通胀调整系数
}

// DefaultDPoSConfig 默认DPoS配置
func DefaultDPoSConfig() *DPoSConfig {
	return &DPoSConfig{
		ValidatorCount:      21,                   // 21个验证者
		BlockInterval:       3 * time.Second,      // 3秒出块
		EpochDuration:       21 * 3 * time.Second, // 21个区块为一轮
		MinStake:            1000000,              // 最小质押100万
		SlashingRate:        0.05,                 // 5%惩罚
		RewardRate:          0.1,                  // 10%奖励
		MaxDelegators:       1000,                 // 最多1000个委托人
		UnbondingPeriod:     7 * 24 * time.Hour,   // 7天解绑期
		InflationRate:       0.08,                 // 8%年通胀率
		MaxInflation:        0.15,                 // 最大15%
		MinInflation:        0.02,                 // 最小2%
		InflationAdjustment: 0.01,                 // 1%调整系数
	}
}

// DPoSConsensus DPoS共识引擎
type DPoSConsensus struct {
	config *DPoSConfig
	mutex  sync.RWMutex

	// 状态数据
	validators    map[types.Address]*ValidatorInfo
	delegations   map[string]*DelegationInfo // key: delegator_address:validator_address
	currentEpoch  uint64
	currentSlot   uint64
	lastBlockTime time.Time
	isProducing   bool

	// 轮转状态
	activeValidators []*ValidatorInfo
	currentProducer  types.Address
	nextSlotTime     time.Time

	// 回调函数
	onValidatorSet    func([]*ValidatorInfo)
	onSlotChange      func(types.Address, uint64)
	onEpochChange     func(uint64)
	onSlashValidator  func(types.Address, uint64)
	onRewardValidator func(types.Address, uint64)

	// 上下文
	ctx    context.Context
	cancel context.CancelFunc
}

// ValidatorInfo 验证者信息
type ValidatorInfo struct {
	Validator      *types.Validator `json:"validator"`
	TotalStake     uint64           `json:"total_stake"`     // 总质押（包含委托）
	SelfStake      uint64           `json:"self_stake"`      // 自质押
	DelegatedStake uint64           `json:"delegated_stake"` // 委托质押
	DelegatorCount int              `json:"delegator_count"` // 委托人数量
	LastBlockSlot  uint64           `json:"last_block_slot"` // 最后出块槽位
	MissedBlocks   uint64           `json:"missed_blocks"`   // 错过区块数
	ProducedBlocks uint64           `json:"produced_blocks"` // 生产区块数
	Commission     float64          `json:"commission"`      // 佣金率
	IsActive       bool             `json:"is_active"`       // 是否活跃
}

// DelegationInfo 委托信息
type DelegationInfo struct {
	Delegator    types.Address `json:"delegator"`
	Validator    types.Address `json:"validator"`
	Amount       uint64        `json:"amount"`
	CreatedAt    time.Time     `json:"created_at"`
	LastRewardAt time.Time     `json:"last_reward_at"`
	UnbondingAt  *time.Time    `json:"unbonding_at,omitempty"`
	IsUnbonding  bool          `json:"is_unbonding"`
}

// NewDPoSConsensus 创建DPoS共识引擎
func NewDPoSConsensus(config *DPoSConfig) *DPoSConsensus {
	if config == nil {
		config = DefaultDPoSConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &DPoSConsensus{
		config:           config,
		validators:       make(map[types.Address]*ValidatorInfo),
		delegations:      make(map[string]*DelegationInfo),
		currentEpoch:     0,
		currentSlot:      0,
		lastBlockTime:    time.Now(),
		isProducing:      false,
		activeValidators: make([]*ValidatorInfo, 0),
		ctx:              ctx,
		cancel:           cancel,
	}
}

// Start 启动DPoS共识
func (dpos *DPoSConsensus) Start() error {
	dpos.mutex.Lock()
	defer dpos.mutex.Unlock()

	if dpos.isProducing {
		return errors.New("DPoS consensus already started")
	}

	// 初始化验证者集合
	if err := dpos.updateValidatorSet(); err != nil {
		return fmt.Errorf("failed to update validator set: %v", err)
	}

	// 计算下一个出块时间
	dpos.calculateNextSlotTime()

	dpos.isProducing = true

	fmt.Printf("DPoS consensus started with %d validators\n", len(dpos.activeValidators))
	return nil
}

// Stop 停止DPoS共识
func (dpos *DPoSConsensus) Stop() error {
	dpos.mutex.Lock()
	defer dpos.mutex.Unlock()

	if !dpos.isProducing {
		return nil
	}

	dpos.cancel()
	dpos.isProducing = false

	fmt.Println("DPoS consensus stopped")
	return nil
}

// IsProducing 检查是否在生产区块
func (dpos *DPoSConsensus) IsProducing() bool {
	dpos.mutex.RLock()
	defer dpos.mutex.RUnlock()
	return dpos.isProducing
}

// GetCurrentProducer 获取当前出块者
func (dpos *DPoSConsensus) GetCurrentProducer() types.Address {
	dpos.mutex.RLock()
	defer dpos.mutex.RUnlock()
	return dpos.currentProducer
}

// GetNextSlotTime 获取下一个出块时间
func (dpos *DPoSConsensus) GetNextSlotTime() time.Time {
	dpos.mutex.RLock()
	defer dpos.mutex.RUnlock()
	return dpos.nextSlotTime
}

// GetCurrentEpoch 获取当前轮次
func (dpos *DPoSConsensus) GetCurrentEpoch() uint64 {
	dpos.mutex.RLock()
	defer dpos.mutex.RUnlock()
	return dpos.currentEpoch
}

// GetCurrentSlot 获取当前槽位
func (dpos *DPoSConsensus) GetCurrentSlot() uint64 {
	dpos.mutex.RLock()
	defer dpos.mutex.RUnlock()
	return dpos.currentSlot
}

// GetActiveValidators 获取活跃验证者
func (dpos *DPoSConsensus) GetActiveValidators() []*ValidatorInfo {
	dpos.mutex.RLock()
	defer dpos.mutex.RUnlock()

	// 返回副本以避免并发修改
	result := make([]*ValidatorInfo, len(dpos.activeValidators))
	copy(result, dpos.activeValidators)
	return result
}

// GetValidatorInfo 获取验证者信息
func (dpos *DPoSConsensus) GetValidatorInfo(address types.Address) (*ValidatorInfo, bool) {
	dpos.mutex.RLock()
	defer dpos.mutex.RUnlock()

	info, exists := dpos.validators[address]
	if !exists {
		return nil, false
	}

	// 返回副本
	return dpos.cloneValidatorInfo(info), true
}

// GetDelegationInfo 获取委托信息
func (dpos *DPoSConsensus) GetDelegationInfo(delegator, validator types.Address) (*DelegationInfo, bool) {
	dpos.mutex.RLock()
	defer dpos.mutex.RUnlock()

	key := fmt.Sprintf("%s:%s", delegator.String(), validator.String())
	info, exists := dpos.delegations[key]
	if !exists {
		return nil, false
	}

	// 返回副本
	return dpos.cloneDelegationInfo(info), true
}

// calculateNextSlotTime 计算下一个出块时间
func (dpos *DPoSConsensus) calculateNextSlotTime() {
	now := time.Now()

	// 计算下一个槽位的时间
	slotDuration := dpos.config.BlockInterval
	nextSlot := ((now.UnixNano() / slotDuration.Nanoseconds()) + 1) * slotDuration.Nanoseconds()
	dpos.nextSlotTime = time.Unix(0, nextSlot)

	// 更新当前槽位 - 使用相对槽位而不是绝对时间
	if dpos.currentSlot == 0 {
		// 初始化时，从0开始
		dpos.currentSlot = 0
		dpos.currentEpoch = 0
	} else {
		// 正常递增
		dpos.currentSlot++
		// 计算当前轮次
		slotsPerEpoch := uint64(dpos.config.EpochDuration / dpos.config.BlockInterval)
		if slotsPerEpoch > 0 {
			dpos.currentEpoch = dpos.currentSlot / slotsPerEpoch
		}
	}

	// 确定当前出块者
	if len(dpos.activeValidators) > 0 {
		validatorIndex := dpos.currentSlot % uint64(len(dpos.activeValidators))
		dpos.currentProducer = dpos.activeValidators[validatorIndex].Validator.Address
	}
}

// updateValidatorSet 更新验证者集合
func (dpos *DPoSConsensus) updateValidatorSet() error {
	// 获取所有活跃验证者
	activeValidators := make([]*ValidatorInfo, 0)

	for _, validator := range dpos.validators {
		if validator.IsActive && validator.TotalStake >= dpos.config.MinStake {
			activeValidators = append(activeValidators, validator)
		}
	}

	// 按质押量排序
	sort.Slice(activeValidators, func(i, j int) bool {
		return activeValidators[i].TotalStake > activeValidators[j].TotalStake
	})

	// 取前N个验证者
	maxValidators := dpos.config.ValidatorCount
	if len(activeValidators) > maxValidators {
		activeValidators = activeValidators[:maxValidators]
	}

	dpos.activeValidators = activeValidators

	// 触发回调
	if dpos.onValidatorSet != nil {
		dpos.onValidatorSet(activeValidators)
	}

	return nil
}

// cloneValidatorInfo 克隆验证者信息
func (dpos *DPoSConsensus) cloneValidatorInfo(info *ValidatorInfo) *ValidatorInfo {
	return &ValidatorInfo{
		Validator:      info.Validator.Clone(),
		TotalStake:     info.TotalStake,
		SelfStake:      info.SelfStake,
		DelegatedStake: info.DelegatedStake,
		DelegatorCount: info.DelegatorCount,
		LastBlockSlot:  info.LastBlockSlot,
		MissedBlocks:   info.MissedBlocks,
		ProducedBlocks: info.ProducedBlocks,
		Commission:     info.Commission,
		IsActive:       info.IsActive,
	}
}

// cloneDelegationInfo 克隆委托信息
func (dpos *DPoSConsensus) cloneDelegationInfo(info *DelegationInfo) *DelegationInfo {
	clone := &DelegationInfo{
		Delegator:    info.Delegator,
		Validator:    info.Validator,
		Amount:       info.Amount,
		CreatedAt:    info.CreatedAt,
		LastRewardAt: info.LastRewardAt,
		IsUnbonding:  info.IsUnbonding,
	}

	if info.UnbondingAt != nil {
		unbondingAt := *info.UnbondingAt
		clone.UnbondingAt = &unbondingAt
	}

	return clone
}

// GetConfig 获取配置
func (dpos *DPoSConsensus) GetConfig() *DPoSConfig {
	return dpos.config
}

// SetCallbacks 设置回调函数
func (dpos *DPoSConsensus) SetCallbacks(
	onValidatorSet func([]*ValidatorInfo),
	onSlotChange func(types.Address, uint64),
	onEpochChange func(uint64),
	onSlashValidator func(types.Address, uint64),
	onRewardValidator func(types.Address, uint64),
) {
	dpos.onValidatorSet = onValidatorSet
	dpos.onSlotChange = onSlotChange
	dpos.onEpochChange = onEpochChange
	dpos.onSlashValidator = onSlashValidator
	dpos.onRewardValidator = onRewardValidator
}
