# ShardMatrix 数据结构设计

## 概述

本文档详细描述了 ShardMatrix 分片区块链平台的核心数据结构设计，包括区块、交易、存储等关键组件。

按阶段实现数据结构设计：
- 第一阶段：实现单分片区块链基础数据结构
- 第二阶段：扩展数据结构以支持智能合约
- 第三阶段：完善数据结构以支持分片机制

## 基础类型定义

### Hash 哈希类型
```go
type Hash [32]byte // 32字节哈希值

// Hash 方法
func (h Hash) String() string {
    return hex.EncodeToString(h[:])
}

func (h Hash) Bytes() []byte {
    return h[:]
}

func (h Hash) IsZero() bool {
    return h == Hash{}
}

// 创建Hash
func NewHash(data []byte) Hash {
    var h Hash
    copy(h[:], data)
    return h
}

// 从字符串创建Hash
func HashFromString(s string) (Hash, error) {
    data, err := hex.DecodeString(s)
    if err != nil {
        return Hash{}, err
    }
    if len(data) != 32 {
        return Hash{}, errors.New("invalid hash length")
    }
    return NewHash(data), nil
}
```

### Address 地址类型
```go
type Address [20]byte // 20字节地址

// Address 方法
func (a Address) String() string {
    return hex.EncodeToString(a[:])
}

func (a Address) Bytes() []byte {
    return a[:]
}

func (a Address) IsZero() bool {
    return a == Address{}
}

// 创建Address
func NewAddress(data []byte) Address {
    var addr Address
    copy(addr[:], data)
    return addr
}

// 从公钥创建Address
func AddressFromPublicKey(pubKey []byte) Address {
    hash := sha256.Sum256(pubKey)
    var addr Address
    copy(addr[:], hash[:20])
    return addr
}

// 从字符串创建Address
func AddressFromString(s string) (Address, error) {
    data, err := hex.DecodeString(s)
    if err != nil {
        return Address{}, err
    }
    if len(data) != 20 {
        return Address{}, errors.New("invalid address length")
    }
    return NewAddress(data), nil
}
```

## 创世区块配置

### GenesisConfig 创世配置
```go
type GenesisConfig struct {
    ShardID           uint64      `json:"shard_id"`           // 分片ID（第一阶段为固定值1）
    Timestamp         int64       `json:"timestamp"`          // 创世时间戳
    InitialValidators []Validator `json:"initial_validators"` // 初始验证者列表
    InitialAccounts   []Account   `json:"initial_accounts"`   // 初始账户
    BlockInterval     uint64      `json:"block_interval"`     // 区块间隔（秒）
    MaxBlockSize      uint64      `json:"max_block_size"`     // 最大区块大小（字节）
}

// 创世区块
type GenesisBlock struct {
    Header       *BlockHeader `json:"header"`
    Transactions []Hash       `json:"transactions"` // 创世交易哈希列表
}
```

## 区块结构

### BlockHeader 区块头

```go
type BlockHeader struct {
    Number         uint64      // 区块序号
    Timestamp      int64       // 时间戳
    PrevHash       Hash        // 前一个区块哈希
    TxRoot         Hash        // 交易Merkle根
    StateRoot      Hash        // 状态Merkle根
    Validator      Address     // 验证者地址
    Signature      []byte      // 验证者签名（创世区块为空）
    ShardID        uint64      // 当前分片ID（第一阶段为固定值，第三阶段为实际分片ID）
    AdjacentHashes [3]Hash     // 相邻分片的区块头哈希数组[父分片, 左子分片, 右子分片]（第一阶段为空）
}
```

**字段说明**:
- `Number`: 区块高度，从0开始递增
- `Timestamp`: 区块创建时间戳（Unix时间戳）
- `PrevHash`: 前一个区块的哈希值，形成链式结构
- `TxRoot`: 交易Merkle根，用于验证交易完整性
- `StateRoot`: 状态Merkle根，用于验证状态完整性
- `Validator`: 创建该区块的验证者地址
- `Signature`: 验证者对区块的签名（创世区块为空）
- `ShardID`: 当前分片的ID（第一阶段为固定值，第三阶段为实际分片ID）
- `AdjacentHashes`: 相邻分片的区块头哈希数组，用于分片间链接（第一阶段为空，第三阶段使用）

### Block 区块

```go
type Block struct {
    Header       *BlockHeader `json:"header"`
    Transactions []Hash       `json:"transactions"` // 交易哈希列表
}
```

**设计原则**:
- 区块只包含交易哈希，不包含完整交易数据
- 这样可以减少区块大小，提高网络传输效率
- 完整交易数据存储在交易池或数据库中
- 符合区块链的设计模式

## 交易结构

### Transaction 交易

```go
type Transaction struct {
    ShardID   uint64  `json:"shard_id"`  // 分片ID（第一阶段为固定值，第二阶段用于合约交易，第三阶段用于跨分片交易）
    From      Address `json:"from"`      // 发送方地址
    To        Address `json:"to"`        // 接收方地址
    Amount    uint64  `json:"amount"`    // 转账金额
    GasPrice  uint64  `json:"gas_price"` // Gas价格
    GasLimit  uint64  `json:"gas_limit"` // Gas限制
    Nonce     uint64  `json:"nonce"`     // 防止重放攻击
    Data      []byte  `json:"data"`      // 交易数据
    Signature []byte  `json:"signature"` // 签名
}
```

**字段说明**:
- `From`: 交易发送方地址
- `To`: 交易接收方地址
- `Amount`: 转账金额（基础货币单位）
- `GasPrice`: Gas价格，用于计算交易费用
- `GasLimit`: Gas限制，交易愿意消耗的最大Gas数量
- `Nonce`: 发送方的交易序号，防止重放攻击
- `Data`: 交易数据（可以是转账金额、合约调用数据等）
- `Signature`: 发送方对交易的数字签名
- `ShardID`: 交易所属的分片ID（第一阶段为固定值，第二阶段用于合约交易，第三阶段用于跨分片交易）

**设计特点**:
- 简化的交易模型，易于理解和实现
- 支持多种交易类型（通过Data字段扩展）
- 包含基本的验证机制
- 第一阶段主要支持基础转账功能
- 第二阶段支持智能合约调用
- 第三阶段支持跨分片交易
- 使用Gas机制提供灵活的费用模型

### ContractTransaction 合约交易（第二阶段）
```go
// 合约交易预编译地址
const (
    ContractPrecompileAddress = "0x0000000000000000000000000000000000000002"
)

// 合约交易结构将由第三方虚拟机定义
// 此处仅定义接口用于与第三方虚拟机交互
type ContractTransactionInterface interface {
    GetContractAddress() Address
    GetMethod() string
    GetArgs() []byte
    GetValue() uint64
    GetGasPrice() uint64
    GetGasLimit() uint64
}
```

### CrossShardTransaction 跨分片交易（第三阶段）

```go
// 跨分片交易预编译地址
const (
    CrossShardPrecompileAddress = "0x0000000000000000000000000000000000000001"
)

type CrossShardTransaction struct {
    TxHash        Hash    `json:"tx_hash"`        // 原始交易哈希
    Transaction   *Transaction `json:"transaction"`   // 原始交易
    SourceShardID uint64  `json:"source_shard_id"` // 源分片ID
    TargetShardID uint64  `json:"target_shard_id"` // 目标分片ID
    Status        string  `json:"status"`         // 处理状态 (pending, processed, failed)
    CreatedAt     int64   `json:"created_at"`     // 创建时间
    ProcessedAt   int64   `json:"processed_at"`   // 处理时间
}

type CrossShardTransactionStatus string

const (
    CrossShardTxPending   CrossShardTransactionStatus = "pending"
    CrossShardTxProcessed CrossShardTransactionStatus = "processed"
    CrossShardTxFailed    CrossShardTransactionStatus = "failed"
)
```

**设计说明**:
- 跨分片交易通过特定的预编译地址标识（To字段为`CrossShardPrecompileAddress`）
- 源分片处理跨分片交易时，将交易记录存储到存储层
- 目标分片定期检查存储层中的跨分片交易记录并处理
- 通过状态字段跟踪跨分片交易的处理进度

### Account 账户结构

```go
type Account struct {
    Address   Address `json:"address"`   // 账户地址
    Balance   uint64  `json:"balance"`   // 账户余额
    Nonce     uint64  `json:"nonce"`     // 交易计数器
    UpdatedAt int64   `json:"updated_at"` // 最后更新时间
    CodeHash  Hash    `json:"code_hash,omitempty"` // 合约代码哈希（第二阶段，由第三方虚拟机管理）
    StorageRoot Hash  `json:"storage_root,omitempty"` // 合约存储根（第二阶段，由第三方虚拟机管理）
}
```

### Contract 合约结构（第二阶段）
```go
// 合约结构将由第三方虚拟机定义和管理
// 此处仅定义接口用于与第三方虚拟机交互
type ContractInterface interface {
    GetAddress() Address
    GetCode() []byte
    GetState(key string) ([]byte, error)
    SetState(key string, value []byte) error
    GetBalance() uint64
    GetCreator() Address
    GetCreatedAt() uint64
}
```

### Validator 验证者结构

```go
type Validator struct {
    Address    Address         `json:"address"`    // 验证者地址
    Stake      uint64          `json:"stake"`      // 质押数量
    Status     ValidatorStatus `json:"status"`     // 验证者状态
    LastBlock  uint64          `json:"last_block"` // 最后出块高度
    TotalVotes uint64          `json:"total_votes"`// 总投票数
}

type ValidatorStatus int
const (
    ValidatorActive ValidatorStatus = iota
    ValidatorInactive
    ValidatorJailed
)

type Delegator struct {
    Address   Address `json:"address"`   // 委托人地址
    Validator Address `json:"validator"` // 委托的验证者地址
    Stake     uint64  `json:"stake"`     // 委托数量
}
```

## 存储设计

### 存储接口定义

```go
// 区块存储接口
type BlockStore interface {
    PutBlock(block *Block) error
    GetBlock(blockHash []byte) (*Block, error)
    GetBlockByHeight(height uint64) (*Block, error)
    HasBlock(blockHash []byte) bool
    GetLatestBlock() (*Block, error)
    PutGenesisBlock(block *GenesisBlock) error
    GetGenesisBlock() (*GenesisBlock, error)
}

// 交易存储接口
type TransactionStore interface {
    PutTransaction(tx *Transaction) error
    GetTransaction(txHash []byte) (*Transaction, error)
    HasTransaction(txHash []byte) bool
    DeleteTransaction(txHash []byte) error
    GetTransactionsByBlock(blockHash []byte) ([]*Transaction, error)
}

// 合约存储接口（第二阶段）
// 注意：合约存储将由第三方虚拟机管理，此处仅定义接口用于与第三方虚拟机交互
type ContractStoreInterface interface {
    PutContract(contract ContractInterface) error
    GetContract(address Address) (ContractInterface, error)
    HasContract(address Address) bool
    DeleteContract(address Address) error
    GetContractCode(address Address) ([]byte, error)
    PutContractState(address Address, key string, value []byte) error
    GetContractState(address Address, key string) ([]byte, error)
}

// 跨分片交易存储接口（第三阶段）
type CrossShardTransactionStore interface {
    PutCrossShardTransaction(tx *CrossShardTransaction) error
    GetCrossShardTransaction(txHash []byte) (*CrossShardTransaction, error)
    GetCrossShardTransactionsByTargetShard(shardID uint64) ([]*CrossShardTransaction, error)
    UpdateCrossShardTransactionStatus(txHash Hash, status CrossShardTransactionStatus) error
    DeleteCrossShardTransaction(txHash []byte) error
}

// 账户状态存储接口
type StateStore interface {
    GetAccount(address Address) (*Account, error)
    UpdateAccount(address Address, account *Account) error
    DeleteAccount(address Address) error
    GetStateRoot() Hash
    Commit() error
    Rollback() error
}

// 验证者存储接口
type ValidatorStore interface {
    GetValidator(address Address) (*Validator, error)
    UpdateValidator(validator *Validator) error
    GetAllValidators() ([]*Validator, error)
    GetActiveValidators() ([]*Validator, error)
    GetDelegators(validator Address) ([]*Delegator, error)
    UpdateDelegator(delegator *Delegator) error
}

// 恶意行为存储接口
type MaliciousBehaviorStore interface {
    PutMaliciousBehavior(behavior *MaliciousBehavior) error
    GetMaliciousBehaviorsByValidator(address Address) ([]*MaliciousBehavior, error)
    GetMaliciousBehaviorsByType(behaviorType string) ([]*MaliciousBehavior, error)
}

// 恶意行为记录
type MaliciousBehavior struct {
    Validator     Address  `json:"validator"`      // 恶意验证者地址
    BehaviorType  string   `json:"behavior_type"`  // 恶意行为类型
    BlockHeight   uint64   `json:"block_height"`   // 区块高度
    Timestamp     int64    `json:"timestamp"`      // 时间戳
    Evidence      []byte   `json:"evidence"`       // 证据数据
    Penalty       uint64   `json:"penalty"`        // 惩罚金额
}

// 惩罚类型
type PenaltyType string

const (
    PenaltyDuplicateBlock    PenaltyType = "duplicate_block"     // 重复区块
    PenaltyInvalidSignature  PenaltyType = "invalid_signature"   // 无效签名
    PenaltyDoubleSign        PenaltyType = "double_sign"         // 双重签名
)
```

### 存储格式

**LevelDB 键值格式**:
```
// 区块存储
Key: "block:{区块哈希}"
Value: 序列化的区块数据

Key: "block_height:{高度}"
Value: 区块哈希

Key: "genesis_block"
Value: 序列化的创世区块数据

// 交易存储
Key: "tx:{交易哈希}"
Value: 序列化的交易数据

// 合约存储（第二阶段）
Key: "contract:{合约地址}"
Value: 序列化的合约数据

Key: "contract_code:{合约地址}"
Value: 合约代码

Key: "contract_state:{合约地址}:{状态键}"
Value: 状态值

// 跨分片交易存储（第三阶段）
Key: "cross_shard_tx:{交易哈希}"
Value: 序列化的跨分片交易数据

Key: "cross_shard_tx_target:{目标分片ID}:{交易哈希}"
Value: 交易哈希（用于按目标分片查询）

// 账户状态
Key: "account:{地址}"
Value: 序列化的账户数据

// 验证者信息
Key: "validator:{地址}"
Value: 序列化的验证者数据

Key: "delegator:{委托人地址}:{验证者地址}"
Value: 序列化的委托信息

// 恶意行为记录
Key: "malicious:{验证者地址}:{时间戳}"
Value: 序列化的恶意行为数据

Key: "malicious_type:{行为类型}:{时间戳}"
Value: 序列化的恶意行为数据
```

## 数据流程

### 第一阶段数据流程（单分片）
```mermaid
graph TD
    A[用户创建交易] --> B[交易验证]
    B --> C[交易池]
    C --> D[区块创建]
    D --> E[交易哈希添加到区块]
    E --> F[计算Merkle根]
    F --> G[区块签名]
    G --> H[区块存储]
    H --> I[交易存储]
    I --> J[状态更新]
    
    T[恶意行为检测] --> U[记录恶意行为]
    U --> V[实施惩罚]
```

### 第二阶段数据流程（智能合约）
```mermaid
graph TD
    A[用户创建交易] --> B[交易验证]
    B --> C[交易池]
    C --> D{是否为合约交易}
    D -->|是| E[合约执行]
    D -->|否| F[普通交易处理]
    E --> G[Gas计费]
    G --> H[状态更新]
    F --> I[普通状态更新]
    H --> J[区块创建]
    I --> J
    J --> K[交易哈希添加到区块]
    K --> L[计算Merkle根]
    L --> M[区块签名]
    M --> N[区块存储]
    N --> O[交易存储]
    O --> P[合约存储]
```

### 第三阶段数据流程（分片）
```mermaid
graph TD
    A[用户创建交易] --> B[交易验证]
    B --> C[交易池]
    C --> D{交易类型}
    D -->|普通交易| E[本地处理]
    D -->|合约交易| F[合约执行]
    D -->|跨分片交易| G[标记跨分片交易]
    E --> H[状态更新]
    F --> I[Gas计费和状态更新]
    G --> J[存储跨分片交易记录]
    J --> K[标记目标分片]
    H --> L[区块创建]
    I --> L
    L --> M[交易哈希添加到区块]
    M --> N[计算Merkle根]
    N --> O[区块签名]
    O --> P[区块存储]
    P --> Q[交易存储]
    Q --> R[合约存储]
    R --> S[跨分片交易存储]
    
    T[目标分片处理] --> U[获取跨分片交易]
    U --> V[执行跨分片交易]
    V --> W{执行结果}
    W -->|成功| X[更新交易状态为已处理]
    W -->|失败| Y[更新交易状态为失败]
    
    Z[相邻分片同步] --> AA[获取区块头哈希]
    AA --> AB[更新相邻哈希数组]
```

## 设计原则
- **简单优先**: 优先选择简单可靠的实现方案
- **类型安全**: 使用强类型和枚举
- **模块化**: 保持良好的模块边界，便于后续扩展
- **可测试**: 所有核心功能必须有完整测试
- **阶段清晰**: 每个阶段的数据结构设计目标明确

## 性能与扩展性
- **基础优化**: 缓存、批量处理
- **阶段扩展**: 每个阶段在前一阶段基础上扩展
- **向后兼容**: 数据结构版本控制
- **分片支持**: 第三阶段支持分片数据隔离