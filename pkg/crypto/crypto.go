// Package crypto 提供ShardMatrix区块链的密码学功能
package crypto

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"math/big"

	"github.com/govm-net/shardmatrix/pkg/types"
)

// KeyPair 密钥对
type KeyPair struct {
	PrivateKey *ecdsa.PrivateKey
	PublicKey  *ecdsa.PublicKey
}

// GenerateKeyPair 生成新的密钥对
func GenerateKeyPair() (*KeyPair, error) {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate private key: %v", err)
	}

	return &KeyPair{
		PrivateKey: privateKey,
		PublicKey:  &privateKey.PublicKey,
	}, nil
}

// GetAddress 从公钥生成地址
func (kp *KeyPair) GetAddress() types.Address {
	return PublicKeyToAddress(&kp.PrivateKey.PublicKey)
}

// SignData 使用私钥对数据进行签名
func (kp *KeyPair) SignData(data []byte) ([]byte, error) {
	return SignData(data, kp.PrivateKey)
}

// VerifySignature 使用公钥验证签名
func (kp *KeyPair) VerifySignature(data []byte, signature []byte) bool {
	return VerifySignature(data, signature, &kp.PrivateKey.PublicKey)
}

// PublicKeyToAddress 从公钥生成地址
func PublicKeyToAddress(pubKey *ecdsa.PublicKey) types.Address {
	// 将公钥坐标转换为字节
	pubKeyBytes := append(pubKey.X.Bytes(), pubKey.Y.Bytes()...)

	// 计算Keccak256哈希（这里使用SHA256代替）
	hash := sha256.Sum256(pubKeyBytes)

	// 取后20字节作为地址
	var address types.Address
	copy(address[:], hash[12:])

	return address
}

// PublicKeyFromBytes 从字节数组恢复公钥
func PublicKeyFromBytes(pubKeyBytes []byte) (*ecdsa.PublicKey, error) {
	if len(pubKeyBytes) != 64 {
		return nil, fmt.Errorf("invalid public key length: expected 64, got %d", len(pubKeyBytes))
	}

	pubKey := &ecdsa.PublicKey{
		Curve: elliptic.P256(),
		X:     new(big.Int).SetBytes(pubKeyBytes[:32]),
		Y:     new(big.Int).SetBytes(pubKeyBytes[32:]),
	}

	return pubKey, nil
}

// SignData 使用私钥对数据进行ECDSA签名
func SignData(data []byte, privateKey *ecdsa.PrivateKey) ([]byte, error) {
	// 计算数据的哈希
	hash := sha256.Sum256(data)

	// 使用ECDSA签名
	r, s, err := ecdsa.Sign(rand.Reader, privateKey, hash[:])
	if err != nil {
		return nil, fmt.Errorf("failed to sign data: %v", err)
	}

	// 将r和s拼接成签名
	signature := append(r.Bytes(), s.Bytes()...)
	return signature, nil
}

// VerifySignature 验证ECDSA签名
func VerifySignature(data []byte, signature []byte, publicKey *ecdsa.PublicKey) bool {
	if len(signature) < 64 {
		return false
	}

	// 分离r和s
	r := new(big.Int).SetBytes(signature[:32])
	s := new(big.Int).SetBytes(signature[32:64])

	// 计算数据的哈希
	hash := sha256.Sum256(data)

	// 验证签名
	return ecdsa.Verify(publicKey, hash[:], r, s)
}

// HashData 计算数据的SHA256哈希
func HashData(data []byte) types.Hash {
	hash := sha256.Sum256(data)
	return types.Hash(hash)
}

// ValidateAddress 验证地址格式是否正确
func ValidateAddress(address types.Address) bool {
	// 检查地址是否为全零
	return !address.IsZero()
}

// RecoverPublicKey 从签名和数据中恢复公钥（简化版本）
func RecoverPublicKey(data []byte, signature []byte) (*ecdsa.PublicKey, error) {
	// 注意：这是一个简化版本，实际的公钥恢复需要更复杂的算法
	// 在实际应用中，应该使用包含恢复ID的签名格式
	return nil, fmt.Errorf("public key recovery not implemented yet")
}

// Signer 签名者接口
type Signer interface {
	Sign(data []byte) ([]byte, error)
	GetPublicKey() *ecdsa.PublicKey
	GetAddress() types.Address
}

// Verifier 验证者接口
type Verifier interface {
	Verify(data []byte, signature []byte, publicKey *ecdsa.PublicKey) bool
}

// DefaultSigner 默认签名者实现
type DefaultSigner struct {
	keyPair *KeyPair
}

// NewSigner 创建新的签名者
func NewSigner(keyPair *KeyPair) *DefaultSigner {
	return &DefaultSigner{
		keyPair: keyPair,
	}
}

// Sign 实现Signer接口
func (s *DefaultSigner) Sign(data []byte) ([]byte, error) {
	return s.keyPair.SignData(data)
}

// GetPublicKey 获取公钥
func (s *DefaultSigner) GetPublicKey() *ecdsa.PublicKey {
	return s.keyPair.PublicKey
}

// GetAddress 获取地址
func (s *DefaultSigner) GetAddress() types.Address {
	return s.keyPair.GetAddress()
}

// DefaultVerifier 默认验证者实现
type DefaultVerifier struct{}

// NewVerifier 创建新的验证者
func NewVerifier() *DefaultVerifier {
	return &DefaultVerifier{}
}

// Verify 实现Verifier接口
func (v *DefaultVerifier) Verify(data []byte, signature []byte, publicKey *ecdsa.PublicKey) bool {
	return VerifySignature(data, signature, publicKey)
}
