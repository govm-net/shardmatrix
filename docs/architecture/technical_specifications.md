# ShardMatrix 技术规格文档

## 项目定位

ShardMatrix 是一个高性能的分片区块链平台，专注于可扩展性和效率。项目采用迭代开发策略，按阶段实现功能：
- 第一阶段：实现单分片区块链基础功能
- 第二阶段：集成第三方智能合约模块
- 第三阶段：实现分片机制和跨分片交易

## 核心技术参数

### 网络参数
- **区块间隔**: 2秒（固定）
- **区块大小**: 2MB（初期），后续可调整
- **验证者数量**: 每个分片21个
- **网络类型**: 分片架构（第三阶段）

### 共识机制
- **算法**: DPoS（委托权益证明）
- **验证者选择**: 基于质押权重排序
- **出块方式**: 验证者轮流出块
- **确认机制**: 即时确认（简化版）
- **最小质押**: 10,000 代币

### 性能目标
- **第一阶段**: 稳定运行，功能完整（单分片）
- **第二阶段**: 支持第三方智能合约模块集成
- **第三阶段**: 支持多分片并行处理，TPS达到10000+
- **交易处理**: 支持基础转账交易和智能合约调用

## 技术栈

### 开发环境
- **开发语言**: Go 1.21+
- **构建工具**: Make
- **代码规范**: Go 标准规范 + golangci-lint

### 核心依赖
```yaml
dependencies:
  storage: LevelDB
  network: libp2p-go
  serialization: binary
  logging: logrus v1.9.3
  cli: cobra v1.8.0
  config: viper v1.18.2
  crypto: Go 标准库 + 第三方扩展
  vm: 第三方虚拟机（第二阶段）
```

### 系统架构
- **架构模式**: 分层架构 + 模块化设计
- **配置管理**: YAML 配置文件
- **数据存储**: LevelDB 存储（第一阶段单分片，第三阶段分片存储）
- **网络通信**: P2P 点对点通信

## 系统组件

### 第一阶段核心模块
1. **节点管理** (pkg/node)
   - 节点生命周期管理
   - 服务协调

2. **共识层** (pkg/consensus)
   - DPoS 共识实现
   - 验证者管理
   - 区块生产

3. **网络层** (pkg/network)
   - P2P 通信
   - 节点发现
   - 消息传播

4. **存储层** (pkg/storage)
   - 区块存储
   - 交易存储
   - 状态管理

5. **交易池** (pkg/txpool)
   - 交易验证
   - 交易排序
   - 内存管理

6. **密码学** (pkg/crypto)
   - 数字签名
   - 哈希算法
   - 密钥管理

7. **API接口** (pkg/api)
   - RESTful API
   - 查询接口
   - 状态接口

### 第二阶段新增模块
8. **虚拟机接口** (pkg/vm)
   - 第三方虚拟机集成接口
   - 合约部署和管理接口
   - Gas计费接口

### 第三阶段新增模块
9. **分片管理** (pkg/shard)
   - 分片路由
   - 分片间通信
   - 分片状态同步

## 数据模型

### 区块结构
```go
type Block struct {
    Header       *BlockHeader
    Transactions []Hash
}

type BlockHeader struct {
    Number         uint64
    Timestamp      int64
    PrevHash       Hash
    TxRoot         Hash
    StateRoot      Hash
    Validator      Address
    Signature      []byte
    ShardID        uint64        // 第一阶段为固定值，第三阶段为实际分片ID
    AdjacentHashes [3]Hash       // 第一阶段为空，第三阶段为相邻分片区块头哈希
}
```

### 交易结构
```go
type Transaction struct {
    From      Address
    To        Address
    Amount    uint64
    GasPrice  uint64
    GasLimit  uint64
    Nonce     uint64
    Data      []byte
    Signature []byte
    ShardID   uint64            // 第一阶段为固定值，第二阶段用于合约交易，第三阶段用于跨分片交易
}
```

### 智能合约结构（第二阶段）
```go
// 智能合约结构将由第三方虚拟机定义
// 此处仅定义接口用于与第三方虚拟机交互
type ContractInterface interface {
    Deploy(code []byte, args []byte) (Address, error)
    Execute(contract Address, method string, args []byte) ([]byte, error)
    GetState(contract Address, key string) ([]byte, error)
    SetState(contract Address, key string, value []byte) error
}
```

### 跨分片交易结构（第三阶段）
```go
type CrossShardTransaction struct {
    TxHash        Hash
    Transaction   *Transaction
    SourceShardID uint64
    TargetShardID uint64
    Status        string        // pending, processed, failed
    CreatedAt     int64
    ProcessedAt   int64
}
```

## 安全模型

### 基础安全
- 数字签名验证
- Nonce 防重放攻击
- 基础交易验证

### 共识安全
- DPoS 验证者机制
- 基础惩罚机制
- 简单的投票权重验证

### 智能合约安全（第二阶段）
- 第三方虚拟机提供的安全机制
- Gas限制防止无限循环
- 合约调用权限控制由第三方虚拟机实现

### 分片安全（第三阶段）
- 分片间通过区块头哈希验证
- 跨分片交易的原子性保证
- 分片状态一致性验证

## 部署要求

### 最低硬件要求
- **CPU**: 2核心
- **内存**: 4GB
- **存储**: 50GB SSD
- **网络**: 10Mbps 带宽

### 推荐硬件配置
- **CPU**: 4核心+
- **内存**: 8GB+
- **存储**: 100GB+ SSD
- **网络**: 50Mbps+ 带宽

## 开发约束

### 第一阶段限制
1. **功能范围**: 实现单分片区块链基础功能
2. **性能目标**: 以稳定性为主，暂不追求极致性能
3. **复杂特性**: 实现基础功能，暂不实现高级特性

### 第二阶段限制
1. **功能范围**: 在单分片基础上集成第三方智能合约模块
2. **虚拟机**: 集成轻量级第三方虚拟机，支持基本合约执行
3. **安全性**: 确保合约执行的安全性由第三方虚拟机保障

### 第三阶段限制
1. **功能范围**: 实现分片机制和跨分片交易
2. **复杂特性**: 实现分片间同步和跨分片交易处理
3. **性能目标**: 实现多分片并行处理，提升TPS

### 设计原则
1. **简单优先**: 优先选择简单可靠的实现方案
2. **模块化**: 保持良好的模块边界，便于后续扩展
3. **可测试**: 所有核心功能必须有完整测试
4. **文档齐全**: 代码和设计文档同步更新
5. **阶段清晰**: 每个阶段目标明确，功能边界清晰

## 质量标准

### 代码质量
- 测试覆盖率 > 80%
- golangci-lint 检查通过
- 所有公开接口有文档注释

### 性能基准
- 节点启动时间 < 30秒
- 基础交易处理延迟 < 1秒
- 系统稳定运行 24小时+
- 第三阶段TPS > 10000（多分片）

### 兼容性
- Go 1.21+ 兼容
- 主流操作系统支持（Linux、macOS、Windows）
- Docker 容器化支持

## 版本控制

### API 版本
- 当前版本: v1
- 版本策略: 语义化版本控制

### 数据格式版本
- 区块格式版本: v1
- 交易格式版本: v1
- 配置格式版本: v1
- 合约接口版本: v1（第二阶段）
- 跨分片交易格式版本: v1（第三阶段）

---

**重要说明**: 本文档定义了 ShardMatrix 各阶段的技术规格，所有架构文档和实现代码都应遵循此规格。任何变更都需要更新此文档并保持全局一致性.