package validator

import (
	"fmt"

	"github.com/govm-net/shardmatrix/pkg/storage"
	"github.com/govm-net/shardmatrix/pkg/types"
)

// Validator 综合验证器
type Validator struct {
	blockValidator       *BlockValidator
	transactionValidator *TransactionValidator
	config               *ValidationConfig
}

// NewValidator 创建综合验证器
func NewValidator(
	config *ValidationConfig,
	blockStore storage.BlockStore,
	transactionStore storage.TransactionStoreInterface,
	validatorStore storage.ValidatorStoreInterface,
	accountStore storage.AccountStoreInterface,
) *Validator {
	if config == nil {
		config = DefaultValidationConfig()
	}

	blockValidator := NewBlockValidator(
		config,
		blockStore,
		transactionStore,
		validatorStore,
		accountStore,
	)

	transactionValidator := NewTransactionValidator(
		config,
		accountStore,
	)

	return &Validator{
		blockValidator:       blockValidator,
		transactionValidator: transactionValidator,
		config:               config,
	}
}

// ValidateNewBlock 验证新区块（包括区块和交易验证）
func (v *Validator) ValidateNewBlock(
	block *types.Block,
	txStore storage.TransactionStoreInterface,
) error {
	if block == nil {
		return NewValidationError("INVALID_BLOCK", "block cannot be nil")
	}

	// 1. 验证区块本身
	if err := v.blockValidator.ValidateBlock(block); err != nil {
		return fmt.Errorf("block validation failed: %w", err)
	}

	// 2. 验证区块中的交易
	if txStore != nil && len(block.Transactions) > 0 {
		if err := v.transactionValidator.ValidateTransactionInBlock(block, txStore); err != nil {
			return fmt.Errorf("transaction validation in block failed: %w", err)
		}

		// 3. 验证交易nonce顺序
		if err := v.validateTransactionNonceInBlock(block, txStore); err != nil {
			return fmt.Errorf("transaction nonce validation failed: %w", err)
		}
	}

	return nil
}

// ValidateTransaction 验证单个交易
func (v *Validator) ValidateTransaction(tx *types.Transaction) error {
	return v.transactionValidator.ValidateTransaction(tx)
}

// ValidateTransactionList 验证交易列表
func (v *Validator) ValidateTransactionList(transactions []*types.Transaction) error {
	return v.transactionValidator.ValidateTransactionList(transactions)
}

// ValidateGenesisBlock 验证创世区块
func (v *Validator) ValidateGenesisBlock(block *types.Block) error {
	return v.blockValidator.ValidateGenesisBlock(block)
}

// validateTransactionNonceInBlock 验证区块中交易的nonce
func (v *Validator) validateTransactionNonceInBlock(
	block *types.Block,
	txStore storage.TransactionStoreInterface,
) error {
	// 收集所有交易
	transactions := make([]*types.Transaction, 0, len(block.Transactions))

	for _, txHash := range block.Transactions {
		tx, err := txStore.GetTransaction(txHash)
		if err != nil {
			return NewValidationError("TRANSACTION_NOT_FOUND",
				fmt.Sprintf("transaction %s not found: %v", txHash.String(), err))
		}
		transactions = append(transactions, tx)
	}

	// 验证nonce顺序
	return v.transactionValidator.ValidateNonceOrder(transactions)
}

// GetConfig 获取验证配置
func (v *Validator) GetConfig() *ValidationConfig {
	return v.config
}

// GetValidatorStore 获取验证者存储
func (v *Validator) GetValidatorStore() storage.ValidatorStoreInterface {
	return v.blockValidator.validatorStore
}

// UpdateConfig 更新验证配置
func (v *Validator) UpdateConfig(config *ValidationConfig) {
	if config != nil {
		v.config = config
		v.blockValidator.config = config
		v.transactionValidator.config = config
	}
}

// ValidateBlockSequence 验证区块序列
func (v *Validator) ValidateBlockSequence(blocks []*types.Block) error {
	if len(blocks) == 0 {
		return nil
	}

	// 验证第一个区块（可能是创世区块）
	firstBlock := blocks[0]
	if firstBlock.Header.Number == 0 {
		if err := v.ValidateGenesisBlock(firstBlock); err != nil {
			return fmt.Errorf("genesis block validation failed: %w", err)
		}
	} else {
		if err := v.blockValidator.ValidateBlock(firstBlock); err != nil {
			return fmt.Errorf("first block validation failed: %w", err)
		}
	}

	// 验证后续区块
	for i := 1; i < len(blocks); i++ {
		currentBlock := blocks[i]
		prevBlock := blocks[i-1]

		// 验证当前区块
		if err := v.blockValidator.ValidateBlock(currentBlock); err != nil {
			return fmt.Errorf("block %d validation failed: %w", i, err)
		}

		// 验证区块连接
		if !currentBlock.Header.PrevHash.Equal(prevBlock.Hash()) {
			return NewValidationError("INVALID_BLOCK_SEQUENCE",
				fmt.Sprintf("block %d does not connect to previous block", i))
		}

		// 验证高度递增
		if currentBlock.Header.Number != prevBlock.Header.Number+1 {
			return NewValidationError("INVALID_BLOCK_HEIGHT",
				fmt.Sprintf("block %d height is not incremental", i))
		}

		// 验证时间戳递增
		if currentBlock.Header.Timestamp <= prevBlock.Header.Timestamp {
			return NewValidationError("INVALID_TIMESTAMP_SEQUENCE",
				fmt.Sprintf("block %d timestamp is not incremental", i))
		}
	}

	return nil
}

// ValidateChainIntegrity 验证链的完整性
func (v *Validator) ValidateChainIntegrity(
	startHeight, endHeight uint64,
	blockStore storage.BlockStore,
) error {
	if startHeight > endHeight {
		return NewValidationError("INVALID_RANGE", "start height cannot be greater than end height")
	}

	var prevBlock *types.Block

	for height := startHeight; height <= endHeight; height++ {
		block, err := blockStore.GetBlockByHeight(height)
		if err != nil {
			return NewValidationError("BLOCK_NOT_FOUND",
				fmt.Sprintf("block at height %d not found: %v", height, err))
		}

		// 验证区块
		if err := v.blockValidator.ValidateBlock(block); err != nil {
			return fmt.Errorf("block %d validation failed: %w", height, err)
		}

		// 验证与前一个区块的连接
		if prevBlock != nil {
			if !block.Header.PrevHash.Equal(prevBlock.Hash()) {
				return NewValidationError("BROKEN_CHAIN",
					fmt.Sprintf("block %d does not connect to previous block", height))
			}
		}

		prevBlock = block
	}

	return nil
}
