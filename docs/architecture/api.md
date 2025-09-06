# ShardMatrix API 接口设计

## 概述

ShardMatrix 提供 RESTful API 和 WebSocket API，支持区块链查询、交易提交和实时数据订阅。

## API 架构

- **REST API**: HTTP/JSON 接口
- **WebSocket API**: 实时数据订阅
- **GraphQL API**: 灵活查询接口

## REST API

### 基础信息
- **基础URL**: `http://localhost:8080/api/v1`
- **认证方式**: API Key / JWT Token
- **数据格式**: JSON

### 区块相关接口

#### 获取最新区块
```http
GET /blocks/latest
```

#### 获取指定区块
```http
GET /blocks/{height}
```

#### 获取区块列表
```http
GET /blocks?limit=10&offset=0
```

### 交易相关接口

#### 提交交易
```http
POST /transactions
```

**请求体**:
```json
{
  "from": "0x1234...",
  "to": "0x5678...",
  "nonce": 1,
  "gas_limit": 21000,
  "gas_price": 20000000000,
  "fee": 420000000000000,
  "type": 0,
  "data": "0x",
  "shard_id": 0
}
```

#### 获取交易信息
```http
GET /transactions/{tx_hash}
```

### 账户相关接口

#### 获取账户信息
```http
GET /accounts/{address}
```

#### 获取账户交易历史
```http
GET /accounts/{address}/transactions?limit=10&offset=0
```

### 共识相关接口

#### 获取验证者列表
```http
GET /validators
```

#### 获取权益信息
```http
GET /stakes/{address}
```

#### 获取委托人信息
```http
GET /delegators/{validator_address}
```

#### 获取投票权重
```http
GET /votes/{validator_address}
```

## WebSocket API

### 连接
```javascript
const ws = new WebSocket('ws://localhost:8080/ws');
```

### 订阅区块
```javascript
ws.send(JSON.stringify({
  "method": "subscribe",
  "params": {
    "type": "blocks",

  },
  "id": 1
}));
```

### 订阅交易
```javascript
ws.send(JSON.stringify({
  "method": "subscribe",
  "params": {
    "type": "transactions",
    "address": "0x1234..."
  },
  "id": 2
}));
```

## GraphQL API

### Schema 定义
```graphql
type Block {
  header: BlockHeader!
  transactions: [String!]!
  receipts: [Receipt!]!
}

type Transaction {
  hash: String!
  from: String!
  to: String!
  nonce: Int!
  gasLimit: Int!
  gasPrice: Int!
  fee: String!
  type: Int!
  data: String!
  status: String!
}

type Query {
  block(height: Int!): Block
  latestBlock: Block
  transaction(hash: String!): Transaction
  account(address: String!): Account
}

type Subscription {
  newBlock: Block!
  newTransaction: Transaction!
}
```

## 错误处理

### 错误码定义
- `1000`: 成功
- `1001`: 参数错误
- `1002`: 资源不存在
- `1003`: 权限不足
- `1004`: 服务器内部错误

### 错误响应格式
```json
{
  "code": 1001,
  "message": "参数错误",
  "data": null,
  "details": {
    "field": "address",
    "reason": "地址格式不正确"
  }
}
```

## 认证授权

### API Key 认证
```http
GET /api/v1/blocks/latest
X-API-Key: your-api-key
```

### JWT Token 认证
```http
GET /api/v1/accounts/0x1234
Authorization: Bearer your-jwt-token
```

## 限流控制

### 限流策略
- **REST API**: 1000 请求/分钟/IP
- **WebSocket**: 100 连接/IP
- **GraphQL**: 500 查询/分钟/IP

## 监控指标

### API 指标
- 请求数量
- 响应时间
- 错误率
- 并发连接数

### 业务指标
- 交易提交成功率
- 查询响应时间
- WebSocket 连接数
