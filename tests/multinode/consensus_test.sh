#!/bin/bash

# ShardMatrix 多节点共识测试脚本
# 用于分析和测试多节点间的区块同步和共识机制

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# 脚本目录
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

echo -e "${BLUE}🔬 ShardMatrix 多节点共识测试${NC}"
echo -e "${BLUE}================================${NC}"

# 检查节点状态
check_nodes_running() {
    echo -e "${CYAN}📊 检查节点运行状态...${NC}"
    
    local all_running=true
    for i in 1 2 3; do
        local api_port=$((8090 + $i))
        if ! curl -s -f http://127.0.0.1:${api_port}/api/v1/blocks/latest > /dev/null; then
            echo -e "${RED}❌ 节点 ${i} API 不可访问 (端口: ${api_port})${NC}"
            all_running=false
        else
            echo -e "${GREEN}✅ 节点 ${i} API 正常 (端口: ${api_port})${NC}"
        fi
    done
    
    if [ "$all_running" = false ]; then
        echo -e "${RED}❌ 部分节点未运行，请先启动多节点网络${NC}"
        echo -e "${YELLOW}提示: 运行 ./tests/multinode/scripts/start_network.sh start${NC}"
        exit 1
    fi
    echo
}

# 获取节点区块信息
get_node_block_info() {
    local node_id=$1
    local api_port=$((8090 + $node_id))
    
    local response=$(curl -s http://127.0.0.1:${api_port}/api/v1/blocks/latest)
    if [ $? -eq 0 ] && [ -n "$response" ]; then
        echo "$response" | jq -r '.data | "\(.height)|\(.hash)|\(.validator)|\(.timestamp)"'
    else
        echo "ERROR|ERROR|ERROR|ERROR"
    fi
}

# 分析区块同步状态
analyze_block_sync() {
    echo -e "${CYAN}🔍 分析区块同步状态...${NC}"
    echo -e "${CYAN}========================${NC}"
    
    # 收集所有节点的区块信息
    # 使用数组存储节点信息
    node_heights=()
    node_hashes=()
    node_validators=()
    node_timestamps=()
    
    echo -e "${BLUE}当前区块状态:${NC}"
    printf "%-8s %-8s %-66s %-42s %-20s\n" "节点" "高度" "区块哈希" "验证者" "时间戳"
    echo "--------------------------------------------------------------------------------------------------------"
    
    for i in 1 2 3; do
        local info=$(get_node_block_info $i)
        local height=$(echo "$info" | cut -d'|' -f1)
        local hash=$(echo "$info" | cut -d'|' -f2)
        local validator=$(echo "$info" | cut -d'|' -f3)
        local timestamp=$(echo "$info" | cut -d'|' -f4)
        
        node_heights[$i]=$height
        node_hashes[$i]=$hash
        node_validators[$i]=$validator
        node_timestamps[$i]=$timestamp
        
        # 时间戳格式化
        local short_time=$(echo "$timestamp" | cut -d'T' -f2 | cut -d'+' -f1)
        
        printf "%-8s %-8s %-66s %-42s %-20s\n" "节点$i" "$height" "$hash" "$validator" "$short_time"
    done
    
    echo
    
    # 分析同步状态
    echo -e "${CYAN}🔍 同步状态分析:${NC}"
    
    # 检查高度差异
    local max_height=0
    local min_height=999999
    
    # 获取所有节点的高度
    local heights=()
    for i in 1 2 3; do
        local info=$(get_node_block_info $i)
        local height=$(echo "$info" | cut -d'|' -f1)
        heights[$i]=$height
        
        if [ "$height" != "ERROR" ] && [ "$height" -gt "$max_height" ]; then
            max_height=$height
        fi
        if [ "$height" != "ERROR" ] && [ "$height" -lt "$min_height" ]; then
            min_height=$height
        fi
    done
    
    local height_diff=$((max_height - min_height))
    echo -e "高度范围: $min_height - $max_height (差异: $height_diff)"
    
    if [ "$height_diff" -eq 0 ]; then
        echo -e "${GREEN}✅ 所有节点区块高度一致${NC}"
    elif [ "$height_diff" -le 2 ]; then
        echo -e "${YELLOW}⚠️  节点间存在小幅高度差异 (可能是正常的网络延迟)${NC}"
    else
        echo -e "${RED}❌ 节点间存在显著高度差异 (可能存在同步问题)${NC}"
    fi
    
    # 检查相同高度的区块哈希
    echo
    echo -e "${CYAN}🔍 区块哈希一致性分析:${NC}"
    
    # 找到共同的高度范围进行比较
    local check_height=$min_height
    local consensus_issues=0
    
    echo "检查高度 $check_height 的区块一致性:"
    
    # 获取所有节点在该高度的区块信息
    for i in 1 2 3; do
        local api_port=$((8090 + $i))
        local block_info=$(curl -s "http://127.0.0.1:${api_port}/api/v1/blocks/height/${check_height}")
        if [ $? -eq 0 ] && [ -n "$block_info" ]; then
            local hash=$(echo "$block_info" | jq -r '.data.hash // "N/A"')
            local validator=$(echo "$block_info" | jq -r '.data.validator // "N/A"')
            echo "  节点$i: $hash (验证者: $(echo $validator | cut -c1-10)...)"
        else
            echo "  节点$i: 无法获取区块信息"
            ((consensus_issues++))
        fi
    done
    
    echo
}

# 测试网络连接
test_network_connectivity() {
    echo -e "${CYAN}🌐 测试网络连接状态...${NC}"
    echo -e "${CYAN}========================${NC}"
    
    for i in 1 2 3; do
        local api_port=$((8090 + $i))
        echo -e "${BLUE}节点 $i 的网络状态:${NC}"
        
        local peers_info=$(curl -s "http://127.0.0.1:${api_port}/api/v1/network/peers")
        if [ $? -eq 0 ] && [ -n "$peers_info" ]; then
            local peer_count=$(echo "$peers_info" | jq -r '.data.summary.total_peers // 0')
            local connected_count=$(echo "$peers_info" | jq -r '.data.summary.connected_peers // 0')
            echo "  对等节点: $connected_count/$peer_count 连接"
        else
            echo "  网络状态: 无法获取"
        fi
        
        # 获取网络健康状态
        local health_info=$(curl -s "http://127.0.0.1:${api_port}/api/v1/network/health")
        if [ $? -eq 0 ] && [ -n "$health_info" ]; then
            local overall_status=$(echo "$health_info" | jq -r '.data.overall_status // "unknown"')
            echo "  网络健康: $overall_status"
        fi
        echo
    done
}

# 测试验证者状态
test_validator_status() {
    echo -e "${CYAN}👑 测试验证者状态...${NC}"
    echo -e "${CYAN}====================${NC}"
    
    echo -e "${BLUE}各节点的验证者信息:${NC}"
    printf "%-8s %-42s %-12s %-12s %-8s\n" "节点" "验证者地址" "权益" "委托数" "状态"
    echo "--------------------------------------------------------------------------------"
    
    for i in 1 2 3; do
        local api_port=$((8090 + $i))
        local validators_info=$(curl -s "http://127.0.0.1:${api_port}/api/v1/consensus/validators")
        
        if [ $? -eq 0 ] && [ -n "$validators_info" ]; then
            # 获取验证者总数
            local validator_count=$(echo "$validators_info" | jq -r '.data.summary.total_validators // 0')
            local active_count=$(echo "$validators_info" | jq -r '.data.summary.active_validators // 0')
            
            echo "节点$i: $active_count/$validator_count 个活跃验证者"
            
            # 显示验证者详情（如果有的话）
            echo "$validators_info" | jq -r '.data.validators[]? | "  \(.address[0:10])... 权益:\(.total_stake) 状态:\(.status)"' 2>/dev/null || echo "  无详细验证者信息"
        else
            echo "节点$i: 无法获取验证者信息"
        fi
        echo
    done
}

# 执行一致性测试
run_consistency_test() {
    echo -e "${CYAN}🧪 执行一致性测试...${NC}"
    echo -e "${CYAN}====================${NC}"
    
    echo "等待 10 秒观察区块生产..."
    
    # 记录初始状态
    local initial_heights=()
    for i in 1 2 3; do
        local info=$(get_node_block_info $i)
        local height=$(echo "$info" | cut -d'|' -f1)
        initial_heights[$i]=$height
        echo "节点$i 初始高度: $height"
    done
    
    sleep 10
    
    echo
    echo "10秒后的状态:"
    
    # 记录最终状态
    local progress_made=false
    for i in 1 2 3; do
        local info=$(get_node_block_info $i)
        local height=$(echo "$info" | cut -d'|' -f1)
        local initial=${initial_heights[$i]}
        local diff=$((height - initial))
        
        echo "节点$i 当前高度: $height (增长: +$diff)"
        
        if [ "$diff" -gt 0 ]; then
            progress_made=true
        fi
    done
    
    echo
    if [ "$progress_made" = true ]; then
        echo -e "${GREEN}✅ 检测到区块生产活动${NC}"
    else
        echo -e "${RED}❌ 未检测到区块生产活动${NC}"
    fi
}

# 生成详细报告
generate_report() {
    echo -e "${CYAN}📋 生成测试报告...${NC}"
    echo -e "${CYAN}==================${NC}"
    
    local report_file="$SCRIPT_DIR/CONSENSUS_TEST_$(date +%Y%m%d_%H%M%S).md"
    
    cat > "$report_file" << EOF
# ShardMatrix 多节点共识测试报告

**测试时间**: $(date)
**测试类型**: 区块同步和共识机制验证

## 测试结果摘要

EOF
    
    # 重新收集数据并写入报告
    echo "### 当前节点状态" >> "$report_file"
    echo >> "$report_file"
    echo "| 节点 | 高度 | 区块哈希 | 验证者 |" >> "$report_file"
    echo "|------|------|----------|---------|" >> "$report_file"
    
    for i in 1 2 3; do
        local info=$(get_node_block_info $i)
        local height=$(echo "$info" | cut -d'|' -f1)
        local hash=$(echo "$info" | cut -d'|' -f2 | cut -c1-16)
        local validator=$(echo "$info" | cut -d'|' -f3 | cut -c1-10)
        
        echo "| 节点$i | $height | $hash... | $validator... |" >> "$report_file"
    done
    
    echo >> "$report_file"
    echo "### 发现的问题" >> "$report_file"
    echo >> "$report_file"
    echo "1. **区块同步问题**: 不同节点产生不同的区块，缺乏统一共识" >> "$report_file"
    echo "2. **验证者选举**: 多个验证者可能同时生产区块" >> "$report_file"
    echo "3. **网络同步**: 节点间区块传播和同步机制需要改进" >> "$report_file"
    echo >> "$report_file"
    echo "### 建议修复方案" >> "$report_file"
    echo >> "$report_file"
    echo "1. 实现严格的验证者轮转机制" >> "$report_file"
    echo "2. 添加区块同步协议" >> "$report_file"
    echo "3. 实现最长链选择算法" >> "$report_file"
    echo "4. 优化网络消息传播机制" >> "$report_file"
    
    echo -e "${GREEN}✅ 报告已生成: $report_file${NC}"
}

# 主要测试流程
main() {
    case "$1" in
        "quick")
            check_nodes_running
            analyze_block_sync
            ;;
        "full")
            check_nodes_running
            analyze_block_sync
            test_network_connectivity
            test_validator_status
            run_consistency_test
            generate_report
            ;;
        "sync")
            check_nodes_running
            analyze_block_sync
            ;;
        "network")
            check_nodes_running
            test_network_connectivity
            ;;
        "validators")
            check_nodes_running
            test_validator_status
            ;;
        *)
            echo -e "${BLUE}ShardMatrix 多节点共识测试工具${NC}"
            echo
            echo -e "${YELLOW}用法:${NC}"
            echo "  $0 quick      - 快速区块同步检查"
            echo "  $0 full       - 完整测试和报告生成"
            echo "  $0 sync       - 区块同步状态分析"
            echo "  $0 network    - 网络连接测试"
            echo "  $0 validators - 验证者状态检查"
            echo
            echo -e "${YELLOW}示例:${NC}"
            echo "  $0 quick      # 快速检查"
            echo "  $0 full       # 完整测试"
            echo "  $0 sync       # 同步分析"
            ;;
    esac
}

# 执行主函数
main "$@"