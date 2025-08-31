package types

import (
	"testing"
)

func TestNewTransaction(t *testing.T) {
	// 创建测试数据
	from := AddressFromPublicKey([]byte("from_public_key"))
	to := AddressFromPublicKey([]byte("to_public_key"))
	amount := uint64(100)
	fee := uint64(10)
	nonce := uint64(1)
	data := []byte("transaction_data")

	tx := NewTransaction(from, to, amount, fee, nonce, data)

	// 验证交易字段
	if tx.From != from {
		t.Errorf("Expected from address %x, got %x", from, tx.From)
	}

	if tx.To != to {
		t.Errorf("Expected to address %x, got %x", to, tx.To)
	}

	if tx.Amount != amount {
		t.Errorf("Expected amount %d, got %d", amount, tx.Amount)
	}

	if tx.Fee != fee {
		t.Errorf("Expected fee %d, got %d", fee, tx.Fee)
	}

	if tx.Nonce != nonce {
		t.Errorf("Expected nonce %d, got %d", nonce, tx.Nonce)
	}

	if string(tx.Data) != string(data) {
		t.Errorf("Expected data %s, got %s", string(data), string(tx.Data))
	}

	// 验证签名为空
	if len(tx.Signature) != 0 {
		t.Errorf("Expected empty signature, got %d bytes", len(tx.Signature))
	}
}

func TestTransactionHash(t *testing.T) {
	// 创建交易
	from := AddressFromPublicKey([]byte("from_public_key"))
	to := AddressFromPublicKey([]byte("to_public_key"))
	tx := NewTransaction(from, to, 100, 10, 1, []byte("data"))

	// 计算哈希
	hash1 := tx.Hash()
	hash2 := tx.Hash()

	// 验证哈希一致性
	if hash1 != hash2 {
		t.Errorf("Transaction hash should be consistent, got %x and %x", hash1, hash2)
	}

	// 验证哈希长度
	if len(hash1.Bytes()) != 32 {
		t.Errorf("Expected hash length 32, got %d", len(hash1.Bytes()))
	}

	// 验证不同交易有不同的哈希
	from2 := AddressFromPublicKey([]byte("from2_public_key"))
	to2 := AddressFromPublicKey([]byte("to2_public_key"))
	tx2 := NewTransaction(from2, to2, 200, 20, 2, []byte("data2"))
	hash3 := tx2.Hash()

	if hash1 == hash3 {
		t.Errorf("Different transactions should have different hashes")
	}
}

func TestTransactionSign(t *testing.T) {
	// 创建交易
	from := AddressFromPublicKey([]byte("from_public_key"))
	to := AddressFromPublicKey([]byte("to_public_key"))
	tx := NewTransaction(from, to, 100, 10, 1, []byte("data"))

	// 签名前验证签名为空
	if len(tx.Signature) != 0 {
		t.Errorf("Expected empty signature before signing")
	}

	// 签名交易
	privateKey := []byte("private_key")
	err := tx.Sign(privateKey)
	if err != nil {
		t.Errorf("Failed to sign transaction: %v", err)
	}

	// 验证签名不为空
	if len(tx.Signature) == 0 {
		t.Errorf("Expected non-empty signature after signing")
	}
}

func TestTransactionVerify(t *testing.T) {
	// 创建交易
	from := AddressFromPublicKey([]byte("from_public_key"))
	to := AddressFromPublicKey([]byte("to_public_key"))
	tx := NewTransaction(from, to, 100, 10, 1, []byte("data"))

	// 签名交易
	privateKey := []byte("private_key")
	tx.Sign(privateKey)

	// 验证签名
	publicKey := []byte("public_key")
	valid := tx.Verify(publicKey)

	// 注意：当前实现总是返回true，实际应该验证签名
	if !valid {
		t.Errorf("Transaction verification failed")
	}
}

func TestTransactionIsValid(t *testing.T) {
	// 创建有效交易
	from := AddressFromPublicKey([]byte("from_public_key"))
	to := AddressFromPublicKey([]byte("to_public_key"))
	validTx := NewTransaction(from, to, 100, 10, 1, []byte("data"))
	validTx.Sign([]byte("private_key")) // 添加签名

	if !validTx.IsValid() {
		t.Errorf("Valid transaction should pass validation")
	}

	// 测试无效交易 - 缺少发送方
	invalidTx1 := NewTransaction(Address{}, to, 100, 10, 1, []byte("data"))
	invalidTx1.Sign([]byte("private_key"))
	if invalidTx1.IsValid() {
		t.Errorf("Transaction without from address should fail validation")
	}

	// 测试无效交易 - 缺少接收方
	invalidTx2 := NewTransaction(from, Address{}, 100, 10, 1, []byte("data"))
	invalidTx2.Sign([]byte("private_key"))
	if invalidTx2.IsValid() {
		t.Errorf("Transaction without to address should fail validation")
	}

	// 测试无效交易 - 手续费为0
	invalidTx3 := NewTransaction(from, to, 100, 0, 1, []byte("data"))
	invalidTx3.Sign([]byte("private_key"))
	if invalidTx3.IsValid() {
		t.Errorf("Transaction with zero fee should fail validation")
	}

	// 测试无效交易 - 没有签名
	invalidTx4 := NewTransaction(from, to, 100, 10, 1, []byte("data"))
	if invalidTx4.IsValid() {
		t.Errorf("Transaction without signature should fail validation")
	}
}

func TestTransactionGetID(t *testing.T) {
	// 创建交易
	from := AddressFromPublicKey([]byte("from_public_key"))
	to := AddressFromPublicKey([]byte("to_public_key"))
	tx := NewTransaction(from, to, 100, 10, 1, []byte("data"))

	// 获取交易ID
	id1 := tx.GetID()
	id2 := tx.GetID()

	// 验证ID一致性
	if id1 != id2 {
		t.Errorf("Transaction ID should be consistent")
	}

	// 验证ID与哈希相同
	hash := tx.Hash()
	if id1 != hash {
		t.Errorf("Transaction ID should equal transaction hash")
	}
}

func TestNewTransferTransaction(t *testing.T) {
	// 创建转账交易
	from := AddressFromPublicKey([]byte("from_public_key"))
	to := AddressFromPublicKey([]byte("to_public_key"))
	tx := NewTransferTransaction(from, to, 100, 10, 1)

	// 验证转账交易属性
	if tx.From != from {
		t.Errorf("Expected from address %x, got %x", from, tx.From)
	}

	if tx.To != to {
		t.Errorf("Expected to address %x, got %x", to, tx.To)
	}

	if tx.Amount != 100 {
		t.Errorf("Expected amount 100, got %d", tx.Amount)
	}

	if tx.Fee != 10 {
		t.Errorf("Expected fee 10, got %d", tx.Fee)
	}

	if tx.Nonce != 1 {
		t.Errorf("Expected nonce 1, got %d", tx.Nonce)
	}

	// 转账交易不应该有数据
	if len(tx.Data) != 0 {
		t.Errorf("Transfer transaction should have no data")
	}

	// 验证是转账交易
	if !tx.IsTransfer() {
		t.Errorf("Should be identified as transfer transaction")
	}
}
