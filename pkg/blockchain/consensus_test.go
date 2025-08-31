package blockchain

import (
	"testing"
	"time"

	"github.com/govm-net/shardmatrix/pkg/consensus"
	"github.com/govm-net/shardmatrix/pkg/storage"
	"github.com/govm-net/shardmatrix/pkg/types"
	"github.com/govm-net/shardmatrix/pkg/validator"
)

func TestBlockchainDPoSIntegration(t *testing.T) {
	// 创建存储
	blockStore := storage.NewMemoryBlockStore()
	txStore := storage.NewMemoryTransactionStore()
	validatorStore := storage.NewMemoryValidatorStore()
	accountStore := storage.NewMemoryAccountStore()

	// 创建验证器
	val := validator.NewValidator(
		validator.DefaultValidationConfig(),
		blockStore,
		txStore,
		validatorStore,
		accountStore,
	)

	// 创建区块链
	blockchain, err := NewBlockchain(DefaultBlockchainConfig(), blockStore, val)
	if err != nil {
		t.Fatalf("Failed to create blockchain: %v", err)
	}

	// 创建DPoS共识
	dposConfig := consensus.DefaultDPoSConfig()
	dposConfig.MinStake = 1000
	dposConfig.ValidatorCount = 3
	dpos := consensus.NewDPoSConsensus(dposConfig)

	// 设置共识
	blockchain.SetConsensus(dpos)

	// 启动共识
	err = blockchain.GetConsensus().Start()
	if err != nil {
		t.Fatalf("Failed to start consensus: %v", err)
	}
	defer blockchain.GetConsensus().Stop()

	// 注册验证者
	validator1 := types.GenerateAddress()
	validator2 := types.GenerateAddress()
	validator3 := types.GenerateAddress()

	err = blockchain.GetConsensus().RegisterValidator(validator1, 5000, 0.1)
	if err != nil {
		t.Fatalf("Failed to register validator1: %v", err)
	}

	err = blockchain.GetConsensus().RegisterValidator(validator2, 4000, 0.15)
	if err != nil {
		t.Fatalf("Failed to register validator2: %v", err)
	}

	err = blockchain.GetConsensus().RegisterValidator(validator3, 3000, 0.2)
	if err != nil {
		t.Fatalf("Failed to register validator3: %v", err)
	}

	// 等待一点时间让验证者集合更新
	time.Sleep(100 * time.Millisecond)

	// 检查活跃验证者
	activeValidators := blockchain.GetConsensus().GetActiveValidators()
	if len(activeValidators) != 3 {
		t.Errorf("Expected 3 active validators, got %d", len(activeValidators))
	}

	// 测试区块生产
	producer := validator1
	transactions := []types.Hash{
		types.CalculateHash([]byte("tx1")),
		types.CalculateHash([]byte("tx2")),
	}

	// 生产区块
	block, err := blockchain.GetConsensus().ProduceBlock(producer, transactions)
	if err != nil {
		// 可能不是当前出块者，这是正常的
		t.Logf("Block production failed (may not be current producer): %v", err)
	} else {
		// 如果成功生产区块，验证区块
		if len(block.Transactions) != len(transactions) {
			t.Errorf("Expected %d transactions, got %d", len(transactions), len(block.Transactions))
		}

		if !block.Header.Validator.Equal(producer) {
			t.Errorf("Expected producer %s, got %s", producer.String(), block.Header.Validator.String())
		}
	}

	// 测试共识信息
	consensusInfo := blockchain.GetConsensus().GetConsensusInfo()
	if !consensusInfo.Enabled {
		t.Error("Consensus should be enabled")
	}

	if consensusInfo.ActiveValidators != 3 {
		t.Errorf("Expected 3 active validators in consensus info, got %d", consensusInfo.ActiveValidators)
	}

	if consensusInfo.TotalValidators != 3 {
		t.Errorf("Expected 3 total validators in consensus info, got %d", consensusInfo.TotalValidators)
	}
}

func TestConsensusValidation(t *testing.T) {
	// 创建测试环境
	blockStore := storage.NewMemoryBlockStore()
	txStore := storage.NewMemoryTransactionStore()
	validatorStore := storage.NewMemoryValidatorStore()
	accountStore := storage.NewMemoryAccountStore()
	val := validator.NewValidator(
		validator.DefaultValidationConfig(),
		blockStore,
		txStore,
		validatorStore,
		accountStore,
	)

	blockchain, err := NewBlockchain(DefaultBlockchainConfig(), blockStore, val)
	if err != nil {
		t.Fatalf("Failed to create blockchain: %v", err)
	}

	// 创建并设置DPoS
	dpos := consensus.NewDPoSConsensus(consensus.DefaultDPoSConfig())
	blockchain.SetConsensus(dpos)
	blockchain.GetConsensus().Start()
	defer blockchain.GetConsensus().Stop()

	// 注册验证者
	validator1 := types.GenerateAddress()
	blockchain.GetConsensus().RegisterValidator(validator1, 5000, 0.1)

	// 创建一个无效的区块（错误的出块者）
	invalidValidator := types.GenerateAddress()
	invalidBlock := types.NewBlock(1, blockchain.chainState.BestBlockHash, invalidValidator)
	invalidBlock.Header.Timestamp = time.Now().Unix()

	// 尝试添加无效区块，应该失败
	err = blockchain.AddBlock(invalidBlock)
	if err == nil {
		t.Error("Expected error when adding block with invalid producer")
	}
}

func TestStakingOperations(t *testing.T) {
	// 创建测试环境
	blockStore := storage.NewMemoryBlockStore()
	txStore := storage.NewMemoryTransactionStore()
	validatorStore := storage.NewMemoryValidatorStore()
	accountStore := storage.NewMemoryAccountStore()
	val := validator.NewValidator(
		validator.DefaultValidationConfig(),
		blockStore,
		txStore,
		validatorStore,
		accountStore,
	)

	blockchain, err := NewBlockchain(DefaultBlockchainConfig(), blockStore, val)
	if err != nil {
		t.Fatalf("Failed to create blockchain: %v", err)
	}

	// 创建DPoS
	dposConfig := consensus.DefaultDPoSConfig()
	dposConfig.MinStake = 1000
	dpos := consensus.NewDPoSConsensus(dposConfig)
	blockchain.SetConsensus(dpos)

	blockchain.GetConsensus().Start()
	defer blockchain.GetConsensus().Stop()

	// 测试验证者注册
	validator1 := types.GenerateAddress()
	err = blockchain.GetConsensus().RegisterValidator(validator1, 5000, 0.1)
	if err != nil {
		t.Fatalf("Failed to register validator: %v", err)
	}

	// 测试委托
	delegator1 := types.GenerateAddress()
	err = blockchain.GetConsensus().Delegate(delegator1, validator1, 2000)
	if err != nil {
		t.Fatalf("Failed to delegate: %v", err)
	}

	// 验证验证者信息
	validatorInfo, exists := dpos.GetValidatorInfo(validator1)
	if !exists {
		t.Fatal("Validator not found")
	}

	if validatorInfo.TotalStake != 7000 { // 5000 + 2000
		t.Errorf("Expected total stake 7000, got %d", validatorInfo.TotalStake)
	}

	if validatorInfo.DelegatedStake != 2000 {
		t.Errorf("Expected delegated stake 2000, got %d", validatorInfo.DelegatedStake)
	}

	// 测试取消委托
	err = blockchain.GetConsensus().Undelegate(delegator1, validator1, 1000)
	if err != nil {
		t.Fatalf("Failed to undelegate: %v", err)
	}

	// 验证更新后的信息
	validatorInfo, _ = dpos.GetValidatorInfo(validator1)
	if validatorInfo.TotalStake != 6000 { // 7000 - 1000
		t.Errorf("Expected total stake 6000 after undelegation, got %d", validatorInfo.TotalStake)
	}
}

func TestProducerElectionIntegration(t *testing.T) {
	// 创建测试环境
	blockStore := storage.NewMemoryBlockStore()
	txStore := storage.NewMemoryTransactionStore()
	validatorStore := storage.NewMemoryValidatorStore()
	accountStore := storage.NewMemoryAccountStore()
	val := validator.NewValidator(
		validator.DefaultValidationConfig(),
		blockStore,
		txStore,
		validatorStore,
		accountStore,
	)

	blockchain, err := NewBlockchain(DefaultBlockchainConfig(), blockStore, val)
	if err != nil {
		t.Fatalf("Failed to create blockchain: %v", err)
	}

	// 创建DPoS
	dposConfig := consensus.DefaultDPoSConfig()
	dposConfig.MinStake = 1000
	dposConfig.ValidatorCount = 3
	dposConfig.BlockInterval = time.Second
	dpos := consensus.NewDPoSConsensus(dposConfig)
	blockchain.SetConsensus(dpos)

	blockchain.GetConsensus().Start()
	defer blockchain.GetConsensus().Stop()

	// 注册多个验证者
	validators := make([]types.Address, 3)
	for i := 0; i < 3; i++ {
		validators[i] = types.GenerateAddress()
		stake := uint64(3000 + i*1000) // 不同的质押量
		err := blockchain.GetConsensus().RegisterValidator(validators[i], stake, 0.1)
		if err != nil {
			t.Fatalf("Failed to register validator %d: %v", i, err)
		}
	}

	// 等待验证者集合更新
	time.Sleep(100 * time.Millisecond)

	// 测试当前出块者获取
	currentProducer := blockchain.GetConsensus().GetCurrentProducer()
	if currentProducer.IsZero() {
		// 在初始化阶段，出块者可能为零，这是正常的
		t.Logf("Current producer is zero (normal during initialization)")
	}

	// 测试出块权限检查
	found := false
	for _, validator := range validators {
		if blockchain.GetConsensus().IsMyTurnToProduce(validator) {
			found = true
			break
		}
	}
	if !found {
		t.Error("At least one validator should be able to produce blocks")
	}

	// 测试验证者统计
	for _, validator := range validators {
		stats, err := blockchain.GetConsensus().GetValidatorStats(validator)
		if err != nil {
			t.Errorf("Failed to get validator stats for %s: %v", validator.String(), err)
		}

		if stats.TotalStake == 0 {
			t.Errorf("Validator %s should have non-zero stake", validator.String())
		}
	}
}

func TestUnbondingProcess(t *testing.T) {
	// 创建测试环境
	blockStore := storage.NewMemoryBlockStore()
	validatorStore := storage.NewMemoryValidatorStore()
	accountStore := storage.NewMemoryAccountStore()
	txStore := storage.NewMemoryTransactionStore()
	val := validator.NewValidator(
		validator.DefaultValidationConfig(),
		blockStore,
		txStore,
		validatorStore,
		accountStore,
	)

	blockchain, err := NewBlockchain(DefaultBlockchainConfig(), blockStore, val)
	if err != nil {
		t.Fatalf("Failed to create blockchain: %v", err)
	}

	// 创建DPoS，设置短的解绑期用于测试
	dposConfig := consensus.DefaultDPoSConfig()
	dposConfig.UnbondingPeriod = 100 * time.Millisecond // 很短的解绑期
	dposConfig.MinStake = 1000                          // 设置低的最小质押量
	dpos := consensus.NewDPoSConsensus(dposConfig)
	blockchain.SetConsensus(dpos)

	blockchain.GetConsensus().Start()
	defer blockchain.GetConsensus().Stop()

	// 注册验证者和委托
	validator1 := types.GenerateAddress()
	delegator1 := types.GenerateAddress()

	err = blockchain.GetConsensus().RegisterValidator(validator1, 3000, 0.1)
	if err != nil {
		t.Fatalf("Failed to register validator: %v", err)
	}

	err = blockchain.GetConsensus().Delegate(delegator1, validator1, 2000)
	if err != nil {
		t.Fatalf("Failed to delegate: %v", err)
	}

	// 完全取消委托
	err = blockchain.GetConsensus().Undelegate(delegator1, validator1, 2000)
	if err != nil {
		t.Fatalf("Failed to undelegate: %v", err)
	}

	// 检查解绑状态
	delegation, err := dpos.GetDelegation(delegator1, validator1)
	if err != nil {
		t.Fatalf("Failed to get delegation: %v", err)
	}

	if !delegation.IsUnbonding {
		t.Error("Delegation should be in unbonding state")
	}

	// 等待解绑期结束
	time.Sleep(200 * time.Millisecond)

	// 处理解绑委托
	completed := blockchain.GetConsensus().ProcessUnbondingDelegations()
	if len(completed) == 0 {
		t.Error("Should have completed unbonding delegations")
	}

	// 验证委托已被删除
	_, err = dpos.GetDelegation(delegator1, validator1)
	if err == nil {
		t.Error("Delegation should have been deleted after unbonding")
	}
}
