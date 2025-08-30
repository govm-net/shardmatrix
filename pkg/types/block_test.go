package types

import (
	"testing"
)

func TestNewBlock(t *testing.T) {
	// 创建新区块
	block := NewBlock(1, []byte("prev_hash"), 1, []byte("validator"))

	// 验证区块头
	if block.Header.Number != 1 {
		t.Errorf("Expected block number 1, got %d", block.Header.Number)
	}

	if string(block.Header.PrevHash) != "prev_hash" {
		t.Errorf("Expected prev hash 'prev_hash', got %s", string(block.Header.PrevHash))
	}

	if block.Header.ChainID != 1 {
		t.Errorf("Expected chain ID 1, got %d", block.Header.ChainID)
	}

	if string(block.Header.Validator) != "validator" {
		t.Errorf("Expected validator 'validator', got %s", string(block.Header.Validator))
	}

	// 验证交易列表为空
	if len(block.Transactions) != 0 {
		t.Errorf("Expected empty transactions, got %d", len(block.Transactions))
	}
}

func TestBlockHash(t *testing.T) {
	// 创建区块
	block := NewBlock(1, []byte("prev_hash"), 1, []byte("validator"))

	// 计算哈希
	hash1 := block.Hash()
	hash2 := block.Hash()

	// 验证哈希一致性
	if string(hash1) != string(hash2) {
		t.Errorf("Block hash should be consistent, got %x and %x", hash1, hash2)
	}

	// 验证哈希长度
	if len(hash1) != 32 {
		t.Errorf("Expected hash length 32, got %d", len(hash1))
	}
}

func TestAddTransaction(t *testing.T) {
	// 创建区块
	block := NewBlock(1, []byte("prev_hash"), 1, []byte("validator"))

	// 创建交易
	tx := NewTransaction([]byte("from"), []byte("to"), 100, []byte("data"))
	txHash := tx.Hash()

	// 添加交易到区块
	block.AddTransaction(txHash)

	// 验证交易已添加
	if len(block.Transactions) != 1 {
		t.Errorf("Expected 1 transaction, got %d", len(block.Transactions))
	}

	if string(block.Transactions[0]) != string(txHash) {
		t.Errorf("Transaction hash mismatch")
	}
}

func TestCalculateTxRoot(t *testing.T) {
	// 创建区块
	block := NewBlock(1, []byte("prev_hash"), 1, []byte("validator"))

	// 空区块的Merkle根
	emptyRoot := block.CalculateTxRoot()
	if len(emptyRoot) != 32 {
		t.Errorf("Expected empty root length 32, got %d", len(emptyRoot))
	}

	// 添加交易
	tx1 := NewTransaction([]byte("from1"), []byte("to1"), 100, []byte("data1"))
	tx2 := NewTransaction([]byte("from2"), []byte("to2"), 200, []byte("data2"))

	block.AddTransaction(tx1.Hash())
	block.AddTransaction(tx2.Hash())

	// 计算Merkle根
	root := block.CalculateTxRoot()
	if len(root) != 32 {
		t.Errorf("Expected root length 32, got %d", len(root))
	}

	// 验证根不为空
	allZero := true
	for _, b := range root {
		if b != 0 {
			allZero = false
			break
		}
	}
	if allZero {
		t.Errorf("Merkle root should not be all zeros")
	}
}
