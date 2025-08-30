# ShardMatrix 监控与运维设计

## 概述

ShardMatrix 提供全面的监控和运维体系，包括性能监控、告警系统、日志管理和故障恢复。

## 监控架构

- **Prometheus**: 指标收集和存储
- **Grafana**: 数据可视化
- **AlertManager**: 告警管理
- **Jaeger**: 分布式追踪
- **ELK Stack**: 日志管理

## 监控指标

### 性能指标
- **TPS**: 每秒交易数
- **区块时间**: 区块生成时间
- **交易延迟**: 交易确认延迟
- **快速确认时间**: 快速确认延迟
- **最终确认时间**: 最终确认延迟
- **网络延迟**: 节点间通信延迟
- **磁盘使用率**: 存储空间使用情况
- **内存使用率**: 内存占用情况
- **CPU使用率**: CPU占用情况

### 业务指标
- **活跃账户数**: 活跃用户数量
- **总交易数**: 历史交易总数
- **验证者参与率**: 共识参与情况
- **委托人数量**: 参与委托的用户数量
- **投票权重分布**: 验证者投票权重分布
- **节点负载**: 各节点负载分布
- **网络负载**: 网络负载分布

### 安全指标
- **攻击检测**: 恶意攻击次数
- **双重签名**: 双重签名检测
- **快速确认失败**: 快速确认失败次数
- **最终确认失败**: 最终确认失败次数
- **共识失败**: 共识机制失败
- **网络攻击**: 网络层面攻击

## 告警系统

### 告警规则
```yaml
groups:
  - name: blockchain_alerts
    rules:
      - alert: HighTPS
        expr: tps > 1500
        for: 5m
        labels:
          severity: warning
      
      - alert: BlockTimeHigh
        expr: block_time > 5
        for: 2m
        labels:
          severity: critical
      
      - alert: FastConfirmFailed
        expr: fast_confirm_failures > 0
        for: 1m
        labels:
          severity: warning
      
      - alert: FinalConfirmFailed
        expr: final_confirm_failures > 0
        for: 5m
        labels:
          severity: critical
      
      - alert: LowValidatorParticipation
        expr: validator_participation < 0.8
        for: 10m
        labels:
          severity: critical
      
      - alert: HighDiskUsage
        expr: disk_usage > 0.9
        for: 5m
        labels:
          severity: warning
      
      - alert: HighMemoryUsage
        expr: memory_usage > 0.85
        for: 5m
        labels:
          severity: warning
```

### 告警通知
- **邮件告警**: 重要告警发送邮件
- **短信告警**: 紧急告警发送短信
- **钉钉告警**: 团队协作告警
- **Webhook**: 自定义告警处理

## 日志管理

### 日志级别
- **Debug**: 调试信息
- **Info**: 一般信息
- **Warn**: 警告信息
- **Error**: 错误信息
- **Fatal**: 致命错误

### 日志格式
```json
{
  "timestamp": "2024-01-01T00:00:00Z",
  "level": "info",
  "component": "consensus",
  "message": "Block created",
  "trace_id": "abc123",
  "fields": {
    "block_height": 12345,
    "validator": "0x1234..."
  }
}
```

### 日志配置
```yaml
logging:
  level: info
  format: json
  output: stdout
  file:
    path: /var/log/shardmatrix
    max_size: 100MB
    max_age: 30d
    max_backups: 10
```

## 分布式追踪

### 追踪配置
- **服务名称**: ShardMatrix
- **采样率**: 10%
- **最大追踪长度**: 1000
- **端点**: http://localhost:16686

### 追踪示例
```go
func (n *Node) ProcessTransaction(tx *Transaction) error {
    span := tracer.StartSpan("process_transaction")
    defer span.Finish()
    
    span.SetTag("tx_hash", hex.EncodeToString(tx.Hash()))
    span.SetTag("from", hex.EncodeToString(tx.From))
    span.SetTag("to", hex.EncodeToString(tx.To))
    
    // 验证交易
    if err := n.validateTransaction(tx); err != nil {
        span.SetTag("error", true)
        span.LogKV("error", err.Error())
        return err
    }
    
    span.SetTag("success", true)
    return nil
}
```

## 健康检查

### 健康检查接口
```http
GET /health
```

**响应示例**:
```json
{
  "status": "healthy",
  "timestamp": "2024-01-01T00:00:00Z",
  "version": "1.0.0",
  "components": {
    "consensus": {
      "status": "healthy",
      "details": {
        "validator_count": 21,
        "participation_rate": 0.95
      }
    },
    "network": {
      "status": "healthy",
      "details": {
        "peer_count": 50,
        "connection_count": 100
      }
    },
    "storage": {
      "status": "healthy",
      "details": {
        "disk_usage": 0.75,
        "database_size": "10GB"
      }
    }
  }
}
```

## 故障恢复

### 自动恢复
- **网络恢复**: 重新连接节点、同步状态
- **存储恢复**: 检查数据完整性、修复损坏数据
- **共识恢复**: 重新同步区块、验证状态
- **应用恢复**: 重启服务、清理状态

### 恢复策略
```go
type RecoveryStrategy interface {
    Name() string
    CanRecover(error) bool
    Recover() error
}
```

## 运维工具

### 命令行工具
```bash
# 节点管理
shardmatrix node start
shardmatrix node stop
shardmatrix node restart
shardmatrix node status

# 监控查询
shardmatrix metrics list
shardmatrix metrics query "tps > 1000"
shardmatrix alerts list

# 日志查询
shardmatrix logs tail
shardmatrix logs search "error"
shardmatrix logs export --start="2024-01-01" --end="2024-01-02"

# 健康检查
shardmatrix health check
shardmatrix health detailed

# 故障恢复
shardmatrix recovery auto
shardmatrix recovery manual --command="restart_network"
```

### 配置管理
```yaml
# 监控配置
monitoring:
  prometheus:
    enabled: true
    port: 9090
    retention: 30d
  
  grafana:
    enabled: true
    port: 3000
    admin_password: "admin"
  
  alertmanager:
    enabled: true
    port: 9093
    email:
      smtp_host: "smtp.gmail.com"
      smtp_port: 587
      username: "alerts@example.com"
      password: "password"
  
  jaeger:
    enabled: true
    port: 16686
    sampling_rate: 0.1

# 日志配置
logging:
  level: info
  format: json
  output: stdout
  file:
    enabled: true
    path: /var/log/shardmatrix
    max_size: 100MB
    max_age: 30d
    max_backups: 10

# 告警配置
alerts:
  rules:
    - name: "high_tps"
      condition: "tps > 1500"
      duration: "5m"
      severity: "warning"
  
  notifications:
    - type: "email"
      recipients: ["admin@example.com"]
    - type: "webhook"
      url: "https://hooks.slack.com/services/xxx"
```

## 性能调优

### 系统调优
```bash
# 网络调优
sysctl -w net.core.rmem_max=16777216
sysctl -w net.core.wmem_max=16777216
sysctl -w net.ipv4.tcp_rmem="4096 87380 16777216"
sysctl -w net.ipv4.tcp_wmem="4096 65536 16777216"

# 文件系统调优
mount -o noatime,nodiratime /dev/sda1 /data

# 内存调优
echo 3 > /proc/sys/vm/drop_caches
```

### 应用调优
```yaml
# 应用配置
application:
  # 并发配置
  max_goroutines: 10000
  max_connections: 1000
  
  # 缓存配置
  cache_size: 100MB
  cache_ttl: 300s
  
  # 数据库配置
  db_max_connections: 100
  db_max_idle_connections: 10
  db_connection_lifetime: 3600s
  
  # 网络配置
  network_timeout: 30s
  network_keepalive: 60s
  network_max_retries: 3
```
