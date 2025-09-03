# ShardMatrix API 文档

ShardMatrix 提供了一套完整的 RESTful API 接口，允许开发者与区块链节点进行交互，包括查询区块链状态、提交交易、获取区块和交易信息等操作。

## OpenAPI 规范

我们提供了完整的 OpenAPI 3.0 规范定义文件，您可以使用该文件生成客户端代码或导入到 API 测试工具中：

- [openapi.yaml](./openapi.yaml) - OpenAPI 3.0 规范文件

## 基础信息

- **API版本**: v1
- **基础URL**: `http://localhost:8081/api/v1`
- **数据格式**: JSON
- **认证方式**: 无（开发阶段）

## API 端点

### 健康检查

#### GET /health
检查API服务器的健康状态。

**响应示例**:
```json
{
  "success": true,
  "data": {
    "status": "ok",
    "timestamp": "2023-01-01T12:00:00Z",
    "version": "1.0.0",
    "uptime": "0s"
  }
}
```

### 节点状态

#### GET /status
获取节点的完整状态信息，包括区块链、网络和共识状态。

**响应示例**:
```json
{
  "success": true,
  "data": {
    "node_info": {
      "version": "1.0.0",
      "node_id": "node-1",
      "start_time": "2023-01-01T12:00:00Z",
      "uptime": "0s"
    },
    "blockchain_info": {
      "chain_id": 1,
      "latest_height": 0,
      "latest_hash": "0x...",
      "latest_time": "2023-01-01T12:00:00Z",
      "total_blocks": 1,
      "total_txs": 0
    },
    "network_info": {
      "peer_count": 0,
      "inbound_peers": 0,
      "outbound_peers": 0,
      "listen_addr": []
    },
    "consensus_info": {
      "type": "dpos",
      "validator_count": 21,
      "active_validators": 1,
      "block_interval": "3s"
    }
  }
}
```

### 区块相关接口

#### GET /blocks/latest
获取最新的区块信息。

**响应示例**:
```json
{
  "success": true,
  "data": {
    "hash": "0x...",
    "height": 0,
    "prev_hash": "0x0000000000000000000000000000000000000000000000000000000000000000",
    "merkle_root": "0x...",
    "timestamp": "2023-01-01T12:00:00Z",
    "validator": "0x...",
    "signature": "0x...",
    "tx_count": 0,
    "transactions": [],
    "size": 0
  }
}
```

#### GET /blocks/height/{height}
根据区块高度获取区块信息。

**参数**:
- `height`: 区块高度（整数）

#### GET /blocks/{hash}
根据区块哈希获取区块信息。

**参数**:
- `hash`: 区块哈希（十六进制字符串）

#### GET /blocks
获取区块列表。

**查询参数**:
- `page`: 页码（默认: 1）
- `limit`: 每页数量（默认: 10，最大: 100）
- `from`: 起始高度
- `to`: 结束高度

### 交易相关接口

#### POST /transactions
提交新的交易。

**请求体**:
```json
{
  "from": "0x1234...",
  "to": "0x5678...",
  "amount": 1000,
  "fee": 10,
  "nonce": 1
}
```

**响应示例**:
```json
{
  "success": true,
  "data": {
    "tx_hash": "0x...",
    "status": "pending",
    "message": "Transaction submitted successfully",
    "timestamp": "2023-01-01T12:00:00Z"
  }
}
```

#### GET /transactions/{hash}
根据交易哈希获取交易信息。

**参数**:
- `hash`: 交易哈希（十六进制字符串）

#### GET /transactions
获取交易列表。

**查询参数**:
- `page`: 页码（默认: 1）
- `limit`: 每页数量（默认: 10，最大: 100）
- `address`: 地址筛选
- `status`: 状态筛选

### 账户相关接口

#### GET /accounts/{address}
获取账户信息。

**参数**:
- `address`: 账户地址（十六进制字符串）

#### GET /accounts/{address}/balance
获取账户余额。

**参数**:
- `address`: 账户地址（十六进制字符串）

#### GET /accounts/{address}/transactions
获取账户的交易历史。

**参数**:
- `address`: 账户地址（十六进制字符串）
- `page`: 页码（默认: 1）
- `limit`: 每页数量（默认: 10，最大: 100）

### 验证者相关接口

#### GET /validators
获取验证者列表。

#### GET /validators/{address}
根据地址获取验证者信息。

**参数**:
- `address`: 验证者地址（十六进制字符串）

#### GET /validators/{address}/delegators
获取验证者的委托人列表。

**参数**:
- `address`: 验证者地址（十六进制字符串）

### 网络相关接口

#### GET /network/info
获取网络信息。

#### GET /network/peers
获取连接的节点列表。

#### GET /network/stats
获取网络统计信息。

#### GET /network/health
获取网络健康状态。

### 共识相关接口

#### GET /consensus/info
获取共识机制信息。

#### GET /consensus/validators
获取共识相关的验证者信息。

## 错误处理

所有API错误都遵循统一的错误格式：

```json
{
  "success": false,
  "error": "错误描述",
  "data": {
    "code": 400,
    "message": "详细错误信息",
    "details": "错误详情（可选）"
  }
}
```

## 使用示例

### 获取最新区块
```bash
curl http://localhost:8081/api/v1/blocks/latest
```

### 提交交易
```bash
curl -X POST http://localhost:8081/api/v1/transactions \
  -H "Content-Type: application/json" \
  -d '{
    "from": "0x1234567890123456789012345678901234567890",
    "to": "0x0987654321098765432109876543210987654321",
    "amount": 1000,
    "fee": 10,
    "nonce": 1
  }'
```

## 开发工具

ShardMatrix 提供了API使用示例，可以在 `examples/api_usage.go` 中找到完整的使用示例。

运行示例：
```bash
# 首先启动节点
go run cmd/node/main.go

# 然后运行API示例
go run examples/api_usage.go
```

或者使用Makefile命令：
```bash
# 启动节点
make run

# 运行API示例（在另一个终端）
make example-api
```