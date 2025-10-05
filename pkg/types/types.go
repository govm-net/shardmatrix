package types

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"time"
)

// Hash 表示32字节的哈希值
type Hash [32]byte

// Address 表示20字节的地址
type Address [20]byte

// Signature 表示Ed25519签名
type Signature [64]byte

// 常量定义
const (
	BlockTime      = 2 * time.Second // 严格2秒出块间隔
	MaxBlockSize   = 2 * 1024 * 1024 // 2MB最大区块大小
	MaxTxPerBlock  = 10000           // 每个区块最大交易数
	ValidatorCount = 21              // DPoS验证者数量
	MinStakeAmount = 10000           // 最小质押金额
	ShardID        = uint64(1)       // 第一阶段固定分片ID
)

// BlockHeader 区块头结构
type BlockHeader struct {
	Number         uint64    `binary:"number"`          // 区块高度
	Timestamp      int64     `binary:"timestamp"`       // 严格时间戳
	PrevHash       Hash      `binary:"prev_hash"`       // 前一区块哈希
	TxRoot         Hash      `binary:"tx_root"`         // 交易Merkle根
	StateRoot      Hash      `binary:"state_root"`      // 状态根
	Validator      Address   `binary:"validator"`       // 验证者地址
	Signature      Signature `binary:"signature"`       // 验证者签名
	ShardID        uint64    `binary:"shard_id"`        // 分片ID
	AdjacentHashes [3]Hash   `binary:"adjacent_hashes"` // 相邻分片哈希
}

// Block 区块结构
type Block struct {
	Header       BlockHeader `binary:"header"`       // 区块头
	Transactions []Hash      `binary:"transactions"` // 交易哈希列表
}

// Transaction 交易结构
type Transaction struct {
	ShardID   uint64    `binary:"shard_id"`  // 分片ID
	From      Address   `binary:"from"`      // 发送方地址
	To        Address   `binary:"to"`        // 接收方地址
	Amount    uint64    `binary:"amount"`    // 转账金额
	GasPrice  uint64    `binary:"gas_price"` // Gas价格
	GasLimit  uint64    `binary:"gas_limit"` // Gas限制
	Nonce     uint64    `binary:"nonce"`     // 交易序号
	Data      []byte    `binary:"data"`      // 交易数据
	Signature Signature `binary:"signature"` // 交易签名
}

// Account 账户结构
type Account struct {
	Address Address `binary:"address"` // 账户地址
	Balance uint64  `binary:"balance"` // 账户余额
	Nonce   uint64  `binary:"nonce"`   // 交易计数器
	Staked  uint64  `binary:"staked"`  // 质押金额
}

// Validator 验证者结构
type Validator struct {
	Address      Address `binary:"address"`       // 验证者地址
	PublicKey    []byte  `binary:"public_key"`    // 公钥
	StakeAmount  uint64  `binary:"stake_amount"`  // 自质押金额
	DelegatedAmt uint64  `binary:"delegated_amt"` // 委托金额
	VotePower    uint64  `binary:"vote_power"`    // 投票权重
	IsActive     bool    `binary:"is_active"`     // 是否活跃
	SlashCount   uint64  `binary:"slash_count"`   // 惩罚次数
}

// ValidatorSet 验证者集合
type ValidatorSet struct {
	Validators []Validator `binary:"validators"` // 验证者列表
	Round      uint64      `binary:"round"`      // 当前轮次
}

// GenesisBlock 创世区块配置
type GenesisBlock struct {
	Timestamp    int64             `binary:"timestamp"`     // 创世时间戳
	InitAccounts []Account         `binary:"init_accounts"` // 初始账户
	Validators   []Validator       `binary:"validators"`    // 初始验证者
	ChainID      string            `binary:"chain_id"`      // 链ID
	Config       map[string]string `binary:"config"`        // 配置参数
}

// 工具方法

// Hash 计算哈希值
func (h Hash) String() string {
	return hex.EncodeToString(h[:])
}

// IsEmpty 检查哈希是否为空
func (h Hash) IsEmpty() bool {
	return h == Hash{}
}

// String 地址字符串表示
func (a Address) String() string {
	return hex.EncodeToString(a[:])
}

// IsEmpty 检查地址是否为空
func (a Address) IsEmpty() bool {
	return a == Address{}
}

// String 签名字符串表示
func (s Signature) String() string {
	return hex.EncodeToString(s[:])
}

// IsEmpty 检查签名是否为空
func (s Signature) IsEmpty() bool {
	return s == Signature{}
}

// Hash 计算区块头哈希
func (h *BlockHeader) Hash() Hash {
	// 简化的哈希计算实现
	var data []byte
	// 使用binary.LittleEndian.PutUint64转换数值类型
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, h.Number)
	data = append(data, buf...)
	binary.LittleEndian.PutUint64(buf, uint64(h.Timestamp))
	data = append(data, buf...)
	data = append(data, h.PrevHash[:]...)
	data = append(data, h.TxRoot[:]...)
	data = append(data, h.StateRoot[:]...)
	data = append(data, h.Validator[:]...)
	return sha256.Sum256(data)
}

// Hash 计算区块哈希
func (b *Block) Hash() Hash {
	return b.Header.Hash()
}

// Hash 计算交易哈希
func (tx *Transaction) Hash() Hash {
	// 简化的交易哈希计算
	var data []byte
	buf := make([]byte, 8)

	binary.LittleEndian.PutUint64(buf, tx.ShardID)
	data = append(data, buf...)
	data = append(data, tx.From[:]...)
	data = append(data, tx.To[:]...)
	binary.LittleEndian.PutUint64(buf, tx.Amount)
	data = append(data, buf...)
	binary.LittleEndian.PutUint64(buf, tx.GasPrice)
	data = append(data, buf...)
	binary.LittleEndian.PutUint64(buf, tx.GasLimit)
	data = append(data, buf...)
	binary.LittleEndian.PutUint64(buf, tx.Nonce)
	data = append(data, buf...)
	data = append(data, tx.Data...)

	return sha256.Sum256(data)
}

// IsEmpty 检查区块是否为空块（无交易）
func (b *Block) IsEmpty() bool {
	return len(b.Transactions) == 0
}

// Size 计算区块大小
func (b *Block) Size() int {
	// 简化实现：估算区块大小
	baseSize := 200                    // 区块头基础大小
	txSize := len(b.Transactions) * 32 // 每个交易哈希32字节
	return baseSize + txSize
}

// Verify 验证交易签名
func (tx *Transaction) Verify() bool {
	// 简化实现：仅检查签名是否为空
	return !tx.Signature.IsEmpty()
}

// GetTotalVotePower 获取验证者集合总投票权重
func (vs *ValidatorSet) GetTotalVotePower() uint64 {
	total := uint64(0)
	for _, v := range vs.Validators {
		if v.IsActive {
			total += v.VotePower
		}
	}
	return total
}

// GetValidator 根据地址获取验证者
func (vs *ValidatorSet) GetValidator(addr Address) *Validator {
	for i := range vs.Validators {
		if vs.Validators[i].Address == addr {
			return &vs.Validators[i]
		}
	}
	return nil
}

// GetCurrentValidator 获取当前轮次的验证者
func (vs *ValidatorSet) GetCurrentValidator(blockHeight uint64) *Validator {
	if len(vs.Validators) == 0 {
		return nil
	}

	// 计算当前应该出块的验证者索引
	index := blockHeight % uint64(len(vs.Validators))
	return &vs.Validators[index]
}

// CalculateBlockTime 计算区块的严格时间戳
func CalculateBlockTime(genesisTime int64, blockNumber uint64) int64 {
	return genesisTime + int64(blockNumber)*int64(BlockTime.Seconds())
}

// ValidateBlockTime 验证区块时间戳是否严格符合要求
func ValidateBlockTime(genesisTime int64, blockNumber uint64, timestamp int64) bool {
	expectedTime := CalculateBlockTime(genesisTime, blockNumber)
	return timestamp == expectedTime
}

// EmptyTxRoot 计算空交易列表的Merkle根
func EmptyTxRoot() Hash {
	return sha256.Sum256([]byte{})
}

// CalculateTxRoot 计算交易Merkle根
func CalculateTxRoot(txHashes []Hash) Hash {
	if len(txHashes) == 0 {
		return EmptyTxRoot()
	}

	// 简化的Merkle根计算（实际实现中需要完整的Merkle树）
	var data []byte
	for _, hash := range txHashes {
		data = append(data, hash[:]...)
	}
	return sha256.Sum256(data)
}

// 测试辅助函数

// RandomHash 生成随机哈希（用于测试）
func RandomHash() Hash {
	var hash Hash
	rand.Read(hash[:])
	return hash
}

// RandomAddress 生成随机地址（用于测试）
func RandomAddress() Address {
	var addr Address
	rand.Read(addr[:])
	return addr
}

// RandomSignature 生成随机签名（用于测试）
func RandomSignature() Signature {
	var sig Signature
	rand.Read(sig[:])
	return sig
}
