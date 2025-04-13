# ShardMatrix

基于Tendermint的简单区块链实现。

## 功能特性

- 基于Tendermint的BFT共识
- 简单的键值存储应用
- 支持P2P网络
- 提供RPC接口

## 快速开始

1. 安装依赖
```bash
go mod download
```

2. 初始化节点
```bash
tendermint init
```

3. 运行节点
```bash
go run cmd/shardmatrixd/main.go
```

## 发送交易

可以通过RPC接口发送交易：

```bash
curl -X POST http://localhost:26657/broadcast_tx_commit -d '{"tx": "{\"key\":\"test\",\"value\":\"hello\"}"}'
```

## 查询状态

可以通过RPC接口查询状态：

```bash
curl -X GET http://localhost:26657/abci_query?data=test
```

## 配置说明

配置文件位于`config/config.toml`，主要配置项包括：

- 共识超时时间
- P2P网络配置
- RPC接口配置
- 数据库配置

## 开发指南

1. 修改ABCI应用逻辑：`pkg/state/application.go`
2. 添加新的交易类型
3. 实现状态查询接口

## 许可证

MIT
