package node

import (
	"fmt"
	"sync"
	"time"
)

// HealthStatus 健康状态枚举
type HealthStatus int

const (
	HealthStatusHealthy HealthStatus = iota
	HealthStatusWarning 
	HealthStatusCritical
	HealthStatusUnknown
)

func (h HealthStatus) String() string {
	switch h {
	case HealthStatusHealthy:
		return "healthy"
	case HealthStatusWarning:
		return "warning"
	case HealthStatusCritical:
		return "critical"
	case HealthStatusUnknown:
		return "unknown"
	default:
		return "unknown"
	}
}

// ComponentHealth 组件健康状态
type ComponentHealth struct {
	Name           string                 `json:"name"`
	Status         HealthStatus           `json:"status"`
	Message        string                 `json:"message"`
	LastCheck      time.Time              `json:"last_check"`
	CheckDuration  time.Duration          `json:"check_duration"`
	ErrorCount     int                    `json:"error_count"`
	Metrics        map[string]interface{} `json:"metrics"`
}

// SystemHealth 系统整体健康状态
type SystemHealth struct {
	OverallStatus  HealthStatus                   `json:"overall_status"`
	Components     map[string]*ComponentHealth    `json:"components"`
	LastUpdate     time.Time                      `json:"last_update"`
	Uptime         time.Duration                  `json:"uptime"`
	StartTime      time.Time                      `json:"start_time"`
	SystemMetrics  map[string]interface{}         `json:"system_metrics"`
}

// HealthChecker 健康检查器接口
type HealthChecker interface {
	CheckHealth() *ComponentHealth
	GetName() string
}

// HealthMonitor 健康监控器
type HealthMonitor struct {
	mu                sync.RWMutex
	checkers          map[string]HealthChecker
	health            *SystemHealth
	checkInterval     time.Duration
	criticalThreshold int           // 连续失败多少次变为critical
	warningThreshold  int           // 连续失败多少次变为warning
	isRunning         bool
	stopCh            chan struct{}
	
	// 配置参数
	config HealthConfig
}

// HealthConfig 健康检查配置
type HealthConfig struct {
	CheckInterval      time.Duration `json:"check_interval"`      // 检查间隔
	CriticalThreshold  int           `json:"critical_threshold"`  // 严重阈值
	WarningThreshold   int           `json:"warning_threshold"`   // 警告阈值
	ComponentTimeout   time.Duration `json:"component_timeout"`   // 组件检查超时
	EnableSystemStats  bool          `json:"enable_system_stats"` // 启用系统统计
	AlertWebhook       string        `json:"alert_webhook"`       // 告警webhook
}

// DefaultHealthConfig 默认健康检查配置
func DefaultHealthConfig() HealthConfig {
	return HealthConfig{
		CheckInterval:     time.Second * 30,
		CriticalThreshold: 3,
		WarningThreshold:  1,
		ComponentTimeout:  time.Second * 5,
		EnableSystemStats: true,
		AlertWebhook:      "",
	}
}

// NewHealthMonitor 创建健康监控器
func NewHealthMonitor(config HealthConfig) *HealthMonitor {
	return &HealthMonitor{
		checkers:          make(map[string]HealthChecker),
		checkInterval:     config.CheckInterval,
		criticalThreshold: config.CriticalThreshold,
		warningThreshold:  config.WarningThreshold,
		stopCh:            make(chan struct{}),
		config:            config,
		health: &SystemHealth{
			Components:    make(map[string]*ComponentHealth),
			StartTime:     time.Now(),
			SystemMetrics: make(map[string]interface{}),
		},
	}
}

// RegisterChecker 注册健康检查器
func (hm *HealthMonitor) RegisterChecker(checker HealthChecker) {
	hm.mu.Lock()
	defer hm.mu.Unlock()
	
	name := checker.GetName()
	hm.checkers[name] = checker
	
	// 初始化组件健康状态
	hm.health.Components[name] = &ComponentHealth{
		Name:      name,
		Status:    HealthStatusUnknown,
		Message:   "Not checked yet",
		LastCheck: time.Time{},
		Metrics:   make(map[string]interface{}),
	}
}

// UnregisterChecker 取消注册健康检查器
func (hm *HealthMonitor) UnregisterChecker(name string) {
	hm.mu.Lock()
	defer hm.mu.Unlock()
	
	delete(hm.checkers, name)
	delete(hm.health.Components, name)
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
	
	// 立即执行一次检查
	go hm.performHealthCheck()
	
	// 启动定期检查
	go hm.healthCheckLoop()
	
	// 如果启用系统统计，启动系统指标收集
	if hm.config.EnableSystemStats {
		go hm.systemMetricsLoop()
	}
	
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

// healthCheckLoop 健康检查循环
func (hm *HealthMonitor) healthCheckLoop() {
	ticker := time.NewTicker(hm.checkInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-hm.stopCh:
			return
		case <-ticker.C:
			hm.performHealthCheck()
		}
	}
}

// performHealthCheck 执行健康检查
func (hm *HealthMonitor) performHealthCheck() {
	hm.mu.RLock()
	checkers := make(map[string]HealthChecker)
	for name, checker := range hm.checkers {
		checkers[name] = checker
	}
	hm.mu.RUnlock()
	
	// 并发检查所有组件
	results := make(chan *ComponentHealth, len(checkers))
	
	for name, checker := range checkers {
		go func(n string, c HealthChecker) {
			health := hm.checkComponent(c)
			results <- health
		}(name, checker)
	}
	
	// 收集结果
	healthyCount := 0
	warningCount := 0
	criticalCount := 0
	
	for i := 0; i < len(checkers); i++ {
		health := <-results
		
		hm.mu.Lock()
		hm.health.Components[health.Name] = health
		hm.mu.Unlock()
		
		switch health.Status {
		case HealthStatusHealthy:
			healthyCount++
		case HealthStatusWarning:
			warningCount++
		case HealthStatusCritical:
			criticalCount++
		}
	}
	
	// 更新整体状态
	hm.mu.Lock()
	hm.health.LastUpdate = time.Now()
	hm.health.Uptime = time.Since(hm.health.StartTime)
	
	// 计算整体健康状态
	if criticalCount > 0 {
		hm.health.OverallStatus = HealthStatusCritical
	} else if warningCount > 0 {
		hm.health.OverallStatus = HealthStatusWarning
	} else if healthyCount > 0 {
		hm.health.OverallStatus = HealthStatusHealthy
	} else {
		hm.health.OverallStatus = HealthStatusUnknown
	}
	hm.mu.Unlock()
	
	// 如果配置了告警webhook，检查是否需要发送告警
	if hm.config.AlertWebhook != "" && (criticalCount > 0 || warningCount > 0) {
		go hm.sendAlert()
	}
}

// checkComponent 检查单个组件
func (hm *HealthMonitor) checkComponent(checker HealthChecker) *ComponentHealth {
	start := time.Now()
	
	// 使用超时控制
	done := make(chan *ComponentHealth, 1)
	go func() {
		health := checker.CheckHealth()
		done <- health
	}()
	
	select {
	case health := <-done:
		health.CheckDuration = time.Since(start)
		health.LastCheck = time.Now()
		
		// 根据检查结果调整错误计数
		hm.mu.RLock()
		if existing, exists := hm.health.Components[health.Name]; exists {
			if health.Status == HealthStatusHealthy {
				health.ErrorCount = 0 // 重置错误计数
			} else {
				health.ErrorCount = existing.ErrorCount + 1
			}
			
			// 根据错误计数调整状态
			if health.ErrorCount >= hm.criticalThreshold {
				health.Status = HealthStatusCritical
			} else if health.ErrorCount >= hm.warningThreshold {
				health.Status = HealthStatusWarning
			}
		}
		hm.mu.RUnlock()
		
		return health
		
	case <-time.After(hm.config.ComponentTimeout):
		// 超时处理
		return &ComponentHealth{
			Name:          checker.GetName(),
			Status:        HealthStatusCritical,
			Message:       "Health check timeout",
			LastCheck:     time.Now(),
			CheckDuration: hm.config.ComponentTimeout,
			ErrorCount:    1,
			Metrics:       make(map[string]interface{}),
		}
	}
}

// systemMetricsLoop 系统指标收集循环
func (hm *HealthMonitor) systemMetricsLoop() {
	ticker := time.NewTicker(time.Minute) // 每分钟收集一次系统指标
	defer ticker.Stop()
	
	for {
		select {
		case <-hm.stopCh:
			return
		case <-ticker.C:
			hm.collectSystemMetrics()
		}
	}
}

// collectSystemMetrics 收集系统指标
func (hm *HealthMonitor) collectSystemMetrics() {
	metrics := make(map[string]interface{})
	
	// 这里可以收集各种系统指标
	// 简化实现，添加一些基本指标
	metrics["timestamp"] = time.Now().Unix()
	metrics["uptime_seconds"] = time.Since(hm.health.StartTime).Seconds()
	metrics["goroutines"] = "runtime.NumGoroutine() would go here"
	metrics["memory_alloc"] = "memory stats would go here"
	
	hm.mu.Lock()
	hm.health.SystemMetrics = metrics
	hm.mu.Unlock()
}

// sendAlert 发送告警
func (hm *HealthMonitor) sendAlert() {
	// 简化实现：打印告警信息
	hm.mu.RLock()
	defer hm.mu.RUnlock()
	
	fmt.Printf("HEALTH ALERT: System status is %s\n", hm.health.OverallStatus.String())
	for name, component := range hm.health.Components {
		if component.Status != HealthStatusHealthy {
			fmt.Printf("  - %s: %s (%s)\n", name, component.Status.String(), component.Message)
		}
	}
}

// GetHealth 获取当前健康状态
func (hm *HealthMonitor) GetHealth() *SystemHealth {
	hm.mu.RLock()
	defer hm.mu.RUnlock()
	
	// 创建副本以避免竞态条件
	health := &SystemHealth{
		OverallStatus: hm.health.OverallStatus,
		Components:    make(map[string]*ComponentHealth),
		LastUpdate:    hm.health.LastUpdate,
		Uptime:        hm.health.Uptime,
		StartTime:     hm.health.StartTime,
		SystemMetrics: make(map[string]interface{}),
	}
	
	// 复制组件健康状态
	for name, component := range hm.health.Components {
		componentCopy := *component
		health.Components[name] = &componentCopy
	}
	
	// 复制系统指标
	for k, v := range hm.health.SystemMetrics {
		health.SystemMetrics[k] = v
	}
	
	return health
}

// GetComponentHealth 获取特定组件的健康状态
func (hm *HealthMonitor) GetComponentHealth(name string) *ComponentHealth {
	hm.mu.RLock()
	defer hm.mu.RUnlock()
	
	if component, exists := hm.health.Components[name]; exists {
		componentCopy := *component
		return &componentCopy
	}
	
	return nil
}

// IsHealthy 检查系统是否健康
func (hm *HealthMonitor) IsHealthy() bool {
	hm.mu.RLock()
	defer hm.mu.RUnlock()
	
	return hm.health.OverallStatus == HealthStatusHealthy
}

// GetHealthSummary 获取健康状态摘要
func (hm *HealthMonitor) GetHealthSummary() map[string]interface{} {
	hm.mu.RLock()
	defer hm.mu.RUnlock()
	
	healthyCount := 0
	warningCount := 0
	criticalCount := 0
	unknownCount := 0
	
	for _, component := range hm.health.Components {
		switch component.Status {
		case HealthStatusHealthy:
			healthyCount++
		case HealthStatusWarning:
			warningCount++
		case HealthStatusCritical:
			criticalCount++
		case HealthStatusUnknown:
			unknownCount++
		}
	}
	
	return map[string]interface{}{
		"overall_status":   hm.health.OverallStatus.String(),
		"total_components": len(hm.health.Components),
		"healthy_count":    healthyCount,
		"warning_count":    warningCount,
		"critical_count":   criticalCount,
		"unknown_count":    unknownCount,
		"last_update":      hm.health.LastUpdate.Unix(),
		"uptime_seconds":   hm.health.Uptime.Seconds(),
	}
}

// 具体的健康检查器实现

// BlockchainHealthChecker 区块链健康检查器
type BlockchainHealthChecker struct {
	manager BlockchainManager
}

type BlockchainManager interface {
	GetCurrentBlock() interface{}
	GetStats() map[string]interface{}
}

func NewBlockchainHealthChecker(manager BlockchainManager) *BlockchainHealthChecker {
	return &BlockchainHealthChecker{manager: manager}
}

func (bhc *BlockchainHealthChecker) GetName() string {
	return "blockchain"
}

func (bhc *BlockchainHealthChecker) CheckHealth() *ComponentHealth {
	health := &ComponentHealth{
		Name:    bhc.GetName(),
		Status:  HealthStatusHealthy,
		Message: "Blockchain is operating normally",
		Metrics: make(map[string]interface{}),
	}
	
	// 检查区块链管理器是否可用
	if bhc.manager == nil {
		health.Status = HealthStatusCritical
		health.Message = "Blockchain manager is not available"
		return health
	}
	
	// 获取当前区块
	currentBlock := bhc.manager.GetCurrentBlock()
	if currentBlock == nil {
		health.Status = HealthStatusWarning
		health.Message = "No current block available"
	}
	
	// 获取统计信息
	if stats := bhc.manager.GetStats(); stats != nil {
		health.Metrics = stats
	}
	
	return health
}

// NetworkHealthChecker 网络健康检查器
type NetworkHealthChecker struct {
	network NetworkManager
}

type NetworkManager interface {
	GetPeerCount() int
	GetHealthyPeerCount() int
	GetNetworkStats() map[string]interface{}
	IsRunning() bool
}

func NewNetworkHealthChecker(network NetworkManager) *NetworkHealthChecker {
	return &NetworkHealthChecker{network: network}
}

func (nhc *NetworkHealthChecker) GetName() string {
	return "network"
}

func (nhc *NetworkHealthChecker) CheckHealth() *ComponentHealth {
	health := &ComponentHealth{
		Name:    nhc.GetName(),
		Status:  HealthStatusHealthy,
		Message: "Network is operating normally",
		Metrics: make(map[string]interface{}),
	}
	
	if nhc.network == nil {
		health.Status = HealthStatusCritical
		health.Message = "Network manager is not available"
		return health
	}
	
	if !nhc.network.IsRunning() {
		health.Status = HealthStatusCritical
		health.Message = "Network is not running"
		return health
	}
	
	peerCount := nhc.network.GetPeerCount()
	healthyPeerCount := nhc.network.GetHealthyPeerCount()
	
	health.Metrics["peer_count"] = peerCount
	health.Metrics["healthy_peer_count"] = healthyPeerCount
	
	if peerCount == 0 {
		health.Status = HealthStatusWarning
		health.Message = "No peers connected"
	} else if healthyPeerCount < peerCount/2 {
		health.Status = HealthStatusWarning
		health.Message = "Less than half of peers are healthy"
	}
	
	// 获取网络统计信息
	if stats := nhc.network.GetNetworkStats(); stats != nil {
		for k, v := range stats {
			health.Metrics[k] = v
		}
	}
	
	return health
}

// TransactionPoolHealthChecker 交易池健康检查器
type TransactionPoolHealthChecker struct {
	txPool TransactionPoolManager
}

type TransactionPoolManager interface {
	GetTransactionCount() int
	GetStats() map[string]interface{}
}

func NewTransactionPoolHealthChecker(txPool TransactionPoolManager) *TransactionPoolHealthChecker {
	return &TransactionPoolHealthChecker{txPool: txPool}
}

func (tphc *TransactionPoolHealthChecker) GetName() string {
	return "transaction_pool"
}

func (tphc *TransactionPoolHealthChecker) CheckHealth() *ComponentHealth {
	health := &ComponentHealth{
		Name:    tphc.GetName(),
		Status:  HealthStatusHealthy,
		Message: "Transaction pool is operating normally",
		Metrics: make(map[string]interface{}),
	}
	
	if tphc.txPool == nil {
		health.Status = HealthStatusCritical
		health.Message = "Transaction pool is not available"
		return health
	}
	
	txCount := tphc.txPool.GetTransactionCount()
	health.Metrics["transaction_count"] = txCount
	
	// 获取交易池统计信息
	if stats := tphc.txPool.GetStats(); stats != nil {
		for k, v := range stats {
			health.Metrics[k] = v
		}
	}
	
	return health
}