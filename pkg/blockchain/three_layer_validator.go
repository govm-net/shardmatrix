package blockchain

import (
	"fmt"
	"time"

	"github.com/lengzhao/shardmatrix/pkg/crypto"
	"github.com/lengzhao/shardmatrix/pkg/consensus"
	"github.com/lengzhao/shardmatrix/pkg/types"
)

// ValidationLevel 验证层级
type ValidationLevel int

const (
	ValidationLevelStructure ValidationLevel = iota // 第一层：结构验证
	ValidationLevelConsensus                        // 第二层：共识验证
	ValidationLevelContent                          // 第三层：内容验证
)

// String 返回验证层级的字符串表示
func (vl ValidationLevel) String() string {
	switch vl {
	case ValidationLevelStructure:
		return "STRUCTURE"
	case ValidationLevelConsensus:
		return "CONSENSUS"
	case ValidationLevelContent:
		return "CONTENT"
	default:
		return "UNKNOWN"
	}
}

// ValidationError 验证错误
type ValidationError struct {
	Level   ValidationLevel // 验证层级
	Code    string         // 错误代码
	Message string         // 错误信息
	Field   string         // 相关字段
}

// Error 实现error接口
func (ve *ValidationError) Error() string {
	return fmt.Sprintf("[%s] %s: %s (field: %s)", 
		ve.Level.String(), ve.Code, ve.Message, ve.Field)
}

// ThreeLayerValidator 三层区块验证器
type ThreeLayerValidator struct {
	consensusEngine  ConsensusEngine     // 共识引擎
	stateManager     StateManager        // 状态管理器
	timeController   *consensus.TimeController // 时间控制器
	strictMode       bool                // 严格模式
	config          *ValidationConfig   // 验证配置
}

// ConsensusEngine 共识引擎接口
type ConsensusEngine interface {
	ValidateBlock(block *types.Block) error
	IsValidator(address types.Address) bool
	IsCurrentValidator(address types.Address, blockHeight uint64) bool
	GetValidatorSet() *types.ValidatorSet
}

// ValidationConfig 验证配置
type ValidationConfig struct {
	EnableStructureValidation bool          // 启用结构验证
	EnableConsensusValidation bool          // 启用共识验证
	EnableContentValidation   bool          // 启用内容验证
	StrictTimeValidation      bool          // 严格时间验证
	MaxBlockSize              int           // 最大区块大小
	MaxTxPerBlock            int           // 每个区块最大交易数
	GasLimitPerBlock         uint64        // 每个区块Gas限制
	TimeToleranceSeconds     int64         // 时间容忍度(秒)
}

// NewThreeLayerValidator 创建三层验证器
func NewThreeLayerValidator(
	consensusEngine ConsensusEngine,
	stateManager StateManager,
	timeController *consensus.TimeController,
) *ThreeLayerValidator {
	return &ThreeLayerValidator{
		consensusEngine: consensusEngine,
		stateManager:    stateManager,
		timeController:  timeController,
		strictMode:      true,
		config: &ValidationConfig{
			EnableStructureValidation: true,
			EnableConsensusValidation: true,
			EnableContentValidation:   true,
			StrictTimeValidation:      true,
			MaxBlockSize:              types.MaxBlockSize,
			MaxTxPerBlock:            types.MaxTxPerBlock,
			GasLimitPerBlock:         1000000,
			TimeToleranceSeconds:     2, // 2秒容忍度
		},
	}
}

// ValidateBlock 完整的三层区块验证
func (tv *ThreeLayerValidator) ValidateBlock(block *types.Block) error {
	// 第一层：结构验证
	if tv.config.EnableStructureValidation {
		if err := tv.validateStructure(block); err != nil {
			return err
		}
	}

	// 第二层：共识验证
	if tv.config.EnableConsensusValidation {
		if err := tv.validateConsensus(block); err != nil {
			return err
		}
	}

	// 第三层：内容验证
	if tv.config.EnableContentValidation {
		if err := tv.validateContent(block); err != nil {
			return err
		}
	}

	return nil
}

// =============== 第一层：结构验证 ===============

// validateStructure 结构验证
func (tv *ThreeLayerValidator) validateStructure(block *types.Block) error {
	// 1. 区块完整性检查
	if err := tv.validateBlockIntegrity(block); err != nil {
		return &ValidationError{
			Level:   ValidationLevelStructure,
			Code:    "BLOCK_INTEGRITY",
			Message: err.Error(),
			Field:   "block",
		}
	}

	// 2. 时间戳严格验证
	if err := tv.validateTimestamp(block); err != nil {
		return &ValidationError{
			Level:   ValidationLevelStructure,
			Code:    "TIMESTAMP",
			Message: err.Error(),
			Field:   "header.timestamp",
		}
	}

	// 3. 区块高度连续性
	if err := tv.validateBlockHeight(block); err != nil {
		return &ValidationError{
			Level:   ValidationLevelStructure,
			Code:    "BLOCK_HEIGHT",
			Message: err.Error(),
			Field:   "header.number",
		}
	}

	// 4. 区块大小限制
	if err := tv.validateBlockSize(block); err != nil {
		return &ValidationError{
			Level:   ValidationLevelStructure,
			Code:    "BLOCK_SIZE",
			Message: err.Error(),
			Field:   "block_size",
		}
	}

	// 5. 交易数量限制
	if err := tv.validateTransactionCount(block); err != nil {
		return &ValidationError{
			Level:   ValidationLevelStructure,
			Code:    "TX_COUNT",
			Message: err.Error(),
			Field:   "transactions",
		}
	}

	return nil
}

// validateBlockIntegrity 验证区块完整性
func (tv *ThreeLayerValidator) validateBlockIntegrity(block *types.Block) error {
	if block == nil {
		return fmt.Errorf("block is nil")
	}

	// 验证区块哈希
	expectedHash := block.Header.Hash()
	if expectedHash.IsEmpty() {
		return fmt.Errorf("invalid block hash")
	}

	// 验证交易根
	expectedTxRoot := types.CalculateTxRoot(block.Transactions)
	if block.Header.TxRoot != expectedTxRoot {
		return fmt.Errorf("invalid transaction root: expected %s, got %s", 
			expectedTxRoot.String(), block.Header.TxRoot.String())
	}

	// 验证分片ID
	if block.Header.ShardID != types.ShardID {
		return fmt.Errorf("invalid shard ID: expected %d, got %d", 
			types.ShardID, block.Header.ShardID)
	}

	return nil
}

// validateTimestamp 验证时间戳
func (tv *ThreeLayerValidator) validateTimestamp(block *types.Block) error {
	now := time.Now().Unix()
	blockTime := block.Header.Timestamp

	// 检查时间戳不能是未来时间
	if blockTime > now+tv.config.TimeToleranceSeconds {
		return fmt.Errorf("block timestamp is too far in the future: %d > %d", 
			blockTime, now)
	}

	// 严格2秒间隔验证（如果启用）
	if tv.config.StrictTimeValidation && tv.timeController != nil {
		expectedTime := tv.timeController.GetBlockTime(block.Header.Number)
		if blockTime != expectedTime {
			return fmt.Errorf("strict time validation failed: expected %d, got %d", 
				expectedTime, blockTime)
		}
	}

	// 验证与前一个区块的时间关系
	if block.Header.Number > 0 {
		prevBlock, err := tv.stateManager.GetBlockByHeight(block.Header.Number - 1)
		if err != nil {
			return fmt.Errorf("failed to get previous block: %w", err)
		}

		expectedInterval := int64(types.BlockTime.Seconds())
		actualInterval := blockTime - prevBlock.Header.Timestamp

		if actualInterval != expectedInterval {
			return fmt.Errorf("invalid time interval: expected %d, got %d", 
				expectedInterval, actualInterval)
		}
	}

	return nil
}

// validateBlockHeight 验证区块高度
func (tv *ThreeLayerValidator) validateBlockHeight(block *types.Block) error {
	if block.Header.Number == 0 {
		// 创世区块特殊处理
		return nil
	}

	// 获取当前最新区块高度
	latestBlock, err := tv.stateManager.GetLastBlock()
	if err != nil {
		return fmt.Errorf("failed to get latest block: %w", err)
	}

	expectedHeight := latestBlock.Header.Number + 1
	if block.Header.Number != expectedHeight {
		return fmt.Errorf("invalid block height: expected %d, got %d", 
			expectedHeight, block.Header.Number)
	}

	// 验证前一区块哈希
	expectedPrevHash := latestBlock.Hash()
	if block.Header.PrevHash != expectedPrevHash {
		return fmt.Errorf("invalid previous hash: expected %s, got %s", 
			expectedPrevHash.String(), block.Header.PrevHash.String())
	}

	return nil
}

// validateBlockSize 验证区块大小
func (tv *ThreeLayerValidator) validateBlockSize(block *types.Block) error {
	blockSize := block.Size()
	if blockSize > tv.config.MaxBlockSize {
		return fmt.Errorf("block size exceeds limit: %d > %d", 
			blockSize, tv.config.MaxBlockSize)
	}
	return nil
}

// validateTransactionCount 验证交易数量
func (tv *ThreeLayerValidator) validateTransactionCount(block *types.Block) error {
	txCount := len(block.Transactions)
	if txCount > tv.config.MaxTxPerBlock {
		return fmt.Errorf("transaction count exceeds limit: %d > %d", 
			txCount, tv.config.MaxTxPerBlock)
	}
	return nil
}

// =============== 第二层：共识验证 ===============

// validateConsensus 共识验证
func (tv *ThreeLayerValidator) validateConsensus(block *types.Block) error {
	// 1. 验证者权限确认
	if err := tv.validateValidatorPermission(block); err != nil {
		return &ValidationError{
			Level:   ValidationLevelConsensus,
			Code:    "VALIDATOR_PERMISSION",
			Message: err.Error(),
			Field:   "header.validator",
		}
	}

	// 2. 签名有效性检查
	if err := tv.validateBlockSignature(block); err != nil {
		return &ValidationError{
			Level:   ValidationLevelConsensus,
			Code:    "SIGNATURE",
			Message: err.Error(),
			Field:   "header.signature",
		}
	}

	// 3. 空块标识验证
	if err := tv.validateEmptyBlockMarker(block); err != nil {
		return &ValidationError{
			Level:   ValidationLevelConsensus,
			Code:    "EMPTY_BLOCK",
			Message: err.Error(),
			Field:   "empty_block_marker",
		}
	}

	return nil
}

// validateValidatorPermission 验证验证者权限
func (tv *ThreeLayerValidator) validateValidatorPermission(block *types.Block) error {
	validator := block.Header.Validator

	// 检查是否为注册的验证者
	if !tv.consensusEngine.IsValidator(validator) {
		return fmt.Errorf("block producer is not a registered validator: %s", 
			validator.String())
	}

	// 检查是否为当前轮次的验证者
	if !tv.consensusEngine.IsCurrentValidator(validator, block.Header.Number) {
		return fmt.Errorf("block producer is not current validator for height %d: %s", 
			block.Header.Number, validator.String())
	}

	return nil
}

// validateBlockSignature 验证区块签名
func (tv *ThreeLayerValidator) validateBlockSignature(block *types.Block) error {
	// 空区块可能没有签名（安全模式）
	if block.IsEmpty() && block.Header.Signature.IsEmpty() {
		return nil
	}

	// 获取验证者公钥
	validatorSet := tv.consensusEngine.GetValidatorSet()
	validator := validatorSet.GetValidator(block.Header.Validator)
	if validator == nil {
		return fmt.Errorf("validator not found in validator set: %s", 
			block.Header.Validator.String())
	}

	// 验证签名
	if !crypto.VerifyBlock(&block.Header, validator.PublicKey) {
		return fmt.Errorf("invalid block signature from validator: %s", 
			block.Header.Validator.String())
	}

	return nil
}

// validateEmptyBlockMarker 验证空区块标识
func (tv *ThreeLayerValidator) validateEmptyBlockMarker(block *types.Block) error {
	isEmpty := block.IsEmpty()
	hasEmptyTxRoot := block.Header.TxRoot == types.EmptyTxRoot()

	// 空区块必须有正确的空交易根
	if isEmpty && !hasEmptyTxRoot {
		return fmt.Errorf("empty block must have empty transaction root")
	}

	// 非空区块不能有空交易根
	if !isEmpty && hasEmptyTxRoot {
		return fmt.Errorf("non-empty block cannot have empty transaction root")
	}

	return nil
}

// =============== 第三层：内容验证 ===============

// validateContent 内容验证
func (tv *ThreeLayerValidator) validateContent(block *types.Block) error {
	// 跳过空区块的内容验证
	if block.IsEmpty() {
		return nil
	}

	// 1. 交易合法性校验
	if err := tv.validateTransactionLegality(block); err != nil {
		return &ValidationError{
			Level:   ValidationLevelContent,
			Code:    "TRANSACTION_LEGALITY",
			Message: err.Error(),
			Field:   "transactions",
		}
	}

	// 2. 状态转换正确性
	if err := tv.validateStateTransition(block); err != nil {
		return &ValidationError{
			Level:   ValidationLevelContent,
			Code:    "STATE_TRANSITION",
			Message: err.Error(),
			Field:   "state_root",
		}
	}

	// 3. Gas限制合规性
	if err := tv.validateGasCompliance(block); err != nil {
		return &ValidationError{
			Level:   ValidationLevelContent,
			Code:    "GAS_COMPLIANCE",
			Message: err.Error(),
			Field:   "gas_usage",
		}
	}

	return nil
}

// validateTransactionLegality 验证交易合法性
func (tv *ThreeLayerValidator) validateTransactionLegality(block *types.Block) error {
	// 简化实现：检查每个交易是否存在且有效
	for i, txHash := range block.Transactions {
		// 这里应该从交易池或存储中获取完整交易进行验证
		// 简化实现中只检查哈希是否为空
		if txHash.IsEmpty() {
			return fmt.Errorf("invalid transaction hash at index %d", i)
		}
	}

	// TODO: 完整实现应包括：
	// - 交易签名验证
	// - 余额充足性检查
	// - Nonce连续性验证
	// - 交易格式验证

	return nil
}

// validateStateTransition 验证状态转换
func (tv *ThreeLayerValidator) validateStateTransition(block *types.Block) error {
	// 简化实现：检查状态根是否有效
	if block.Header.StateRoot.IsEmpty() {
		return fmt.Errorf("invalid state root")
	}

	// TODO: 完整实现应包括：
	// - 重新执行所有交易
	// - 计算新的状态根
	// - 与区块中的状态根对比

	return nil
}

// validateGasCompliance 验证Gas合规性
func (tv *ThreeLayerValidator) validateGasCompliance(block *types.Block) error {
	// 简化实现：计算总Gas使用量
	totalGasUsed := uint64(0)

	// 这里应该累计所有交易的Gas使用量
	// 简化实现中假设每个交易使用1000 Gas
	for range block.Transactions {
		totalGasUsed += 1000
	}

	if totalGasUsed > tv.config.GasLimitPerBlock {
		return fmt.Errorf("total gas used exceeds block limit: %d > %d", 
			totalGasUsed, tv.config.GasLimitPerBlock)
	}

	return nil
}

// =============== 序列验证方法 ===============

// ValidateSequence 验证区块序列
func (tv *ThreeLayerValidator) ValidateSequence(blocks []*types.Block) error {
	if len(blocks) == 0 {
		return nil
	}

	// 验证每个区块
	for i, block := range blocks {
		if err := tv.ValidateBlock(block); err != nil {
			return fmt.Errorf("block %d validation failed: %w", i, err)
		}
	}

	// 验证区块间连续性
	for i := 1; i < len(blocks); i++ {
		prevBlock := blocks[i-1]
		currentBlock := blocks[i]

		// 验证高度连续性
		if currentBlock.Header.Number != prevBlock.Header.Number+1 {
			return fmt.Errorf("block height not continuous: %d -> %d", 
				prevBlock.Header.Number, currentBlock.Header.Number)
		}

		// 验证哈希链接
		if currentBlock.Header.PrevHash != prevBlock.Hash() {
			return fmt.Errorf("block hash chain broken at height %d", 
				currentBlock.Header.Number)
		}

		// 验证时间间隔
		expectedInterval := int64(types.BlockTime.Seconds())
		actualInterval := currentBlock.Header.Timestamp - prevBlock.Header.Timestamp
		if actualInterval != expectedInterval {
			return fmt.Errorf("time interval validation failed at height %d: expected %d, got %d", 
				currentBlock.Header.Number, expectedInterval, actualInterval)
		}
	}

	return nil
}

// =============== 配置和工具方法 ===============

// SetStrictMode 设置严格验证模式
func (tv *ThreeLayerValidator) SetStrictMode(strict bool) {
	tv.strictMode = strict
}

// SetConfig 设置验证配置
func (tv *ThreeLayerValidator) SetConfig(config *ValidationConfig) {
	tv.config = config
}

// GetValidationStats 获取验证统计信息
func (tv *ThreeLayerValidator) GetValidationStats() map[string]interface{} {
	return map[string]interface{}{
		"strict_mode":               tv.strictMode,
		"structure_validation":      tv.config.EnableStructureValidation,
		"consensus_validation":      tv.config.EnableConsensusValidation,
		"content_validation":        tv.config.EnableContentValidation,
		"strict_time_validation":    tv.config.StrictTimeValidation,
		"max_block_size":           tv.config.MaxBlockSize,
		"max_tx_per_block":         tv.config.MaxTxPerBlock,
		"gas_limit_per_block":      tv.config.GasLimitPerBlock,
		"time_tolerance_seconds":   tv.config.TimeToleranceSeconds,
	}
}