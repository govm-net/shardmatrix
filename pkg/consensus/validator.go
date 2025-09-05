package consensus

import (
	"errors"
	"fmt"

	"github.com/govm-net/shardmatrix/pkg/types"
)

// ValidatorManagerError 验证者管理错误类型
type ValidatorManagerError struct {
	Code    string
	Message string
}

func (e *ValidatorManagerError) Error() string {
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// 错误代码
const (
	ErrValidatorExists      = "VALIDATOR_EXISTS"
	ErrValidatorNotFound    = "VALIDATOR_NOT_FOUND"
	ErrInsufficientStake    = "INSUFFICIENT_STAKE"
	ErrValidatorInactive    = "VALIDATOR_INACTIVE"
	ErrValidatorJailed      = "VALIDATOR_JAILED"
	ErrInvalidCommission    = "INVALID_COMMISSION"
	ErrMaxValidatorsReached = "MAX_VALIDATORS_REACHED"
)

// RegisterValidator 注册验证者
func (dpos *DPoSConsensus) RegisterValidator(address types.Address, stake uint64, commission float64) error {
	dpos.mutex.Lock()
	defer dpos.mutex.Unlock()

	// 检查验证者是否已存在
	if _, exists := dpos.validators[address]; exists {
		return &ValidatorManagerError{
			Code:    ErrValidatorExists,
			Message: fmt.Sprintf("validator %s already exists", address.String()),
		}
	}

	// 检查最小质押量
	if stake < dpos.config.MinStake {
		return &ValidatorManagerError{
			Code:    ErrInsufficientStake,
			Message: fmt.Sprintf("stake %d is less than minimum %d", stake, dpos.config.MinStake),
		}
	}

	// 检查佣金率
	if commission < 0 || commission > 1 {
		return &ValidatorManagerError{
			Code:    ErrInvalidCommission,
			Message: fmt.Sprintf("commission rate %f must be between 0 and 1", commission),
		}
	}

	// 创建验证者（使用空公钥，实际应用中应该提供真实公钥）
	validator := types.NewValidator(address, nil, stake)
	validatorInfo := &ValidatorInfo{
		Validator:      validator,
		TotalStake:     stake,
		SelfStake:      stake,
		DelegatedStake: 0,
		DelegatorCount: 0,
		LastBlockSlot:  0,
		MissedBlocks:   0,
		ProducedBlocks: 0,
		Commission:     commission,
		IsActive:       true,
	}

	dpos.validators[address] = validatorInfo

	// 更新验证者集合
	if err := dpos.updateValidatorSet(); err != nil {
		// 回滚
		delete(dpos.validators, address)
		return fmt.Errorf("failed to update validator set: %v", err)
	}

	fmt.Printf("Validator %s registered with stake %d and commission %f\n",
		address.String(), stake, commission)

	return nil
}

// UnregisterValidator 注销验证者
func (dpos *DPoSConsensus) UnregisterValidator(address types.Address) error {
	dpos.mutex.Lock()
	defer dpos.mutex.Unlock()

	validatorInfo, exists := dpos.validators[address]
	if !exists {
		return &ValidatorManagerError{
			Code:    ErrValidatorNotFound,
			Message: fmt.Sprintf("validator %s not found", address.String()),
		}
	}

	// 检查是否有委托
	if validatorInfo.DelegatorCount > 0 {
		return errors.New("cannot unregister validator with active delegations")
	}

	// 设置为非活跃状态
	validatorInfo.IsActive = false
	validatorInfo.Validator.SetStatus(types.ValidatorInactive)

	// 更新验证者集合
	if err := dpos.updateValidatorSet(); err != nil {
		return fmt.Errorf("failed to update validator set: %v", err)
	}

	fmt.Printf("Validator %s unregistered\n", address.String())
	return nil
}

// AddValidatorStake 增加验证者质押
func (dpos *DPoSConsensus) AddValidatorStake(address types.Address, amount uint64) error {
	dpos.mutex.Lock()
	defer dpos.mutex.Unlock()

	validatorInfo, exists := dpos.validators[address]
	if !exists {
		return &ValidatorManagerError{
			Code:    ErrValidatorNotFound,
			Message: fmt.Sprintf("validator %s not found", address.String()),
		}
	}

	// 更新质押
	validatorInfo.SelfStake += amount
	validatorInfo.TotalStake += amount
	validatorInfo.Validator.AddStake(amount)

	// 更新验证者集合
	if err := dpos.updateValidatorSet(); err != nil {
		return fmt.Errorf("failed to update validator set: %v", err)
	}

	fmt.Printf("Added %d stake to validator %s, total stake: %d\n",
		amount, address.String(), validatorInfo.TotalStake)

	return nil
}

// SubValidatorStake 减少验证者质押
func (dpos *DPoSConsensus) SubValidatorStake(address types.Address, amount uint64) error {
	dpos.mutex.Lock()
	defer dpos.mutex.Unlock()

	validatorInfo, exists := dpos.validators[address]
	if !exists {
		return &ValidatorManagerError{
			Code:    ErrValidatorNotFound,
			Message: fmt.Sprintf("validator %s not found", address.String()),
		}
	}

	// 检查质押是否足够
	if validatorInfo.SelfStake < amount {
		return &ValidatorManagerError{
			Code: ErrInsufficientStake,
			Message: fmt.Sprintf("insufficient self stake: have %d, need %d",
				validatorInfo.SelfStake, amount),
		}
	}

	// 检查减少后是否满足最小质押要求
	newSelfStake := validatorInfo.SelfStake - amount
	newTotalStake := validatorInfo.TotalStake - amount
	if newTotalStake < dpos.config.MinStake {
		return &ValidatorManagerError{
			Code: ErrInsufficientStake,
			Message: fmt.Sprintf("total stake after reduction %d would be less than minimum %d",
				newTotalStake, dpos.config.MinStake),
		}
	}

	// 更新质押
	validatorInfo.SelfStake = newSelfStake
	validatorInfo.TotalStake = newTotalStake
	validatorInfo.Validator.SubStake(amount)

	// 更新验证者集合
	if err := dpos.updateValidatorSet(); err != nil {
		return fmt.Errorf("failed to update validator set: %v", err)
	}

	fmt.Printf("Removed %d stake from validator %s, total stake: %d\n",
		amount, address.String(), validatorInfo.TotalStake)

	return nil
}

// SetValidatorCommission 设置验证者佣金率
func (dpos *DPoSConsensus) SetValidatorCommission(address types.Address, commission float64) error {
	dpos.mutex.Lock()
	defer dpos.mutex.Unlock()

	validatorInfo, exists := dpos.validators[address]
	if !exists {
		return &ValidatorManagerError{
			Code:    ErrValidatorNotFound,
			Message: fmt.Sprintf("validator %s not found", address.String()),
		}
	}

	// 检查佣金率
	if commission < 0 || commission > 1 {
		return &ValidatorManagerError{
			Code:    ErrInvalidCommission,
			Message: fmt.Sprintf("commission rate %f must be between 0 and 1", commission),
		}
	}

	validatorInfo.Commission = commission

	fmt.Printf("Set validator %s commission to %f\n", address.String(), commission)
	return nil
}

// JailValidator 监禁验证者
func (dpos *DPoSConsensus) JailValidator(address types.Address, reason string) error {
	dpos.mutex.Lock()
	defer dpos.mutex.Unlock()

	validatorInfo, exists := dpos.validators[address]
	if !exists {
		return &ValidatorManagerError{
			Code:    ErrValidatorNotFound,
			Message: fmt.Sprintf("validator %s not found", address.String()),
		}
	}

	// 设置为监禁状态
	validatorInfo.IsActive = false
	validatorInfo.Validator.SetStatus(types.ValidatorJailed)

	// 更新验证者集合
	if err := dpos.updateValidatorSet(); err != nil {
		return fmt.Errorf("failed to update validator set: %v", err)
	}

	// 触发惩罚回调
	slashAmount := uint64(float64(validatorInfo.SelfStake) * dpos.config.SlashingRate)
	if dpos.onSlashValidator != nil {
		dpos.onSlashValidator(address, slashAmount)
	}

	fmt.Printf("Validator %s jailed for reason: %s, slashed %d tokens\n",
		address.String(), reason, slashAmount)

	return nil
}

// UnjailValidator 解除监禁验证者
func (dpos *DPoSConsensus) UnjailValidator(address types.Address) error {
	dpos.mutex.Lock()
	defer dpos.mutex.Unlock()

	validatorInfo, exists := dpos.validators[address]
	if !exists {
		return &ValidatorManagerError{
			Code:    ErrValidatorNotFound,
			Message: fmt.Sprintf("validator %s not found", address.String()),
		}
	}

	// 检查是否被监禁
	if validatorInfo.Validator.Status != types.ValidatorJailed {
		return errors.New("validator is not jailed")
	}

	// 检查是否满足最小质押要求
	if validatorInfo.TotalStake < dpos.config.MinStake {
		return &ValidatorManagerError{
			Code: ErrInsufficientStake,
			Message: fmt.Sprintf("total stake %d is less than minimum %d",
				validatorInfo.TotalStake, dpos.config.MinStake),
		}
	}

	// 恢复活跃状态
	validatorInfo.IsActive = true
	validatorInfo.Validator.SetStatus(types.ValidatorActive)

	// 重置错过区块计数
	validatorInfo.MissedBlocks = 0

	// 更新验证者集合
	if err := dpos.updateValidatorSet(); err != nil {
		return fmt.Errorf("failed to update validator set: %v", err)
	}

	fmt.Printf("Validator %s unjailed\n", address.String())
	return nil
}

// GetValidatorList 获取验证者列表
func (dpos *DPoSConsensus) GetValidatorList() []*ValidatorInfo {
	dpos.mutex.RLock()
	defer dpos.mutex.RUnlock()

	validators := make([]*ValidatorInfo, 0, len(dpos.validators))
	for _, validator := range dpos.validators {
		validators = append(validators, dpos.cloneValidatorInfo(validator))
	}

	return validators
}

// GetValidatorCount 获取验证者数量
func (dpos *DPoSConsensus) GetValidatorCount() int {
	dpos.mutex.RLock()
	defer dpos.mutex.RUnlock()

	return len(dpos.validators)
}

// GetActiveValidatorCount 获取活跃验证者数量
func (dpos *DPoSConsensus) GetActiveValidatorCount() int {
	dpos.mutex.RLock()
	defer dpos.mutex.RUnlock()

	count := 0
	for _, validator := range dpos.validators {
		if validator.IsActive {
			count++
		}
	}

	return count
}

// IsValidator 检查地址是否为验证者
func (dpos *DPoSConsensus) IsValidator(address types.Address) bool {
	dpos.mutex.RLock()
	defer dpos.mutex.RUnlock()

	_, exists := dpos.validators[address]
	return exists
}

// IsActiveValidator 检查地址是否为活跃验证者
func (dpos *DPoSConsensus) IsActiveValidator(address types.Address) bool {
	dpos.mutex.RLock()
	defer dpos.mutex.RUnlock()

	validator, exists := dpos.validators[address]
	if !exists {
		return false
	}

	return validator.IsActive
}

// RecordBlockProduced 记录验证者生产区块
func (dpos *DPoSConsensus) RecordBlockProduced(validator types.Address, slot uint64) error {
	dpos.mutex.Lock()
	defer dpos.mutex.Unlock()

	validatorInfo, exists := dpos.validators[validator]
	if !exists {
		return &ValidatorManagerError{
			Code:    ErrValidatorNotFound,
			Message: fmt.Sprintf("validator %s not found", validator.String()),
		}
	}

	validatorInfo.ProducedBlocks++
	validatorInfo.LastBlockSlot = slot
	validatorInfo.Validator.UpdateLastBlock(slot)

	return nil
}

// RecordBlockMissed 记录验证者错过区块
func (dpos *DPoSConsensus) RecordBlockMissed(validator types.Address) error {
	dpos.mutex.Lock()
	defer dpos.mutex.Unlock()

	validatorInfo, exists := dpos.validators[validator]
	if !exists {
		return &ValidatorManagerError{
			Code:    ErrValidatorNotFound,
			Message: fmt.Sprintf("validator %s not found", validator.String()),
		}
	}

	validatorInfo.MissedBlocks++

	// 检查是否需要监禁（错过太多区块）
	maxMissedBlocks := uint64(10) // 最多错过10个区块
	if validatorInfo.MissedBlocks >= maxMissedBlocks {
		return dpos.jailValidatorUnsafe(validator, "missed too many blocks")
	}

	return nil
}

// jailValidatorUnsafe 监禁验证者（不加锁版本）
func (dpos *DPoSConsensus) jailValidatorUnsafe(address types.Address, reason string) error {
	validatorInfo := dpos.validators[address]

	// 设置为监禁状态
	validatorInfo.IsActive = false
	validatorInfo.Validator.SetStatus(types.ValidatorJailed)

	// 更新验证者集合
	if err := dpos.updateValidatorSet(); err != nil {
		return fmt.Errorf("failed to update validator set: %v", err)
	}

	// 触发惩罚回调
	slashAmount := uint64(float64(validatorInfo.SelfStake) * dpos.config.SlashingRate)
	if dpos.onSlashValidator != nil {
		dpos.onSlashValidator(address, slashAmount)
	}

	fmt.Printf("Validator %s jailed for reason: %s, slashed %d tokens\n",
		address.String(), reason, slashAmount)

	return nil
}
