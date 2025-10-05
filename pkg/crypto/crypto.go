package crypto

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/lengzhao/shardmatrix/pkg/types"
)

// 错误定义
var (
	ErrInvalidSignature  = errors.New("invalid signature")
	ErrInvalidPublicKey  = errors.New("invalid public key")
	ErrInvalidPrivateKey = errors.New("invalid private key")
)

// KeyPair 密钥对结构
type KeyPair struct {
	PrivateKey ed25519.PrivateKey
	PublicKey  ed25519.PublicKey
	Address    types.Address
}

// GenerateKeyPair 生成新的密钥对
func GenerateKeyPair() (*KeyPair, error) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate key pair: %w", err)
	}

	address := PublicKeyToAddress(publicKey)

	return &KeyPair{
		PrivateKey: privateKey,
		PublicKey:  publicKey,
		Address:    address,
	}, nil
}

// NewKeyPairFromPrivateKey 从私钥创建密钥对
func NewKeyPairFromPrivateKey(privateKey ed25519.PrivateKey) (*KeyPair, error) {
	if len(privateKey) != ed25519.PrivateKeySize {
		return nil, ErrInvalidPrivateKey
	}

	publicKey := privateKey.Public().(ed25519.PublicKey)
	address := PublicKeyToAddress(publicKey)

	return &KeyPair{
		PrivateKey: privateKey,
		PublicKey:  publicKey,
		Address:    address,
	}, nil
}

// NewKeyPairFromHex 从十六进制字符串创建密钥对
func NewKeyPairFromHex(privateKeyHex string) (*KeyPair, error) {
	privateKeyBytes, err := hex.DecodeString(privateKeyHex)
	if err != nil {
		return nil, fmt.Errorf("failed to decode private key hex: %w", err)
	}

	return NewKeyPairFromPrivateKey(ed25519.PrivateKey(privateKeyBytes))
}

// Sign 使用私钥签名数据
func (kp *KeyPair) Sign(data []byte) types.Signature {
	signature := ed25519.Sign(kp.PrivateKey, data)
	var sig types.Signature
	copy(sig[:], signature)
	return sig
}

// PrivateKeyHex 返回私钥的十六进制表示
func (kp *KeyPair) PrivateKeyHex() string {
	return hex.EncodeToString(kp.PrivateKey)
}

// PublicKeyHex 返回公钥的十六进制表示
func (kp *KeyPair) PublicKeyHex() string {
	return hex.EncodeToString(kp.PublicKey)
}

// Hash 计算数据的SHA256哈希
func Hash(data []byte) types.Hash {
	return sha256.Sum256(data)
}

// HashString 计算字符串的SHA256哈希
func HashString(s string) types.Hash {
	return Hash([]byte(s))
}

// PublicKeyToAddress 从公钥生成地址
func PublicKeyToAddress(publicKey ed25519.PublicKey) types.Address {
	hash := sha256.Sum256(publicKey)
	var addr types.Address
	copy(addr[:], hash[12:]) // 取后20字节作为地址
	return addr
}

// AddressFromHex 从十六进制字符串创建地址
func AddressFromHex(hexStr string) (types.Address, error) {
	bytes, err := hex.DecodeString(hexStr)
	if err != nil {
		return types.Address{}, fmt.Errorf("failed to decode address hex: %w", err)
	}

	if len(bytes) != 20 {
		return types.Address{}, fmt.Errorf("invalid address length: expected 20, got %d", len(bytes))
	}

	var addr types.Address
	copy(addr[:], bytes)
	return addr, nil
}

// HashFromHex 从十六进制字符串创建哈希
func HashFromHex(hexStr string) (types.Hash, error) {
	bytes, err := hex.DecodeString(hexStr)
	if err != nil {
		return types.Hash{}, fmt.Errorf("failed to decode hash hex: %w", err)
	}

	if len(bytes) != 32 {
		return types.Hash{}, fmt.Errorf("invalid hash length: expected 32, got %d", len(bytes))
	}

	var hash types.Hash
	copy(hash[:], bytes)
	return hash, nil
}

// SignatureFromHex 从十六进制字符串创建签名
func SignatureFromHex(hexStr string) (types.Signature, error) {
	bytes, err := hex.DecodeString(hexStr)
	if err != nil {
		return types.Signature{}, fmt.Errorf("failed to decode signature hex: %w", err)
	}

	if len(bytes) != 64 {
		return types.Signature{}, fmt.Errorf("invalid signature length: expected 64, got %d", len(bytes))
	}

	var sig types.Signature
	copy(sig[:], bytes)
	return sig, nil
}

// VerifySignature 验证签名
func VerifySignature(publicKey ed25519.PublicKey, data []byte, signature types.Signature) bool {
	return ed25519.Verify(publicKey, data, signature[:])
}

// VerifySignatureWithAddress 使用地址验证签名（需要公钥恢复机制）
func VerifySignatureWithAddress(address types.Address, data []byte, signature types.Signature, publicKey ed25519.PublicKey) bool {
	// 验证公钥是否对应该地址
	expectedAddr := PublicKeyToAddress(publicKey)
	if expectedAddr != address {
		return false
	}

	// 验证签名
	return VerifySignature(publicKey, data, signature)
}

// SignTransaction 签名交易
func (kp *KeyPair) SignTransaction(tx *types.Transaction) error {
	// 创建交易副本，清空签名字段
	txCopy := *tx
	txCopy.Signature = types.Signature{}

	// 序列化交易数据
	data, err := SerializeForSigning(&txCopy)
	if err != nil {
		return fmt.Errorf("failed to serialize transaction for signing: %w", err)
	}

	// 签名
	signature := kp.Sign(data)
	tx.Signature = signature

	return nil
}

// VerifyTransaction 验证交易签名
func VerifyTransaction(tx *types.Transaction, publicKey ed25519.PublicKey) bool {
	// 创建交易副本，清空签名字段
	txCopy := *tx
	txCopy.Signature = types.Signature{}

	// 序列化交易数据
	data, err := SerializeForSigning(&txCopy)
	if err != nil {
		return false
	}

	// 验证签名
	return VerifySignature(publicKey, data, tx.Signature)
}

// SignBlock 签名区块头
func (kp *KeyPair) SignBlock(header *types.BlockHeader) error {
	// 创建区块头副本，清空签名字段
	headerCopy := *header
	headerCopy.Signature = types.Signature{}

	// 序列化区块头数据
	data, err := SerializeForSigning(&headerCopy)
	if err != nil {
		return fmt.Errorf("failed to serialize block header for signing: %w", err)
	}

	// 签名
	signature := kp.Sign(data)
	header.Signature = signature

	return nil
}

// VerifyBlock 验证区块签名
func VerifyBlock(header *types.BlockHeader, publicKey ed25519.PublicKey) bool {
	// 创建区块头副本，清空签名字段
	headerCopy := *header
	headerCopy.Signature = types.Signature{}

	// 序列化区块头数据
	data, err := SerializeForSigning(&headerCopy)
	if err != nil {
		return false
	}

	// 验证签名
	return VerifySignature(publicKey, data, header.Signature)
}

// SerializeForSigning 为签名序列化数据（使用binary库）
func SerializeForSigning(data interface{}) ([]byte, error) {
	// 这里需要导入binary库
	// 为了避免循环依赖，暂时使用简单的实现
	switch v := data.(type) {
	case *types.Transaction:
		return serializeTransaction(v), nil
	case *types.BlockHeader:
		return serializeBlockHeader(v), nil
	default:
		return nil, fmt.Errorf("unsupported data type for signing")
	}
}

// serializeTransaction 序列化交易（简化版本）
func serializeTransaction(tx *types.Transaction) []byte {
	var data []byte
	// 这是一个简化的序列化实现
	// 实际项目中应该使用binary.Marshal
	data = append(data, byte(tx.ShardID))
	data = append(data, tx.From[:]...)
	data = append(data, tx.To[:]...)
	data = append(data, byte(tx.Amount))
	data = append(data, byte(tx.GasPrice))
	data = append(data, byte(tx.GasLimit))
	data = append(data, byte(tx.Nonce))
	data = append(data, tx.Data...)
	return data
}

// serializeBlockHeader 序列化区块头（简化版本）
func serializeBlockHeader(header *types.BlockHeader) []byte {
	var data []byte
	// 这是一个简化的序列化实现
	// 实际项目中应该使用binary.Marshal
	data = append(data, byte(header.Number))
	data = append(data, byte(header.Timestamp))
	data = append(data, header.PrevHash[:]...)
	data = append(data, header.TxRoot[:]...)
	data = append(data, header.StateRoot[:]...)
	data = append(data, header.Validator[:]...)
	data = append(data, byte(header.ShardID))
	for _, hash := range header.AdjacentHashes {
		data = append(data, hash[:]...)
	}
	return data
}

// 辅助函数

// RandomHash 生成随机哈希（用于测试）
func RandomHash() types.Hash {
	var hash types.Hash
	rand.Read(hash[:])
	return hash
}

// RandomAddress 生成随机地址（用于测试）
func RandomAddress() types.Address {
	var addr types.Address
	rand.Read(addr[:])
	return addr
}

// RandomSignature 生成随机签名（用于测试）
func RandomSignature() types.Signature {
	var sig types.Signature
	rand.Read(sig[:])
	return sig
}

// ZeroHash 返回零哈希
func ZeroHash() types.Hash {
	return types.Hash{}
}

// ZeroAddress 返回零地址
func ZeroAddress() types.Address {
	return types.Address{}
}

// IsValidAddress 检查地址是否有效
func IsValidAddress(addr types.Address) bool {
	return !addr.IsEmpty()
}

// IsValidHash 检查哈希是否有效
func IsValidHash(hash types.Hash) bool {
	return !hash.IsEmpty()
}

// IsValidSignature 检查签名是否有效
func IsValidSignature(sig types.Signature) bool {
	return !sig.IsEmpty()
}
