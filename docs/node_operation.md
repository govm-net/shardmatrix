# 节点操作指南

## 目录
- [环境准备](#环境准备)
- [编译安装](#编译安装)
- [节点初始化](#节点初始化)
- [创世节点设置](#创世节点设置)
- [验证者节点设置](#验证者节点设置)
- [节点启动](#节点启动)
- [常见问题](#常见问题)

## 环境准备

1. 安装 Go 1.23+
```bash
# 下载并安装 Go
wget https://golang.org/dl/go1.23.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.23.linux-amd64.tar.gz

# 设置环境变量
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
echo 'export GOPATH=$HOME/go' >> ~/.bashrc
source ~/.bashrc
```

2. 安装 CometBFT
```bash
# 下载 CometBFT
git clone https://github.com/cometbft/cometbft.git
cd cometbft

# 编译安装
make install
```

3. 克隆代码仓库
```bash
git clone https://github.com/govm-net/shardmatrix.git
cd shardmatrix
```

## 编译安装

### 1. 编译二进制文件

```bash
# 编译 shardmatrixd
make build

# 或者手动编译
go build -o build/shardmatrixd ./cmd/shardmatrixd
```

### 2. 安装到系统路径

```bash
# 创建安装目录
sudo mkdir -p /usr/local/bin

# 复制二进制文件
sudo cp build/shardmatrixd /usr/local/bin/

# 设置执行权限
sudo chmod +x /usr/local/bin/shardmatrixd
```

### 3. 验证安装

```bash
# 检查版本
shardmatrixd version

# 检查帮助信息
shardmatrixd --help
```

### 4. 可选：创建符号链接

```bash
# 创建符号链接
sudo ln -s /usr/local/bin/shardmatrixd /usr/bin/shardmatrixd
```

## 节点初始化

### 1. 初始化创世节点

```bash
# 设置环境变量
export HOME_DIR=".shardmatrix"
export CHAIN_ID="shardmatrix-1"

# 运行初始化脚本
./scripts/init.sh
```

初始化过程会：
- 创建必要的目录结构
- 生成节点密钥
- 创建创世文件
- 设置初始验证者

### 2. 配置创世节点

编辑 `$HOME_DIR/config/config.toml`：
```toml
# P2P 配置
p2p.laddr = "tcp://0.0.0.0:26656"
p2p.seeds = ""  # 创世节点不需要 seeds

# RPC 配置
rpc.laddr = "tcp://0.0.0.0:26657"

# 共识配置
consensus.timeout_commit = "1s"
```

### 3. 启动创世节点

```bash
# 设置环境变量
export LOG_LEVEL="info"
export P2P_PORT="26656"
export RPC_PORT="26657"

# 启动节点
./scripts/start.sh
```

## 验证者节点设置

### 1. 初始化验证者节点

```bash
# 设置环境变量
export HOME_DIR=".shardmatrix-validator"
export CHAIN_ID="shardmatrix-1"

# 运行初始化脚本
./scripts/init.sh
```

### 2. 配置验证者节点

编辑 `$HOME_DIR/config/config.toml`：
```toml
# P2P 配置
p2p.laddr = "tcp://0.0.0.0:26656"
p2p.seeds = "node_id@genesis_node_ip:26656"  # 添加创世节点作为 seed

# RPC 配置
rpc.laddr = "tcp://0.0.0.0:26657"
```

### 3. 添加验证者

1. 在创世节点上创建验证者交易：
```bash
shardmatrixd tx staking create-validator \
    --amount=1000000000000000000stake \
    --pubkey=$(shardmatrixd tendermint show-validator) \
    --moniker="validator" \
    --chain-id=$CHAIN_ID \
    --commission-rate="0.10" \
    --commission-max-rate="0.20" \
    --commission-max-change-rate="0.01" \
    --min-self-delegation="1" \
    --from=validator \
    --home=$HOME_DIR
```

2. 将验证者交易广播到网络：
```bash
shardmatrixd tx broadcast signed_tx.json
```

### 4. 启动验证者节点

```bash
# 设置环境变量
export LOG_LEVEL="info"
export P2P_PORT="26656"
export RPC_PORT="26657"

# 启动节点
./scripts/start.sh
```

## 节点管理

### 查看节点状态
```bash
shardmatrixd status
```

### 查看验证者列表
```bash
shardmatrixd query staking validators
```

### 查看账户余额
```bash
shardmatrixd query bank balances $(shardmatrixd keys show validator -a)
```

### 停止节点
```bash
pkill shardmatrixd
```

## 常见问题

### 1. 节点无法连接
- 检查防火墙设置
- 确认 P2P 端口是否开放
- 验证 seed 节点地址是否正确

### 2. 同步失败
- 检查网络连接
- 确认创世文件一致
- 查看日志排查具体错误

### 3. 验证者被惩罚
- 检查节点是否在线
- 确认签名密钥是否正确
- 查看惩罚原因和时长

## 安全建议

1. 使用防火墙限制端口访问
2. 定期备份节点数据
3. 使用强密码保护密钥
4. 监控节点状态和资源使用
5. 及时更新软件版本 