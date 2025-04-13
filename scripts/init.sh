#!/bin/bash

# 设置默认值
HOME_DIR=${HOME_DIR:-".shardmatrix"}
CHAIN_ID=${CHAIN_ID:-"shardmatrix-1"}

# 创建目录结构
mkdir -p $HOME_DIR/config
mkdir -p $HOME_DIR/data

# 初始化节点配置
shardmatrixd init local --chain-id $CHAIN_ID --home $HOME_DIR

# 生成验证者密钥
shardmatrixd tendermint gen-validator --home $HOME_DIR

# 创建创世账户
shardmatrixd keys add validator --home $HOME_DIR

# 添加创世账户到创世文件
shardmatrixd add-genesis-account validator 1000000000000000000stake --home $HOME_DIR

# 创建创世交易
shardmatrixd gentx validator 1000000000000000000stake --chain-id $CHAIN_ID --home $HOME_DIR

# 收集创世交易
shardmatrixd collect-gentxs --home $HOME_DIR

echo "Node initialized successfully in $HOME_DIR" 