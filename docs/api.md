# ShardMatrix API 文档

## 概述

本文档描述了ShardMatrix系统的API接口。所有API都遵循RESTful设计规范，使用JSON作为数据交换格式。

## 基础信息

- 基础URL: `http://localhost:26657`
- 认证方式: JWT Token
- 数据格式: JSON

## 通用响应格式

```json
{
    "jsonrpc": "2.0",
    "id": "1",
    "result": {
        "code": 0,
        "data": {},
        "message": "success"
    }
}
```

## API接口

### 1. 分片管理

#### 1.1 获取分片信息
- 请求路径: `/shard/info`
- 请求方法: GET
- 请求参数: 无
- 响应示例:
```json
{
    "jsonrpc": "2.0",
    "id": "1",
    "result": {
        "code": 0,
        "data": {
            "shard_id": 1,
            "parent_id": 0,
            "child_ids": [2, 3],
            "validators": ["validator1", "validator2"]
        },
        "message": "success"
    }
}
```

### 2. 共识管理

#### 2.1 获取共识状态
- 请求路径: `/consensus/status`
- 请求方法: GET
- 请求参数: 无
- 响应示例:
```json
{
    "jsonrpc": "2.0",
    "id": "1",
    "result": {
        "code": 0,
        "data": {
            "height": 100,
            "round": 0,
            "step": "RoundStepPropose",
            "validators": {
                "validator1": {
                    "voting_power": 10,
                    "proposer_priority": 0
                }
            }
        },
        "message": "success"
    }
}
```

#### 2.2 获取验证器集合
- 请求路径: `/consensus/validators`
- 请求方法: GET
- 请求参数: 无
- 响应示例:
```json
{
    "jsonrpc": "2.0",
    "id": "1",
    "result": {
        "code": 0,
        "data": {
            "validators": [
                {
                    "address": "validator1",
                    "pub_key": "pubkey1",
                    "voting_power": 10
                }
            ]
        },
        "message": "success"
    }
}
```

### 3. 网络管理

#### 3.1 获取网络信息
- 请求路径: `/net_info`
- 请求方法: GET
- 请求参数: 无
- 响应示例:
```json
{
    "jsonrpc": "2.0",
    "id": "1",
    "result": {
        "code": 0,
        "data": {
            "listening": true,
            "listeners": ["tcp://0.0.0.0:26656"],
            "n_peers": 4,
            "peers": [
                {
                    "node_info": {
                        "id": "peer1",
                        "listen_addr": "tcp://0.0.0.0:26656"
                    },
                    "is_outbound": true
                }
            ]
        },
        "message": "success"
    }
}
```

#### 3.2 获取节点信息
- 请求路径: `/status`
- 请求方法: GET
- 请求参数: 无
- 响应示例:
```json
{
    "jsonrpc": "2.0",
    "id": "1",
    "result": {
        "code": 0,
        "data": {
            "node_info": {
                "id": "node1",
                "listen_addr": "tcp://0.0.0.0:26656",
                "network": "shardmatrix",
                "version": "0.1.0"
            },
            "sync_info": {
                "latest_block_height": 100,
                "latest_block_time": "2024-01-01T00:00:00Z"
            }
        },
        "message": "success"
    }
}
```

### 4. 智能合约

#### 4.1 部署合约
- 请求路径: `/contract/deploy`
- 请求方法: POST
- 请求参数:
```json
{
    "code": "contract code",
    "args": ["arg1", "arg2"]
}
```
- 响应示例:
```json
{
    "jsonrpc": "2.0",
    "id": "1",
    "result": {
        "code": 0,
        "data": {
            "contract_address": "0x123...",
            "gas_used": 1000
        },
        "message": "success"
    }
}
```

#### 4.2 调用合约
- 请求路径: `/contract/call`
- 请求方法: POST
- 请求参数:
```json
{
    "contract_address": "0x123...",
    "method": "transfer",
    "args": ["to", "amount"]
}
```
- 响应示例:
```json
{
    "jsonrpc": "2.0",
    "id": "1",
    "result": {
        "code": 0,
        "data": {
            "result": "success",
            "gas_used": 500
        },
        "message": "success"
    }
}
```

### 5. 交易管理

#### 5.1 发送交易
- 请求路径: `/tx/broadcast`
- 请求方法: POST
- 请求参数:
```json
{
    "tx": "transaction data",
    "mode": "sync"
}
```
- 响应示例:
```json
{
    "jsonrpc": "2.0",
    "id": "1",
    "result": {
        "code": 0,
        "data": {
            "hash": "0x456...",
            "height": 100
        },
        "message": "success"
    }
}
```

#### 5.2 查询交易
- 请求路径: `/tx/{hash}`
- 请求方法: GET
- 请求参数: 无
- 响应示例:
```json
{
    "jsonrpc": "2.0",
    "id": "1",
    "result": {
        "code": 0,
        "data": {
            "hash": "0x456...",
            "height": 100,
            "tx": "transaction data",
            "result": {
                "code": 0,
                "data": "result data"
            }
        },
        "message": "success"
    }
}
```

## 错误码

| 错误码 | 说明 |
|--------|------|
| 0 | 成功 |
| 1 | 参数错误 |
| 2 | 内部错误 |
| 3 | 权限错误 |
| 4 | 资源不存在 |
| 5 | 资源已存在 |
| 6 | 操作超时 |
| 7 | 网络错误 |
| 8 | 共识错误 |
| 9 | 合约执行错误 |

## 安全说明

1. 所有API请求都需要JWT Token认证
2. 建议使用HTTPS协议
3. 敏感操作需要额外的权限验证
4. 建议使用API限流措施

## 更新日志

### v0.1.0 (2024-01-01)
- 初始版本
- 支持基本的分片管理
- 集成Tendermint共识
- 支持智能合约
- 提供RESTful API 