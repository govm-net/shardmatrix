package crypto

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"testing"

	"github.com/govm-net/shardmatrix/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKeyPairGeneration(t *testing.T) {
	keyPair, err := GenerateKeyPair()
	require.NoError(t, err)
	assert.NotNil(t, keyPair.PrivateKey)
	assert.NotNil(t, keyPair.PublicKey)
}

func TestSignAndVerifyData(t *testing.T) {
	keyPair, err := GenerateKeyPair()
	require.NoError(t, err)

	data := []byte("test data for signing")
	signature, err := keyPair.SignData(data)
	require.NoError(t, err)
	assert.NotEmpty(t, signature)

	// 验证正确签名
	valid := keyPair.VerifySignature(data, signature)
	assert.True(t, valid)

	// 验证错误签名
	invalidData := []byte("different data")
	valid = keyPair.VerifySignature(invalidData, signature)
	assert.False(t, valid)
}

func TestAddressGeneration(t *testing.T) {
	keyPair, err := GenerateKeyPair()
	require.NoError(t, err)

	address := keyPair.GetAddress()
	assert.False(t, address.IsZero())
}

func TestPublicKeyFromBytes(t *testing.T) {
	// 生成一个公钥
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	// 将公钥转换为字节
	pubKeyBytes := append(privateKey.PublicKey.X.Bytes(), privateKey.PublicKey.Y.Bytes()...)

	// 确保长度为64字节
	for len(pubKeyBytes) < 64 {
		pubKeyBytes = append([]byte{0}, pubKeyBytes...)
	}
	if len(pubKeyBytes) > 64 {
		pubKeyBytes = pubKeyBytes[len(pubKeyBytes)-64:]
	}

	// 从字节恢复公钥
	recoveredPubKey, err := PublicKeyFromBytes(pubKeyBytes)
	require.NoError(t, err)
	assert.Equal(t, privateKey.PublicKey.X, recoveredPubKey.X)
	assert.Equal(t, privateKey.PublicKey.Y, recoveredPubKey.Y)
}

func TestTransactionSignAndVerify(t *testing.T) {
	// 生成密钥对
	keyPair, err := GenerateKeyPair()
	require.NoError(t, err)

	// 创建交易
	from := keyPair.GetAddress()
	to := types.Address{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20}
	tx := types.NewTransaction(from, to, 100, 10, 1, []byte("test data"))

	// 签名交易
	err = tx.Sign(keyPair.PrivateKey)
	require.NoError(t, err)
	assert.NotEmpty(t, tx.Signature)

	// 验证交易签名
	valid := tx.Verify(&keyPair.PrivateKey.PublicKey)
	assert.True(t, valid)
}

func TestBlockSignAndVerify(t *testing.T) {
	// 生成密钥对
	keyPair, err := GenerateKeyPair()
	require.NoError(t, err)

	// 创建区块
	validator := keyPair.GetAddress()
	prevHash := types.EmptyHash()
	block := types.NewBlock(1, prevHash, validator)

	// 签名区块
	err = block.SignBlock(keyPair.PrivateKey)
	require.NoError(t, err)
	assert.NotEmpty(t, block.Header.Signature)

	// 验证区块签名
	valid := block.VerifyBlockSignature(&keyPair.PrivateKey.PublicKey)
	assert.True(t, valid)
}

func TestBlockHash(t *testing.T) {
	// 创建区块
	validator := types.Address{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20}
	prevHash := types.EmptyHash()
	block := types.NewBlock(1, prevHash, validator)

	// 计算哈希
	hash1 := block.Hash()
	hash2 := block.Hash()

	// 验证哈希一致性
	assert.Equal(t, hash1, hash2)
	assert.False(t, hash1.IsZero())
}

func TestBlockTransactions(t *testing.T) {
	// 创建区块
	validator := types.Address{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20}
	prevHash := types.EmptyHash()
	block := types.NewBlock(1, prevHash, validator)

	// 添加交易
	txHash1 := types.Hash{1, 2, 3}
	txHash2 := types.Hash{4, 5, 6}
	block.AddTransaction(txHash1)
	block.AddTransaction(txHash2)

	// 验证交易数量
	assert.Equal(t, 2, block.GetTransactionCount())

	// 验证交易存在
	assert.True(t, block.HasTransaction(txHash1))
	assert.True(t, block.HasTransaction(txHash2))
	assert.False(t, block.HasTransaction(types.Hash{7, 8, 9}))
}
