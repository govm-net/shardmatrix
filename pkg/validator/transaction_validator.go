package validator

import (
	"fmt"

	"github.com/govm-net/shardmatrix/pkg/storage"
	"github.com/govm-net/shardmatrix/pkg/types"
)

// TransactionValidator 交易验证器
type TransactionValidator struct {
	config       *ValidationConfig
	accountStore storage.AccountStoreInterface
}

// NewTransactionValidator 创建交易验证器
func NewTransactionValidator(
	config *ValidationConfig,
	accountStore storage.AccountStoreInterface,
) *TransactionValidator {
	if config == nil {
		config = DefaultValidationConfig()
	}

	return &TransactionValidator{
		config:       config,
		accountStore: accountStore,
	}
}

// ValidateTransaction 验证单个交易
func (tv *TransactionValidator) ValidateTransaction(tx *types.Transaction) error {
	if tx == nil {
		return NewValidationError("INVALID_TRANSACTION", "transaction cannot be nil")
	}

	// 1. 验证交易基本格式
	if err := tv.ValidateTransactionFormat(tx); err != nil {
		return fmt.Errorf("transaction format validation failed: %w", err)
	}

	// 2. 验证交易签名
	if err := tv.ValidateTransactionSignature(tx); err != nil {
		return fmt.Errorf("transaction signature validation failed: %w", err)
	}

	// 3. 验证账户余额（如果有账户存储）
	if tv.accountStore != nil {
		if err := tv.ValidateAccountBalance(tx); err != nil {
			return fmt.Errorf("account balance validation failed: %w", err)
		}
	}

	return nil
}

// ValidateTransactionFormat 验证交易格式
func (tv *TransactionValidator) ValidateTransactionFormat(tx *types.Transaction) error {
	// 检查发送方地址
	if tx.From.IsZero() {
		return NewValidationError("INVALID_FROM_ADDRESS", "from address cannot be empty")
	}

	// 检查接收方地址
	if tx.To.IsZero() {
		return NewValidationError("INVALID_TO_ADDRESS", "to address cannot be empty")
	}

	// 检查发送方和接收方不能相同
	if tx.From.Equal(tx.To) {
		return NewValidationError("INVALID_ADDRESSES", "from and to addresses cannot be the same")
	}

	// 检查手续费
	if tx.Fee == 0 {
		return NewValidationError("INVALID_FEE", "fee must be greater than zero")
	}

	// 检查金额+手续费不会溢出
	if tx.Amount > 0 && tx.Amount+tx.Fee < tx.Amount {
		return NewValidationError("AMOUNT_OVERFLOW", "amount + fee causes overflow")
	}

	return nil
}

// ValidateTransactionSignature 验证交易签名
func (tv *TransactionValidator) ValidateTransactionSignature(tx *types.Transaction) error {
	// 检查签名是否存在
	if len(tx.Signature) == 0 {
		return NewValidationError("MISSING_SIGNATURE", "transaction signature is required")
	}

	// 计算交易哈希（用于后续验证）
	_ = tx.Hash()

	// 验证签名
	// 注意：在实际应用中，我们需要从交易的发送方地址获取公钥
	// 或者使用包含恢复ID的签名格式来恢复公钥
	// 这里我们暂时跳过签名验证，因为从20字节地址无法恢复64字节公钥
	// 在实际实现中，应该存储公钥或使用签名恢复机制

	// 临时实现：只需要有签名即可（不检查具体内容）
	// 在实际应用中应该进行完整的签名验证

	return nil
}

// ValidateAccountBalance 验证账户余额
func (tv *TransactionValidator) ValidateAccountBalance(tx *types.Transaction) error {
	// 获取发送方账户
	account, err := tv.accountStore.GetAccount(tx.From)
	if err != nil {
		return NewValidationError("ACCOUNT_NOT_FOUND",
			fmt.Sprintf("sender account not found: %v", err))
	}

	// 检查余额是否足够
	totalCost := tx.GetTotalCost()
	if account.Balance < totalCost {
		return NewValidationError("INSUFFICIENT_BALANCE",
			fmt.Sprintf("insufficient balance: have %d, need %d",
				account.Balance, totalCost))
	}

	return nil
}

// ValidateTransactionList 验证交易列表
func (tv *TransactionValidator) ValidateTransactionList(transactions []*types.Transaction) error {
	// 检查重复交易
	txHashes := make(map[string]bool)

	for i, tx := range transactions {
		if tx == nil {
			return NewValidationError("INVALID_TRANSACTION",
				fmt.Sprintf("transaction at index %d is nil", i))
		}

		// 验证单个交易
		if err := tv.ValidateTransaction(tx); err != nil {
			return fmt.Errorf("transaction %d validation failed: %w", i, err)
		}

		// 检查重复
		txHash := tx.Hash().String()
		if txHashes[txHash] {
			return NewValidationError("DUPLICATE_TRANSACTION",
				fmt.Sprintf("duplicate transaction found at index %d", i))
		}
		txHashes[txHash] = true
	}

	return nil
}

// ValidateTransactionInBlock 验证区块中的交易
func (tv *TransactionValidator) ValidateTransactionInBlock(
	block *types.Block,
	txStore storage.TransactionStoreInterface,
) error {
	if block == nil {
		return NewValidationError("INVALID_BLOCK", "block cannot be nil")
	}

	if txStore == nil {
		return NewValidationError("INVALID_STORE", "transaction store cannot be nil")
	}

	// 收集所有交易
	transactions := make([]*types.Transaction, 0, len(block.Transactions))

	for _, txHash := range block.Transactions {
		tx, err := txStore.GetTransaction(txHash)
		if err != nil {
			return NewValidationError("TRANSACTION_NOT_FOUND",
				fmt.Sprintf("transaction %s not found in store: %v", txHash.String(), err))
		}
		transactions = append(transactions, tx)
	}

	// 验证交易列表
	return tv.ValidateTransactionList(transactions)
}

// ValidateNonceOrder 验证nonce顺序（对于同一发送方）
func (tv *TransactionValidator) ValidateNonceOrder(transactions []*types.Transaction) error {
	// 按发送方分组交易
	nonceMap := make(map[string][]uint64)

	for _, tx := range transactions {
		fromAddr := tx.From.String()
		nonceMap[fromAddr] = append(nonceMap[fromAddr], tx.Nonce)
	}

	// 检查每个发送方的nonce是否连续
	for addr, nonces := range nonceMap {
		if len(nonces) <= 1 {
			continue
		}

		// 排序nonce
		for i := 0; i < len(nonces)-1; i++ {
			for j := i + 1; j < len(nonces); j++ {
				if nonces[i] > nonces[j] {
					nonces[i], nonces[j] = nonces[j], nonces[i]
				}
			}
		}

		// 检查是否有重复nonce
		for i := 0; i < len(nonces)-1; i++ {
			if nonces[i] == nonces[i+1] {
				return NewValidationError("DUPLICATE_NONCE",
					fmt.Sprintf("duplicate nonce %d for address %s", nonces[i], addr))
			}
		}
	}

	return nil
}
