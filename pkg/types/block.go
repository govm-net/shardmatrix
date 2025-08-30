package types

import (
	"bytes"
	"crypto/sha256"
	"encoding/gob"
	"time"
)

// BlockHeader 区块头结构
type BlockHeader struct {
	Number    uint64 `json:"number"`     // 区块序号
	Timestamp int64  `json:"timestamp"`  // 时间戳
	PrevHash  []byte `json:"prev_hash"`  // 前一个区块哈希
	ChainID   uint64 `json:"chain_id"`   // 链ID
	TxRoot    []byte `json:"tx_root"`    // 交易RootHash
	StateRoot []byte `json:"state_root"` // 状态RootHash
	Validator []byte `json:"validator"`  // 验证者地址
	Signature []byte `json:"signature"`  // 验证者签名
}

// Block 区块结构
type Block struct {
	Header       *BlockHeader `json:"header"`
	Transactions [][]byte     `json:"transactions"` // 交易哈希列表
}

// NewBlock 创建新区块
func NewBlock(number uint64, prevHash []byte, chainID uint64, validator []byte) *Block {
	return &Block{
		Header: &BlockHeader{
			Number:    number,
			Timestamp: time.Now().Unix(),
			PrevHash:  prevHash,
			ChainID:   chainID,
			Validator: validator,
		},
		Transactions: make([][]byte, 0),
	}
}

// Hash 计算区块哈希
func (b *Block) Hash() []byte {
	// 使用gob序列化区块头
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	enc.Encode(b.Header)

	hash := sha256.Sum256(buf.Bytes())
	return hash[:]
}

// AddTransaction 添加交易到区块
func (b *Block) AddTransaction(txHash []byte) {
	b.Transactions = append(b.Transactions, txHash)
}

// CalculateTxRoot 计算交易Merkle根
func (b *Block) CalculateTxRoot() []byte {
	if len(b.Transactions) == 0 {
		return make([]byte, 32)
	}

	// 直接使用交易哈希计算Merkle根
	root := calculateMerkleRoot(b.Transactions)
	return root
}

// calculateMerkleRoot 计算Merkle根
func calculateMerkleRoot(hashes [][]byte) []byte {
	if len(hashes) == 0 {
		return make([]byte, 32)
	}

	if len(hashes) == 1 {
		return hashes[0]
	}

	// 如果奇数个，复制最后一个
	if len(hashes)%2 == 1 {
		hashes = append(hashes, hashes[len(hashes)-1])
	}

	// 计算下一层
	nextLevel := make([][]byte, len(hashes)/2)
	for i := 0; i < len(hashes); i += 2 {
		combined := append(hashes[i], hashes[i+1]...)
		hash := sha256.Sum256(combined)
		nextLevel[i/2] = hash[:]
	}

	return calculateMerkleRoot(nextLevel)
}
