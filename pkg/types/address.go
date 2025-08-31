package types

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"
)

// Address 20字节地址类型
type Address [20]byte

// String 返回地址的十六进制字符串表示
func (a Address) String() string {
	return "0x" + hex.EncodeToString(a[:])
}

// Bytes 返回地址的字节切片
func (a Address) Bytes() []byte {
	return a[:]
}

// IsZero 检查地址是否为零值
func (a Address) IsZero() bool {
	return a == Address{}
}

// Equal 比较两个地址是否相等
func (a Address) Equal(other Address) bool {
	return a == other
}

// NewAddress 从字节数据创建Address
func NewAddress(data []byte) Address {
	var addr Address
	if len(data) >= 20 {
		copy(addr[:], data[:20])
	} else {
		copy(addr[:], data)
	}
	return addr
}

// AddressFromString 从十六进制字符串创建Address
func AddressFromString(s string) (Address, error) {
	// 去除0x前缀
	if len(s) >= 2 && s[:2] == "0x" {
		s = s[2:]
	}

	if len(s) != 40 { // 20字节 = 40个十六进制字符
		return Address{}, fmt.Errorf("invalid address string length: expected 40, got %d", len(s))
	}

	data, err := hex.DecodeString(s)
	if err != nil {
		return Address{}, fmt.Errorf("invalid hex string: %w", err)
	}

	return NewAddress(data), nil
}

// AddressFromPublicKey 从公钥创建Address
func AddressFromPublicKey(pubKey []byte) Address {
	// 对公钥进行哈希，取前20字节作为地址
	hash := sha256.Sum256(pubKey)
	return NewAddress(hash[:20])
}

// GenerateRandomAddress 生成随机地址（仅用于测试）
func GenerateRandomAddress() Address {
	// 使用crypto/rand生成真正的随机数据
	data := make([]byte, 20)
	if _, err := rand.Read(data); err != nil {
		// 如果随机数生成失败，使用时间戳作为种子
		now := time.Now().UnixNano()
		hash := sha256.Sum256([]byte(fmt.Sprintf("%d", now)))
		copy(data, hash[:20])
	}
	return NewAddress(data)
}

// GenerateAddress 生成地址（测试用别名）
func GenerateAddress() Address {
	return GenerateRandomAddress()
}

// EmptyAddress 返回空地址
func EmptyAddress() Address {
	return Address{}
}
