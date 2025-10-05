package node

import (
	"fmt"
	"sync"
	"time"

	"github.com/lengzhao/shardmatrix/pkg/types"
)

// HealthStatus 节点健康状态
type HealthStatus int

const (
	HealthStatusActive   HealthStatus = iota // 活跃状态
	HealthStatusInactive                     // 非活跃状态
	HealthStatusUnknown                      // 未知状态
)

// NodeHealth 节点健康信息
type NodeHealth struct {
	Address         types.Address `json:"address"`          // 节点地址
	LastHeartbeat   int64         `json:"last_heartbeat"`   // 最后心跳时间
	LastBlockTime   int64         `json:"last_block_time"`  // 最后出块时间
	ConnectionCount int           `json:"connection_count"` // 连接数
	Status          HealthStatus  `json:"status"`           // 健康状态
	BlocksProduced  uint64        `json:"blocks_produced"`  // 生产区块数
	MissedBlocks    uint64        `json:"missed_blocks"`    // 错过区块数
	Responsiveness  float64       `json:"responsiveness"`   // 响应度评分(0-1)
}

// HealthMonitor 节点健康监控器
type HealthMonitor struct {
	mu                 sync.RWMutex
	nodeHealthMap      map[types.Address]*NodeHealth // 节点健康状态映射
	heartbeatTimeout   time.Duration                 // 心跳超时时间
	blockTimeout       time.Duration                 // 出块超时时间
	minActiveNodes     int                           // 最小活跃节点数(60%要求)
	totalValidators    int                           // 总验证者数量
	isRunning          bool
	stopCh             chan struct{}
	callbacks          []ActiveNodeChangeCallback
}

// ActiveNodeChangeCallback 活跃节点变化回调
type ActiveNodeChangeCallback func(activeCount int, totalCount int, isAboveThreshold bool)

// NewHealthMonitor 创建健康监控器
func NewHealthMonitor(totalValidators int) *HealthMonitor {
	minActiveNodes := int(float64(totalValidators) * 0.6) // 60%活跃度要求
	if minActiveNodes < 1 {
		minActiveNodes = 1
	}

	return &HealthMonitor{
		nodeHealthMap:    make(map[types.Address]*NodeHealth),
		heartbeatTimeout: 30 * time.Second,  // 30秒心跳超时
		blockTimeout:     6 * time.Second,   // 3个区块周期超时
		minActiveNodes:   minActiveNodes,
		totalValidators:  totalValidators,
		stopCh:          make(chan struct{}),
		callbacks:       make([]ActiveNodeChangeCallback, 0),
	}
}

// Start 启动健康监控
func (hm *HealthMonitor) Start() error {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	if hm.isRunning {
		return fmt.Errorf("health monitor already running")
	}

	hm.isRunning = true
	hm.stopCh = make(chan struct{})

	go hm.monitorLoop()
	return nil
}

// Stop 停止健康监控
func (hm *HealthMonitor) Stop() {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	if !hm.isRunning {
		return
	}

	hm.isRunning = false
	close(hm.stopCh)
}

// RegisterNode 注册节点
func (hm *HealthMonitor) RegisterNode(address types.Address) {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	if _, exists := hm.nodeHealthMap[address]; !exists {
		hm.nodeHealthMap[address] = &NodeHealth{
			Address:         address,
			LastHeartbeat:   time.Now().Unix(),
			LastBlockTime:   0,
			ConnectionCount: 0,
			Status:          HealthStatusUnknown,
			BlocksProduced:  0,
			MissedBlocks:    0,
			Responsiveness:  0.5, // 初始评分
		}
	}
}

// UpdateHeartbeat 更新节点心跳
func (hm *HealthMonitor) UpdateHeartbeat(address types.Address, connections int) {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	health, exists := hm.nodeHealthMap[address]
	if !exists {
		hm.RegisterNode(address)
		health = hm.nodeHealthMap[address]
	}

	now := time.Now().Unix()
	health.LastHeartbeat = now
	health.ConnectionCount = connections

	// 更新响应度评分
	hm.updateResponsiveness(health)
}

// UpdateBlockProduction 更新节点出块信息
func (hm *HealthMonitor) UpdateBlockProduction(address types.Address, blockTime int64, produced bool) {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	health, exists := hm.nodeHealthMap[address]
	if !exists {
		hm.RegisterNode(address)
		health = hm.nodeHealthMap[address]
	}

	health.LastBlockTime = blockTime
	if produced {
		health.BlocksProduced++
	} else {
		health.MissedBlocks++
	}

	// 更新响应度评分
	hm.updateResponsiveness(health)
}

// updateResponsiveness 更新节点响应度评分
func (hm *HealthMonitor) updateResponsiveness(health *NodeHealth) {
	now := time.Now().Unix()
	
	// 心跳响应度 (40%权重)
	heartbeatScore := 1.0
	if now-health.LastHeartbeat > int64(hm.heartbeatTimeout.Seconds()) {
		heartbeatScore = 0.0
	} else if now-health.LastHeartbeat > int64(hm.heartbeatTimeout.Seconds()/2) {
		heartbeatScore = 0.5
	}

	// 出块响应度 (35%权重)
	blockScore := 1.0
	if health.LastBlockTime > 0 && now-health.LastBlockTime > int64(hm.blockTimeout.Seconds()) {
		blockScore = 0.0
	}

	// 区块生产率 (25%权重)
	productionScore := 1.0
	totalBlocks := health.BlocksProduced + health.MissedBlocks
	if totalBlocks > 0 {
		productionScore = float64(health.BlocksProduced) / float64(totalBlocks)
	}

	// 综合评分
	health.Responsiveness = heartbeatScore*0.4 + blockScore*0.35 + productionScore*0.25
}

// monitorLoop 监控循环
func (hm *HealthMonitor) monitorLoop() {
	ticker := time.NewTicker(10 * time.Second) // 每10秒检查一次
	defer ticker.Stop()

	for {
		select {
		case <-hm.stopCh:
			return
		case <-ticker.C:
			hm.checkNodeHealth()
		}
	}
}

// checkNodeHealth 检查所有节点健康状态
func (hm *HealthMonitor) checkNodeHealth() {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	now := time.Now().Unix()
	activeCount := 0
	totalCount := len(hm.nodeHealthMap)

	for _, health := range hm.nodeHealthMap {
		oldStatus := health.Status

		// 更新健康状态
		if hm.isNodeActive(health, now) {
			health.Status = HealthStatusActive
			activeCount++
		} else {
			health.Status = HealthStatusInactive
		}

		// 如果状态发生变化，记录日志
		if oldStatus != health.Status {
			fmt.Printf("Node %s status changed: %v -> %v (responsiveness: %.2f)\n", 
				health.Address.String(), oldStatus, health.Status, health.Responsiveness)
		}
	}

	// 检查是否满足60%活跃度要求
	isAboveThreshold := activeCount >= hm.minActiveNodes

	// 触发回调
	for _, callback := range hm.callbacks {
		go callback(activeCount, totalCount, isAboveThreshold)
	}

	// 记录活跃度状态
	if !isAboveThreshold {
		fmt.Printf("⚠️  Active nodes below threshold: %d/%d (required: %d)\n", 
			activeCount, totalCount, hm.minActiveNodes)
	}
}

// isNodeActive 判断节点是否活跃
func (hm *HealthMonitor) isNodeActive(health *NodeHealth, currentTime int64) bool {
	// 多维度活跃度检测
	
	// 1. 心跳检测
	heartbeatActive := currentTime-health.LastHeartbeat <= int64(hm.heartbeatTimeout.Seconds())
	
	// 2. 网络连接检测
	connectionActive := health.ConnectionCount >= 3 // 至少3个连接
	
	// 3. 响应度评分检测
	responsivenessActive := health.Responsiveness >= 0.6 // 60%以上响应度
	
	// 综合判断：必须同时满足心跳和响应度要求，连接数作为参考
	return heartbeatActive && responsivenessActive && connectionActive
}

// GetActiveNodes 获取活跃节点列表
func (hm *HealthMonitor) GetActiveNodes() []types.Address {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	var activeNodes []types.Address
	for address, health := range hm.nodeHealthMap {
		if health.Status == HealthStatusActive {
			activeNodes = append(activeNodes, address)
		}
	}

	return activeNodes
}

// GetActiveNodeCount 获取活跃节点数量
func (hm *HealthMonitor) GetActiveNodeCount() int {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	activeCount := 0
	for _, health := range hm.nodeHealthMap {
		if health.Status == HealthStatusActive {
			activeCount++
		}
	}

	return activeCount
}

// IsAboveThreshold 检查是否达到60%活跃度要求
func (hm *HealthMonitor) IsAboveThreshold() bool {
	activeCount := hm.GetActiveNodeCount()
	return activeCount >= hm.minActiveNodes
}

// GetNodeHealth 获取指定节点健康信息
func (hm *HealthMonitor) GetNodeHealth(address types.Address) *NodeHealth {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	if health, exists := hm.nodeHealthMap[address]; exists {
		// 返回副本以防止并发修改
		healthCopy := *health
		return &healthCopy
	}

	return nil
}

// GetAllNodeHealth 获取所有节点健康信息
func (hm *HealthMonitor) GetAllNodeHealth() map[types.Address]*NodeHealth {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	result := make(map[types.Address]*NodeHealth)
	for address, health := range hm.nodeHealthMap {
		healthCopy := *health
		result[address] = &healthCopy
	}

	return result
}

// RegisterCallback 注册活跃节点变化回调
func (hm *HealthMonitor) RegisterCallback(callback ActiveNodeChangeCallback) {
	hm.mu.Lock()
	defer hm.mu.Unlock()
	hm.callbacks = append(hm.callbacks, callback)
}

// GetHealthStats 获取健康统计信息
func (hm *HealthMonitor) GetHealthStats() map[string]interface{} {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	activeCount := 0
	totalCount := len(hm.nodeHealthMap)
	avgResponsiveness := 0.0

	for _, health := range hm.nodeHealthMap {
		if health.Status == HealthStatusActive {
			activeCount++
		}
		avgResponsiveness += health.Responsiveness
	}

	if totalCount > 0 {
		avgResponsiveness /= float64(totalCount)
	}

	return map[string]interface{}{
		"active_nodes":        activeCount,
		"total_nodes":         totalCount,
		"active_percentage":   float64(activeCount) / float64(totalCount) * 100,
		"min_required":        hm.minActiveNodes,
		"above_threshold":     activeCount >= hm.minActiveNodes,
		"avg_responsiveness":  avgResponsiveness,
		"heartbeat_timeout":   hm.heartbeatTimeout.Seconds(),
		"block_timeout":       hm.blockTimeout.Seconds(),
	}
}

// SetThresholds 设置监控阈值
func (hm *HealthMonitor) SetThresholds(heartbeatTimeout, blockTimeout time.Duration) {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	hm.heartbeatTimeout = heartbeatTimeout
	hm.blockTimeout = blockTimeout
}

// RemoveNode 移除节点监控
func (hm *HealthMonitor) RemoveNode(address types.Address) {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	delete(hm.nodeHealthMap, address)
}