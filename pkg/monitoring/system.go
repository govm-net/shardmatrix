package monitoring

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/lengzhao/shardmatrix/pkg/types"
)

// MetricType 指标类型
type MetricType string

const (
	MetricTypeCounter   MetricType = "counter"   // 计数器
	MetricTypeGauge     MetricType = "gauge"     // 仪表盘
	MetricTypeHistogram MetricType = "histogram" // 直方图
	MetricTypeSummary   MetricType = "summary"   // 摘要
)

// Severity 告警严重级别
type Severity string

const (
	SeverityInfo     Severity = "info"     // 信息
	SeverityWarning  Severity = "warning"  // 警告
	SeverityError    Severity = "error"    // 错误
	SeverityCritical Severity = "critical" // 严重
)

// Metric 监控指标
type Metric struct {
	Name        string            `json:"name"`        // 指标名称
	Type        MetricType        `json:"type"`        // 指标类型
	Value       float64           `json:"value"`       // 当前值
	Labels      map[string]string `json:"labels"`      // 标签
	Timestamp   int64             `json:"timestamp"`   // 时间戳
	Description string            `json:"description"` // 描述
	Unit        string            `json:"unit"`        // 单位
	Tags        []string          `json:"tags"`        // 标签
}

// Alert 告警信息
type Alert struct {
	ID         string                 `json:"id"`          // 告警ID
	MetricName string                 `json:"metric_name"` // 指标名称
	Severity   Severity               `json:"severity"`    // 严重级别
	Message    string                 `json:"message"`     // 告警消息
	Details    map[string]interface{} `json:"details"`     // 详细信息
	Timestamp  int64                  `json:"timestamp"`   // 告警时间
	ResolvedAt int64                  `json:"resolved_at"` // 解决时间
	IsActive   bool                   `json:"is_active"`   // 是否活跃
	Count      int                    `json:"count"`       // 触发次数
}

// HealthStatus 健康状态
type HealthStatus = types.HealthStatus

const (
	HealthStatusHealthy                = types.HealthStatusHealthy  // 健康
	HealthStatusDegraded  HealthStatus = 4                          // 降级
	HealthStatusUnhealthy HealthStatus = 5                          // 不健康
	HealthStatusCritical               = types.HealthStatusCritical // 严重
	HealthStatusUnknown                = types.HealthStatusUnknown  // 未知
)

// ComponentHealth 组件健康状态
type ComponentHealth struct {
	Name         string                 `json:"name"`         // 组件名称
	Status       HealthStatus           `json:"status"`       // 健康状态
	Message      string                 `json:"message"`      // 状态消息
	LastCheck    int64                  `json:"last_check"`   // 最后检查时间
	CheckCount   int                    `json:"check_count"`  // 检查次数
	FailCount    int                    `json:"fail_count"`   // 失败次数
	Metrics      map[string]interface{} `json:"metrics"`      // 相关指标
	Dependencies []string               `json:"dependencies"` // 依赖组件
}

// SystemDiagnostics 系统诊断信息
type SystemDiagnostics struct {
	OverallHealth HealthStatus                `json:"overall_health"` // 总体健康状态
	Components    map[string]*ComponentHealth `json:"components"`     // 组件健康状态
	ActiveAlerts  []*Alert                    `json:"active_alerts"`  // 活跃告警
	Metrics       map[string]*Metric          `json:"metrics"`        // 系统指标
	Performance   *PerformanceMetrics         `json:"performance"`    // 性能指标
	Network       *NetworkDiagnostics         `json:"network"`        // 网络诊断
	Storage       *StorageDiagnostics         `json:"storage"`        // 存储诊断
	Consensus     *ConsensusDiagnostics       `json:"consensus"`      // 共识诊断
	Timestamp     int64                       `json:"timestamp"`      // 诊断时间
}

// PerformanceMetrics 性能指标
type PerformanceMetrics struct {
	CPU                 float64 `json:"cpu"`                   // CPU使用率
	Memory              float64 `json:"memory"`                // 内存使用率
	Disk                float64 `json:"disk"`                  // 磁盘使用率
	NetworkIO           float64 `json:"network_io"`            // 网络IO
	BlockProcessingRate float64 `json:"block_processing_rate"` // 区块处理速率
	TxProcessingRate    float64 `json:"tx_processing_rate"`    // 交易处理速率
	SyncSpeed           float64 `json:"sync_speed"`            // 同步速度
	Latency             float64 `json:"latency"`               // 延迟
}

// NetworkDiagnostics 网络诊断
type NetworkDiagnostics struct {
	PeerCount        int     `json:"peer_count"`        // 对等节点数
	HealthyPeers     int     `json:"healthy_peers"`     // 健康节点数
	ConnectionErrors int     `json:"connection_errors"` // 连接错误数
	MessageDropRate  float64 `json:"message_drop_rate"` // 消息丢失率
	Bandwidth        float64 `json:"bandwidth"`         // 带宽使用
	Latency          float64 `json:"latency"`           // 网络延迟
	CongestionLevel  float64 `json:"congestion_level"`  // 拥塞级别
}

// StorageDiagnostics 存储诊断
type StorageDiagnostics struct {
	DiskUsage       float64 `json:"disk_usage"`       // 磁盘使用率
	IOLatency       float64 `json:"io_latency"`       // IO延迟
	BlockHeight     uint64  `json:"block_height"`     // 区块高度
	StateSize       int64   `json:"state_size"`       // 状态大小
	CheckpointCount int     `json:"checkpoint_count"` // 检查点数量
	CompactionRate  float64 `json:"compaction_rate"`  // 压缩率
	ErrorRate       float64 `json:"error_rate"`       // 错误率
}

// ConsensusDiagnostics 共识诊断
type ConsensusDiagnostics struct {
	ActiveValidators    int     `json:"active_validators"`     // 活跃验证者数
	ValidatorRatio      float64 `json:"validator_ratio"`       // 验证者比例
	SafetyMode          bool    `json:"safety_mode"`           // 安全模式
	ConsensusLatency    float64 `json:"consensus_latency"`     // 共识延迟
	BlockProductionRate float64 `json:"block_production_rate"` // 出块速率
	MissedBlocks        int     `json:"missed_blocks"`         // 错过区块数
	EmptyBlocks         int     `json:"empty_blocks"`          // 空区块数
}

// MonitoringSystem 监控系统
type MonitoringSystem struct {
	mu                sync.RWMutex
	metrics           map[string]*Metric          // 指标存储
	alerts            map[string]*Alert           // 告警存储
	componentHealth   map[string]*ComponentHealth // 组件健康状态
	alertRules        []*AlertRule                // 告警规则
	collectors        []MetricCollector           // 指标收集器
	alertHandlers     []AlertHandler              // 告警处理器
	healthCheckers    []HealthChecker             // 健康检查器
	diagnosticsBuffer []*SystemDiagnostics        // 诊断历史缓冲
	isRunning         bool                        // 运行状态
	stopCh            chan struct{}               // 停止信号
	config            *MonitoringConfig           // 监控配置
}

// AlertRule 告警规则
type AlertRule struct {
	ID            string            `json:"id"`             // 规则ID
	MetricName    string            `json:"metric_name"`    // 指标名称
	Condition     string            `json:"condition"`      // 条件表达式
	Threshold     float64           `json:"threshold"`      // 阈值
	Severity      Severity          `json:"severity"`       // 严重级别
	Message       string            `json:"message"`        // 告警消息
	Enabled       bool              `json:"enabled"`        // 是否启用
	Cooldown      time.Duration     `json:"cooldown"`       // 冷却时间
	Labels        map[string]string `json:"labels"`         // 标签
	LastTriggered int64             `json:"last_triggered"` // 最后触发时间
}

// MetricCollector 指标收集器接口
type MetricCollector interface {
	CollectMetrics() ([]*Metric, error)
	GetName() string
}

// AlertHandler 告警处理器接口
type AlertHandler interface {
	HandleAlert(alert *Alert) error
	GetName() string
}

// HealthChecker 健康检查器接口
type HealthChecker interface {
	CheckHealth() (*ComponentHealth, error)
	GetComponentName() string
}

// MonitoringConfig 监控配置
type MonitoringConfig struct {
	MetricRetention     time.Duration `json:"metric_retention"`      // 指标保留时间
	AlertRetention      time.Duration `json:"alert_retention"`       // 告警保留时间
	CollectionInterval  time.Duration `json:"collection_interval"`   // 收集间隔
	HealthCheckInterval time.Duration `json:"health_check_interval"` // 健康检查间隔
	DiagnosticInterval  time.Duration `json:"diagnostic_interval"`   // 诊断间隔
	BufferSize          int           `json:"buffer_size"`           // 缓冲区大小
	EnableAlerts        bool          `json:"enable_alerts"`         // 启用告警
	EnableDiagnostics   bool          `json:"enable_diagnostics"`    // 启用诊断
	LogLevel            string        `json:"log_level"`             // 日志级别
}

// NewMonitoringSystem 创建监控系统
func NewMonitoringSystem() *MonitoringSystem {
	return &MonitoringSystem{
		metrics:           make(map[string]*Metric),
		alerts:            make(map[string]*Alert),
		componentHealth:   make(map[string]*ComponentHealth),
		alertRules:        make([]*AlertRule, 0),
		collectors:        make([]MetricCollector, 0),
		alertHandlers:     make([]AlertHandler, 0),
		healthCheckers:    make([]HealthChecker, 0),
		diagnosticsBuffer: make([]*SystemDiagnostics, 0),
		stopCh:            make(chan struct{}),
		config: &MonitoringConfig{
			MetricRetention:     24 * time.Hour,     // 24小时
			AlertRetention:      7 * 24 * time.Hour, // 7天
			CollectionInterval:  30 * time.Second,   // 30秒
			HealthCheckInterval: 10 * time.Second,   // 10秒
			DiagnosticInterval:  60 * time.Second,   // 60秒
			BufferSize:          1000,
			EnableAlerts:        true,
			EnableDiagnostics:   true,
			LogLevel:            "info",
		},
	}
}

// Start 启动监控系统
func (ms *MonitoringSystem) Start() error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	if ms.isRunning {
		return fmt.Errorf("monitoring system already running")
	}

	ms.isRunning = true
	ms.stopCh = make(chan struct{})

	// 立即生成一次初始诊断数据
	go func() {
		time.Sleep(100 * time.Millisecond) // 稍微延迟确保其他组件已启动
		ms.performDiagnostics()
	}()

	// 启动各种循环
	go ms.metricCollectionLoop()
	go ms.healthCheckLoop()
	go ms.alertProcessingLoop()
	if ms.config.EnableDiagnostics {
		go ms.diagnosticsLoop()
	}
	go ms.cleanupLoop()

	fmt.Println("✓ Monitoring system started")
	return nil
}

// Stop 停止监控系统
func (ms *MonitoringSystem) Stop() {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	if !ms.isRunning {
		return
	}

	ms.isRunning = false
	close(ms.stopCh)
	fmt.Println("✓ Monitoring system stopped")
}

// RegisterCollector 注册指标收集器
func (ms *MonitoringSystem) RegisterCollector(collector MetricCollector) {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	ms.collectors = append(ms.collectors, collector)
}

// RegisterAlertHandler 注册告警处理器
func (ms *MonitoringSystem) RegisterAlertHandler(handler AlertHandler) {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	ms.alertHandlers = append(ms.alertHandlers, handler)
}

// RegisterHealthChecker 注册健康检查器
func (ms *MonitoringSystem) RegisterHealthChecker(checker HealthChecker) {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	ms.healthCheckers = append(ms.healthCheckers, checker)
}

// AddAlertRule 添加告警规则
func (ms *MonitoringSystem) AddAlertRule(rule *AlertRule) {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	ms.alertRules = append(ms.alertRules, rule)
}

// metricCollectionLoop 指标收集循环
func (ms *MonitoringSystem) metricCollectionLoop() {
	ticker := time.NewTicker(ms.config.CollectionInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ms.stopCh:
			return
		case <-ticker.C:
			ms.collectMetrics()
		}
	}
}

// collectMetrics 收集指标
func (ms *MonitoringSystem) collectMetrics() {
	ms.mu.Lock()
	collectors := make([]MetricCollector, len(ms.collectors))
	copy(collectors, ms.collectors)
	ms.mu.Unlock()

	for _, collector := range collectors {
		metrics, err := collector.CollectMetrics()
		if err != nil {
			fmt.Printf("Failed to collect metrics from %s: %v\n", collector.GetName(), err)
			continue
		}

		ms.mu.Lock()
		for _, metric := range metrics {
			ms.metrics[metric.Name] = metric
		}
		ms.mu.Unlock()
	}
}

// healthCheckLoop 健康检查循环
func (ms *MonitoringSystem) healthCheckLoop() {
	ticker := time.NewTicker(ms.config.HealthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ms.stopCh:
			return
		case <-ticker.C:
			ms.performHealthChecks()
		}
	}
}

// performHealthChecks 执行健康检查
func (ms *MonitoringSystem) performHealthChecks() {
	ms.mu.Lock()
	checkers := make([]HealthChecker, len(ms.healthCheckers))
	copy(checkers, ms.healthCheckers)
	ms.mu.Unlock()

	for _, checker := range checkers {
		health, err := checker.CheckHealth()
		if err != nil {
			fmt.Printf("Health check failed for %s: %v\n", checker.GetComponentName(), err)
			continue
		}

		ms.mu.Lock()
		ms.componentHealth[health.Name] = health
		ms.mu.Unlock()
	}
}

// alertProcessingLoop 告警处理循环
func (ms *MonitoringSystem) alertProcessingLoop() {
	ticker := time.NewTicker(10 * time.Second) // 每10秒检查告警
	defer ticker.Stop()

	for {
		select {
		case <-ms.stopCh:
			return
		case <-ticker.C:
			if ms.config.EnableAlerts {
				ms.processAlerts()
			}
		}
	}
}

// processAlerts 处理告警
func (ms *MonitoringSystem) processAlerts() {
	ms.mu.Lock()
	rules := make([]*AlertRule, len(ms.alertRules))
	copy(rules, ms.alertRules)
	metrics := make(map[string]*Metric)
	for k, v := range ms.metrics {
		metrics[k] = v
	}
	ms.mu.Unlock()

	for _, rule := range rules {
		if !rule.Enabled {
			continue
		}

		metric, exists := metrics[rule.MetricName]
		if !exists {
			continue
		}

		// 检查冷却时间
		if time.Now().Unix()-rule.LastTriggered < int64(rule.Cooldown.Seconds()) {
			continue
		}

		// 评估告警条件
		if ms.evaluateCondition(metric, rule) {
			alert := &Alert{
				ID:         fmt.Sprintf("%s_%d", rule.ID, time.Now().Unix()),
				MetricName: rule.MetricName,
				Severity:   rule.Severity,
				Message:    rule.Message,
				Details: map[string]interface{}{
					"metric_value": metric.Value,
					"threshold":    rule.Threshold,
					"condition":    rule.Condition,
				},
				Timestamp: time.Now().Unix(),
				IsActive:  true,
				Count:     1,
			}

			ms.mu.Lock()
			ms.alerts[alert.ID] = alert
			rule.LastTriggered = time.Now().Unix()
			ms.mu.Unlock()

			// 处理告警
			ms.handleAlert(alert)
		}
	}
}

// evaluateCondition 评估告警条件
func (ms *MonitoringSystem) evaluateCondition(metric *Metric, rule *AlertRule) bool {
	switch rule.Condition {
	case "greater_than", ">":
		return metric.Value > rule.Threshold
	case "less_than", "<":
		return metric.Value < rule.Threshold
	case "greater_equal", ">=":
		return metric.Value >= rule.Threshold
	case "less_equal", "<=":
		return metric.Value <= rule.Threshold
	case "equal", "==":
		return metric.Value == rule.Threshold
	case "not_equal", "!=":
		return metric.Value != rule.Threshold
	default:
		return false
	}
}

// handleAlert 处理告警
func (ms *MonitoringSystem) handleAlert(alert *Alert) {
	ms.mu.Lock()
	handlers := make([]AlertHandler, len(ms.alertHandlers))
	copy(handlers, ms.alertHandlers)
	ms.mu.Unlock()

	for _, handler := range handlers {
		go func(h AlertHandler) {
			if err := h.HandleAlert(alert); err != nil {
				fmt.Printf("Alert handler %s failed: %v\n", h.GetName(), err)
			}
		}(handler)
	}
}

// diagnosticsLoop 诊断循环
func (ms *MonitoringSystem) diagnosticsLoop() {
	ticker := time.NewTicker(ms.config.DiagnosticInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ms.stopCh:
			return
		case <-ticker.C:
			ms.performDiagnostics()
		}
	}
}

// performDiagnostics 执行系统诊断
func (ms *MonitoringSystem) performDiagnostics() {
	diagnostics := &SystemDiagnostics{
		OverallHealth: ms.calculateOverallHealth(),
		Components:    make(map[string]*ComponentHealth),
		ActiveAlerts:  ms.getActiveAlerts(),
		Metrics:       make(map[string]*Metric),
		Performance:   ms.collectPerformanceMetrics(),
		Network:       ms.collectNetworkDiagnostics(),
		Storage:       ms.collectStorageDiagnostics(),
		Consensus:     ms.collectConsensusDiagnostics(),
		Timestamp:     time.Now().Unix(),
	}

	ms.mu.Lock()
	// 复制组件健康状态
	for k, v := range ms.componentHealth {
		healthCopy := *v
		diagnostics.Components[k] = &healthCopy
	}

	// 复制关键指标
	for k, v := range ms.metrics {
		metricCopy := *v
		diagnostics.Metrics[k] = &metricCopy
	}

	// 添加到缓冲区
	ms.diagnosticsBuffer = append(ms.diagnosticsBuffer, diagnostics)
	if len(ms.diagnosticsBuffer) > ms.config.BufferSize {
		ms.diagnosticsBuffer = ms.diagnosticsBuffer[1:]
	}
	ms.mu.Unlock()
}

// calculateOverallHealth 计算总体健康状态
func (ms *MonitoringSystem) calculateOverallHealth() HealthStatus {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	if len(ms.componentHealth) == 0 {
		return HealthStatusUnknown
	}

	healthyCounts := make(map[HealthStatus]int)
	for _, health := range ms.componentHealth {
		healthyCounts[health.Status]++
	}

	total := len(ms.componentHealth)
	criticalCount := healthyCounts[HealthStatusCritical]
	unhealthyCount := healthyCounts[HealthStatusUnhealthy]
	degradedCount := healthyCounts[HealthStatusDegraded]

	// 如果有严重问题
	if criticalCount > 0 {
		return HealthStatusCritical
	}

	// 如果超过30%组件不健康
	if float64(unhealthyCount)/float64(total) > 0.3 {
		return HealthStatusUnhealthy
	}

	// 如果超过50%组件降级
	if float64(degradedCount)/float64(total) > 0.5 {
		return HealthStatusDegraded
	}

	return HealthStatusHealthy
}

// getActiveAlerts 获取活跃告警
func (ms *MonitoringSystem) getActiveAlerts() []*Alert {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	var activeAlerts []*Alert
	for _, alert := range ms.alerts {
		if alert.IsActive {
			alertCopy := *alert
			activeAlerts = append(activeAlerts, &alertCopy)
		}
	}

	return activeAlerts
}

// collectPerformanceMetrics 收集性能指标
func (ms *MonitoringSystem) collectPerformanceMetrics() *PerformanceMetrics {
	// 简化实现，实际应该从系统收集真实指标
	return &PerformanceMetrics{
		CPU:                 50.0,  // 50% CPU使用率
		Memory:              60.0,  // 60% 内存使用率
		Disk:                30.0,  // 30% 磁盘使用率
		NetworkIO:           100.0, // 100 MB/s 网络IO
		BlockProcessingRate: 0.5,   // 0.5 blocks/s
		TxProcessingRate:    10.0,  // 10 tx/s
		SyncSpeed:           2.0,   // 2 blocks/s
		Latency:             50.0,  // 50ms 延迟
	}
}

// collectNetworkDiagnostics 收集网络诊断
func (ms *MonitoringSystem) collectNetworkDiagnostics() *NetworkDiagnostics {
	// 简化实现
	return &NetworkDiagnostics{
		PeerCount:        5,
		HealthyPeers:     4,
		ConnectionErrors: 1,
		MessageDropRate:  0.01, // 1%
		Bandwidth:        80.0, // 80% 带宽使用
		Latency:          25.0, // 25ms
		CongestionLevel:  0.3,  // 30% 拥塞
	}
}

// collectStorageDiagnostics 收集存储诊断
func (ms *MonitoringSystem) collectStorageDiagnostics() *StorageDiagnostics {
	// 简化实现
	return &StorageDiagnostics{
		DiskUsage:       45.0,              // 45% 磁盘使用
		IOLatency:       5.0,               // 5ms IO延迟
		BlockHeight:     12345,             // 区块高度
		StateSize:       1024 * 1024 * 100, // 100MB 状态大小
		CheckpointCount: 123,
		CompactionRate:  0.6,  // 60% 压缩率
		ErrorRate:       0.01, // 1% 错误率
	}
}

// collectConsensusDiagnostics 收集共识诊断
func (ms *MonitoringSystem) collectConsensusDiagnostics() *ConsensusDiagnostics {
	// 简化实现
	return &ConsensusDiagnostics{
		ActiveValidators:    21,
		ValidatorRatio:      0.85, // 85% 验证者活跃
		SafetyMode:          false,
		ConsensusLatency:    100.0, // 100ms 共识延迟
		BlockProductionRate: 0.5,   // 0.5 blocks/s
		MissedBlocks:        2,
		EmptyBlocks:         5,
	}
}

// cleanupLoop 清理循环
func (ms *MonitoringSystem) cleanupLoop() {
	ticker := time.NewTicker(1 * time.Hour) // 每小时清理一次
	defer ticker.Stop()

	for {
		select {
		case <-ms.stopCh:
			return
		case <-ticker.C:
			ms.cleanup()
		}
	}
}

// cleanup 清理过期数据
func (ms *MonitoringSystem) cleanup() {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	now := time.Now().Unix()

	// 清理过期指标
	for name, metric := range ms.metrics {
		if now-metric.Timestamp > int64(ms.config.MetricRetention.Seconds()) {
			delete(ms.metrics, name)
		}
	}

	// 清理过期告警
	for id, alert := range ms.alerts {
		if now-alert.Timestamp > int64(ms.config.AlertRetention.Seconds()) {
			delete(ms.alerts, id)
		}
	}
}

// GetSystemDiagnostics 获取系统诊断信息
func (ms *MonitoringSystem) GetSystemDiagnostics() *SystemDiagnostics {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	if len(ms.diagnosticsBuffer) == 0 {
		// 如果缓冲区为空，立即生成一个基本诊断
		return ms.generateFallbackDiagnostics()
	}

	// 返回最新的诊断信息副本
	latest := ms.diagnosticsBuffer[len(ms.diagnosticsBuffer)-1]
	diagnosticsCopy := *latest
	return &diagnosticsCopy
}

// generateFallbackDiagnostics 生成备用诊断信息
func (ms *MonitoringSystem) generateFallbackDiagnostics() *SystemDiagnostics {
	diagnostics := &SystemDiagnostics{
		OverallHealth: HealthStatusHealthy,
		Components:    make(map[string]*ComponentHealth),
		ActiveAlerts:  []*Alert{},
		Metrics:       make(map[string]*Metric),
		Performance:   ms.collectPerformanceMetrics(),
		Network:       ms.collectNetworkDiagnostics(),
		Storage:       ms.collectStorageDiagnostics(),
		Consensus:     ms.collectConsensusDiagnostics(),
		Timestamp:     time.Now().Unix(),
	}

	// 复制组件健康状态
	for k, v := range ms.componentHealth {
		healthCopy := *v
		diagnostics.Components[k] = &healthCopy
	}

	// 复制关键指标
	for k, v := range ms.metrics {
		metricCopy := *v
		diagnostics.Metrics[k] = &metricCopy
	}

	return diagnostics
}

// GetMetrics 获取所有指标
func (ms *MonitoringSystem) GetMetrics() map[string]*Metric {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	result := make(map[string]*Metric)
	for k, v := range ms.metrics {
		metricCopy := *v
		result[k] = &metricCopy
	}

	return result
}

// GetAlerts 获取所有告警
func (ms *MonitoringSystem) GetAlerts() []*Alert {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	var alerts []*Alert
	for _, alert := range ms.alerts {
		alertCopy := *alert
		alerts = append(alerts, &alertCopy)
	}

	return alerts
}

// GetComponentHealth 获取组件健康状态
func (ms *MonitoringSystem) GetComponentHealth() map[string]*ComponentHealth {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	result := make(map[string]*ComponentHealth)
	for k, v := range ms.componentHealth {
		healthCopy := *v
		result[k] = &healthCopy
	}

	return result
}

// ExportMetrics 导出指标数据
func (ms *MonitoringSystem) ExportMetrics(format string) ([]byte, error) {
	metrics := ms.GetMetrics()

	switch format {
	case "json":
		return json.MarshalIndent(metrics, "", "  ")
	default:
		return nil, fmt.Errorf("unsupported format: %s", format)
	}
}

// SetConfig 设置监控配置
func (ms *MonitoringSystem) SetConfig(config *MonitoringConfig) {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	ms.config = config
}

// GetConfig 获取监控配置
func (ms *MonitoringSystem) GetConfig() *MonitoringConfig {
	ms.mu.RLock()
	defer ms.mu.RUnlock()
	configCopy := *ms.config
	return &configCopy
}
