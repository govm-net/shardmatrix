package validator

import (
	"fmt"
	"time"

	"github.com/govm-net/shardmatrix/pkg/storage"
	"github.com/govm-net/shardmatrix/pkg/types"
)

// ValidationError 验证错误类型
type ValidationError struct {
	Type    string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation error [%s]: %s", e.Type, e.Message)
}

// NewValidationError 创建验证错误
func NewValidationError(errorType, message string) *ValidationError {
	return &ValidationError{
		Type:    errorType,
		Message: message,
	}
}

// ValidationConfig 验证配置
type ValidationConfig struct {
	MaxBlockSize        int           // 最大区块大小（字节）
	MaxTransactions     int           // 最大交易数量
	BlockTimeWindow     time.Duration // 区块时间窗口
	MinTimestamp        int64         // 最小时间戳
	RequireSignature    bool          // 是否需要验证者签名
	AllowEmptyBlocks    bool          // 是否允许空区块
	ValidatorStakeCheck bool          // 是否检查验证者质押
}

// DefaultValidationConfig 默认验证配置
func DefaultValidationConfig() *ValidationConfig {
	return &ValidationConfig{
		MaxBlockSize:        1024 * 1024, // 1MB
		MaxTransactions:     1000,
		BlockTimeWindow:     time.Minute * 10, // 10分钟时间窗口
		MinTimestamp:        0,
		RequireSignature:    true,
		AllowEmptyBlocks:    true,
		ValidatorStakeCheck: true,
	}
}

// BlockValidator 区块验证器
type BlockValidator struct {
	config           *ValidationConfig
	blockStore       storage.BlockStore
	transactionStore storage.TransactionStoreInterface
	validatorStore   storage.ValidatorStoreInterface
	accountStore     storage.AccountStoreInterface
}

// NewBlockValidator 创建区块验证器
func NewBlockValidator(
	config *ValidationConfig,
	blockStore storage.BlockStore,
	transactionStore storage.TransactionStoreInterface,
	validatorStore storage.ValidatorStoreInterface,
	accountStore storage.AccountStoreInterface,
) *BlockValidator {
	if config == nil {
		config = DefaultValidationConfig()
	}

	return &BlockValidator{
		config:           config,
		blockStore:       blockStore,
		transactionStore: transactionStore,
		validatorStore:   validatorStore,
		accountStore:     accountStore,
	}
}

// ValidateBlock 验证区块
func (bv *BlockValidator) ValidateBlock(block *types.Block) error {
	if block == nil {
		return NewValidationError("INVALID_BLOCK", "block cannot be nil")
	}

	// 1. 验证区块头
	if err := bv.ValidateBlockHeader(block.Header); err != nil {
		return fmt.Errorf("block header validation failed: %w", err)
	}

	// 2. 验证区块大小
	if err := bv.ValidateBlockSize(block); err != nil {
		return fmt.Errorf("block size validation failed: %w", err)
	}

	// 3. 验证交易数量
	if err := bv.ValidateTransactionCount(block); err != nil {
		return fmt.Errorf("transaction count validation failed: %w", err)
	}

	// 4. 验证交易Merkle根
	if err := bv.ValidateTransactionRoot(block); err != nil {
		return fmt.Errorf("transaction root validation failed: %w", err)
	}

	// 5. 验证时间戳
	if err := bv.ValidateTimestamp(block); err != nil {
		return fmt.Errorf("timestamp validation failed: %w", err)
	}

	// 6. 验证前一个区块
	if err := bv.ValidatePreviousBlock(block); err != nil {
		return fmt.Errorf("previous block validation failed: %w", err)
	}

	// 7. 验证验证者
	if err := bv.ValidateValidator(block); err != nil {
		return fmt.Errorf("validator validation failed: %w", err)
	}

	// 8. 验证签名（创世区块不需要签名）
	if bv.config.RequireSignature && block.Header.Number > 0 {
		if err := bv.ValidateSignature(block); err != nil {
			return fmt.Errorf("signature validation failed: %w", err)
		}
	}

	return nil
}

// ValidateBlockHeader 验证区块头
func (bv *BlockValidator) ValidateBlockHeader(header *types.BlockHeader) error {
	if header == nil {
		return NewValidationError("INVALID_HEADER", "block header cannot be nil")
	}

	// 检查验证者地址
	if header.Validator.IsZero() {
		return NewValidationError("INVALID_VALIDATOR", "validator address cannot be empty")
	}

	// 检查时间戳
	if header.Timestamp <= 0 {
		return NewValidationError("INVALID_TIMESTAMP", "timestamp must be positive")
	}

	// 检查最小时间戳
	if header.Timestamp < bv.config.MinTimestamp {
		return NewValidationError("INVALID_TIMESTAMP",
			fmt.Sprintf("timestamp %d is less than minimum %d",
				header.Timestamp, bv.config.MinTimestamp))
	}

	return nil
}

// ValidateBlockSize 验证区块大小
func (bv *BlockValidator) ValidateBlockSize(block *types.Block) error {
	size := block.Size()
	if size > bv.config.MaxBlockSize {
		return NewValidationError("BLOCK_TOO_LARGE",
			fmt.Sprintf("block size %d exceeds maximum %d",
				size, bv.config.MaxBlockSize))
	}
	return nil
}

// ValidateTransactionCount 验证交易数量
func (bv *BlockValidator) ValidateTransactionCount(block *types.Block) error {
	txCount := len(block.Transactions)

	// 检查是否允许空区块
	if !bv.config.AllowEmptyBlocks && txCount == 0 {
		return NewValidationError("EMPTY_BLOCK", "empty blocks are not allowed")
	}

	// 检查最大交易数量
	if txCount > bv.config.MaxTransactions {
		return NewValidationError("TOO_MANY_TRANSACTIONS",
			fmt.Sprintf("transaction count %d exceeds maximum %d",
				txCount, bv.config.MaxTransactions))
	}

	return nil
}

// ValidateTransactionRoot 验证交易Merkle根
func (bv *BlockValidator) ValidateTransactionRoot(block *types.Block) error {
	expectedRoot := block.CalculateTxRoot()
	if !block.Header.TxRoot.Equal(expectedRoot) {
		return NewValidationError("INVALID_TX_ROOT",
			"transaction merkle root mismatch")
	}
	return nil
}

// ValidateTimestamp 验证时间戳
func (bv *BlockValidator) ValidateTimestamp(block *types.Block) error {
	now := time.Now().Unix()
	blockTime := block.Header.Timestamp

	// 检查时间戳不能太未来
	if blockTime > now+int64(bv.config.BlockTimeWindow.Seconds()) {
		return NewValidationError("TIMESTAMP_TOO_FUTURE",
			fmt.Sprintf("block timestamp %d is too far in the future", blockTime))
	}

	// 对于非创世区块，检查时间戳顺序
	if block.Header.Number > 0 {
		prevBlock, err := bv.blockStore.GetBlockByHeight(block.Header.Number - 1)
		if err != nil {
			return NewValidationError("PREV_BLOCK_NOT_FOUND",
				fmt.Sprintf("previous block not found: %v", err))
		}

		if blockTime <= prevBlock.Header.Timestamp {
			return NewValidationError("INVALID_TIMESTAMP_ORDER",
				"block timestamp must be greater than previous block")
		}
	}

	return nil
}

// ValidatePreviousBlock 验证前一个区块
func (bv *BlockValidator) ValidatePreviousBlock(block *types.Block) error {
	// 创世区块特殊处理
	if block.Header.Number == 0 {
		if !block.Header.PrevHash.IsZero() {
			return NewValidationError("INVALID_GENESIS",
				"genesis block must have empty previous hash")
		}
		return nil
	}

	// 检查前一个区块是否存在
	if !bv.blockStore.HasBlock(block.Header.PrevHash) {
		return NewValidationError("PREV_BLOCK_NOT_FOUND",
			"previous block not found in store")
	}

	// 获取前一个区块并检查高度
	prevBlock, err := bv.blockStore.GetBlock(block.Header.PrevHash)
	if err != nil {
		return NewValidationError("PREV_BLOCK_ERROR",
			fmt.Sprintf("failed to get previous block: %v", err))
	}

	if prevBlock.Header.Number != block.Header.Number-1 {
		return NewValidationError("INVALID_BLOCK_HEIGHT",
			"block height must be previous block height + 1")
	}

	return nil
}

// ValidateValidator 验证验证者
func (bv *BlockValidator) ValidateValidator(block *types.Block) error {
	if bv.validatorStore == nil {
		return nil // 如果没有验证者存储，跳过验证
	}

	// 检查验证者是否存在
	if !bv.validatorStore.HasValidator(block.Header.Validator) {
		return NewValidationError("VALIDATOR_NOT_FOUND",
			"validator not found in validator set")
	}

	// 检查验证者质押（如果启用）
	if bv.config.ValidatorStakeCheck {
		validator, err := bv.validatorStore.GetValidator(block.Header.Validator)
		if err != nil {
			return NewValidationError("VALIDATOR_ERROR",
				fmt.Sprintf("failed to get validator: %v", err))
		}

		if !validator.CanValidate() {
			return NewValidationError("VALIDATOR_INACTIVE",
				"validator is not active or has insufficient stake")
		}
	}

	return nil
}

// ValidateSignature 验证区块签名
func (bv *BlockValidator) ValidateSignature(block *types.Block) error {
	if len(block.Header.Signature) == 0 {
		return NewValidationError("MISSING_SIGNATURE",
			"block signature is required")
	}

	// 计算区块哈希（不包含签名）
	tempHeader := *block.Header
	tempHeader.Signature = nil
	headerData, err := (&tempHeader).Serialize()
	if err != nil {
		return NewValidationError("SERIALIZATION_ERROR",
			fmt.Sprintf("failed to serialize header: %v", err))
	}

	// 验证签名（需要验证者的公钥）
	// 这里需要从验证者信息中获取公钥
	if bv.validatorStore != nil {
		validator, err := bv.validatorStore.GetValidator(block.Header.Validator)
		if err != nil {
			return NewValidationError("VALIDATOR_ERROR",
				fmt.Sprintf("failed to get validator for signature verification: %v", err))
		}

		// 假设验证者有公钥字段（需要扩展Validator结构）
		// 这里使用地址作为临时公钥
		// TODO: 实现正确的公钥恢复和签名验证
		_ = validator  // 避免未使用变量警告
		_ = headerData // 避免未使用变量警告
	}

	return nil
}

// ValidateGenesisBlock 验证创世区块
func (bv *BlockValidator) ValidateGenesisBlock(block *types.Block) error {
	if block.Header.Number != 0 {
		return NewValidationError("NOT_GENESIS", "not a genesis block")
	}

	if !block.Header.PrevHash.IsZero() {
		return NewValidationError("INVALID_GENESIS",
			"genesis block must have empty previous hash")
	}

	// 其他基本验证
	if err := bv.ValidateBlockHeader(block.Header); err != nil {
		return err
	}

	if err := bv.ValidateBlockSize(block); err != nil {
		return err
	}

	if err := bv.ValidateTransactionRoot(block); err != nil {
		return err
	}

	return nil
}
