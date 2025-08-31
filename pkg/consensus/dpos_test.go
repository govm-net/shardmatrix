package consensus

import (
	"testing"
	"time"

	"github.com/govm-net/shardmatrix/pkg/types"
)

func TestDPoSConsensus_BasicFunctionality(t *testing.T) {
	// 创建DPoS共识引擎
	config := DefaultDPoSConfig()
	config.ValidatorCount = 3
	config.MinStake = 1000
	dpos := NewDPoSConsensus(config)

	// 启动共识
	if err := dpos.Start(); err != nil {
		t.Fatalf("Failed to start DPoS consensus: %v", err)
	}
	defer dpos.Stop()

	// 检查初始状态
	if dpos.GetValidatorCount() != 0 {
		t.Errorf("Expected 0 validators initially, got %d", dpos.GetValidatorCount())
	}

	if dpos.GetCurrentEpoch() != 0 {
		t.Errorf("Expected epoch 0 initially, got %d", dpos.GetCurrentEpoch())
	}
}

func TestDPoSConsensus_ValidatorRegistration(t *testing.T) {
	config := DefaultDPoSConfig()
	config.MinStake = 1000
	dpos := NewDPoSConsensus(config)

	if err := dpos.Start(); err != nil {
		t.Fatalf("Failed to start DPoS consensus: %v", err)
	}
	defer dpos.Stop()

	// 创建测试地址
	validator1 := types.GenerateAddress()
	validator2 := types.GenerateAddress()

	// 注册验证者
	err := dpos.RegisterValidator(validator1, 5000, 0.1)
	if err != nil {
		t.Fatalf("Failed to register validator1: %v", err)
	}

	err = dpos.RegisterValidator(validator2, 3000, 0.15)
	if err != nil {
		t.Fatalf("Failed to register validator2: %v", err)
	}

	// 检查验证者数量
	if dpos.GetValidatorCount() != 2 {
		t.Errorf("Expected 2 validators, got %d", dpos.GetValidatorCount())
	}

	// 检查活跃验证者数量
	if dpos.GetActiveValidatorCount() != 2 {
		t.Errorf("Expected 2 active validators, got %d", dpos.GetActiveValidatorCount())
	}

	// 获取验证者信息
	info1, exists := dpos.GetValidatorInfo(validator1)
	if !exists {
		t.Fatal("Validator1 not found")
	}

	if info1.TotalStake != 5000 {
		t.Errorf("Expected validator1 stake 5000, got %d", info1.TotalStake)
	}

	if info1.Commission != 0.1 {
		t.Errorf("Expected validator1 commission 0.1, got %f", info1.Commission)
	}

	// 测试重复注册
	err = dpos.RegisterValidator(validator1, 1000, 0.05)
	if err == nil {
		t.Error("Expected error when registering duplicate validator")
	}

	// 测试质押不足
	validator3 := types.GenerateAddress()
	err = dpos.RegisterValidator(validator3, 500, 0.1) // 小于最小质押
	if err == nil {
		t.Error("Expected error when registering validator with insufficient stake")
	}
}

func TestDPoSConsensus_ValidatorStakeManagement(t *testing.T) {
	config := DefaultDPoSConfig()
	config.MinStake = 1000
	dpos := NewDPoSConsensus(config)

	if err := dpos.Start(); err != nil {
		t.Fatalf("Failed to start DPoS consensus: %v", err)
	}
	defer dpos.Stop()

	validator := types.GenerateAddress()

	// 注册验证者
	err := dpos.RegisterValidator(validator, 2000, 0.1)
	if err != nil {
		t.Fatalf("Failed to register validator: %v", err)
	}

	// 增加质押
	err = dpos.AddValidatorStake(validator, 1000)
	if err != nil {
		t.Fatalf("Failed to add validator stake: %v", err)
	}

	info, _ := dpos.GetValidatorInfo(validator)
	if info.TotalStake != 3000 {
		t.Errorf("Expected total stake 3000, got %d", info.TotalStake)
	}

	// 减少质押
	err = dpos.SubValidatorStake(validator, 500)
	if err != nil {
		t.Fatalf("Failed to subtract validator stake: %v", err)
	}

	info, _ = dpos.GetValidatorInfo(validator)
	if info.TotalStake != 2500 {
		t.Errorf("Expected total stake 2500, got %d", info.TotalStake)
	}

	// 测试减少过多质押
	err = dpos.SubValidatorStake(validator, 3000)
	if err == nil {
		t.Error("Expected error when subtracting too much stake")
	}

	// 测试减少质押至低于最小值
	err = dpos.SubValidatorStake(validator, 2000) // 剩余500，小于最小质押1000
	if err == nil {
		t.Error("Expected error when reducing stake below minimum")
	}
}

func TestDPoSConsensus_Delegation(t *testing.T) {
	config := DefaultDPoSConfig()
	config.MinStake = 1000
	config.MaxDelegators = 100
	dpos := NewDPoSConsensus(config)

	if err := dpos.Start(); err != nil {
		t.Fatalf("Failed to start DPoS consensus: %v", err)
	}
	defer dpos.Stop()

	validator := types.GenerateAddress()
	delegator1 := types.GenerateAddress()
	delegator2 := types.GenerateAddress()

	// 注册验证者
	err := dpos.RegisterValidator(validator, 2000, 0.1)
	if err != nil {
		t.Fatalf("Failed to register validator: %v", err)
	}

	// 委托
	err = dpos.Delegate(delegator1, validator, 1000)
	if err != nil {
		t.Fatalf("Failed to delegate: %v", err)
	}

	err = dpos.Delegate(delegator2, validator, 1500)
	if err != nil {
		t.Fatalf("Failed to delegate: %v", err)
	}

	// 检查验证者信息
	info, _ := dpos.GetValidatorInfo(validator)
	if info.TotalStake != 4500 { // 2000 + 1000 + 1500
		t.Errorf("Expected total stake 4500, got %d", info.TotalStake)
	}
	if info.DelegatedStake != 2500 { // 1000 + 1500
		t.Errorf("Expected delegated stake 2500, got %d", info.DelegatedStake)
	}
	if info.DelegatorCount != 2 {
		t.Errorf("Expected 2 delegators, got %d", info.DelegatorCount)
	}

	// 检查委托信息
	delegation1, exists := dpos.GetDelegationInfo(delegator1, validator)
	if !exists {
		t.Fatal("Delegation1 not found")
	}
	if delegation1.Amount != 1000 {
		t.Errorf("Expected delegation1 amount 1000, got %d", delegation1.Amount)
	}

	// 增加委托
	err = dpos.Delegate(delegator1, validator, 500)
	if err != nil {
		t.Fatalf("Failed to add delegation: %v", err)
	}

	delegation1, _ = dpos.GetDelegationInfo(delegator1, validator)
	if delegation1.Amount != 1500 {
		t.Errorf("Expected delegation1 amount 1500, got %d", delegation1.Amount)
	}

	// 测试自委托
	err = dpos.Delegate(validator, validator, 1000)
	if err == nil {
		t.Error("Expected error for self delegation")
	}
}

func TestDPoSConsensus_Undelegation(t *testing.T) {
	config := DefaultDPoSConfig()
	config.MinStake = 1000
	config.UnbondingPeriod = time.Hour // 1小时解绑期
	dpos := NewDPoSConsensus(config)

	if err := dpos.Start(); err != nil {
		t.Fatalf("Failed to start DPoS consensus: %v", err)
	}
	defer dpos.Stop()

	validator := types.GenerateAddress()
	delegator := types.GenerateAddress()

	// 注册验证者并委托
	dpos.RegisterValidator(validator, 2000, 0.1)
	dpos.Delegate(delegator, validator, 1000)

	// 部分取消委托
	err := dpos.Undelegate(delegator, validator, 300)
	if err != nil {
		t.Fatalf("Failed to undelegate: %v", err)
	}

	delegation, _ := dpos.GetDelegationInfo(delegator, validator)
	if delegation.Amount != 700 {
		t.Errorf("Expected remaining delegation 700, got %d", delegation.Amount)
	}

	// 完全取消委托
	err = dpos.Undelegate(delegator, validator, 700)
	if err != nil {
		t.Fatalf("Failed to undelegate completely: %v", err)
	}

	delegation, _ = dpos.GetDelegationInfo(delegator, validator)
	if !delegation.IsUnbonding {
		t.Error("Expected delegation to be unbonding")
	}

	// 测试取消过多委托
	dpos.Delegate(delegator, validator, 500)
	err = dpos.Undelegate(delegator, validator, 1000)
	if err == nil {
		t.Error("Expected error when undelegating more than available")
	}
}

func TestDPoSConsensus_ProducerElection(t *testing.T) {
	config := DefaultDPoSConfig()
	config.ValidatorCount = 3
	config.MinStake = 1000
	config.BlockInterval = time.Second
	dpos := NewDPoSConsensus(config)

	if err := dpos.Start(); err != nil {
		t.Fatalf("Failed to start DPoS consensus: %v", err)
	}
	defer dpos.Stop()

	// 注册验证者
	validators := make([]types.Address, 3)
	for i := 0; i < 3; i++ {
		validators[i] = types.GenerateAddress()
		stake := uint64(2000 + i*1000) // 不同质押量
		err := dpos.RegisterValidator(validators[i], stake, 0.1)
		if err != nil {
			t.Fatalf("Failed to register validator %d: %v", i, err)
		}
	}

	// 检查活跃验证者
	activeValidators := dpos.GetActiveValidators()
	if len(activeValidators) != 3 {
		t.Errorf("Expected 3 active validators, got %d", len(activeValidators))
	}

	// 测试槽位生产者选择
	now := time.Now()
	producer, slot, err := dpos.GetCurrentProducerForTime(now)
	if err != nil {
		t.Fatalf("Failed to get current producer: %v", err)
	}

	// 验证生产者是否为有效验证者
	isValid := false
	for _, validator := range validators {
		if producer.Equal(validator) {
			isValid = true
			break
		}
	}
	if !isValid {
		t.Errorf("Producer %s is not a valid validator", producer.String())
	}

	// 测试槽位时间验证
	if !dpos.IsSlotTime(now) {
		t.Error("Current time should be valid slot time")
	}

	// 测试生产者权限验证
	if !dpos.CanProduceBlock(producer, now) {
		t.Error("Current producer should be able to produce block")
	}

	// 其他验证者不应该能够出块
	for _, validator := range validators {
		if !validator.Equal(producer) && dpos.CanProduceBlock(validator, now) {
			t.Errorf("Validator %s should not be able to produce block at slot %d",
				validator.String(), slot)
		}
	}
}

func TestDPoSConsensus_ValidatorJailing(t *testing.T) {
	config := DefaultDPoSConfig()
	config.MinStake = 1000
	dpos := NewDPoSConsensus(config)

	if err := dpos.Start(); err != nil {
		t.Fatalf("Failed to start DPoS consensus: %v", err)
	}
	defer dpos.Stop()

	validator := types.GenerateAddress()

	// 注册验证者
	err := dpos.RegisterValidator(validator, 2000, 0.1)
	if err != nil {
		t.Fatalf("Failed to register validator: %v", err)
	}

	// 验证初始状态
	if !dpos.IsActiveValidator(validator) {
		t.Error("Validator should be active initially")
	}

	// 监禁验证者
	err = dpos.JailValidator(validator, "misbehavior")
	if err != nil {
		t.Fatalf("Failed to jail validator: %v", err)
	}

	// 验证监禁状态
	if dpos.IsActiveValidator(validator) {
		t.Error("Validator should not be active after jailing")
	}

	info, _ := dpos.GetValidatorInfo(validator)
	if info.Validator.Status != types.ValidatorJailed {
		t.Error("Validator status should be jailed")
	}

	// 解除监禁
	err = dpos.UnjailValidator(validator)
	if err != nil {
		t.Fatalf("Failed to unjail validator: %v", err)
	}

	// 验证恢复状态
	if !dpos.IsActiveValidator(validator) {
		t.Error("Validator should be active after unjailing")
	}

	info, _ = dpos.GetValidatorInfo(validator)
	if info.Validator.Status != types.ValidatorActive {
		t.Error("Validator status should be active")
	}

	if info.MissedBlocks != 0 {
		t.Error("Missed blocks should be reset after unjailing")
	}
}

func TestDPoSConsensus_RewardCalculation(t *testing.T) {
	config := DefaultDPoSConfig()
	config.MinStake = 1000
	config.RewardRate = 0.1
	dpos := NewDPoSConsensus(config)

	if err := dpos.Start(); err != nil {
		t.Fatalf("Failed to start DPoS consensus: %v", err)
	}
	defer dpos.Stop()

	validator := types.GenerateAddress()
	delegator := types.GenerateAddress()

	// 注册验证者和委托
	dpos.RegisterValidator(validator, 3000, 0.2) // 20%佣金
	dpos.Delegate(delegator, validator, 2000)

	// 创建测试区块
	block := types.NewBlock(1, types.EmptyHash(), validator)

	// 计算奖励
	distribution, err := dpos.CalculateBlockReward(block)
	if err != nil {
		t.Fatalf("Failed to calculate block reward: %v", err)
	}

	// 验证奖励分配
	if distribution.TotalReward == 0 {
		t.Error("Total reward should be greater than 0")
	}

	validatorReward, exists := distribution.ValidatorRewards[validator]
	if !exists {
		t.Fatal("Validator reward not found")
	}

	if validatorReward == 0 {
		t.Error("Validator reward should be greater than 0")
	}

	// 验证委托者奖励
	if len(distribution.DelegatorRewards) == 0 {
		t.Error("Should have delegator rewards")
	}

	// 验证社区奖励
	if distribution.CommunityReward == 0 {
		t.Error("Community reward should be greater than 0")
	}

	// 计算APY
	apy, err := dpos.CalculateAPY(validator)
	if err != nil {
		t.Fatalf("Failed to calculate APY: %v", err)
	}

	if apy <= 0 || apy > 1 {
		t.Errorf("APY should be between 0 and 1, got %f", apy)
	}
}

func TestDPoSConsensus_ConcurrentOperations(t *testing.T) {
	config := DefaultDPoSConfig()
	config.MinStake = 1000
	dpos := NewDPoSConsensus(config)

	if err := dpos.Start(); err != nil {
		t.Fatalf("Failed to start DPoS consensus: %v", err)
	}
	defer dpos.Stop()

	// 并发注册验证者
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(index int) {
			validator := types.GenerateAddress()
			stake := uint64(1000 + index*100)
			err := dpos.RegisterValidator(validator, stake, 0.1)
			if err != nil {
				t.Errorf("Failed to register validator %d: %v", index, err)
			}
			done <- true
		}(i)
	}

	// 等待所有操作完成
	for i := 0; i < 10; i++ {
		<-done
	}

	// 验证结果
	if dpos.GetValidatorCount() != 10 {
		t.Errorf("Expected 10 validators, got %d", dpos.GetValidatorCount())
	}
}

// 基准测试
func BenchmarkDPoSConsensus_RegisterValidator(b *testing.B) {
	config := DefaultDPoSConfig()
	config.MinStake = 1000
	dpos := NewDPoSConsensus(config)
	dpos.Start()
	defer dpos.Stop()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		validator := types.GenerateAddress()
		dpos.RegisterValidator(validator, 2000, 0.1)
	}
}

func BenchmarkDPoSConsensus_Delegate(b *testing.B) {
	config := DefaultDPoSConfig()
	config.MinStake = 1000
	dpos := NewDPoSConsensus(config)
	dpos.Start()
	defer dpos.Stop()

	// 准备验证者
	validator := types.GenerateAddress()
	dpos.RegisterValidator(validator, 10000, 0.1)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		delegator := types.GenerateAddress()
		dpos.Delegate(delegator, validator, 1000)
	}
}

func BenchmarkDPoSConsensus_GetProducerForSlot(b *testing.B) {
	config := DefaultDPoSConfig()
	config.MinStake = 1000
	dpos := NewDPoSConsensus(config)
	dpos.Start()
	defer dpos.Stop()

	// 注册验证者
	for i := 0; i < 10; i++ {
		validator := types.GenerateAddress()
		dpos.RegisterValidator(validator, 2000, 0.1)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		dpos.GetProducerForSlot(uint64(i))
	}
}
