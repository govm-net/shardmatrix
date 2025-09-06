# ShardMatrix 架构概览

## 项目简介

ShardMatrix 是一个高性能的分片区块链平台，专注于可扩展性和效率。采用简化的 DPoS 共识机制，支持稳定的交易处理。

**注意**: 项目采用迭代开发策略，第一阶段专注于实现基础区块链功能和分片机制，后续阶段将逐步添加高级特性。

## 核心特性

### 性能指标
- **区块间隔**: 2秒（固定）
- **区块大小**: 2MB（可调整）
- **确认时间**: 即时确认（简化版）
- **验证节点**: 21个验证节点
- **分片数量**: 可动态扩展
- **性能目标**: 第一阶段以稳定性为主，后续优化性能

### 技术栈
- **开发语言**: Go 1.21+
- **数据库**: LevelDB
- **网络库**: libp2p-go
- **序列化**: binary
- **共识算法**: DPoS（委托权益证明）
- **加密算法**: Ed25519 + SHA256

## 系统架构

``mermaid
graph TB
    subgraph "应用层"
        A[客户端应用]
        B[钱包应用]
        C[RPC API接口]
    end
    
    subgraph "网络层"
        H[P2P网络]
        I[节点发现]
        J[消息传播]
    end
    
    subgraph "共识层"
        L[DPoS共识]
        M[验证者管理]
        N[区块生产]
    end
    
    subgraph "核心层"
        P[区块管理]
        Q[交易处理]
        R[状态管理]
    end
    
    subgraph "存储层"
        T[LevelDB]
        V[索引存储]
    end
    
    subgraph "分片层"
        S1[分片1]
        S2[分片2]
        S3[分片3]
        S4[...更多分片]
    end
    
    A --> C
    B --> C
    C --> P
    H --> I
    I --> J
    J --> L
    L --> M
    M --> N
    N --> P
    P --> Q
    Q --> R
    R --> T
    T --> V
    S1 --> H
    S2 --> H
    S3 --> H
    S4 --> H
```

## 核心组件

### 1. 区块结构
```go
type BlockHeader struct {
    Number         uint64      // 区块序号
    Timestamp      int64       // 时间戳
    PrevHash       Hash        // 前一个区块哈希
    TxRoot         Hash        // 交易Merkle根
    StateRoot      Hash        // 状态Merkle根
    Validator      Address     // 验证者地址
    Signature      []byte      // 验证者签名
    ShardID        uint64      // 当前分片ID
    AdjacentHashes [3]Hash     // 相邻分片的区块头哈希数组[父分片, 左子分片, 右子分片]
}

type Block struct {
    Header        *BlockHeader `json:"header"`
    Transactions  []Hash       `json:"transactions"` // 交易哈希列表
}
```

### 2. 分片设计
ShardMatrix 采用创新的分片架构，其中：
- 每个分片都是一条独立的区块链
- 每个分片维护自己的状态和交易
- 每个分片只保存相邻分片的区块头哈希（长度为3的数组）
- 相邻分片的定义：
  - 父分片：`shardID = self.ShardID / 2`
  - 左子分片：`shardID = self.ShardID * 2`
  - 右子分片：`shardID = self.ShardID * 2 + 1`

分片树结构示例：
``mermaid
graph TD
    Shard1["Shard 1<br/>(ID: 1)"] --> Shard2["Shard 2<br/>(ID: 2)"]
    Shard1 --> Shard3["Shard 3<br/>(ID: 3)"]
    Shard2 --> Shard4["Shard 4<br/>(ID: 4)"]
    Shard2 --> Shard5["Shard 5<br/>(ID: 5)"]
    Shard3 --> Shard6["Shard 6<br/>(ID: 6)"]
    Shard3 --> Shard7["Shard 7<br/>(ID: 7)"]
    
    classDef shardStyle fill:#e1f5fe,stroke:#01579b,stroke-width:2px;
    class Shard1,Shard2,Shard3,Shard4,Shard5,Shard6,Shard7 shardStyle;
```

相邻分片关系示例：
- 分片1的相邻分片：无父分片，左子分片2，右子分片3
- 分片2的相邻分片：父分片1，左子分片4，右子分片5
- 分片3的相邻分片：父分片1，左子分片6，右子分片7
- 分片4的相邻分片：父分片2，无左子分片，无右子分片

区块头中的相邻哈希数组结构：
```go
type AdjacentHashes [3]Hash  // [父分片哈希, 左子分片哈希, 右子分片哈希]
```

当某个相邻分片不存在时，对应的哈希值为空哈希（全0）。

### 3. 交易模型
```go
type Transaction struct {
    From      Address   // 发送方地址
    To        Address   // 接收方地址
    Amount    uint64    // 转账金额
    GasPrice  uint64    // Gas价格
    GasLimit  uint64    // Gas限制
    Nonce     uint64    // 防止重放攻击
    Data      []byte    // 交易数据
    Signature []byte    // 签名
    ShardID   uint64    // 分片ID（可选，用于跨分片交易）
}
```

### 4. 委托权益证明共识机制（DPoS）
- **验证节点数量**: 每个分片21个验证节点
- **出块机制**: 基于投票权重的验证者选择 + 轮流出块
- **确认机制**: 即时确认（简化版）
- **惩罚机制**: 基础惩罚机制
- **投票管理**: 动态委托、解委托

## 数据流

``mermaid
sequenceDiagram
    participant Client as 客户端
    participant Node as 节点
    participant Consensus as 共识层
    participant Storage as 存储层
    participant Shard1 as 分片1
    participant Shard2 as 分片2
    
    Client->>Node: 提交交易/合约调用
    Node->>Node: 验证交易格式
    Node->>Node: 检查Nonce和余额/Energy
    Node->>Node: 确定交易/合约所属分片
    Node->>Shard1: 加入交易池/合约执行队列
    
    Note over Consensus: 每2秒生成新区块
    Consensus->>Consensus: 选择验证者
    Consensus->>Shard1: 创建新区块
    Shard1->>Shard1: 收集相邻分片区块头哈希
    Shard1->>Consensus: 广播区块
    Consensus->>Storage: 存储区块
    Storage->>Shard1: 更新状态
    Shard1->>Client: 返回确认结果/合约执行结果
    
    Note over Shard1,Shard2: 分片间同步区块头哈希信息
    Shard1->>Shard2: 同步区块头哈希
    Shard2->>Shard1: 同步区块头哈希
```

## 网络拓扑

``mermaid
graph LR
    subgraph "分片1验证节点集群"
        V1[验证节点1]
        V2[验证节点2]
        V3[验证节点3]
        V21[...验证节点21]
    end
    
    subgraph "分片2验证节点集群"
        V4[验证节点1]
        V5[验证节点2]
        V6[验证节点3]
        V22[...验证节点21]
    end
    
    subgraph "全节点网络"
        N1[全节点1]
        N2[全节点2]
        N3[全节点3]
        N4[全节点4]
    end
    
    V1 --- V2
    V2 --- V3
    V3 --- V21
    V21 --- V1
    
    V4 --- V5
    V5 --- V6
    V6 --- V22
    V22 --- V4
    
    V1 --- N1
    V2 --- N2
    V4 --- N3
    V5 --- N4
    
    N1 --- N2
    N2 --- N3
    N3 --- N4
    N4 --- N1
```

## 存储设计

### 存储结构
```
blocks/          # 区块数据
├── shard_1/     # 分片1区块
│   ├── {height} # 按高度存储
│   └── {hash}   # 按哈希存储
├── shard_2/     # 分片2区块
│   ├── {height} # 按高度存储
│   └── {hash}   # 按哈希存储
└── shard_n/     # 分片n区块
    ├── {height} # 按高度存储
    └── {hash}   # 按哈希存储

transactions/    # 交易数据
├── shard_1/     # 分片1交易
│   ├── {txid}   # 按交易ID存储
│   └── {block}  # 按区块存储
├── shard_2/     # 分片2交易
│   ├── {txid}   # 按交易ID存储
│   └── {block}  # 按区块存储
└── shard_n/     # 分片n交易
    ├── {txid}   # 按交易ID存储
    └── {block}  # 按区块存储

accounts/        # 账户状态
├── shard_1/     # 分片1账户
│   ├── {address}# 按地址存储
│   └── balance_index # 余额索引
├── shard_2/     # 分片2账户
│   ├── {address}# 按地址存储
│   └── balance_index # 余额索引
└── shard_n/     # 分片n账户
    ├── {address}# 按地址存储
    └── balance_index # 余额索引

validators/      # 验证者信息
├── shard_1/     # 分片1验证者
│   ├── validators# 验证者列表
│   ├── stakes   # 权益信息
│   └── delegators# 委托人信息
├── shard_2/     # 分片2验证者
│   ├── validators# 验证者列表
│   ├── stakes   # 权益信息
│   └── delegators# 委托人信息
└── shard_n/     # 分片n验证者
    ├── validators# 验证者列表
    ├── stakes   # 权益信息
    └── delegators# 委托人信息

indexes/         # 索引数据
├── shard_1/     # 分片1索引
│   ├── tx_index # 交易索引
│   └── block_index# 区块索引
├── shard_2/     # 分片2索引
│   ├── tx_index # 交易索引
│   └── block_index# 区块索引
└── shard_n/     # 分片n索引
    ├── tx_index # 交易索引
    └── block_index# 区块索引
```

## 创世区块设计

### 设计原则
ShardMatrix 的创世区块设计遵循以下原则：
1. **第一个分片优先**: ShardID=1 作为网络的第一个分片，通过配置文件保证网络的创世区块生成
2. **子分片动态创建**: 子分片通过在第一分片的特殊交易创建
3. **公示期机制**: 新分片创建后需要经过一段时间的公示期
4. **参数确认**: 公示期过后，参数确定，确认指定分片的创世区块所需的所有信息
5. **无需签名**: 所有的创世区块都不需要签名
6. **时间确定机制**:
   - 第一个分片基于配置确定创世区块时间
   - 其他分片基于创建分片的交易事件确定创世区块时间

### 实现机制
1. **第一个分片创世区块**:
   - ShardID=1 作为网络的第一个分片
   - 通过配置文件定义初始验证者、初始账户等信息
   - 创世区块时间由配置文件确定
   - 确保网络启动时能够生成第一个创世区块

2. **子分片创建流程**:
   - 通过在 ShardID=1 分片提交特殊交易来创建新分片
   - 交易中包含新分片的配置信息（如初始验证者列表等）
   - 交易被确认后，进入公示期

3. **公示期机制**:
   - 新分片创建后需要经过一段时间的公示期
   - 在公示期内，网络中的节点可以验证新分片的配置
   - 公示期结束后，新分片的创世区块正式生效
   - 创世区块时间基于创建分片的交易事件确定

4. **创世区块信息确认**:
   - 公示期过后，所有相关参数被确认
   - 系统根据确认的参数生成指定分片的创世区块
   - 创世区块包含初始状态、验证者列表等必要信息

## 分片间交互机制

### 跨分片交易处理
ShardMatrix 采用创新的跨分片交易处理机制：
1. **交易标识**: 跨分片交易通过特定的预编译地址标识（To字段为指定的预编译地址）
2. **交易记录**: 源分片处理跨分片交易时，将交易记录存储到存储层，标记目标分片ID和交易详情
3. **交易查询**: 目标分片定期检查存储层中的跨分片交易记录
4. **交易执行**: 目标分片获取并执行相应的跨分片交易
5. **失败处理**: 跨分片交易失败时，只记录失败，不做任何回滚

### 分片间同步机制
分片间通过以下机制实现数据同步：
1. **区块头哈希同步**: 每个分片定期同步相邻分片的区块头哈希信息
2. **跨分片交易同步**: 
   - 每个分片都会存储跨分片交易，标记目标分片和交易详情
   - 目标分片基于区块头里包含的相邻分片的区块hash，获取对应区块里的跨分片交易记录
   - 目标分片处理对应的交易
3. **状态一致性验证**: 通过区块头哈希信息实现分片间的轻量级状态验证
4. **同步优化**: 保存相邻分片的区块hash，要求间隔30个区块（即1分钟），尽量保证对应分片已经最终确认

## 安全机制

### 恶意行为检测与惩罚
ShardMatrix 设计了完善的恶意行为检测与惩罚机制：
1. **重复区块检测**: 检测相同验证者多次提交相同高度的区块（区块内容不同）
2. **惩罚机制**: 对恶意行为的验证者实施惩罚，包括但不限于：
   - 扣除部分质押
   - 暂时或永久禁止出块权限
   - 公开记录恶意行为
3. **网络保护**: 防止恶意区块对网络造成影响

### 交易安全
- 数字签名验证（Ed25519）
- Nonce防重放攻击
- Gas机制防止DoS攻击
- 交易池大小限制

### 共识安全
- DPoS 验证者机制
- 基础惩罚机制
- 简单的投票权重验证

### 网络安全
- P2P加密通信
- 节点身份验证
- 网络分区检测
- 消息传播优化

### 分片安全
- 分片间通过区块头哈希验证
- 跨分片交易的原子性保证
- 分片状态一致性验证
- 分片故障隔离和恢复

## 分片间交互

### 区块头哈希同步
- 每个分片定期同步相邻分片的区块头哈希信息
- 区块头哈希信息包含在本分片的区块中
- 通过区块头哈希信息实现分片间的轻量级验证
- 保存相邻分片的区块hash，要求间隔n个区块，尽量保证对应分片已经最终确认

### 跨分片交易
- 跨分片交易需要在源分片和目标分片都进行验证
- 通过区块头哈希信息验证目标分片的状态
- 实现原子性的跨分片交易处理
- 要求在源分片验证通过后才能记录下来
- 目标分片的执行不保证成功，特别是复杂的交易

## API接口设计

### 跨分片数据查询
ShardMatrix 提供了专门的API接口用于跨分片数据查询：
1. **查询机制**: 通过API，使用对应的区块hash，查询跨链交易
2. **接口设计**: 
   - 跨分片交易查询接口
   - 分片间状态验证接口
   - 区块头哈希查询接口
3. **数据隔离**: 简单地将不同的分片都单独存储，确保数据隔离性
