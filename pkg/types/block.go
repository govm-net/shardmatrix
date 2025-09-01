package types

import (
	"crypto/sha256"
	"encoding/json"
	"time"
)

// BlockHeader 区块头结构
type BlockHeader struct {
	Number    uint64  `json:"number"`     // 区块序号
	Timestamp int64   `json:"timestamp"`  // 时间戳
	PrevHash  Hash    `json:"prev_hash"`  // 前一个区块哈希
	TxRoot    Hash    `json:"tx_root"`    // 交易RootHash
	StateRoot Hash    `json:"state_root"` // 状态RootHash
	Validator Address `json:"validator"`  // 验证者地址
	Signature []byte  `json:"signature"`  // 验证者签名
}

// Block 区块结构
type Block struct {
	Header       *BlockHeader `json:"header"`
	Transactions []Hash       `json:"transactions"` // 交易哈希列表
}

// NewBlock 创建新区块
func NewBlock(number uint64, prevHash Hash, validator Address) *Block {
	return &Block{
		Header: &BlockHeader{
			Number:    number,
			Timestamp: time.Now().Unix(),
			PrevHash:  prevHash,
			Validator: validator,
			TxRoot:    EmptyHash(),
			StateRoot: EmptyHash(),
		},
		Transactions: make([]Hash, 0),
	}
}

// NewGenesisBlock 创建初始区块
func NewGenesisBlock(validator Address) *Block {
	block := NewBlock(0, EmptyHash(), validator)

	// 设置固定的创世区块时间戳
	block.Header.Timestamp = 1700000000 // 固定时间戳，确保所有节点生成相同的创世区块

	// 清空签名字段，创世区块不需要签名
	block.Header.Signature = nil

	return block
}

// Hash 计算区块哈希
func (b *Block) Hash() Hash {
	// 序列化区块头进行哈希计算
	headerBytes, err := json.Marshal(b.Header)
	if err != nil {
		// 如果序列化失败，返回空哈希
		return EmptyHash()
	}

	return CalculateHash(headerBytes)
}

// AddTransaction 添加交易到区块
func (b *Block) AddTransaction(txHash Hash) {
	b.Transactions = append(b.Transactions, txHash)
	// 更新交易Merkle根
	b.Header.TxRoot = b.CalculateTxRoot()
}

// CalculateTxRoot 计算交易Merkle根
func (b *Block) CalculateTxRoot() Hash {
	if len(b.Transactions) == 0 {
		return EmptyHash()
	}

	// 转换为[][]byte进行计算
	hashes := make([][]byte, len(b.Transactions))
	for i, txHash := range b.Transactions {
		hashes[i] = txHash.Bytes()
	}

	root := calculateMerkleRoot(hashes)
	return NewHash(root)
}

// IsValid 验证区块是否有效
func (b *Block) IsValid() bool {
	if b.Header == nil {
		return false
	}

	// 检查时间戳
	if b.Header.Timestamp <= 0 {
		return false
	}

	// 检查验证者地址
	if b.Header.Validator.IsZero() {
		return false
	}

	// 验证交易Merkle根
	expectedTxRoot := b.CalculateTxRoot()
	if !b.Header.TxRoot.Equal(expectedTxRoot) {
		return false
	}

	return true
}

// GetTransactionCount 获取交易数量
func (b *Block) GetTransactionCount() int {
	return len(b.Transactions)
}

// HasTransaction 检查区块是否包含指定交易
func (b *Block) HasTransaction(txHash Hash) bool {
	for _, hash := range b.Transactions {
		if hash.Equal(txHash) {
			return true
		}
	}
	return false
}

// Size 获取区块大小（字节）
func (b *Block) Size() int {
	data, err := json.Marshal(b)
	if err != nil {
		return 0
	}
	return len(data)
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

// Serialize 序列化区块为JSON
func (b *Block) Serialize() ([]byte, error) {
	return json.Marshal(b)
}

// Deserialize 从 JSON 反序列化区块
func (b *Block) Deserialize(data []byte) error {
	return json.Unmarshal(data, b)
}

// DeserializeBlock 从 JSON 创建区块
func DeserializeBlock(data []byte) (*Block, error) {
	var block Block
	err := json.Unmarshal(data, &block)
	if err != nil {
		return nil, err
	}
	return &block, nil
}

// Serialize 序列化区块头为JSON
func (bh *BlockHeader) Serialize() ([]byte, error) {
	return json.Marshal(bh)
}

// Deserialize 从 JSON 反序列化区块头
func (bh *BlockHeader) Deserialize(data []byte) error {
	return json.Unmarshal(data, bh)
}

// DeserializeBlockHeader 从 JSON 创建区块头
func DeserializeBlockHeader(data []byte) (*BlockHeader, error) {
	var header BlockHeader
	err := json.Unmarshal(data, &header)
	if err != nil {
		return nil, err
	}
	return &header, nil
}
