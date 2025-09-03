# ShardMatrix 网络分区检测功能

## 概述

ShardMatrix 现在支持网络分区检测功能，能够监控网络连接状态并在检测到网络分区时采取相应的措施。

## 功能实现

### 1. 网络监控器 (NetworkMonitor)

在网络节点中集成了网络监控器，定期检查网络连接状态：

```go
type NetworkMonitor struct {
    node           *Node
    status         NetworkStatus
    lastPeerCount  int
    partitionTimer *time.Timer
    mutex          sync.RWMutex
}
```

### 2. 网络状态跟踪

每10秒钟检查一次网络连接状态：

```go
func (nm *NetworkMonitor) checkNetworkStatus() {
    // 获取当前连接的节点数量
    currentPeers := len(nm.node.network.GetPeers())
    
    // 检测网络分区
    if nm.lastPeerCount > 0 && currentPeers == 0 {
        // 检测到可能的网络分区
        nm.handleNetworkPartition()
    } else if nm.lastPeerCount == 0 && currentPeers > 0 {
        // 网络连接恢复
        nm.handleNetworkRecovery()
    }
}
```

### 3. 分区处理

当检测到网络分区时，系统会记录事件并采取相应措施：

```go
func (nm *NetworkMonitor) handleNetworkPartition() {
    // 记录分区事件
    // 暂停某些网络相关操作
    // 保持区块生产和交易处理
}
```

### 4. 分区恢复

当网络连接恢复时，系统会触发区块同步：

```go
func (nm *NetworkMonitor) handleNetworkRecovery() {
    // 触发区块同步
    // 重新建立网络连接
}
```

## API 接口

节点信息API现在包含网络状态信息：

```json
{
  "network_status": {
    "connected_peers": 10,
    "is_partitioned": false,
    "partition_since": "2025-01-01T10:00:00Z",
    "reconnect_count": 2,
    "last_update": "2025-01-01T10:05:00Z"
  }
}
```

## 测试验证

通过多节点测试验证了网络监控功能的正确性：

1. 节点启动和连接建立
2. 网络状态监控
3. 连接状态变化检测
4. 网络状态信息报告

## 使用说明

网络分区检测功能默认启用，无需额外配置。系统会自动监控网络状态并在控制台输出相关信息。

## 未来改进

1. 实现更智能的分区恢复机制
2. 添加网络分区事件的日志记录
3. 实现自动重连机制
4. 添加网络健康度评估