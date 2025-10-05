package monitoring

import (
	"runtime"
	"time"

	"github.com/lengzhao/shardmatrix/pkg/blockchain"
	"github.com/lengzhao/shardmatrix/pkg/consensus"
	"github.com/lengzhao/shardmatrix/pkg/network"
	"github.com/lengzhao/shardmatrix/pkg/node"
	"github.com/lengzhao/shardmatrix/pkg/storage"
)

// SystemMetricsCollector 系统指标收集器
type SystemMetricsCollector struct {
	name string
}

// NewSystemMetricsCollector 创建系统指标收集器
func NewSystemMetricsCollector() *SystemMetricsCollector {
	return &SystemMetricsCollector{
		name: "system_metrics",
	}
}

// GetName 获取收集器名称
func (smc *SystemMetricsCollector) GetName() string {
	return smc.name
}

// CollectMetrics 收集系统指标
func (smc *SystemMetricsCollector) CollectMetrics() ([]*Metric, error) {
	var metrics []*Metric
	now := time.Now().Unix()

	// 内存使用情况
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	metrics = append(metrics, &Metric{
		Name:        "system.memory.heap_used",
		Type:        MetricTypeGauge,
		Value:       float64(m.HeapInuse),
		Timestamp:   now,
		Description: "Heap memory in use",
		Unit:        "bytes",
		Labels:      map[string]string{"component": "system"},
	})

	metrics = append(metrics, &Metric{
		Name:        "system.memory.heap_allocated",
		Type:        MetricTypeGauge,
		Value:       float64(m.HeapAlloc),
		Timestamp:   now,
		Description: "Heap memory allocated",
		Unit:        "bytes",
		Labels:      map[string]string{"component": "system"},
	})

	// 垃圾回收统计
	metrics = append(metrics, &Metric{
		Name:        "system.gc.collections",
		Type:        MetricTypeCounter,
		Value:       float64(m.NumGC),
		Timestamp:   now,
		Description: "Number of garbage collections",
		Unit:        "count",
		Labels:      map[string]string{"component": "system"},
	})

	// Goroutine 数量
	metrics = append(metrics, &Metric{
		Name:        "system.goroutines",
		Type:        MetricTypeGauge,
		Value:       float64(runtime.NumGoroutine()),
		Timestamp:   now,
		Description: "Number of goroutines",
		Unit:        "count",
		Labels:      map[string]string{"component": "system"},
	})

	return metrics, nil
}

// BlockchainMetricsCollector 区块链指标收集器
type BlockchainMetricsCollector struct {
	name           string
	blockchainMgr  *blockchain.Manager
	blockProducer  *blockchain.BlockProducer
	synchronizer   *blockchain.BlockSynchronizer
}

// NewBlockchainMetricsCollector 创建区块链指标收集器
func NewBlockchainMetricsCollector(
	blockchainMgr *blockchain.Manager,
	blockProducer *blockchain.BlockProducer,
	synchronizer *blockchain.BlockSynchronizer,
) *BlockchainMetricsCollector {
	return &BlockchainMetricsCollector{
		name:          "blockchain_metrics",
		blockchainMgr: blockchainMgr,
		blockProducer: blockProducer,
		synchronizer:  synchronizer,
	}
}

// GetName 获取收集器名称
func (bmc *BlockchainMetricsCollector) GetName() string {
	return bmc.name
}

// CollectMetrics 收集区块链指标
func (bmc *BlockchainMetricsCollector) CollectMetrics() ([]*Metric, error) {
	var metrics []*Metric
	now := time.Now().Unix()

	// 区块链统计
	if bmc.blockchainMgr != nil {
		stats := bmc.blockchainMgr.GetStats()
		
		if height, ok := stats["current_height"].(uint64); ok {
			metrics = append(metrics, &Metric{
				Name:        "blockchain.block_height",
				Type:        MetricTypeGauge,
				Value:       float64(height),
				Timestamp:   now,
				Description: "Current blockchain height",
				Unit:        "blocks",
				Labels:      map[string]string{"component": "blockchain"},
			})
		}

		if accountCount, ok := stats["account_count"].(int); ok {
			metrics = append(metrics, &Metric{
				Name:        "blockchain.accounts",
				Type:        MetricTypeGauge,
				Value:       float64(accountCount),
				Timestamp:   now,
				Description: "Number of accounts",
				Unit:        "count",
				Labels:      map[string]string{"component": "blockchain"},
			})
		}
	}

	// 区块生产器统计
	if bmc.blockProducer != nil {
		producerStats := bmc.blockProducer.GetStats()
		
		if isRunning, ok := producerStats["is_running"].(bool); ok {
			var value float64
			if isRunning {
				value = 1.0
			}
			metrics = append(metrics, &Metric{
				Name:        "blockchain.producer_running",
				Type:        MetricTypeGauge,
				Value:       value,
				Timestamp:   now,
				Description: "Block producer running status",
				Unit:        "boolean",
				Labels:      map[string]string{"component": "blockchain"},
			})
		}

		if txCount, ok := producerStats["pending_tx_count"].(int); ok {
			metrics = append(metrics, &Metric{
				Name:        "blockchain.pending_transactions",
				Type:        MetricTypeGauge,
				Value:       float64(txCount),
				Timestamp:   now,
				Description: "Number of pending transactions",
				Unit:        "count",
				Labels:      map[string]string{"component": "blockchain"},
			})
		}

		// 空区块统计
		emptyBlockStats := bmc.blockProducer.GetEmptyBlockStats()
		if enabled, ok := emptyBlockStats["empty_block_enabled"].(bool); ok {
			var value float64
			if enabled {
				value = 1.0
			}
			metrics = append(metrics, &Metric{
				Name:        "blockchain.empty_blocks_enabled",
				Type:        MetricTypeGauge,
				Value:       value,
				Timestamp:   now,
				Description: "Empty block generation enabled",
				Unit:        "boolean",
				Labels:      map[string]string{"component": "blockchain"},
			})
		}
	}

	// 同步器统计
	if bmc.synchronizer != nil {
		syncStats := bmc.synchronizer.GetSyncStatus()
		
		if currentHeight, ok := syncStats["current_height"].(uint64); ok {
			metrics = append(metrics, &Metric{
				Name:        "blockchain.sync_height",
				Type:        MetricTypeGauge,
				Value:       float64(currentHeight),
				Timestamp:   now,
				Description: "Current sync height",
				Unit:        "blocks",
				Labels:      map[string]string{"component": "blockchain"},
			})
		}

		if progress, ok := syncStats["progress"].(float64); ok {
			metrics = append(metrics, &Metric{
				Name:        "blockchain.sync_progress",
				Type:        MetricTypeGauge,
				Value:       progress,
				Timestamp:   now,
				Description: "Sync progress percentage",
				Unit:        "percent",
				Labels:      map[string]string{"component": "blockchain"},
			})
		}

		if errorCount, ok := syncStats["error_count"].(int); ok {
			metrics = append(metrics, &Metric{
				Name:        "blockchain.sync_errors",
				Type:        MetricTypeCounter,
				Value:       float64(errorCount),
				Timestamp:   now,
				Description: "Number of sync errors",
				Unit:        "count",
				Labels:      map[string]string{"component": "blockchain"},
			})
		}
	}

	return metrics, nil
}

// NetworkMetricsCollector 网络指标收集器
type NetworkMetricsCollector struct {
	name    string
	network *network.Network
}

// NewNetworkMetricsCollector 创建网络指标收集器
func NewNetworkMetricsCollector(net *network.Network) *NetworkMetricsCollector {
	return &NetworkMetricsCollector{
		name:    "network_metrics",
		network: net,
	}
}

// GetName 获取收集器名称
func (nmc *NetworkMetricsCollector) GetName() string {
	return nmc.name
}

// CollectMetrics 收集网络指标
func (nmc *NetworkMetricsCollector) CollectMetrics() ([]*Metric, error) {
	var metrics []*Metric
	now := time.Now().Unix()

	if nmc.network == nil {
		return metrics, nil
	}

	// 网络统计
	networkStats := nmc.network.GetNetworkStats()

	if peerCount, ok := networkStats["peer_count"].(int); ok {
		metrics = append(metrics, &Metric{
			Name:        "network.peer_count",
			Type:        MetricTypeGauge,
			Value:       float64(peerCount),
			Timestamp:   now,
			Description: "Number of connected peers",
			Unit:        "count",
			Labels:      map[string]string{"component": "network"},
		})
	}

	if healthyPeers, ok := networkStats["healthy_peers"].(int); ok {
		metrics = append(metrics, &Metric{
			Name:        "network.healthy_peers",
			Type:        MetricTypeGauge,
			Value:       float64(healthyPeers),
			Timestamp:   now,
			Description: "Number of healthy peers",
			Unit:        "count",
			Labels:      map[string]string{"component": "network"},
		})
	}

	if messagesSent, ok := networkStats["messages_sent"].(uint64); ok {
		metrics = append(metrics, &Metric{
			Name:        "network.messages_sent",
			Type:        MetricTypeCounter,
			Value:       float64(messagesSent),
			Timestamp:   now,
			Description: "Total messages sent",
			Unit:        "count",
			Labels:      map[string]string{"component": "network"},
		})
	}

	if messagesReceived, ok := networkStats["messages_received"].(uint64); ok {
		metrics = append(metrics, &Metric{
			Name:        "network.messages_received",
			Type:        MetricTypeCounter,
			Value:       float64(messagesReceived),
			Timestamp:   now,
			Description: "Total messages received",
			Unit:        "count",
			Labels:      map[string]string{"component": "network"},
		})
	}

	if bytesSent, ok := networkStats["bytes_sent"].(uint64); ok {
		metrics = append(metrics, &Metric{
			Name:        "network.bytes_sent",
			Type:        MetricTypeCounter,
			Value:       float64(bytesSent),
			Timestamp:   now,
			Description: "Total bytes sent",
			Unit:        "bytes",
			Labels:      map[string]string{"component": "network"},
		})
	}

	if bytesReceived, ok := networkStats["bytes_received"].(uint64); ok {
		metrics = append(metrics, &Metric{
			Name:        "network.bytes_received",
			Type:        MetricTypeCounter,
			Value:       float64(bytesReceived),
			Timestamp:   now,
			Description: "Total bytes received",
			Unit:        "bytes",
			Labels:      map[string]string{"component": "network"},
		})
	}

	// 队列统计
	queueStats := nmc.network.GetQueueStats()
	if queueStats != nil {
		if totalQueue, ok := queueStats["total_queue_size"].(int); ok {
			metrics = append(metrics, &Metric{
				Name:        "network.queue_size",
				Type:        MetricTypeGauge,
				Value:       float64(totalQueue),
				Timestamp:   now,
				Description: "Total message queue size",
				Unit:        "count",
				Labels:      map[string]string{"component": "network"},
			})
		}

		if droppedMessages, ok := queueStats["dropped_messages"].(uint64); ok {
			metrics = append(metrics, &Metric{
				Name:        "network.dropped_messages",
				Type:        MetricTypeCounter,
				Value:       float64(droppedMessages),
				Timestamp:   now,
				Description: "Total dropped messages",
				Unit:        "count",
				Labels:      map[string]string{"component": "network"},
			})
		}

		if congestionLevel, ok := queueStats["congestion_level"].(float64); ok {
			metrics = append(metrics, &Metric{
				Name:        "network.congestion_level",
				Type:        MetricTypeGauge,
				Value:       congestionLevel,
				Timestamp:   now,
				Description: "Network congestion level",
				Unit:        "ratio",
				Labels:      map[string]string{"component": "network"},
			})
		}
	}

	// 带宽使用
	bandwidthUsage := nmc.network.GetBandwidthUsage()
	if bandwidthUsage != nil {
		if usagePercent, ok := bandwidthUsage["usage_percent"].(float64); ok {
			metrics = append(metrics, &Metric{
				Name:        "network.bandwidth_usage",
				Type:        MetricTypeGauge,
				Value:       usagePercent,
				Timestamp:   now,
				Description: "Bandwidth usage percentage",
				Unit:        "percent",
				Labels:      map[string]string{"component": "network"},
			})
		}
	}

	return metrics, nil
}

// ConsensusMetricsCollector 共识指标收集器
type ConsensusMetricsCollector struct {
	name      string
	consensus *consensus.DPoSConsensus
}

// NewConsensusMetricsCollector 创建共识指标收集器
func NewConsensusMetricsCollector(cons *consensus.DPoSConsensus) *ConsensusMetricsCollector {
	return &ConsensusMetricsCollector{
		name:      "consensus_metrics",
		consensus: cons,
	}
}

// GetName 获取收集器名称
func (cmc *ConsensusMetricsCollector) GetName() string {
	return cmc.name
}

// CollectMetrics 收集共识指标
func (cmc *ConsensusMetricsCollector) CollectMetrics() ([]*Metric, error) {
	var metrics []*Metric
	now := time.Now().Unix()

	if cmc.consensus == nil {
		return metrics, nil
	}

	// 共识统计
	consensusInfo := cmc.consensus.GetConsensusInfo()

	if isActive, ok := consensusInfo["is_active"].(bool); ok {
		var value float64
		if isActive {
			value = 1.0
		}
		metrics = append(metrics, &Metric{
			Name:        "consensus.is_active",
			Type:        MetricTypeGauge,
			Value:       value,
			Timestamp:   now,
			Description: "Consensus active status",
			Unit:        "boolean",
			Labels:      map[string]string{"component": "consensus"},
		})
	}

	if totalValidators, ok := consensusInfo["total_validators"].(int); ok {
		metrics = append(metrics, &Metric{
			Name:        "consensus.total_validators",
			Type:        MetricTypeGauge,
			Value:       float64(totalValidators),
			Timestamp:   now,
			Description: "Total number of validators",
			Unit:        "count",
			Labels:      map[string]string{"component": "consensus"},
		})
	}

	if activeValidators, ok := consensusInfo["active_validators"].(int); ok {
		metrics = append(metrics, &Metric{
			Name:        "consensus.active_validators",
			Type:        MetricTypeGauge,
			Value:       float64(activeValidators),
			Timestamp:   now,
			Description: "Number of active validators",
			Unit:        "count",
			Labels:      map[string]string{"component": "consensus"},
		})
	}

	if totalVotePower, ok := consensusInfo["total_vote_power"].(uint64); ok {
		metrics = append(metrics, &Metric{
			Name:        "consensus.total_vote_power",
			Type:        MetricTypeGauge,
			Value:       float64(totalVotePower),
			Timestamp:   now,
			Description: "Total voting power",
			Unit:        "power",
			Labels:      map[string]string{"component": "consensus"},
		})
	}

	// 活跃验证者比例
	activeRatio := cmc.consensus.GetActiveValidatorRatio()
	metrics = append(metrics, &Metric{
		Name:        "consensus.active_validator_ratio",
		Type:        MetricTypeGauge,
		Value:       activeRatio,
		Timestamp:   now,
		Description: "Active validator ratio",
		Unit:        "ratio",
		Labels:      map[string]string{"component": "consensus"},
	})

	// 安全模式状态
	inSafetyMode := cmc.consensus.IsInSafetyMode()
	var safetyModeValue float64
	if inSafetyMode {
		safetyModeValue = 1.0
	}
	metrics = append(metrics, &Metric{
		Name:        "consensus.safety_mode",
		Type:        MetricTypeGauge,
		Value:       safetyModeValue,
		Timestamp:   now,
		Description: "Safety mode status",
		Unit:        "boolean",
		Labels:      map[string]string{"component": "consensus"},
	})

	// 健康统计
	healthStats := cmc.consensus.GetHealthStats()
	if healthStats != nil {
		if avgScore, ok := healthStats["avg_score"].(float64); ok {
			metrics = append(metrics, &Metric{
				Name:        "consensus.avg_health_score",
				Type:        MetricTypeGauge,
				Value:       avgScore,
				Timestamp:   now,
				Description: "Average validator health score",
				Unit:        "score",
				Labels:      map[string]string{"component": "consensus"},
			})
		}
	}

	return metrics, nil
}

// StorageMetricsCollector 存储指标收集器  
type StorageMetricsCollector struct {
	name    string
	storage storage.Storage
}

// NewStorageMetricsCollector 创建存储指标收集器
func NewStorageMetricsCollector(stor storage.Storage) *StorageMetricsCollector {
	return &StorageMetricsCollector{
		name:    "storage_metrics",
		storage: stor,
	}
}

// GetName 获取收集器名称
func (smc *StorageMetricsCollector) GetName() string {
	return smc.name
}

// CollectMetrics 收集存储指标
func (smc *StorageMetricsCollector) CollectMetrics() ([]*Metric, error) {
	var metrics []*Metric
	now := time.Now().Unix()

	if smc.storage == nil {
		return metrics, nil
	}

	// 存储统计
	var storageStats map[string]interface{}
	if statsProvider, ok := smc.storage.(interface{ GetStats() map[string]interface{} }); ok {
		storageStats = statsProvider.GetStats()
	} else {
		// 如果存储实现没有GetStats方法，返回空指标
		return metrics, nil
	}

	if readsTotal, ok := storageStats["reads_total"].(int64); ok {
		metrics = append(metrics, &Metric{
			Name:        "storage.reads_total",
			Type:        MetricTypeCounter,
			Value:       float64(readsTotal),
			Timestamp:   now,
			Description: "Total storage reads",
			Unit:        "count",
			Labels:      map[string]string{"component": "storage"},
		})
	}

	if writesTotal, ok := storageStats["writes_total"].(int64); ok {
		metrics = append(metrics, &Metric{
			Name:        "storage.writes_total",
			Type:        MetricTypeCounter,
			Value:       float64(writesTotal),
			Timestamp:   now,
			Description: "Total storage writes",
			Unit:        "count",
			Labels:      map[string]string{"component": "storage"},
		})
	}

	if bytesRead, ok := storageStats["bytes_read"].(int64); ok {
		metrics = append(metrics, &Metric{
			Name:        "storage.bytes_read",
			Type:        MetricTypeCounter,
			Value:       float64(bytesRead),
			Timestamp:   now,
			Description: "Total bytes read",
			Unit:        "bytes",
			Labels:      map[string]string{"component": "storage"},
		})
	}

	if bytesWritten, ok := storageStats["bytes_written"].(int64); ok {
		metrics = append(metrics, &Metric{
			Name:        "storage.bytes_written",
			Type:        MetricTypeCounter,
			Value:       float64(bytesWritten),
			Timestamp:   now,
			Description: "Total bytes written",
			Unit:        "bytes",
			Labels:      map[string]string{"component": "storage"},
		})
	}

	if batchesTotal, ok := storageStats["batches_total"].(int64); ok {
		metrics = append(metrics, &Metric{
			Name:        "storage.batches_total",
			Type:        MetricTypeCounter,
			Value:       float64(batchesTotal),
			Timestamp:   now,
			Description: "Total batch operations",
			Unit:        "count",
			Labels:      map[string]string{"component": "storage"},
		})
	}

	if checkpointHeight, ok := storageStats["checkpoint_height"].(uint64); ok {
		metrics = append(metrics, &Metric{
			Name:        "storage.checkpoint_height",
			Type:        MetricTypeGauge,
			Value:       float64(checkpointHeight),
			Timestamp:   now,
			Description: "Latest checkpoint height",
			Unit:        "blocks",
			Labels:      map[string]string{"component": "storage"},
		})
	}

	return metrics, nil
}

// NodeMetricsCollector 节点指标收集器
type NodeMetricsCollector struct {
	name         string
	node         *node.Node
	healthMonitor *node.HealthMonitor
}

// NewNodeMetricsCollector 创建节点指标收集器
func NewNodeMetricsCollector(n *node.Node, hm *node.HealthMonitor) *NodeMetricsCollector {
	return &NodeMetricsCollector{
		name:          "node_metrics",
		node:          n,
		healthMonitor: hm,
	}
}

// GetName 获取收集器名称
func (nmc *NodeMetricsCollector) GetName() string {
	return nmc.name
}

// CollectMetrics 收集节点指标
func (nmc *NodeMetricsCollector) CollectMetrics() ([]*Metric, error) {
	var metrics []*Metric
	now := time.Now().Unix()

	// 节点统计
	if nmc.node != nil {
		nodeStats := nmc.node.GetStats()

		if isRunning, ok := nodeStats["is_running"].(bool); ok {
			var value float64
			if isRunning {
				value = 1.0
			}
			metrics = append(metrics, &Metric{
				Name:        "node.is_running",
				Type:        MetricTypeGauge,
				Value:       value,
				Timestamp:   now,
				Description: "Node running status",
				Unit:        "boolean",
				Labels:      map[string]string{"component": "node"},
			})
		}

		if isValidator, ok := nodeStats["is_validator"].(bool); ok {
			var value float64
			if isValidator {
				value = 1.0
			}
			metrics = append(metrics, &Metric{
				Name:        "node.is_validator",
				Type:        MetricTypeGauge,
				Value:       value,
				Timestamp:   now,
				Description: "Node validator status",
				Unit:        "boolean",
				Labels:      map[string]string{"component": "node"},
			})
		}
	}

	// 健康监控统计
	if nmc.healthMonitor != nil {
		healthStats := nmc.healthMonitor.GetHealthStats()

		if activeNodes, ok := healthStats["active_nodes"].(int); ok {
			metrics = append(metrics, &Metric{
				Name:        "node.active_nodes",
				Type:        MetricTypeGauge,
				Value:       float64(activeNodes),
				Timestamp:   now,
				Description: "Number of active nodes",
				Unit:        "count",
				Labels:      map[string]string{"component": "node"},
			})
		}

		if totalNodes, ok := healthStats["total_nodes"].(int); ok {
			metrics = append(metrics, &Metric{
				Name:        "node.total_nodes",
				Type:        MetricTypeGauge,
				Value:       float64(totalNodes),
				Timestamp:   now,
				Description: "Total number of nodes",
				Unit:        "count",
				Labels:      map[string]string{"component": "node"},
			})
		}

		if activePercentage, ok := healthStats["active_percentage"].(float64); ok {
			metrics = append(metrics, &Metric{
				Name:        "node.active_percentage",
				Type:        MetricTypeGauge,
				Value:       activePercentage,
				Timestamp:   now,
				Description: "Active nodes percentage",
				Unit:        "percent",
				Labels:      map[string]string{"component": "node"},
			})
		}

		if aboveThreshold, ok := healthStats["above_threshold"].(bool); ok {
			var value float64
			if aboveThreshold {
				value = 1.0
			}
			metrics = append(metrics, &Metric{
				Name:        "node.above_threshold",
				Type:        MetricTypeGauge,
				Value:       value,
				Timestamp:   now,
				Description: "Above activity threshold status",
				Unit:        "boolean",
				Labels:      map[string]string{"component": "node"},
			})
		}
	}

	return metrics, nil
}