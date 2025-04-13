#!/bin/bash

# 设置默认值
HOME_DIR=${HOME_DIR:-".shardmatrix"}
LOG_LEVEL=${LOG_LEVEL:-"info"}
P2P_PORT=${P2P_PORT:-"26656"}
RPC_PORT=${RPC_PORT:-"26657"}

# 启动节点
shardmatrixd start \
    --home $HOME_DIR \
    --log_level $LOG_LEVEL \
    --p2p.laddr "tcp://0.0.0.0:$P2P_PORT" \
    --rpc.laddr "tcp://0.0.0.0:$RPC_PORT" 