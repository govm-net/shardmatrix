package blockchain

import (
	"fmt"
	"time"

	"github.com/govm-net/shardmatrix/pkg/consensus"
	"github.com/govm-net/shardmatrix/pkg/types"
)

// ConsensusIntegration DPoS共识集成
type ConsensusIntegration struct {
	blockchain *Blockchain
	dpos       *consensus.DPoSConsensus
	
	// 区块生产状态
	isProducing     bool
	currentProducer types.Address
	
	// 配置
	enableConsensus bool
}

// NewConsensusIntegration 创建共识集成
func NewConsensusIntegration(blockchain *Blockchain, dpos *consensus.DPoSConsensus) *ConsensusIntegration {
	ci := &ConsensusIntegration{
		blockchain:      blockchain,
		dpos:           dpos,
		enableConsensus: true,
	}

	// 设置DPoS回调
	ci.setupCallbacks()

	return ci
}

// setupCallbacks 设置DPoS回调
func (ci *ConsensusIntegration) setupCallbacks() {
	ci.dpos.SetCallbacks(
		ci.onValidatorSetChanged,
		ci.onSlotChanged,
		ci.onEpochChanged,
		ci.onValidatorSlashed,
		ci.onValidatorRewarded,
	)
}

// onValidatorSetChanged 验证者集合变化回调
func (ci *ConsensusIntegration) onValidatorSetChanged(validators []*consensus.ValidatorInfo) {
	fmt.Printf("Validator set changed: %d active validators\n", len(validators))
	
	// 可以在这里触发其他系统的更新
}

// onSlotChanged 槽位变化回调
func (ci *ConsensusIntegration) onSlotChanged(producer types.Address, slot uint64) {
	ci.currentProducer = producer
	fmt.Printf("Slot %d: Producer %s\n", slot, producer.String())
}

// onEpochChanged 轮次变化回调
func (ci *ConsensusIntegration) onEpochChanged(epoch uint64) {
	fmt.Printf("Epoch changed to %d\n", epoch)
}

// onValidatorSlashed 验证者被惩罚回调
func (ci *ConsensusIntegration) onValidatorSlashed(validator types.Address, amount uint64) {
	fmt.Printf("Validator %s slashed %d tokens\n", validator.String(), amount)
}

// onValidatorRewarded 验证者获得奖励回调
func (ci *ConsensusIntegration) onValidatorRewarded(validator types.Address, amount uint64) {
	fmt.Printf("Validator %s rewarded %d tokens\n", validator.String(), amount)
}

// ValidateBlockWithConsensus 使用DPoS验证区块
func (ci *ConsensusIntegration) ValidateBlockWithConsensus(block *types.Block) error {
	if !ci.enableConsensus {
		return nil // 如果未启用共识，跳过验证
	}

	// 验证区块时间
	if err := ci.dpos.ValidateBlockTiming(block); err != nil {
		return fmt.Errorf("block timing validation failed: %v", err)
	}

	// 验证出块者权限
	blockTime := time.Unix(block.Header.Timestamp, 0)
	if !ci.dpos.CanProduceBlock(block.Header.Validator, blockTime) {
		return fmt.Errorf("validator %s cannot produce block at timestamp %v", 
			block.Header.Validator.String(), blockTime)
	}

	// 记录区块生产
	slot := uint64(blockTime.Unix()) / uint64(ci.dpos.GetConfig().BlockInterval.Seconds())
	if err := ci.dpos.RecordBlockProduced(block.Header.Validator, slot); err != nil {
		return fmt.Errorf("failed to record block production: %v", err)
	}

	return nil
}

// ProduceBlock 生产区块
func (ci *ConsensusIntegration) ProduceBlock(producer types.Address, transactions []types.Hash) (*types.Block, error) {
	if !ci.enableConsensus {
		return nil, fmt.Errorf("consensus not enabled")
	}

	// 检查是否为当前出块者
	now := time.Now()
	if !ci.dpos.CanProduceBlock(producer, now) {
		return nil, fmt.Errorf("not current block producer")
	}

	// 获取链状态
	ci.blockchain.mutex.RLock()
	chainState := ci.blockchain.chainState
	latestBlock, err := ci.blockchain.blockStore.GetLatestBlock()
	ci.blockchain.mutex.RUnlock()

	if err != nil {
		return nil, fmt.Errorf("failed to get latest block: %v", err)
	}

	// 创建新区块
	block := types.NewBlock(
		chainState.Height+1,
		latestBlock.Hash(),
		producer,
	)

	// 设置时间戳
	block.Header.Timestamp = now.Unix()

	// 添加交易
	for _, txHash := range transactions {
		block.AddTransaction(txHash)
	}

	// 验证区块
	if err := ci.ValidateBlockWithConsensus(block); err != nil {
		return nil, fmt.Errorf("block validation failed: %v", err)
	}

	return block, nil
}

// ProcessBlockReward 处理区块奖励
func (ci *ConsensusIntegration) ProcessBlockReward(block *types.Block) error {
	if !ci.enableConsensus {
		return nil
	}

	// 计算奖励分配
	distribution, err := ci.dpos.CalculateBlockReward(block)
	if err != nil {
		return fmt.Errorf("failed to calculate block reward: %v", err)
	}

	// 分发奖励
	if err := ci.dpos.DistributeRewards(distribution); err != nil {
		return fmt.Errorf("failed to distribute rewards: %v", err)
	}

	return nil
}

// CheckMissedBlocks 检查错过的区块
func (ci *ConsensusIntegration) CheckMissedBlocks() error {
	if !ci.enableConsensus {
		return nil
	}

	// 检查当前时间是否已超过出块时间
	now := time.Now()
	nextSlotTime := ci.dpos.GetNextSlotTime()

	if now.After(nextSlotTime) {
		// 获取应该出块的验证者
		producer, slot, err := ci.dpos.GetCurrentProducerForTime(nextSlotTime)
		if err != nil {
			return fmt.Errorf("failed to get producer: %v", err)
		}

		// 记录错过区块
		if err := ci.dpos.RecordBlockMissed(producer); err != nil {
			return fmt.Errorf("failed to record missed block: %v", err)
		}

		fmt.Printf("Block missed at slot %d by validator %s\n", slot, producer.String())

		// 推进到下一个槽位
		if err := ci.dpos.AdvanceSlot(); err != nil {
			return fmt.Errorf("failed to advance slot: %v", err)
		}
	}

	return nil
}

// GetCurrentProducer 获取当前出块者
func (ci *ConsensusIntegration) GetCurrentProducer() types.Address {
	return ci.currentProducer
}

// IsMyTurnToProduce 检查是否轮到指定验证者出块
func (ci *ConsensusIntegration) IsMyTurnToProduce(validator types.Address) bool {
	if !ci.enableConsensus {
		return false
	}

	return ci.dpos.CanProduceBlock(validator, time.Now())
}

// GetTimeToNextSlot 获取到下一个出块时间的间隔
func (ci *ConsensusIntegration) GetTimeToNextSlot() time.Duration {
	if !ci.enableConsensus {
		return 0
	}

	return ci.dpos.GetTimeToNextSlot()
}

// GetValidatorStats 获取验证者统计
func (ci *ConsensusIntegration) GetValidatorStats(validator types.Address) (*consensus.ProducerStats, error) {
	if !ci.enableConsensus {
		return nil, fmt.Errorf("consensus not enabled")
	}

	return ci.dpos.GetProducerStats(validator)
}

// RegisterValidator 注册验证者
func (ci *ConsensusIntegration) RegisterValidator(address types.Address, stake uint64, commission float64) error {
	if !ci.enableConsensus {
		return fmt.Errorf("consensus not enabled")
	}

	return ci.dpos.RegisterValidator(address, stake, commission)
}

// Delegate 委托权益
func (ci *ConsensusIntegration) Delegate(delegator types.Address, validator types.Address, amount uint64) error {
	if !ci.enableConsensus {
		return fmt.Errorf("consensus not enabled")
	}

	return ci.dpos.Delegate(delegator, validator, amount)
}

// Undelegate 取消委托
func (ci *ConsensusIntegration) Undelegate(delegator types.Address, validator types.Address, amount uint64) error {
	if !ci.enableConsensus {
		return fmt.Errorf("consensus not enabled")
	}

	return ci.dpos.Undelegate(delegator, validator, amount)
}

// GetActiveValidators 获取活跃验证者
func (ci *ConsensusIntegration) GetActiveValidators() []*consensus.ValidatorInfo {
	if !ci.enableConsensus {
		return make([]*consensus.ValidatorInfo, 0)
	}

	return ci.dpos.GetActiveValidators()
}

// EnableConsensus 启用/禁用共识
func (ci *ConsensusIntegration) EnableConsensus(enable bool) {
	ci.enableConsensus = enable
}

// IsConsensusEnabled 检查共识是否启用
func (ci *ConsensusIntegration) IsConsensusEnabled() bool {
	return ci.enableConsensus
}

// Start 启动共识集成
func (ci *ConsensusIntegration) Start() error {
	if !ci.enableConsensus {
		return nil
	}

	return ci.dpos.Start()
}

// Stop 停止共识集成
func (ci *ConsensusIntegration) Stop() error {
	if !ci.enableConsensus {
		return nil
	}

	return ci.dpos.Stop()
}

// ProcessUnbondingDelegations 处理解绑委托
func (ci *ConsensusIntegration) ProcessUnbondingDelegations() []string {
	if !ci.enableConsensus {
		return make([]string, 0)
	}

	return ci.dpos.ProcessUnbondingDelegations()
}

// GetConsensusInfo 获取共识信息
func (ci *ConsensusIntegration) GetConsensusInfo() *ConsensusInfo {
	if !ci.enableConsensus {
		return &ConsensusInfo{
			Enabled: false,
		}
	}

	return &ConsensusInfo{
		Enabled:            true,
		CurrentEpoch:       ci.dpos.GetCurrentEpoch(),
		CurrentSlot:        ci.dpos.GetCurrentSlot(),
		CurrentProducer:    ci.currentProducer,
		NextSlotTime:       ci.dpos.GetNextSlotTime(),
		ActiveValidators:   len(ci.dpos.GetActiveValidators()),
		TotalValidators:    ci.dpos.GetValidatorCount(),
		TotalStake:         ci.dpos.GetTotalStake(),
		IsProducing:        ci.dpos.IsProducing(),
	}
}

// ConsensusInfo 共识信息
type ConsensusInfo struct {
	Enabled            bool          `json:"enabled"`
	CurrentEpoch       uint64        `json:"current_epoch"`
	CurrentSlot        uint64        `json:"current_slot"`
	CurrentProducer    types.Address `json:"current_producer"`
	NextSlotTime       time.Time     `json:"next_slot_time"`
	ActiveValidators   int           `json:"active_validators"`
	TotalValidators    int           `json:"total_validators"`
	TotalStake         uint64        `json:"total_stake"`
	IsProducing        bool          `json:"is_producing"`
}