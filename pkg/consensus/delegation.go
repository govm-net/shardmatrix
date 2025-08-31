package consensus

import (
	"errors"
	"fmt"
	"time"

	"github.com/govm-net/shardmatrix/pkg/types"
)

// DelegationManagerError 委托管理错误类型
type DelegationManagerError struct {
	Code    string
	Message string
}

func (e *DelegationManagerError) Error() string {
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// 委托相关错误代码
const (
	ErrDelegationExists     = "DELEGATION_EXISTS"
	ErrDelegationNotFound   = "DELEGATION_NOT_FOUND"
	ErrInsufficientAmount   = "INSUFFICIENT_AMOUNT"
	ErrMaxDelegatorsReached = "MAX_DELEGATORS_REACHED"
	ErrSelfDelegation       = "SELF_DELEGATION"
	ErrDelegationUnbonding  = "DELEGATION_UNBONDING"
	ErrUnbondingPending     = "UNBONDING_PENDING"
)

// Delegate 委托权益给验证者
func (dpos *DPoSConsensus) Delegate(delegator types.Address, validator types.Address, amount uint64) error {
	dpos.mutex.Lock()
	defer dpos.mutex.Unlock()

	// 检查验证者是否存在
	validatorInfo, exists := dpos.validators[validator]
	if !exists {
		return &DelegationManagerError{
			Code:    ErrValidatorNotFound,
			Message: fmt.Sprintf("validator %s not found", validator.String()),
		}
	}

	// 检查验证者是否活跃
	if !validatorInfo.IsActive {
		return &DelegationManagerError{
			Code:    ErrValidatorInactive,
			Message: fmt.Sprintf("validator %s is not active", validator.String()),
		}
	}

	// 检查是否为自委托（验证者委托给自己）
	if delegator.Equal(validator) {
		return &DelegationManagerError{
			Code:    ErrSelfDelegation,
			Message: "validator cannot delegate to itself",
		}
	}

	// 检查委托金额
	if amount == 0 {
		return &DelegationManagerError{
			Code:    ErrInsufficientAmount,
			Message: "delegation amount must be greater than 0",
		}
	}

	// 检查是否已达到最大委托人数
	if validatorInfo.DelegatorCount >= dpos.config.MaxDelegators {
		return &DelegationManagerError{
			Code:    ErrMaxDelegatorsReached,
			Message: fmt.Sprintf("validator %s has reached maximum delegators limit", validator.String()),
		}
	}

	// 创建委托键
	delegationKey := fmt.Sprintf("%s:%s", delegator.String(), validator.String())

	// 检查是否已存在委托
	existingDelegation, exists := dpos.delegations[delegationKey]
	if exists {
		// 如果正在解绑，不允许新增委托
		if existingDelegation.IsUnbonding {
			return &DelegationManagerError{
				Code:    ErrDelegationUnbonding,
				Message: "existing delegation is unbonding, cannot add more",
			}
		}

		// 增加委托金额
		existingDelegation.Amount += amount
		existingDelegation.LastRewardAt = time.Now()
	} else {
		// 创建新委托
		delegation := &DelegationInfo{
			Delegator:    delegator,
			Validator:    validator,
			Amount:       amount,
			CreatedAt:    time.Now(),
			LastRewardAt: time.Now(),
			IsUnbonding:  false,
		}
		dpos.delegations[delegationKey] = delegation

		// 增加委托人数量
		validatorInfo.DelegatorCount++
	}

	// 更新验证者委托质押
	validatorInfo.DelegatedStake += amount
	validatorInfo.TotalStake += amount

	// 更新验证者集合
	if err := dpos.updateValidatorSet(); err != nil {
		return fmt.Errorf("failed to update validator set: %v", err)
	}

	fmt.Printf("Delegated %d tokens from %s to validator %s\n",
		amount, delegator.String(), validator.String())

	return nil
}

// Undelegate 取消委托
func (dpos *DPoSConsensus) Undelegate(delegator types.Address, validator types.Address, amount uint64) error {
	dpos.mutex.Lock()
	defer dpos.mutex.Unlock()

	// 创建委托键
	delegationKey := fmt.Sprintf("%s:%s", delegator.String(), validator.String())

	// 检查委托是否存在
	delegation, exists := dpos.delegations[delegationKey]
	if !exists {
		return &DelegationManagerError{
			Code: ErrDelegationNotFound,
			Message: fmt.Sprintf("delegation from %s to %s not found",
				delegator.String(), validator.String()),
		}
	}

	// 检查是否已在解绑中
	if delegation.IsUnbonding {
		return &DelegationManagerError{
			Code:    ErrDelegationUnbonding,
			Message: "delegation is already unbonding",
		}
	}

	// 检查委托金额是否足够
	if delegation.Amount < amount {
		return &DelegationManagerError{
			Code: ErrInsufficientAmount,
			Message: fmt.Sprintf("insufficient delegation amount: have %d, need %d",
				delegation.Amount, amount),
		}
	}

	// 获取验证者信息
	validatorInfo, exists := dpos.validators[validator]
	if !exists {
		return &DelegationManagerError{
			Code:    ErrValidatorNotFound,
			Message: fmt.Sprintf("validator %s not found", validator.String()),
		}
	}

	// 如果是部分取消委托
	if delegation.Amount > amount {
		// 减少委托金额
		delegation.Amount -= amount
	} else {
		// 完全取消委托，设置解绑状态
		delegation.IsUnbonding = true
		unbondingTime := time.Now().Add(dpos.config.UnbondingPeriod)
		delegation.UnbondingAt = &unbondingTime

		// 减少委托人数量
		validatorInfo.DelegatorCount--
	}

	// 更新验证者委托质押
	validatorInfo.DelegatedStake -= amount
	validatorInfo.TotalStake -= amount

	// 更新验证者集合
	if err := dpos.updateValidatorSet(); err != nil {
		return fmt.Errorf("failed to update validator set: %v", err)
	}

	fmt.Printf("Undelegated %d tokens from %s to validator %s\n",
		amount, delegator.String(), validator.String())

	return nil
}

// CompleteDelegation 完成解绑委托
func (dpos *DPoSConsensus) CompleteDelegation(delegator types.Address, validator types.Address) error {
	dpos.mutex.Lock()
	defer dpos.mutex.Unlock()

	// 创建委托键
	delegationKey := fmt.Sprintf("%s:%s", delegator.String(), validator.String())

	// 检查委托是否存在
	delegation, exists := dpos.delegations[delegationKey]
	if !exists {
		return &DelegationManagerError{
			Code: ErrDelegationNotFound,
			Message: fmt.Sprintf("delegation from %s to %s not found",
				delegator.String(), validator.String()),
		}
	}

	// 检查是否在解绑中
	if !delegation.IsUnbonding {
		return errors.New("delegation is not unbonding")
	}

	// 检查解绑期是否已过
	if delegation.UnbondingAt == nil || time.Now().Before(*delegation.UnbondingAt) {
		return &DelegationManagerError{
			Code: ErrUnbondingPending,
			Message: fmt.Sprintf("unbonding period not yet complete, available at %v",
				delegation.UnbondingAt),
		}
	}

	// 删除委托记录
	delete(dpos.delegations, delegationKey)

	fmt.Printf("Completed unbonding delegation from %s to validator %s\n",
		delegator.String(), validator.String())

	return nil
}

// GetDelegation 获取单个委托信息
func (dpos *DPoSConsensus) GetDelegation(delegator types.Address, validator types.Address) (*DelegationInfo, error) {
	dpos.mutex.RLock()
	defer dpos.mutex.RUnlock()

	delegationKey := fmt.Sprintf("%s:%s", delegator.String(), validator.String())
	delegation, exists := dpos.delegations[delegationKey]
	if !exists {
		return nil, &DelegationManagerError{
			Code: ErrDelegationNotFound,
			Message: fmt.Sprintf("delegation from %s to %s not found",
				delegator.String(), validator.String()),
		}
	}

	return dpos.cloneDelegationInfo(delegation), nil
}

// GetDelegatorDelegations 获取委托人的所有委托
func (dpos *DPoSConsensus) GetDelegatorDelegations(delegator types.Address) []*DelegationInfo {
	dpos.mutex.RLock()
	defer dpos.mutex.RUnlock()

	delegations := make([]*DelegationInfo, 0)
	for _, delegation := range dpos.delegations {
		if delegation.Delegator.Equal(delegator) {
			delegations = append(delegations, dpos.cloneDelegationInfo(delegation))
		}
	}

	return delegations
}

// GetValidatorDelegations 获取验证者的所有委托
func (dpos *DPoSConsensus) GetValidatorDelegations(validator types.Address) []*DelegationInfo {
	dpos.mutex.RLock()
	defer dpos.mutex.RUnlock()

	delegations := make([]*DelegationInfo, 0)
	for _, delegation := range dpos.delegations {
		if delegation.Validator.Equal(validator) {
			delegations = append(delegations, dpos.cloneDelegationInfo(delegation))
		}
	}

	return delegations
}

// GetAllDelegations 获取所有委托
func (dpos *DPoSConsensus) GetAllDelegations() []*DelegationInfo {
	dpos.mutex.RLock()
	defer dpos.mutex.RUnlock()

	delegations := make([]*DelegationInfo, 0, len(dpos.delegations))
	for _, delegation := range dpos.delegations {
		delegations = append(delegations, dpos.cloneDelegationInfo(delegation))
	}

	return delegations
}

// GetDelegationCount 获取委托总数
func (dpos *DPoSConsensus) GetDelegationCount() int {
	dpos.mutex.RLock()
	defer dpos.mutex.RUnlock()

	return len(dpos.delegations)
}

// GetDelegatorCount 获取委托人总数
func (dpos *DPoSConsensus) GetDelegatorCount() int {
	dpos.mutex.RLock()
	defer dpos.mutex.RUnlock()

	delegators := make(map[string]bool)
	for _, delegation := range dpos.delegations {
		delegators[delegation.Delegator.String()] = true
	}

	return len(delegators)
}

// GetUnbondingDelegations 获取解绑中的委托
func (dpos *DPoSConsensus) GetUnbondingDelegations() []*DelegationInfo {
	dpos.mutex.RLock()
	defer dpos.mutex.RUnlock()

	unbondingDelegations := make([]*DelegationInfo, 0)
	for _, delegation := range dpos.delegations {
		if delegation.IsUnbonding {
			unbondingDelegations = append(unbondingDelegations, dpos.cloneDelegationInfo(delegation))
		}
	}

	return unbondingDelegations
}

// ProcessUnbondingDelegations 处理到期的解绑委托
func (dpos *DPoSConsensus) ProcessUnbondingDelegations() []string {
	dpos.mutex.Lock()
	defer dpos.mutex.Unlock()

	now := time.Now()
	completedDelegations := make([]string, 0)

	for key, delegation := range dpos.delegations {
		if delegation.IsUnbonding &&
			delegation.UnbondingAt != nil &&
			now.After(*delegation.UnbondingAt) {

			// 删除已完成解绑的委托
			delete(dpos.delegations, key)
			completedDelegations = append(completedDelegations, key)

			fmt.Printf("Completed unbonding delegation: %s\n", key)
		}
	}

	return completedDelegations
}

// CalculateDelegatorReward 计算委托人奖励
func (dpos *DPoSConsensus) CalculateDelegatorReward(
	delegator types.Address,
	validator types.Address,
	totalReward uint64,
) (uint64, error) {
	dpos.mutex.RLock()
	defer dpos.mutex.RUnlock()

	// 获取委托信息
	delegationKey := fmt.Sprintf("%s:%s", delegator.String(), validator.String())
	delegation, exists := dpos.delegations[delegationKey]
	if !exists {
		return 0, &DelegationManagerError{
			Code: ErrDelegationNotFound,
			Message: fmt.Sprintf("delegation from %s to %s not found",
				delegator.String(), validator.String()),
		}
	}

	// 获取验证者信息
	validatorInfo, exists := dpos.validators[validator]
	if !exists {
		return 0, &DelegationManagerError{
			Code:    ErrValidatorNotFound,
			Message: fmt.Sprintf("validator %s not found", validator.String()),
		}
	}

	// 计算委托人在总委托中的比例
	delegationRatio := float64(delegation.Amount) / float64(validatorInfo.DelegatedStake)

	// 计算分配给委托人的总奖励（扣除验证者佣金）
	delegatorTotalReward := float64(totalReward) * (1 - validatorInfo.Commission)

	// 计算该委托人的奖励
	delegatorReward := uint64(delegatorTotalReward * delegationRatio)

	return delegatorReward, nil
}

// GetTotalDelegatedStake 获取总委托质押量
func (dpos *DPoSConsensus) GetTotalDelegatedStake() uint64 {
	dpos.mutex.RLock()
	defer dpos.mutex.RUnlock()

	totalDelegated := uint64(0)
	for _, delegation := range dpos.delegations {
		if !delegation.IsUnbonding {
			totalDelegated += delegation.Amount
		}
	}

	return totalDelegated
}

// GetValidatorDelegatedStake 获取验证者的委托质押量
func (dpos *DPoSConsensus) GetValidatorDelegatedStake(validator types.Address) uint64 {
	dpos.mutex.RLock()
	defer dpos.mutex.RUnlock()

	validatorInfo, exists := dpos.validators[validator]
	if !exists {
		return 0
	}

	return validatorInfo.DelegatedStake
}
