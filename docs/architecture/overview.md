# ShardMatrix 架构概览

## 项目简介

ShardMatrix 是一个高性能的单链区块链平台，专注于可扩展性和效率。采用简化的 DPoS 共识机制，支持稳定的交易处理。

**注意**: 项目采用迭代开发策略，第一阶段专注于实现基础区块链功能，后续阶段将逐步添加高级特性。

## 核心特性

### 性能指标
- **区块间隔**: 固定3秒
- **区块大小**: 2MB（可调整）
- **确认时间**: 即时确认（简化版）
- **验证节点**: 21个验证节点
- **性能目标**: 第一阶段以稳定性为主，后续优化性能


### 技术栈
- **开发语言**: Go 1.21+
- **数据库**: LevelDB
- **网络库**: libp2p-go
- **序列化**: gob
- **共识算法**: DPoS（委托权益证明）
- **加密算法**: Ed25519 + SHA256

## 系统架构

```mermaid
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
```

## 核心组件

### 1. 区块结构
```go
type BlockHeader struct {
    Number        uint64    // 区块序号
    Timestamp     int64     // 时间戳
    PrevHash      Hash      // 前一个区块哈希
    TxRoot        Hash      // 交易Merkle根
    StateRoot     Hash      // 状态Merkle根
    Validator     Address   // 验证者地址
    Signature     []byte    // 验证者签名
}

type Block struct {
    Header        *BlockHeader `json:"header"`
    Transactions  []Hash       `json:"transactions"` // 交易哈希列表
}
```

### 2. 交易模型
```go
type Transaction struct {
    From      Address   // 发送方地址
    To        Address   // 接收方地址
    Amount    uint64    // 转账金额
    Fee       uint64    // 手续费
    Nonce     uint64    // 防止重放攻击
    Data      []byte    // 交易数据
    Signature []byte    // 签名
}
```

### 3. 委托权益证明共识机制（DPoS）
- **验证节点数量**: 21个验证节点
- **出块机制**: 基于投票权重的验证者选择 + 轮流出块
- **确认机制**: 即时确认（简化版）
- **惩罚机制**: 基础惩罚机制
- **投票管理**: 动态委托、解委托

## 数据流

```mermaid
sequenceDiagram
    participant Client as 客户端
    participant Node as 节点
    participant Consensus as 共识层
    participant Storage as 存储层
    
    Client->>Node: 提交交易
    Node->>Node: 验证交易格式
    Node->>Node: 检查Nonce和余额
    Node->>Node: 加入交易池
    
    Note over Consensus: 每3秒生成新区块
    Consensus->>Consensus: 选择验证者
    Consensus->>Node: 创建新区块
    Node->>Consensus: 广播区块
    Consensus->>Storage: 存储区块
    Storage->>Node: 更新状态
    Node->>Client: 返回确认结果
```

## 网络拓扑

```mermaid
graph LR
    subgraph "验证节点集群"
        V1[验证节点1]
        V2[验证节点2]
        V3[验证节点3]
        V21[...验证节点21]
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
    
    V1 --- N1
    V2 --- N2
    V3 --- N3
    
    N1 --- N2
    N2 --- N3
    N3 --- N4
    N4 --- N1
```

## 存储设计

### 存储结构
```
blocks/          # 区块数据
├── {height}     # 按高度存储
└── {hash}       # 按哈希存储

transactions/    # 交易数据
├── {txid}       # 按交易ID存储
└── {block}      # 按区块存储

accounts/        # 账户状态
├── {address}    # 按地址存储
└── balance_index # 余额索引

validators/      # 验证者信息
├── validators   # 验证者列表
├── stakes       # 权益信息
└── delegators   # 委托人信息

indexes/         # 索引数据
├── tx_index     # 交易索引
└── block_index  # 区块索引
```

## 安全机制

参考 [design_summary.md](./design_summary.md) 中的详细描述，包括交易安全、共识安全和网络安全。

## 性能优化

### 1. 并行处理
- 交易验证并行化
- 状态更新并行化
- 网络消息并行处理

### 2. 缓存策略
- 热点账户缓存
- 区块头缓存
- 验证者信息缓存

### 3. 存储优化
- LevelDB批量写入
- 数据压缩
- 索引优化

### 4. 网络优化
- 消息压缩
- 连接复用
- 带宽控制

## 监控指标

### 1. 性能指标
- 区块生成时间
- 交易确认延迟
- 网络延迟

### 2. 系统指标
- CPU使用率
- 内存使用量
- 磁盘I/O
- 网络带宽

### 3. 业务指标
- 活跃账户数
- 交易成功率
- 验证者参与率
- 权益分布

### 4. 安全指标
- 双重签名检测
- 恶意行为检测
- 网络攻击检测

## 部署架构

```mermaid
graph TB
    subgraph "生产环境"
        LB[负载均衡器]
        subgraph "验证节点集群"
            VN1[验证节点1]
            VN2[验证节点2]
            VN3[验证节点3]
        end
        subgraph "全节点集群"
            FN1[全节点1]
            FN2[全节点2]
        end
        subgraph "监控系统"
            P[Prometheus]
            G[Grafana]
        end
    end
    
    LB --> VN1
    LB --> VN2
    LB --> VN3
    VN1 --> FN1
    VN2 --> FN2
    FN1 --> P
    FN2 --> P
    P --> G
```

## 故障恢复

### 1. 节点故障
- 自动故障检测
- 验证者自动替换
- 状态快速同步

### 2. 网络分区
- 分区检测机制
- 自动重连策略
- 消息重传机制

### 3. 数据损坏
- 数据完整性检查
- 自动备份恢复
- 数据修复工具
