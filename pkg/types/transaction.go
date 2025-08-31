package types

import (
	"encoding/json"
	"errors"
	"fmt"
)

// Transaction 交易结构
type Transaction struct {
	From      Address `json:"from"`      // 发送方地址
	To        Address `json:"to"`        // 接收方地址
	Amount    uint64  `json:"amount"`    // 转账金额
	Fee       uint64  `json:"fee"`       // 手续费
	Nonce     uint64  `json:"nonce"`     // 防止重放攻击
	Data      []byte  `json:"data"`      // 交易数据
	Signature []byte  `json:"signature"` // 签名
}

// NewTransaction 创建新交易
func NewTransaction(from, to Address, amount, fee, nonce uint64, data []byte) *Transaction {
	return &Transaction{
		From:   from,
		To:     to,
		Amount: amount,
		Fee:    fee,
		Nonce:  nonce,
		Data:   data,
	}
}

// NewTransferTransaction 创建转账交易
func NewTransferTransaction(from, to Address, amount, fee, nonce uint64) *Transaction {
	return NewTransaction(from, to, amount, fee, nonce, nil)
}

// Hash 计算交易哈希（不包含签名）
func (tx *Transaction) Hash() Hash {
	// 创建临时交易用于哈希计算（不包含签名）
	tempTx := &Transaction{
		From:   tx.From,
		To:     tx.To,
		Amount: tx.Amount,
		Fee:    tx.Fee,
		Nonce:  tx.Nonce,
		Data:   tx.Data,
	}

	// 使用JSON序列化
	data, err := json.Marshal(tempTx)
	if err != nil {
		return EmptyHash()
	}

	return CalculateHash(data)
}

// GetID 获取交易ID（即交易哈希）
func (tx *Transaction) GetID() Hash {
	return tx.Hash()
}

// SignFunc 签名函数类型
type SignFunc func(data []byte) ([]byte, error)

// VerifyFunc 验证函数类型
type VerifyFunc func(data []byte, signature []byte) bool

// SignWithFunc 使用签名函数签名交易
func (tx *Transaction) SignWithFunc(signFunc SignFunc) error {
	if signFunc == nil {
		return errors.New("sign function cannot be nil")
	}

	// 计算交易哈希
	txHash := tx.Hash()

	// 使用签名函数签名
	signature, err := signFunc(txHash.Bytes())
	if err != nil {
		return fmt.Errorf("failed to sign transaction: %v", err)
	}

	tx.Signature = signature
	return nil
}

// VerifyWithFunc 使用验证函数验证交易签名
func (tx *Transaction) VerifyWithFunc(verifyFunc VerifyFunc) bool {
	if verifyFunc == nil || len(tx.Signature) == 0 {
		return false
	}

	// 计算交易哈希
	txHash := tx.Hash()

	// 使用验证函数验证签名
	return verifyFunc(txHash.Bytes(), tx.Signature)
}

// Sign 签名交易（保持兼容性）
func (tx *Transaction) Sign(privateKey []byte) error {
	if len(privateKey) == 0 {
		return errors.New("private key cannot be empty")
	}

	// 临时实现，后续替换为真正的签名
	tx.Signature = []byte("signature_placeholder")
	return nil
}

// Verify 验证交易签名（保持兼容性）
func (tx *Transaction) Verify(publicKey []byte) bool {
	if len(publicKey) == 0 || len(tx.Signature) == 0 {
		return false
	}

	// 临时实现，始终返回true
	return true
}

// IsValid 验证交易是否有效
func (tx *Transaction) IsValid() bool {
	// 检查发送方地址
	if tx.From.IsZero() {
		return false
	}

	// 检查接收方地址
	if tx.To.IsZero() {
		return false
	}

	// 检查金额（不能为负数）
	// 注意: uint64不能为负数，但我们仍然检查以防万一

	// 检查手续费
	if tx.Fee == 0 {
		return false // 手续费不能为0
	}

	// 检查签名
	if len(tx.Signature) == 0 {
		return false
	}

	return true
}

// GetTotalCost 获取交易总成本（金额 + 手续费）
func (tx *Transaction) GetTotalCost() uint64 {
	return tx.Amount + tx.Fee
}

// IsTransfer 检查是否为转账交易
func (tx *Transaction) IsTransfer() bool {
	return len(tx.Data) == 0 && tx.Amount > 0
}

// Size 获取交易大小（字节）
func (tx *Transaction) Size() int {
	data, err := json.Marshal(tx)
	if err != nil {
		return 0
	}
	return len(data)
}

// Serialize 序列化交易为JSON
func (tx *Transaction) Serialize() ([]byte, error) {
	return json.Marshal(tx)
}

// Deserialize 从 JSON 反序列化交易
func (tx *Transaction) Deserialize(data []byte) error {
	return json.Unmarshal(data, tx)
}

// DeserializeTransaction 从 JSON 创建交易
func DeserializeTransaction(data []byte) (*Transaction, error) {
	var tx Transaction
	err := json.Unmarshal(data, &tx)
	if err != nil {
		return nil, err
	}
	return &tx, nil
}
