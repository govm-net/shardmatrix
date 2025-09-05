package types

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 生成测试用的密钥对（避免导入crypto包导致循环依赖）
func generateTestBlockKeyPair() (*ecdsa.PrivateKey, *ecdsa.PublicKey, error) {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, err
	}
	return privateKey, &privateKey.PublicKey, nil
}

func TestBlockSignAndVerify(t *testing.T) {
	// 生成密钥对
	privateKey, publicKey, err := generateTestBlockKeyPair()
	require.NoError(t, err)

	// 创建区块
	validator := Address{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20}
	prevHash := EmptyHash()
	block := NewBlock(1, prevHash, validator)

	// 签名区块
	err = block.SignBlock(privateKey)
	require.NoError(t, err)
	assert.NotEmpty(t, block.Header.Signature)

	// 验证区块签名
	valid := block.VerifyBlockSignature(publicKey)
	assert.True(t, valid)

	// 测试无效签名
	invalidKey, invalidPublicKey, err := generateTestBlockKeyPair()
	require.NoError(t, err)
	valid = block.VerifyBlockSignature(invalidPublicKey)
	assert.False(t, valid)
	_ = invalidKey // 避免未使用变量警告
}

func TestBlockHash(t *testing.T) {
	// 创建区块
	validator := Address{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20}
	prevHash := EmptyHash()
	block := NewBlock(1, prevHash, validator)

	// 计算哈希
	hash1 := block.Hash()
	hash2 := block.Hash()

	// 验证哈希一致性
	assert.Equal(t, hash1, hash2)
	assert.False(t, hash1.IsZero())
}

func TestBlockTransactions(t *testing.T) {
	// 创建区块
	validator := Address{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20}
	prevHash := EmptyHash()
	block := NewBlock(1, prevHash, validator)

	// 添加交易
	txHash1 := Hash{1, 2, 3}
	txHash2 := Hash{4, 5, 6}
	block.AddTransaction(txHash1)
	block.AddTransaction(txHash2)

	// 验证交易数量
	assert.Equal(t, 2, block.GetTransactionCount())

	// 验证交易存在
	assert.True(t, block.HasTransaction(txHash1))
	assert.True(t, block.HasTransaction(txHash2))
	assert.False(t, block.HasTransaction(Hash{7, 8, 9}))
}

func TestCalculateTxRoot(t *testing.T) {
	// 创建区块
	validator := Address{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20}
	prevHash := EmptyHash()
	block := NewBlock(1, prevHash, validator)

	// 添加交易
	txHash1 := Hash{1, 2, 3}
	txHash2 := Hash{4, 5, 6}
	block.AddTransaction(txHash1)
	block.AddTransaction(txHash2)

	// 计算交易根
	txRoot := block.CalculateTxRoot()
	assert.False(t, txRoot.IsZero())

	// 空交易的根
	emptyBlock := NewBlock(2, prevHash, validator)
	emptyTxRoot := emptyBlock.CalculateTxRoot()
	assert.True(t, emptyTxRoot.IsZero())
}

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
