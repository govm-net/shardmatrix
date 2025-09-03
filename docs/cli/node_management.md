# ShardMatrix 节点管理命令

## 概述

ShardMatrix 现在提供了完整的命令行节点管理功能，允许用户通过命令行界面监控和管理区块链节点。

## 命令结构

```
shardmatrix-node node [command]
```

## 可用命令

### 1. 节点状态 (status)

获取节点的当前状态信息。

#### 用法
```bash
shardmatrix-node node status [flags]
```

#### 标志
- `-c, --config string`: 配置文件路径 (默认 "config.yaml")
- `-f, --format string`: 输出格式 (text, json) (默认 "text")

#### 示例
```bash
# 获取默认格式的节点状态
shardmatrix-node node status

# 获取JSON格式的节点状态
shardmatrix-node node status --format json

# 使用自定义配置文件
shardmatrix-node node status --config /path/to/config.yaml
```

#### 输出信息
- 节点ID
- 版本信息
- 运行状态和运行时间
- 连接的节点数量
- 区块链高度和最佳区块
- 链健康状态
- 网络状态信息

### 2. 节点配置 (config)

管理节点配置。

#### 用法
```bash
shardmatrix-node node config [command]
```

#### 子命令

##### 显示配置 (show)
显示当前节点配置。

```bash
shardmatrix-node node config show [flags]
```

**标志:**
- `-c, --config string`: 配置文件路径 (默认 "config.yaml")
- `-f, --format string`: 输出格式 (yaml, json) (默认 "yaml")

**示例:**
```bash
# 显示YAML格式配置
shardmatrix-node node config show

# 显示JSON格式配置
shardmatrix-node node config show --format json
```

##### 验证配置 (validate)
验证配置文件的有效性。

```bash
shardmatrix-node node config validate [flags]
```

**标志:**
- `-c, --config string`: 配置文件路径 (默认 "config.yaml")

**示例:**
```bash
# 验证默认配置文件
shardmatrix-node node config validate

# 验证自定义配置文件
shardmatrix-node node config validate --config /path/to/config.yaml
```

### 3. 节点连接 (peers)

显示当前连接的节点列表。

#### 用法
```bash
shardmatrix-node node peers [flags]
```

#### 标志
- `-c, --config string`: 配置文件路径 (默认 "config.yaml")
- `-f, --format string`: 输出格式 (text, json) (默认 "text")

#### 示例
```bash
# 获取默认格式的节点列表
shardmatrix-node node peers

# 获取JSON格式的节点列表
shardmatrix-node node peers --format json
```

### 4. 重启节点 (restart)

重启节点（提供重启信息）。

#### 用法
```bash
shardmatrix-node node restart [flags]
```

#### 标志
- `-c, --config string`: 配置文件路径 (默认 "config.yaml")
- `-f, --force`: 强制重启无需确认

#### 示例
```bash
# 重启节点（需要确认）
shardmatrix-node node restart

# 强制重启节点
shardmatrix-node node restart --force
```

## 使用示例

### 查看节点状态
```bash
$ shardmatrix-node node status
=== ShardMatrix Node Status ===
Node ID: 12D3KooWKZNpLnnhcwuk4nWJdp75BzMoeXUxEki97pWBN7mkM1z5
Version: 1.0.0
Running: false
Uptime: 0s
Connected Peers: 0
Blockchain Height: 0
Best Block: 163db733f58ad0a7e485c1aec8c617fde110291e22400317de71197198a648ae
Chain Health: stale
Is Syncing: false

--- Network Status ---
Connected Peers: 0
Partitioned: false
Reconnect Count: 0
```

### 查看JSON格式状态
```bash
$ shardmatrix-node node status --format json
{
  "best_block": "163db733f58ad0a7e485c1aec8c617fde110291e22400317de71197198a648ae",
  "chain_health": "stale",
  "chain_height": 0,
  "is_running": false,
  "is_syncing": false,
  "network_status": {
    "connected_peers": 0,
    "is_partitioned": false,
    "last_update": "2025-01-01T10:00:00Z",
    "partition_since": "0001-01-01T00:00:00Z",
    "reconnect_count": 0
  },
  "node_id": "12D3KooWKZNpLnnhcwuk4nWJdp75BzMoeXUxEki97pWBN7mkM1z5",
  "peer_count": 0,
  "uptime": "0s",
  "version": "1.0.0"
}
```

### 验证配置
```bash
$ shardmatrix-node node config validate
✅ Configuration is valid!
Configuration file: config.yaml
Chain ID: 1
Network port: 8080
Storage type: leveldb
Data directory: ./data
```

## 实现细节

### 网络状态监控
节点管理命令集成了网络状态监控功能，能够提供：
- 连接节点数量
- 网络分区检测
- 重新连接计数
- 分区持续时间

### 配置管理
支持多种格式的配置显示和验证：
- YAML格式（默认）
- JSON格式
- 配置文件有效性检查

### 安全性考虑
- 重启命令需要用户确认（除非使用--force标志）
- 所有命令都使用安全的配置文件加载机制
- 错误处理和用户友好的错误消息

## 未来改进

1. 添加更多的节点管理功能
2. 实现节点远程管理能力
3. 添加节点性能监控和告警
4. 增强配置管理功能