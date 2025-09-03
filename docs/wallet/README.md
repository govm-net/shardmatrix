# ShardMatrix 钱包功能

ShardMatrix 提供了内置的钱包功能，允许用户创建和管理区块链账户。

## 功能特性

- 创建新的钱包账户
- 列出已有的钱包
- 查询账户余额
- 交易签名（开发中）

## 命令行工具

### 创建钱包

```bash
# 创建新钱包并保存到默认文件 wallet.json
./shardmatrix-node wallet create

# 创建新钱包并保存到指定文件
./shardmatrix-node wallet create --file my_wallets.json
```

### 列出钱包

```bash
# 列出默认钱包文件中的所有钱包
./shardmatrix-node wallet list

# 列出指定文件中的所有钱包
./shardmatrix-node wallet list --file my_wallets.json
```

### 查询余额

```bash
# 查询指定地址的余额
./shardmatrix-node wallet balance --address 0x1234567890123456789012345678901234567890

# 从指定钱包文件中查询余额
./shardmatrix-node wallet balance --file my_wallets.json --address 0x1234567890123456789012345678901234567890
```

## 钱包存储

钱包信息以JSON格式存储在文件中，包含以下信息：

```json
{
  "0x1234567890123456789012345678901234567890": {
    "address": "0x1234567890123456789012345678901234567890",
    "private_key": "...",
    "public_key": "..."
  }
}
```

**注意**: 在生产环境中，私钥应该进行加密存储，而不是明文存储。

## 安全建议

1. 妥善保管钱包文件，避免泄露
2. 定期备份钱包文件
3. 在生产环境中使用加密存储私钥
4. 不要在不安全的网络环境中操作钱包

## API 接口

钱包功能也可以通过API接口访问：

- `GET /accounts/{address}` - 获取账户信息
- `GET /accounts/{address}/balance` - 获取账户余额

详细API文档请参考 [API文档](../api/README.md)。