package types

import (
	"testing"
)

func TestNewBlock(t *testing.T) {
	// 创建测试数据
	prevHash := NewHash([]byte("prev_hash_data"))
	validator := AddressFromPublicKey([]byte("validator_public_key"))

	// 创建新区块
	block := NewBlock(1, prevHash, validator)

	// 验证区块头
	if block.Header.Number != 1 {
		t.Errorf("Expected block number 1, got %d", block.Header.Number)
	}

	if block.Header.PrevHash != prevHash {
		t.Errorf("Expected prev hash %x, got %x", prevHash, block.Header.PrevHash)
	}

	if block.Header.Validator != validator {
		t.Errorf("Expected validator %x, got %x", validator, block.Header.Validator)
	}

	// 验证时间戳不为0
	if block.Header.Timestamp <= 0 {
		t.Errorf("Expected positive timestamp, got %d", block.Header.Timestamp)
	}

	// 验证交易列表为空
	if len(block.Transactions) != 0 {
		t.Errorf("Expected empty transactions, got %d", len(block.Transactions))
	}
}

func TestBlockHash(t *testing.T) {
	// 创建测试数据
	prevHash := NewHash([]byte("prev_hash_data"))
	validator := AddressFromPublicKey([]byte("validator_public_key"))

	// 创建区块
	block := NewBlock(1, prevHash, validator)

	// 计算哈希
	hash1 := block.Hash()
	hash2 := block.Hash()

	// 验证哈希一致性
	if hash1 != hash2 {
		t.Errorf("Block hash should be consistent, got %x and %x", hash1, hash2)
	}

	// 验证哈希长度
	if len(hash1.Bytes()) != 32 {
		t.Errorf("Expected hash length 32, got %d", len(hash1.Bytes()))
	}

	// 验证哈希不为空
	if hash1.IsZero() {
		t.Errorf("Block hash should not be zero")
	}
}

func TestAddTransaction(t *testing.T) {
	// 创建测试数据
	prevHash := NewHash([]byte("prev_hash_data"))
	validator := AddressFromPublicKey([]byte("validator_public_key"))

	// 创建区块
	block := NewBlock(1, prevHash, validator)

	// 创建交易
	from := AddressFromPublicKey([]byte("from_public_key"))
	to := AddressFromPublicKey([]byte("to_public_key"))
	tx := NewTransaction(from, to, 100, 10, 1, []byte("data"))
	txHash := tx.Hash()

	// 添加交易到区块
	block.AddTransaction(txHash)

	// 验证交易已添加
	if len(block.Transactions) != 1 {
		t.Errorf("Expected 1 transaction, got %d", len(block.Transactions))
	}

	if block.Transactions[0] != txHash {
		t.Errorf("Transaction hash mismatch")
	}

	// 验证TxRoot已更新
	if block.Header.TxRoot.IsZero() {
		t.Errorf("TxRoot should be updated after adding transaction")
	}
}

func TestCalculateTxRoot(t *testing.T) {
	// 创建测试数据
	prevHash := NewHash([]byte("prev_hash_data"))
	validator := AddressFromPublicKey([]byte("validator_public_key"))

	// 创建区块
	block := NewBlock(1, prevHash, validator)

	// 空区块的Merkle根
	emptyRoot := block.CalculateTxRoot()
	if len(emptyRoot.Bytes()) != 32 {
		t.Errorf("Expected empty root length 32, got %d", len(emptyRoot.Bytes()))
	}

	// 空区块的根应该是空哈希
	if !emptyRoot.IsZero() {
		t.Errorf("Empty block should have zero hash root")
	}

	// 添加交易
	from1 := AddressFromPublicKey([]byte("from1_public_key"))
	to1 := AddressFromPublicKey([]byte("to1_public_key"))
	tx1 := NewTransaction(from1, to1, 100, 10, 1, []byte("data1"))

	from2 := AddressFromPublicKey([]byte("from2_public_key"))
	to2 := AddressFromPublicKey([]byte("to2_public_key"))
	tx2 := NewTransaction(from2, to2, 200, 20, 1, []byte("data2"))

	block.AddTransaction(tx1.Hash())
	block.AddTransaction(tx2.Hash())

	// 计算Merkle根
	root := block.CalculateTxRoot()
	if len(root.Bytes()) != 32 {
		t.Errorf("Expected root length 32, got %d", len(root.Bytes()))
	}

	// 验证根不为空
	if root.IsZero() {
		t.Errorf("Merkle root should not be zero hash with transactions")
	}

	// 验证区块头中的TxRoot已更新
	if block.Header.TxRoot != root {
		t.Errorf("Block header TxRoot should match calculated root")
	}
}

func TestNewGenesisBlock(t *testing.T) {
	// 创建测试数据
	validator := AddressFromPublicKey([]byte("genesis_validator_public_key"))

	// 创建初始区块
	genesisBlock := NewGenesisBlock(validator)

	// 验证初始区块属性
	if genesisBlock.Header.Number != 0 {
		t.Errorf("Expected genesis block number 0, got %d", genesisBlock.Header.Number)
	}

	if !genesisBlock.Header.PrevHash.IsZero() {
		t.Errorf("Genesis block should have zero prev hash")
	}

	if genesisBlock.Header.Validator != validator {
		t.Errorf("Expected validator %x, got %x", validator, genesisBlock.Header.Validator)
	}

	if len(genesisBlock.Transactions) != 0 {
		t.Errorf("Genesis block should have no transactions")
	}
}

func TestBlockIsValid(t *testing.T) {
	// 创建有效区块
	validator := AddressFromPublicKey([]byte("validator_public_key"))
	validBlock := NewBlock(1, NewHash([]byte("prev_hash")), validator)

	if !validBlock.IsValid() {
		t.Errorf("Valid block should pass validation")
	}

	// 测试无效区块 - 没有区块头
	invalidBlock1 := &Block{Header: nil}
	if invalidBlock1.IsValid() {
		t.Errorf("Block without header should fail validation")
	}

	// 测试无效区块 - 时间戳为0
	invalidBlock2 := NewBlock(1, NewHash([]byte("prev_hash")), validator)
	invalidBlock2.Header.Timestamp = 0
	if invalidBlock2.IsValid() {
		t.Errorf("Block with zero timestamp should fail validation")
	}

	// 测试无效区块 - 验证者地址为空
	invalidBlock3 := NewBlock(1, NewHash([]byte("prev_hash")), Address{})
	if invalidBlock3.IsValid() {
		t.Errorf("Block with zero validator address should fail validation")
	}
}
