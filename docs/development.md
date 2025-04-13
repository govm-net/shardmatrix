# ShardMatrix 开发指南

## 项目结构

```
shardmatrix/
├── cmd/
│   └── shardmatrix/          # 主程序入口
├── internal/
│   ├── shard/               # 分片管理
│   │   ├── topology.go      # 分片拓扑结构
│   │   ├── router.go        # 分片路由
│   │   └── cross_shard.go   # 跨分片通信
│   ├── consensus/           # 共识机制
│   │   ├── tendermint.go    # Tendermint Core集成
│   │   ├── abci.go         # ABCI应用实现
│   │   └── state.go        # 状态管理
│   ├── contract/           # 智能合约
│   │   ├── engine.go       # 合约执行引擎
│   │   ├── state.go        # 状态管理
│   │   └── storage.go      # 合约存储
│   └── network/            # 网络层
│       ├── p2p.go          # Tendermint P2P网络
│       ├── message.go      # 消息处理
│       └── discovery.go    # 节点发现
├── pkg/
│   ├── crypto/             # 密码学相关
│   ├── common/             # 公共组件
│   └── types/              # 类型定义
├── docs/                   # 文档
├── go.mod
└── go.sum
```

## 开发环境要求

- Go 1.18+
- Git
- Make
- Docker (可选，用于测试环境)
- Tendermint Core (用于共识和网络)

## 开发步骤

1. 环境准备
```bash
# 克隆项目
git clone https://github.com/your-org/shardmatrix.git

# 安装依赖
go mod download

# 安装Tendermint
go install github.com/tendermint/tendermint/cmd/tendermint@latest
```

2. 代码规范
- 遵循Go标准代码规范
- 使用gofmt格式化代码
- 编写单元测试
- 提交前运行测试

3. 开发流程
```bash
# 创建新分支
git checkout -b feature/your-feature

# 开发完成后提交
git add .
git commit -m "feat: add your feature"

# 推送到远程
git push origin feature/your-feature
```

## 主要模块开发指南

### 1. 分片管理模块

```go
// topology.go
type ShardTopology struct {
    RootShard    *Shard
    ChildShards  []*Shard
    ShardMap     map[uint32]*Shard
}

// router.go
type ShardRouter struct {
    Topology     *ShardTopology
    RouteTable   map[string]uint32
}
```

### 2. 共识机制模块

```go
// tendermint.go
type TendermintCore struct {
    Node         *tendermint.Node
    Config       *tendermint.Config
    PrivVal      *tendermint.PrivValidator
}

// abci.go
type ABCIApplication struct {
    State        *State
    ValidatorSet *tendermint.ValidatorSet
}

// state.go
type State struct {
    Height       int64
    AppHash      []byte
    Validators   *tendermint.ValidatorSet
}
```

### 3. 智能合约模块

```go
// engine.go
type ContractEngine struct {
    VM           *VM
    StateDB      *StateDB
    GasLimit     uint64
}

// state.go
type StateDB struct {
    Storage      map[string][]byte
    Accounts     map[string]*Account
}
```

### 4. 网络层模块

```go
// p2p.go
type P2PNetwork struct {
    Node         *tendermint.Node
    Config       *tendermint.Config
    PeerManager  *tendermint.PeerManager
}

// message.go
type Message struct {
    Type         MessageType
    Data         []byte
    From         string
    To           string
}
```

## 测试指南

1. 单元测试
```bash
# 运行所有测试
go test ./...

# 运行特定包的测试
go test ./internal/shard
```

2. 集成测试
```bash
# 运行集成测试
make test-integration
```

3. 性能测试
```bash
# 运行性能测试
make benchmark
```

## 部署指南

1. 本地部署
```bash
# 构建
make build

# 初始化Tendermint
tendermint init

# 运行
./bin/shardmatrix
```

2. Docker部署
```bash
# 构建镜像
make docker-build

# 运行容器
make docker-run
```

## 贡献指南

1. Fork项目
2. 创建特性分支
3. 提交更改
4. 推送到分支
5. 创建Pull Request

## 常见问题

1. 如何添加新的分片？
2. 如何配置Tendermint参数？
3. 如何部署智能合约？
4. 如何处理跨分片交易？

## 联系方式

- 项目主页：https://github.com/your-org/shardmatrix
- 问题反馈：https://github.com/your-org/shardmatrix/issues
- 邮件列表：shardmatrix@your-org.com 