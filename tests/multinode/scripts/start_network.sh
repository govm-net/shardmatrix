#!/bin/bash

# ShardMatrix 多节点网络测试启动脚本
# 用于启动3个节点的DPoS测试网络

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 脚本目录
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"
MULTINODE_DIR="$PROJECT_ROOT/tests/multinode"
BINARY="$PROJECT_ROOT/build/shardmatrix-node"

echo -e "${BLUE}🚀 ShardMatrix 多节点网络测试${NC}"
echo -e "${BLUE}==============================${NC}"

# 检查二进制文件是否存在
if [ ! -f "$BINARY" ]; then
    echo -e "${RED}❌ 未找到二进制文件，正在构建...${NC}"
    cd "$PROJECT_ROOT"
    make build
    if [ $? -ne 0 ]; then
        echo -e "${RED}❌ 构建失败${NC}"
        exit 1
    fi
fi

# 创建必要的目录
mkdir -p "$MULTINODE_DIR/data/node1" "$MULTINODE_DIR/data/node2" "$MULTINODE_DIR/data/node3" "$MULTINODE_DIR/logs"

# 清理之前的日志和数据（可选）
if [ "$1" = "clean" ]; then
    echo -e "${YELLOW}🧹 清理之前的数据和日志...${NC}"
    rm -rf "$MULTINODE_DIR/data/"*
    rm -rf "$MULTINODE_DIR/logs/"*
    mkdir -p "$MULTINODE_DIR/data/node1" "$MULTINODE_DIR/data/node2" "$MULTINODE_DIR/data/node3" "$MULTINODE_DIR/logs"
fi

# 启动节点函数
start_node() {
    local node_id=$1
    local config_file="$MULTINODE_DIR/node${node_id}.yaml"
    local log_file="$MULTINODE_DIR/logs/node${node_id}.log"
    local pid_file="$MULTINODE_DIR/logs/node${node_id}.pid"
    
    echo -e "${GREEN}🌟 启动节点 ${node_id}...${NC}"
    
    # 启动节点（后台运行）
    nohup "$BINARY" --config "$config_file" --log-level debug > "$log_file" 2>&1 &
    local pid=$!
    echo $pid > "$pid_file"
    
    echo -e "${GREEN}✅ 节点 ${node_id} 已启动 (PID: $pid)${NC}"
    echo -e "   配置文件: $config_file"
    echo -e "   日志文件: $log_file"
    echo -e "   数据目录: $MULTINODE_DIR/data/node${node_id}"
    echo
}

# 停止所有节点函数
stop_all_nodes() {
    echo -e "${YELLOW}🛑 停止所有节点...${NC}"
    
    for i in 1 2 3; do
        local pid_file="$MULTINODE_DIR/logs/node${i}.pid"
        if [ -f "$pid_file" ]; then
            local pid=$(cat "$pid_file")
            if kill -0 "$pid" 2>/dev/null; then
                echo -e "${YELLOW}⏹️  停止节点 ${i} (PID: $pid)${NC}"
                kill "$pid"
                sleep 2
                if kill -0 "$pid" 2>/dev/null; then
                    echo -e "${RED}💀 强制停止节点 ${i}${NC}"
                    kill -9 "$pid"
                fi
            fi
            rm -f "$pid_file"
        fi
    done
    
    echo -e "${GREEN}✅ 所有节点已停止${NC}"
}

# 检查节点状态函数
check_nodes_status() {
    echo -e "${BLUE}📊 节点状态检查${NC}"
    echo -e "${BLUE}=================${NC}"
    
    for i in 1 2 3; do
        local pid_file="$MULTINODE_DIR/logs/node${i}.pid"
        local log_file="$MULTINODE_DIR/logs/node${i}.log"
        
        if [ -f "$pid_file" ]; then
            local pid=$(cat "$pid_file")
            if kill -0 "$pid" 2>/dev/null; then
                echo -e "${GREEN}✅ 节点 ${i} 运行中 (PID: $pid)${NC}"
                
                # API端口检查
                local api_port=$((8090 + $i))
                if command -v nc >/dev/null 2>&1; then
                    if nc -z 127.0.0.1 "$api_port" 2>/dev/null; then
                        echo -e "   🌐 API端口 $api_port 可访问"
                    else
                        echo -e "   ⚠️  API端口 $api_port 不可访问"
                    fi
                fi
                
                # 网络端口检查
                local network_port=$((9000 + $i))
                if command -v nc >/dev/null 2>&1; then
                    if nc -z 127.0.0.1 "$network_port" 2>/dev/null; then
                        echo -e "   🔗 网络端口 $network_port 可访问"
                    else
                        echo -e "   ⚠️  网络端口 $network_port 不可访问"
                    fi
                fi
                
                # 显示最新日志
                if [ -f "$log_file" ]; then
                    echo -e "   📝 最新日志: $(tail -1 "$log_file" 2>/dev/null | head -c 100)..."
                fi
            else
                echo -e "${RED}❌ 节点 ${i} 已停止${NC}"
            fi
        else
            echo -e "${RED}❌ 节点 ${i} 未启动${NC}"
        fi
        echo
    done
}

# 显示实时日志函数
show_logs() {
    local node_id=${1:-"all"}
    
    if [ "$node_id" = "all" ]; then
        echo -e "${BLUE}📝 显示所有节点的实时日志...${NC}"
        echo -e "${YELLOW}提示: 按 Ctrl+C 停止查看日志${NC}"
        sleep 2
        tail -f "$MULTINODE_DIR/logs/node"*.log
    else
        echo -e "${BLUE}📝 显示节点 ${node_id} 的实时日志...${NC}"
        echo -e "${YELLOW}提示: 按 Ctrl+C 停止查看日志${NC}"
        sleep 2
        tail -f "$MULTINODE_DIR/logs/node${node_id}.log"
    fi
}

# 主要启动流程
case "$1" in
    "start")
        echo -e "${GREEN}🚀 启动多节点网络...${NC}"
        
        # 依次启动节点
        start_node 1
        sleep 3  # 等待bootstrap节点启动
        
        start_node 2
        sleep 2  # 等待节点连接
        
        start_node 3
        sleep 2
        
        echo -e "${GREEN}🎉 多节点网络启动完成！${NC}"
        echo
        check_nodes_status
        
        echo -e "${BLUE}💡 有用的命令:${NC}"
        echo -e "  查看状态: $0 status"
        echo -e "  查看日志: $0 logs [node_id]"
        echo -e "  停止网络: $0 stop"
        ;;
        
    "stop")
        stop_all_nodes
        ;;
        
    "restart")
        stop_all_nodes
        sleep 3
        "$0" start
        ;;
        
    "status")
        check_nodes_status
        ;;
        
    "logs")
        show_logs "$2"
        ;;
        
    "clean")
        echo -e "${YELLOW}🧹 清理数据并重新启动...${NC}"
        stop_all_nodes
        echo -e "${YELLOW}🧹 清理之前的数据和日志...${NC}"
        rm -rf "$MULTINODE_DIR/data/"*
        rm -rf "$MULTINODE_DIR/logs/"*
        mkdir -p "$MULTINODE_DIR/data/node1" "$MULTINODE_DIR/data/node2" "$MULTINODE_DIR/data/node3" "$MULTINODE_DIR/logs"
        sleep 3
        "$0" start
        ;;
        
    *)
        echo -e "${BLUE}ShardMatrix 多节点网络测试工具${NC}"
        echo
        echo -e "${YELLOW}用法:${NC}"
        echo "  $0 start         - 启动3节点网络"
        echo "  $0 stop          - 停止所有节点"
        echo "  $0 restart       - 重启网络"
        echo "  $0 status        - 检查节点状态"
        echo "  $0 logs [node_id] - 查看日志 (node_id: 1,2,3 或 all)"
        echo "  $0 clean         - 清理数据并重新启动"
        echo
        echo -e "${YELLOW}示例:${NC}"
        echo "  $0 start         # 启动网络"
        echo "  $0 logs 1        # 查看节点1日志"
        echo "  $0 logs all      # 查看所有节点日志"
        echo "  $0 status        # 检查状态"
        echo "  $0 stop          # 停止网络"
        ;;
esac