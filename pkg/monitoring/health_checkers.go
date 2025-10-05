package monitoring

import (
	"fmt"
	"time"

	"github.com/lengzhao/shardmatrix/pkg/blockchain"
	"github.com/lengzhao/shardmatrix/pkg/consensus"
	"github.com/lengzhao/shardmatrix/pkg/network"
	"github.com/lengzhao/shardmatrix/pkg/node"
	"github.com/lengzhao/shardmatrix/pkg/storage"
)

// NodeHealthChecker 节点健康检查器
type NodeHealthChecker struct {
	node *node.Node
}

// NewNodeHealthChecker 创建节点健康检查器
func NewNodeHealthChecker(n *node.Node) *NodeHealthChecker {
	return &NodeHealthChecker{node: n}
}

// GetComponentName 获取组件名称
func (nhc *NodeHealthChecker) GetComponentName() string {
	return "node"
}

// CheckHealth 检查节点健康
func (nhc *NodeHealthChecker) CheckHealth() (*ComponentHealth, error) {
	health := &ComponentHealth{
		Name:         "node",
		LastCheck:    time.Now().Unix(),
		CheckCount:   1,
		Metrics:      make(map[string]interface{}),
		Dependencies: []string{"network", "storage", "consensus"},
	}

	if nhc.node == nil {
		health.Status = HealthStatusCritical
		health.Message = "Node instance is nil"
		health.FailCount = 1
		return health, nil
	}

	// 检查节点是否运行
	stats := nhc.node.GetStats()
	isRunning, ok := stats["is_running"].(bool)
	if !ok || !isRunning {
		health.Status = HealthStatusCritical
		health.Message = "Node is not running"
		health.FailCount = 1
		return health, nil
	}

	health.Status = HealthStatusHealthy
	health.Message = "Node is running normally"
	health.Metrics = stats

	return health, nil
}

// BlockchainHealthChecker 区块链健康检查器
type BlockchainHealthChecker struct {
	manager       *blockchain.Manager
	blockProducer *blockchain.BlockProducer
	synchronizer  *blockchain.BlockSynchronizer
}

// NewBlockchainHealthChecker 创建区块链健康检查器
func NewBlockchainHealthChecker(
	manager *blockchain.Manager,
	producer *blockchain.BlockProducer,
	sync *blockchain.BlockSynchronizer,
) *BlockchainHealthChecker {
	return &BlockchainHealthChecker{
		manager:       manager,
		blockProducer: producer,
		synchronizer:  sync,
	}
}

// GetComponentName 获取组件名称
func (bhc *BlockchainHealthChecker) GetComponentName() string {
	return "blockchain"
}

// CheckHealth 检查区块链健康
func (bhc *BlockchainHealthChecker) CheckHealth() (*ComponentHealth, error) {
	health := &ComponentHealth{
		Name:         "blockchain",
		LastCheck:    time.Now().Unix(),
		CheckCount:   1,
		Metrics:      make(map[string]interface{}),
		Dependencies: []string{"storage", "consensus"},
	}

	var issues []string

	// 检查区块链管理器
	if bhc.manager != nil {
		stats := bhc.manager.GetStats()
		health.Metrics["manager"] = stats

		isInitialized, ok := stats["is_initialized"].(bool)
		if !ok || !isInitialized {
			issues = append(issues, "blockchain not initialized")
		}
	} else {
		issues = append(issues, "blockchain manager is nil")
	}

	// 检查区块生产器
	if bhc.blockProducer != nil {
		producerStats := bhc.blockProducer.GetStats()
		health.Metrics["producer"] = producerStats

		isRunning, ok := producerStats["is_running"].(bool)
		if ok && !isRunning {
			issues = append(issues, "block producer not running")
		}
	}

	// 检查同步器
	if bhc.synchronizer != nil {
		syncStats := bhc.synchronizer.GetSyncStatus()
		health.Metrics["synchronizer"] = syncStats

		isRunning, ok := syncStats["is_running"].(bool)
		if !ok || !isRunning {
			issues = append(issues, "synchronizer not running")
		}

		errorCount, ok := syncStats["error_count"].(int)
		if ok && errorCount > 10 {
			issues = append(issues, fmt.Sprintf("too many sync errors: %d", errorCount))
		}
	}

	// 确定健康状态
	if len(issues) == 0 {
		health.Status = HealthStatusHealthy
		health.Message = "Blockchain components are healthy"
	} else if len(issues) <= 2 {
		health.Status = HealthStatusDegraded
		health.Message = fmt.Sprintf("Minor issues: %v", issues)
		health.FailCount = len(issues)
	} else {
		health.Status = HealthStatusUnhealthy
		health.Message = fmt.Sprintf("Multiple issues: %v", issues)
		health.FailCount = len(issues)
	}

	return health, nil
}

// NetworkHealthChecker 网络健康检查器
type NetworkHealthChecker struct {
	network *network.Network
}

// NewNetworkHealthChecker 创建网络健康检查器
func NewNetworkHealthChecker(net *network.Network) *NetworkHealthChecker {
	return &NetworkHealthChecker{network: net}
}

// GetComponentName 获取组件名称
func (nhc *NetworkHealthChecker) GetComponentName() string {
	return "network"
}

// CheckHealth 检查网络健康
func (nhc *NetworkHealthChecker) CheckHealth() (*ComponentHealth, error) {
	health := &ComponentHealth{
		Name:         "network",
		LastCheck:    time.Now().Unix(),
		CheckCount:   1,
		Metrics:      make(map[string]interface{}),
		Dependencies: []string{},
	}

	if nhc.network == nil {
		health.Status = HealthStatusCritical
		health.Message = "Network instance is nil"
		health.FailCount = 1
		return health, nil
	}

	// 检查网络运行状态
	if !nhc.network.IsRunning() {
		health.Status = HealthStatusCritical
		health.Message = "Network is not running"
		health.FailCount = 1
		return health, nil
	}

	// 获取网络统计
	networkStats := nhc.network.GetNetworkStats()
	health.Metrics["network_stats"] = networkStats

	// 获取队列统计
	queueStats := nhc.network.GetQueueStats()
	health.Metrics["queue_stats"] = queueStats

	// 获取带宽使用
	bandwidthUsage := nhc.network.GetBandwidthUsage()
	health.Metrics["bandwidth_usage"] = bandwidthUsage

	var issues []string

	// 检查对等节点数量
	if peerCount, ok := networkStats["peer_count"].(int); ok {
		if peerCount < 3 {
			issues = append(issues, fmt.Sprintf("low peer count: %d", peerCount))
		}
	}

	// 检查健康节点比例
	if healthyPeers, ok := networkStats["healthy_peers"].(int); ok {
		if totalPeers, ok := networkStats["peer_count"].(int); ok && totalPeers > 0 {
			healthyRatio := float64(healthyPeers) / float64(totalPeers)
			if healthyRatio < 0.6 {
				issues = append(issues, fmt.Sprintf("low healthy peer ratio: %.2f", healthyRatio))
			}
		}
	}

	// 检查拥塞级别
	if queueStats != nil {
		if congestionLevel, ok := queueStats["congestion_level"].(float64); ok {
			if congestionLevel > 0.8 {
				issues = append(issues, fmt.Sprintf("high congestion: %.2f", congestionLevel))
			}
		}

		// 检查消息丢失
		if droppedMessages, ok := queueStats["dropped_messages"].(uint64); ok {
			if droppedMessages > 1000 {
				issues = append(issues, fmt.Sprintf("high message drop count: %d", droppedMessages))
			}
		}
	}

	// 检查带宽使用
	if bandwidthUsage != nil {
		if usagePercent, ok := bandwidthUsage["usage_percent"].(float64); ok {
			if usagePercent > 90 {
				issues = append(issues, fmt.Sprintf("high bandwidth usage: %.1f%%", usagePercent))
			}
		}
	}

	// 确定健康状态
	if len(issues) == 0 {
		health.Status = HealthStatusHealthy
		health.Message = "Network is healthy"
	} else if len(issues) <= 2 {
		health.Status = HealthStatusDegraded
		health.Message = fmt.Sprintf("Network issues: %v", issues)
		health.FailCount = len(issues)
	} else {
		health.Status = HealthStatusUnhealthy
		health.Message = fmt.Sprintf("Multiple network issues: %v", issues)
		health.FailCount = len(issues)
	}

	return health, nil
}

// ConsensusHealthChecker 共识健康检查器
type ConsensusHealthChecker struct {
	consensus *consensus.DPoSConsensus
}

// NewConsensusHealthChecker 创建共识健康检查器
func NewConsensusHealthChecker(cons *consensus.DPoSConsensus) *ConsensusHealthChecker {
	return &ConsensusHealthChecker{consensus: cons}
}

// GetComponentName 获取组件名称
func (chc *ConsensusHealthChecker) GetComponentName() string {
	return "consensus"
}

// CheckHealth 检查共识健康
func (chc *ConsensusHealthChecker) CheckHealth() (*ComponentHealth, error) {
	health := &ComponentHealth{
		Name:         "consensus",
		LastCheck:    time.Now().Unix(),
		CheckCount:   1,
		Metrics:      make(map[string]interface{}),
		Dependencies: []string{"network"},
	}

	if chc.consensus == nil {
		health.Status = HealthStatusCritical
		health.Message = "Consensus instance is nil"
		health.FailCount = 1
		return health, nil
	}

	// 检查共识是否活跃
	if !chc.consensus.IsActive() {
		health.Status = HealthStatusCritical
		health.Message = "Consensus is not active"
		health.FailCount = 1
		return health, nil
	}

	// 获取共识信息
	consensusInfo := chc.consensus.GetConsensusInfo()
	health.Metrics["consensus_info"] = consensusInfo

	// 获取健康统计
	healthStats := chc.consensus.GetHealthStats()
	health.Metrics["health_stats"] = healthStats

	var issues []string

	// 检查验证者数量
	if totalValidators, ok := consensusInfo["total_validators"].(int); ok {
		if totalValidators == 0 {
			issues = append(issues, "no validators")
		}
	}

	// 检查活跃验证者比例
	activeRatio := chc.consensus.GetActiveValidatorRatio()
	if activeRatio < 0.6 {
		issues = append(issues, fmt.Sprintf("low active validator ratio: %.2f", activeRatio))
	}

	// 检查是否处于安全模式
	if chc.consensus.IsInSafetyMode() {
		issues = append(issues, "consensus in safety mode")
	}

	// 检查平均健康分数
	if healthStats != nil {
		if avgScore, ok := healthStats["avg_score"].(float64); ok {
			if avgScore < 0.5 {
				issues = append(issues, fmt.Sprintf("low average health score: %.2f", avgScore))
			}
		}
	}

	// 确定健康状态
	if len(issues) == 0 {
		health.Status = HealthStatusHealthy
		health.Message = "Consensus is healthy"
	} else if len(issues) == 1 && issues[0] == "consensus in safety mode" {
		health.Status = HealthStatusDegraded
		health.Message = "Consensus in safety mode"
		health.FailCount = 1
	} else if len(issues) <= 2 {
		health.Status = HealthStatusDegraded
		health.Message = fmt.Sprintf("Consensus issues: %v", issues)
		health.FailCount = len(issues)
	} else {
		health.Status = HealthStatusUnhealthy
		health.Message = fmt.Sprintf("Multiple consensus issues: %v", issues)
		health.FailCount = len(issues)
	}

	return health, nil
}

// StorageHealthChecker 存储健康检查器
type StorageHealthChecker struct {
	storage storage.Storage
}

// NewStorageHealthChecker 创建存储健康检查器
func NewStorageHealthChecker(stor storage.Storage) *StorageHealthChecker {
	return &StorageHealthChecker{storage: stor}
}

// GetComponentName 获取组件名称
func (shc *StorageHealthChecker) GetComponentName() string {
	return "storage"
}

// CheckHealth 检查存储健康
func (shc *StorageHealthChecker) CheckHealth() (*ComponentHealth, error) {
	health := &ComponentHealth{
		Name:         "storage",
		Status:       HealthStatusUnknown, // 初始设置为未知状态
		LastCheck:    time.Now().Unix(),
		CheckCount:   1,
		Metrics:      make(map[string]interface{}),
		Dependencies: []string{},
	}

	if shc.storage == nil {
		health.Status = HealthStatusCritical
		health.Message = "Storage instance is nil"
		health.FailCount = 1
		return health, nil
	}

	// 获取存储统计
	var storageStats map[string]interface{}
	if statsProvider, ok := shc.storage.(interface{ GetStats() map[string]interface{} }); ok {
		storageStats = statsProvider.GetStats()
		health.Metrics["storage_stats"] = storageStats
	} else {
		// 如果存储实现没有GetStats方法，使用空统计
		storageStats = make(map[string]interface{})
		health.Metrics["storage_stats"] = storageStats
	}

	// 获取存储健康检查（如果实现了该方法）
	if levelDBStorage, ok := shc.storage.(*storage.LevelDBStorage); ok {
		storageHealth := levelDBStorage.GetStorageHealth()
		health.Metrics["storage_health"] = storageHealth

		// 检查读写状态
		if readWriteOk, ok := storageHealth["read_write_ok"].(bool); ok {
			if !readWriteOk {
				health.Status = HealthStatusCritical
				health.Message = "Storage read/write failed"
				health.FailCount = 1
				return health, nil
			}
		}

		// 检查批量队列健康
		if batchQueueHealthy, ok := storageHealth["batch_queue_healthy"].(bool); ok {
			if !batchQueueHealthy {
				health.Status = HealthStatusDegraded
				health.Message = "Batch queue near capacity"
				health.FailCount = 1
			}
		}
	}

	var issues []string

	// 检查是否有运行状态
	if isRunning, ok := storageStats["is_running"].(bool); ok {
		if !isRunning {
			issues = append(issues, "storage not running")
		}
	}

	// 确定健康状态
	if len(issues) == 0 && health.Status == HealthStatusUnknown {
		health.Status = HealthStatusHealthy
		health.Message = "Storage is healthy"
	} else if len(issues) > 0 {
		health.Status = HealthStatusUnhealthy
		health.Message = fmt.Sprintf("Storage issues: %v", issues)
		health.FailCount = len(issues)
	}

	return health, nil
}