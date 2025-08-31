package types

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

// Hash 32字节哈希类型
type Hash [32]byte

// String 返回哈希的十六进制字符串表示
func (h Hash) String() string {
	return hex.EncodeToString(h[:])
}

// Bytes 返回哈希的字节切片
func (h Hash) Bytes() []byte {
	return h[:]
}

// IsZero 检查哈希是否为零值
func (h Hash) IsZero() bool {
	return h == Hash{}
}

// Equal 比较两个哈希是否相等
func (h Hash) Equal(other Hash) bool {
	return h == other
}

// NewHash 从字节数据创建Hash
func NewHash(data []byte) Hash {
	if len(data) != 32 {
		// 如果数据不是32字节，进行哈希计算
		sum := sha256.Sum256(data)
		return sum
	}

	var h Hash
	copy(h[:], data)
	return h
}

// HashFromString 从十六进制字符串创建Hash
func HashFromString(s string) (Hash, error) {
	if len(s) != 64 { // 32字节 = 64个十六进制字符
		return Hash{}, fmt.Errorf("invalid hash string length: expected 64, got %d", len(s))
	}

	data, err := hex.DecodeString(s)
	if err != nil {
		return Hash{}, fmt.Errorf("invalid hex string: %w", err)
	}

	return NewHash(data), nil
}

// CalculateHash 计算数据的SHA256哈希
func CalculateHash(data []byte) Hash {
	return sha256.Sum256(data)
}

// EmptyHash 返回空哈希
func EmptyHash() Hash {
	return Hash{}
}
