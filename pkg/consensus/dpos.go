package consensus

import (
	"fmt"
	"sort"
	"sync"

	"github.com/lengzhao/shardmatrix/pkg/crypto"
	"github.com/lengzhao/shardmatrix/pkg/storage"
	"github.com/lengzhao/shardmatrix/pkg/types"
)

// DPoSConsensus DPoS共识引擎
type DPoSConsensus struct {
	mu             sync.RWMutex
	storage        storage.Storage
	validatorSet   *types.ValidatorSet
	currentRound   uint64
	timeController *TimeController
	isActive       bool
	keyPair        *crypto.KeyPair
}

// NewDPoSConsensus 创建新的DPoS共识引擎
func NewDPoSConsensus(
	storage storage.Storage,
	timeController *TimeController,
	keyPair *crypto.KeyPair,
) *DPoSConsensus {
	return &DPoSConsensus{
		storage:        storage,
		timeController: timeController,
		keyPair:        keyPair,
		validatorSet: &types.ValidatorSet{
			Validators: make([]types.Validator, 0),
			Round:      0,
		},
	}
}

// Start 启动DPoS共识
func (dpos *DPoSConsensus) Start() error {
	dpos.mu.Lock()
	defer dpos.mu.Unlock()
	
	if dpos.isActive {
		return fmt.Errorf("DPoS consensus already active")
	}
	
	// 加载验证者集合
	err := dpos.loadValidatorSet()
	if err != nil {
		return fmt.Errorf("failed to load validator set: %w", err)
	}
	
	dpos.isActive = true
	return nil
}

// Stop 停止DPoS共识
func (dpos *DPoSConsensus) Stop() {
	dpos.mu.Lock()
	defer dpos.mu.Unlock()
	dpos.isActive = false
}

// loadValidatorSet 加载验证者集合
func (dpos *DPoSConsensus) loadValidatorSet() error {
	validators, err := dpos.storage.GetAllValidators()
	if err != nil {
		return fmt.Errorf("failed to get validators from storage: %w", err)
	}
	
	// 如果没有验证者，创建默认验证者（使用当前节点）
	if len(validators) == 0 {
		defaultValidator := &types.Validator{
			Address:      dpos.keyPair.Address,
			PublicKey:    dpos.keyPair.PublicKey,
			StakeAmount:  1000000, // 默认质押金额
			DelegatedAmt: 0,
			VotePower:    1000000,
			IsActive:     true,
			SlashCount:   0,
		}
		
		err = dpos.storage.SaveValidator(defaultValidator)
		if err != nil {
			return fmt.Errorf("failed to save default validator: %w", err)
		}
		
		validators = []*types.Validator{defaultValidator}
	}
	
	// 转换为ValidatorSet
	validatorList := make([]types.Validator, len(validators))
	for i, v := range validators {
		validatorList[i] = *v
	}
	
	dpos.validatorSet = &types.ValidatorSet{
		Validators: validatorList,
		Round:      0,
	}
	
	// 排序验证者（按投票权重降序）
	dpos.sortValidators()
	
	return nil
}

// sortValidators 排序验证者
func (dpos *DPoSConsensus) sortValidators() {
	sort.Slice(dpos.validatorSet.Validators, func(i, j int) bool {
		return dpos.validatorSet.Validators[i].VotePower > dpos.validatorSet.Validators[j].VotePower
	})
}

// GetCurrentValidator 获取当前应该出块的验证者
func (dpos *DPoSConsensus) GetCurrentValidator(blockHeight uint64) (*types.Validator, error) {
	dpos.mu.RLock()
	defer dpos.mu.RUnlock()
	
	activeValidators := dpos.getActiveValidators()
	if len(activeValidators) == 0 {
		return nil, fmt.Errorf("no active validators")
	}
	
	// 计算当前轮次的验证者索引
	index := blockHeight % uint64(len(activeValidators))
	return &activeValidators[index], nil
}

// getActiveValidators 获取活跃的验证者列表
func (dpos *DPoSConsensus) getActiveValidators() []types.Validator {
	var activeValidators []types.Validator
	for _, v := range dpos.validatorSet.Validators {
		if v.IsActive {
			activeValidators = append(activeValidators, v)
		}
	}
	return activeValidators
}

// IsValidator 检查地址是否为验证者
func (dpos *DPoSConsensus) IsValidator(address types.Address) bool {
	dpos.mu.RLock()
	defer dpos.mu.RUnlock()
	
	validator := dpos.validatorSet.GetValidator(address)
	return validator != nil && validator.IsActive
}

// IsCurrentValidator 检查地址是否为当前轮次的验证者
func (dpos *DPoSConsensus) IsCurrentValidator(address types.Address, blockHeight uint64) bool {
	currentValidator, err := dpos.GetCurrentValidator(blockHeight)
	if err != nil {
		return false
	}
	
	return currentValidator.Address == address
}

// ValidateBlock 验证区块
func (dpos *DPoSConsensus) ValidateBlock(block *types.Block) error {
	dpos.mu.RLock()
	defer dpos.mu.RUnlock()
	
	// 验证区块生产者是否为合法验证者
	validator := dpos.validatorSet.GetValidator(block.Header.Validator)
	if validator == nil {
		return fmt.Errorf("block producer is not a validator: %s", block.Header.Validator.String())
	}
	
	if !validator.IsActive {
		return fmt.Errorf("block producer is not active: %s", block.Header.Validator.String())
	}
	
	// 验证是否为当前轮次的验证者
	if !dpos.IsCurrentValidator(block.Header.Validator, block.Header.Number) {
		return fmt.Errorf("block producer is not current validator for height %d", block.Header.Number)
	}
	
	// 验证区块签名
	if !crypto.VerifyBlock(&block.Header, validator.PublicKey) {
		return fmt.Errorf("invalid block signature")
	}
	
	// 验证时间戳
	if dpos.timeController != nil {
		validator := NewStrictTimeValidator(dpos.timeController)
		if err := validator.ValidateBlock(block); err != nil {
			return fmt.Errorf("block time validation failed: %w", err)
		}
	}
	
	return nil
}

// AddValidator 添加验证者
func (dpos *DPoSConsensus) AddValidator(validator *types.Validator) error {
	dpos.mu.Lock()
	defer dpos.mu.Unlock()
	
	// 检查验证者是否已存在
	if dpos.validatorSet.GetValidator(validator.Address) != nil {
		return fmt.Errorf("validator already exists: %s", validator.Address.String())
	}
	
	// 验证最小质押金额
	if validator.StakeAmount < types.MinStakeAmount {
		return fmt.Errorf("insufficient stake amount: %d < %d", validator.StakeAmount, types.MinStakeAmount)
	}
	
	// 计算投票权重
	validator.VotePower = validator.StakeAmount + validator.DelegatedAmt
	validator.IsActive = true
	
	// 添加到验证者集合
	dpos.validatorSet.Validators = append(dpos.validatorSet.Validators, *validator)
	
	// 重新排序
	dpos.sortValidators()
	
	// 限制验证者数量为21个
	if len(dpos.validatorSet.Validators) > types.ValidatorCount {
		// 移除投票权重最小的验证者
		removed := dpos.validatorSet.Validators[types.ValidatorCount:]
		dpos.validatorSet.Validators = dpos.validatorSet.Validators[:types.ValidatorCount]
		
		// 将被移除的验证者设为非活跃状态
		for _, v := range removed {
			v.IsActive = false
			dpos.storage.SaveValidator(&v)
		}
	}
	
	// 保存验证者到存储
	err := dpos.storage.SaveValidator(validator)
	if err != nil {
		return fmt.Errorf("failed to save validator: %w", err)
	}
	
	return nil
}

// RemoveValidator 移除验证者
func (dpos *DPoSConsensus) RemoveValidator(address types.Address) error {
	dpos.mu.Lock()
	defer dpos.mu.Unlock()
	
	// 查找验证者索引
	index := -1
	for i, v := range dpos.validatorSet.Validators {
		if v.Address == address {
			index = i
			break
		}
	}
	
	if index == -1 {
		return fmt.Errorf("validator not found: %s", address.String())
	}
	
	// 设置为非活跃状态
	dpos.validatorSet.Validators[index].IsActive = false
	
	// 保存更新到存储
	err := dpos.storage.SaveValidator(&dpos.validatorSet.Validators[index])
	if err != nil {
		return fmt.Errorf("failed to update validator: %w", err)
	}
	
	return nil
}

// Delegate 委托质押
func (dpos *DPoSConsensus) Delegate(delegator types.Address, validator types.Address, amount uint64) error {
	dpos.mu.Lock()
	defer dpos.mu.Unlock()
	
	// 获取验证者
	v := dpos.validatorSet.GetValidator(validator)
	if v == nil {
		return fmt.Errorf("validator not found: %s", validator.String())
	}
	
	// 更新委托金额和投票权重
	v.DelegatedAmt += amount
	v.VotePower = v.StakeAmount + v.DelegatedAmt
	
	// 重新排序
	dpos.sortValidators()
	
	// 保存更新
	err := dpos.storage.SaveValidator(v)
	if err != nil {
		return fmt.Errorf("failed to update validator: %w", err)
	}
	
	return nil
}

// Undelegate 取消委托
func (dpos *DPoSConsensus) Undelegate(delegator types.Address, validator types.Address, amount uint64) error {
	dpos.mu.Lock()
	defer dpos.mu.Unlock()
	
	// 获取验证者
	v := dpos.validatorSet.GetValidator(validator)
	if v == nil {
		return fmt.Errorf("validator not found: %s", validator.String())
	}
	
	// 检查委托金额是否足够
	if v.DelegatedAmt < amount {
		return fmt.Errorf("insufficient delegated amount: %d < %d", v.DelegatedAmt, amount)
	}
	
	// 更新委托金额和投票权重
	v.DelegatedAmt -= amount
	v.VotePower = v.StakeAmount + v.DelegatedAmt
	
	// 重新排序
	dpos.sortValidators()
	
	// 保存更新
	err := dpos.storage.SaveValidator(v)
	if err != nil {
		return fmt.Errorf("failed to update validator: %w", err)
	}
	
	return nil
}

// SlashValidator 惩罚验证者
func (dpos *DPoSConsensus) SlashValidator(address types.Address, reason string) error {
	dpos.mu.Lock()
	defer dpos.mu.Unlock()
	
	// 获取验证者
	v := dpos.validatorSet.GetValidator(address)
	if v == nil {
		return fmt.Errorf("validator not found: %s", address.String())
	}
	
	// 增加惩罚计数
	v.SlashCount++
	
	// 根据惩罚次数决定处理方式
	if v.SlashCount >= 3 {
		// 多次违规，设为非活跃状态
		v.IsActive = false
	} else {
		// 减少部分质押作为惩罚
		slashAmount := v.StakeAmount / 10 // 惩罚10%
		if v.StakeAmount > slashAmount {
			v.StakeAmount -= slashAmount
			v.VotePower = v.StakeAmount + v.DelegatedAmt
		}
	}
	
	// 重新排序
	dpos.sortValidators()
	
	// 保存更新
	err := dpos.storage.SaveValidator(v)
	if err != nil {
		return fmt.Errorf("failed to update validator: %w", err)
	}
	
	return nil
}

// GetValidatorSet 获取验证者集合
func (dpos *DPoSConsensus) GetValidatorSet() *types.ValidatorSet {
	dpos.mu.RLock()
	defer dpos.mu.RUnlock()
	
	// 返回副本以避免并发修改
	validatorsCopy := make([]types.Validator, len(dpos.validatorSet.Validators))
	copy(validatorsCopy, dpos.validatorSet.Validators)
	
	return &types.ValidatorSet{
		Validators: validatorsCopy,
		Round:      dpos.validatorSet.Round,
	}
}

// GetValidatorByIndex 根据索引获取验证者
func (dpos *DPoSConsensus) GetValidatorByIndex(index int) *types.Validator {
	dpos.mu.RLock()
	defer dpos.mu.RUnlock()
	
	activeValidators := dpos.getActiveValidators()
	if index < 0 || index >= len(activeValidators) {
		return nil
	}
	
	return &activeValidators[index]
}

// GetActiveValidatorCount 获取活跃验证者数量
func (dpos *DPoSConsensus) GetActiveValidatorCount() int {
	dpos.mu.RLock()
	defer dpos.mu.RUnlock()
	
	return len(dpos.getActiveValidators())
}

// GetTotalVotePower 获取总投票权重
func (dpos *DPoSConsensus) GetTotalVotePower() uint64 {
	dpos.mu.RLock()
	defer dpos.mu.RUnlock()
	
	return dpos.validatorSet.GetTotalVotePower()
}

// GetConsensusInfo 获取共识信息
func (dpos *DPoSConsensus) GetConsensusInfo() map[string]interface{} {
	dpos.mu.RLock()
	defer dpos.mu.RUnlock()
	
	activeValidators := dpos.getActiveValidators()
	
	return map[string]interface{}{
		"consensus_type":        "DPoS",
		"is_active":             dpos.isActive,
		"total_validators":      len(dpos.validatorSet.Validators),
		"active_validators":     len(activeValidators),
		"current_round":         dpos.validatorSet.Round,
		"total_vote_power":      dpos.GetTotalVotePower(),
		"validator_addresses":   dpos.getValidatorAddresses(activeValidators),
	}
}

// getValidatorAddresses 获取验证者地址列表
func (dpos *DPoSConsensus) getValidatorAddresses(validators []types.Validator) []string {
	addresses := make([]string, len(validators))
	for i, v := range validators {
		addresses[i] = v.Address.String()
	}
	return addresses
}

// UpdateRound 更新轮次
func (dpos *DPoSConsensus) UpdateRound(round uint64) {
	dpos.mu.Lock()
	defer dpos.mu.Unlock()
	dpos.validatorSet.Round = round
}

// IsActive 检查共识是否活跃
func (dpos *DPoSConsensus) IsActive() bool {
	dpos.mu.RLock()
	defer dpos.mu.RUnlock()
	return dpos.isActive
}