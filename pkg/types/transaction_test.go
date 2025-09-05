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
func generateTestTxKeyPair() (*ecdsa.PrivateKey, *ecdsa.PublicKey, error) {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, err
	}
	return privateKey, &privateKey.PublicKey, nil
}

func TestTransactionSignAndVerify(t *testing.T) {
	// 生成密钥对
	privateKey, publicKey, err := generateTestTxKeyPair()
	require.NoError(t, err)

	// 创建交易
	from := Address{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20}
	to := Address{21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33, 34, 35, 36, 37, 38, 39, 40}
	tx := NewTransaction(from, to, 100, 10, 1, []byte("test data"))

	// 签名交易
	err = tx.Sign(privateKey)
	require.NoError(t, err)
	assert.NotEmpty(t, tx.Signature)

	// 验证交易签名
	valid := tx.Verify(publicKey)
	assert.True(t, valid)

	// 测试无效签名
	_, invalidPublicKey, err := generateTestTxKeyPair()
	require.NoError(t, err)
	valid = tx.Verify(invalidPublicKey)
	assert.False(t, valid)
}

func TestTransactionHash(t *testing.T) {
	// 创建交易
	from := Address{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20}
	to := Address{21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33, 34, 35, 36, 37, 38, 39, 40}
	tx1 := NewTransaction(from, to, 100, 10, 1, []byte("test data"))
	tx2 := NewTransaction(from, to, 100, 10, 1, []byte("test data"))

	// 验证哈希一致性
	hash1 := tx1.Hash()
	hash2 := tx2.Hash()
	assert.Equal(t, hash1, hash2)
	assert.False(t, hash1.IsZero())
}

func TestTransactionValidation(t *testing.T) {
	// 创建有效交易
	from := Address{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20}
	to := Address{21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33, 34, 35, 36, 37, 38, 39, 40}
	tx := NewTransaction(from, to, 100, 10, 1, []byte("test data"))

	// 签名交易
	privateKey, _, err := generateTestTxKeyPair()
	require.NoError(t, err)
	tx.From = from
	err = tx.Sign(privateKey)
	require.NoError(t, err)

	// 验证有效交易
	assert.True(t, tx.IsValid())

	// 测试无效交易 - 空发送方
	invalidTx := NewTransaction(EmptyAddress(), to, 100, 10, 1, []byte("test data"))
	assert.False(t, invalidTx.IsValid())

	// 测试无效交易 - 空接收方
	invalidTx = NewTransaction(from, EmptyAddress(), 100, 10, 1, []byte("test data"))
	assert.False(t, invalidTx.IsValid())

	// 测试无效交易 - 零手续费
	invalidTx = NewTransaction(from, to, 100, 0, 1, []byte("test data"))
	assert.False(t, invalidTx.IsValid())

	// 测试无效交易 - 无签名
	invalidTx = NewTransaction(from, to, 100, 10, 1, []byte("test data"))
	invalidTx.Signature = nil
	assert.False(t, invalidTx.IsValid())
}

func TestTransactionCost(t *testing.T) {
	// 创建交易
	from := Address{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20}
	to := Address{21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33, 34, 35, 36, 37, 38, 39, 40}
	tx := NewTransaction(from, to, 100, 10, 1, []byte("test data"))

	// 验证总成本
	assert.Equal(t, uint64(110), tx.GetTotalCost()) // 金额100 + 手续费10
}
