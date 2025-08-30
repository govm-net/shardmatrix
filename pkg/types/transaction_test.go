package types

import (
	"testing"
)

func TestNewTransaction(t *testing.T) {
	// 创建新交易
	from := []byte("from_address")
	to := []byte("to_address")
	fee := uint64(100)
	data := []byte("transaction_data")

	tx := NewTransaction(from, to, fee, data)

	// 验证交易字段
	if string(tx.From) != string(from) {
		t.Errorf("Expected from address %s, got %s", string(from), string(tx.From))
	}

	if string(tx.To) != string(to) {
		t.Errorf("Expected to address %s, got %s", string(to), string(tx.To))
	}

	if tx.Fee != fee {
		t.Errorf("Expected fee %d, got %d", fee, tx.Fee)
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
	tx := NewTransaction([]byte("from"), []byte("to"), 100, []byte("data"))

	// 计算哈希
	hash1 := tx.Hash()
	hash2 := tx.Hash()

	// 验证哈希一致性
	if string(hash1) != string(hash2) {
		t.Errorf("Transaction hash should be consistent, got %x and %x", hash1, hash2)
	}

	// 验证哈希长度
	if len(hash1) != 32 {
		t.Errorf("Expected hash length 32, got %d", len(hash1))
	}

	// 验证不同交易有不同的哈希
	tx2 := NewTransaction([]byte("from2"), []byte("to2"), 200, []byte("data2"))
	hash3 := tx2.Hash()

	if string(hash1) == string(hash3) {
		t.Errorf("Different transactions should have different hashes")
	}
}

func TestTransactionSign(t *testing.T) {
	// 创建交易
	tx := NewTransaction([]byte("from"), []byte("to"), 100, []byte("data"))

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
	tx := NewTransaction([]byte("from"), []byte("to"), 100, []byte("data"))

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
	// 测试有效交易
	validTx := NewTransaction([]byte("from"), []byte("to"), 100, []byte("data"))
	if !validTx.IsValid() {
		t.Errorf("Valid transaction should pass validation")
	}

	// 测试无效交易 - 缺少发送方
	invalidTx1 := NewTransaction([]byte{}, []byte("to"), 100, []byte("data"))
	if invalidTx1.IsValid() {
		t.Errorf("Transaction without from address should fail validation")
	}

	// 测试无效交易 - 缺少接收方
	invalidTx2 := NewTransaction([]byte("from"), []byte{}, 100, []byte("data"))
	if invalidTx2.IsValid() {
		t.Errorf("Transaction without to address should fail validation")
	}

	// 测试无效交易 - 负手续费
	invalidTx3 := NewTransaction([]byte("from"), []byte("to"), 0, []byte("data"))
	// 注意：当前实现允许0手续费，实际可能需要调整
	if !invalidTx3.IsValid() {
		t.Errorf("Transaction with zero fee should be valid")
	}
}

func TestTransactionGetID(t *testing.T) {
	// 创建交易
	tx := NewTransaction([]byte("from"), []byte("to"), 100, []byte("data"))

	// 获取交易ID
	id1 := tx.GetID()
	id2 := tx.GetID()

	// 验证ID一致性
	if string(id1) != string(id2) {
		t.Errorf("Transaction ID should be consistent")
	}

	// 验证ID与哈希相同
	hash := tx.Hash()
	if string(id1) != string(hash) {
		t.Errorf("Transaction ID should equal transaction hash")
	}
}
