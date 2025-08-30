package types

import (
	"bytes"
	"crypto/sha256"
	"encoding/gob"
)

// Transaction 交易结构
type Transaction struct {
	From      []byte `json:"from"`      // 发送方地址
	To        []byte `json:"to"`        // 接收方地址
	Fee       uint64 `json:"fee"`       // 手续费
	Data      []byte `json:"data"`      // 交易数据
	Signature []byte `json:"signature"` // 签名
}

// NewTransaction 创建新交易
func NewTransaction(from, to []byte, fee uint64, data []byte) *Transaction {
	return &Transaction{
		From: from,
		To:   to,
		Fee:  fee,
		Data: data,
	}
}

// Hash 计算交易哈希
func (tx *Transaction) Hash() []byte {
	// 创建临时交易用于哈希计算（不包含签名）
	tempTx := &Transaction{
		From: tx.From,
		To:   tx.To,
		Fee:  tx.Fee,
		Data: tx.Data,
	}

	// 使用gob序列化
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	enc.Encode(tempTx)

	hash := sha256.Sum256(buf.Bytes())
	return hash[:]
}

// Sign 签名交易
func (tx *Transaction) Sign(privateKey []byte) error {
	// TODO: 实现数字签名
	// 这里应该使用私钥对交易哈希进行签名
	tx.Signature = []byte("signature_placeholder")
	return nil
}

// Verify 验证交易签名
func (tx *Transaction) Verify(publicKey []byte) bool {
	// TODO: 实现签名验证
	// 这里应该验证交易签名是否有效
	return true
}

// IsValid 验证交易是否有效
func (tx *Transaction) IsValid() bool {
	// 检查基本格式
	if len(tx.From) == 0 {
		return false
	}

	if len(tx.To) == 0 {
		return false
	}

	// 检查手续费
	if tx.Fee < 0 {
		return false
	}

	// TODO: 添加更多验证逻辑
	return true
}

// GetID 获取交易ID（即交易哈希）
func (tx *Transaction) GetID() []byte {
	return tx.Hash()
}
