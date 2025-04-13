package tx

import (
	"context"
	"encoding/binary"
	"fmt"
	"sync"

	db "github.com/cometbft/cometbft-db"
	cometbftcryptoed25519 "github.com/cometbft/cometbft/crypto/ed25519"
)

// Transaction 交易结构
type Transaction struct {
	Type      byte   // 交易类型
	Data      []byte // 交易数据
	Signature []byte // 交易签名
	PublicKey []byte // 公钥
}

// Encode 编码交易
func (tx *Transaction) Encode() []byte {
	buf := make([]byte, 1+4+len(tx.Data)+4+len(tx.Signature)+4+len(tx.PublicKey))
	pos := 0

	// Type
	buf[pos] = tx.Type
	pos++

	// Data
	binary.BigEndian.PutUint32(buf[pos:], uint32(len(tx.Data)))
	pos += 4
	copy(buf[pos:], tx.Data)
	pos += len(tx.Data)

	// Signature
	binary.BigEndian.PutUint32(buf[pos:], uint32(len(tx.Signature)))
	pos += 4
	copy(buf[pos:], tx.Signature)
	pos += len(tx.Signature)

	// PublicKey
	binary.BigEndian.PutUint32(buf[pos:], uint32(len(tx.PublicKey)))
	pos += 4
	copy(buf[pos:], tx.PublicKey)

	return buf
}

// Decode 解码交易
func (tx *Transaction) Decode(data []byte) error {
	if len(data) < 1 {
		return fmt.Errorf("invalid transaction data")
	}

	pos := 0
	tx.Type = data[pos]
	pos++

	// Data
	if len(data) < pos+4 {
		return fmt.Errorf("invalid transaction data")
	}
	dataLen := binary.BigEndian.Uint32(data[pos:])
	pos += 4
	if len(data) < pos+int(dataLen) {
		return fmt.Errorf("invalid transaction data")
	}
	tx.Data = make([]byte, dataLen)
	copy(tx.Data, data[pos:pos+int(dataLen)])
	pos += int(dataLen)

	// Signature
	if len(data) < pos+4 {
		return fmt.Errorf("invalid transaction data")
	}
	sigLen := binary.BigEndian.Uint32(data[pos:])
	pos += 4
	if len(data) < pos+int(sigLen) {
		return fmt.Errorf("invalid transaction data")
	}
	tx.Signature = make([]byte, sigLen)
	copy(tx.Signature, data[pos:pos+int(sigLen)])
	pos += int(sigLen)

	// PublicKey
	if len(data) < pos+4 {
		return fmt.Errorf("invalid transaction data")
	}
	pubKeyLen := binary.BigEndian.Uint32(data[pos:])
	pos += 4
	if len(data) < pos+int(pubKeyLen) {
		return fmt.Errorf("invalid transaction data")
	}
	tx.PublicKey = make([]byte, pubKeyLen)
	copy(tx.PublicKey, data[pos:pos+int(pubKeyLen)])

	return nil
}

// Verify 验证交易签名
func (tx *Transaction) Verify() bool {
	if len(tx.PublicKey) != cometbftcryptoed25519.PubKeySize {
		return false
	}
	if len(tx.Signature) != cometbftcryptoed25519.SignatureSize {
		return false
	}

	pubKey := cometbftcryptoed25519.PubKey(tx.PublicKey)
	return pubKey.VerifySignature(tx.Data, tx.Signature)
}

// TxProcess 交易处理器接口
type TxProcess interface {
	// Type 返回交易类型
	Type() byte
	// Validate 验证交易
	Validate(ctx context.Context, db db.DB, txBytes []byte) error
	// Execute 执行交易
	Execute(ctx context.Context, db db.DB, txBytes []byte) error
}

// TxRegistry 交易处理器注册表
type TxRegistry struct {
	mu         sync.RWMutex
	processors map[byte]TxProcess
}

// 交易类型常量
const (
	TxTypeTransfer byte = 0x01
	TxTypeContract byte = 0x02
)

var DefaultTxRegistry = NewTxRegistry()

// NewTxRegistry 创建交易注册表
func NewTxRegistry() *TxRegistry {
	return &TxRegistry{
		processors: make(map[byte]TxProcess),
	}
}

// Register 注册交易处理器
func (r *TxRegistry) Register(p TxProcess) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.processors[p.Type()] = p
}

// GetProcessor 获取交易处理器
func (r *TxRegistry) GetProcessor(txType byte) (TxProcess, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	process, exists := r.processors[txType]
	return process, exists
}
