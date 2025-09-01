# ShardMatrix

ShardMatrix 是一个高性能的分片区块链平台，专注于可扩展性和效率。采用 PoS 共识机制，支持高吞吐量交易处理。

## 项目特性

### 核心指标
- **目标TPS**: >3000 交易/秒
- **区块间隔**: 2秒
- **区块大小**: 10MB
- **最终确认时间**: 2-4秒（正常），42秒（最坏情况）
- **验证节点**: 21个

### 技术栈
- **开发语言**: Go
- **数据库**: LevelDB
- **网络库**: libp2p-go
- **序列化**: gob
- **共识算法**: PoS（权益证明）

## 项目结构

```
shardmatrix/
├── cmd/                    # 主程序入口
│   └── node/              # 区块链节点程序
├── pkg/                   # 核心包
│   ├── types/             # 数据类型定义
│   ├── config/            # 配置管理
│   ├── storage/           # 数据存储
│   ├── consensus/         # 共识算法
│   ├── network/           # 网络通信
│   ├── crypto/            # 密码学
│   ├── wallet/            # 钱包功能
│   └── api/              # API接口
├── docs/                  # 文档
│   ├── architecture/      # 架构文档
│   └── development/       # 开发文档
├── examples/              # 示例代码
├── config.yaml           # 配置文件
└── go.mod               # Go模块文件
```

## 快速开始

### 环境要求
- Go 1.21+
- LevelDB

### 安装依赖
```bash
go mod download
```

### 运行节点
```bash
go run cmd/node/main.go
```

### 运行示例
```bash
# 运行基本使用示例
go run examples/basic_usage.go

# 运行 LevelDB 存储示例
make example-leveldb
```

### 配置说明
编辑 `config.yaml` 文件来配置节点参数：

```yaml
# 网络配置
network:
  port: 8080
  host: "0.0.0.0"

# 区块链配置
blockchain:
  chain_id: 1
  block_interval: 2
  max_block_size: 10485760

# 共识配置
consensus:
  type: "pos"
  validator_count: 21
```

## 开发状态

### 已完成
- [x] 项目架构设计
- [x] 基础数据结构定义（区块、交易）
- [x] 配置管理系统
- [x] 存储层实现（LevelDB 区块存储、交易存储、账户存储、验证者存储）
- [x] 密码学模块
- [x] 交易池管理
- [x] 区块链管理器
- [x] DPoS 共识机制
- [x] P2P 网络通信

### 进行中
- [ ] API 接口
- [ ] 钱包功能
- [ ] 分片机制
- [ ] 智能合约支持

### 计划中
- [ ] 性能优化
- [ ] 监控和运维工具
- [ ] 部署和打包

## 文档

- [架构概览](docs/architecture/overview.md) - 系统架构设计
- [存储层设计](docs/architecture/storage.md) - 存储系统架构
- [LevelDB 区块存储实现](docs/storage/leveldb_block_store.md) - LevelDB 区块存储详细实现
- [开发路线图](docs/development/roadmap.md) - 详细开发计划
- [API文档](docs/api/) - API接口文档（待完善）

## 贡献

欢迎提交 Issue 和 Pull Request！

## 许可证

[LICENSE](LICENSE)