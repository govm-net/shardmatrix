package consensus

import (
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/lengzhao/shardmatrix/pkg/crypto"
	"github.com/lengzhao/shardmatrix/pkg/storage"
	"github.com/lengzhao/shardmatrix/pkg/types"
)

// DPoSConsensus DPoS共识引擎
type DPoSConsensus struct {
	mu             sync.RWMutex
	storage        storage.Storage
	validatorSet   *types.ValidatorSet
	currentRound   uint64
	timeController *TimeController
	isActive       bool
	keyPair        *crypto.KeyPair
	// 新增：节点活跃度管理
	activeThreshold float64                         // 活跃度阈值 (0.6 = 60%)
	nodeHealthMap   map[types.Address]*NodeHealthInfo // 节点健康信息
	safetyMode      bool                            // 安全模式标志
	lastHealthCheck int64                           // 最后健康检查时间
}

// NodeHealthInfo 节点健康信息
type NodeHealthInfo struct {
	LastHeartbeat   int64     // 最后心跳时间
	LastBlockTime   int64     // 最后出块时间
	BlocksProduced  uint64    // 生产区块数
	MissedBlocks    uint64    // 错过区块数
	NetworkLatency  int64     // 网络延迟(ms)
	IsResponsive    bool      // 是否响应
	ActiveScore     float64   // 活跃度评分(0-1)
}

// NewDPoSConsensus 创建新的DPoS共识引擎
func NewDPoSConsensus(
	storage storage.Storage,
	timeController *TimeController,
	keyPair *crypto.KeyPair,
) *DPoSConsensus {
	return &DPoSConsensus{
		storage:        storage,
		timeController: timeController,
		keyPair:        keyPair,
		validatorSet: &types.ValidatorSet{
			Validators: make([]types.Validator, 0),
			Round:      0,
		},
		// 初始化活跃度管理
		activeThreshold: 0.6, // 60%活跃度要求
		nodeHealthMap:   make(map[types.Address]*NodeHealthInfo),
		safetyMode:      false,
		lastHealthCheck: time.Now().Unix(),
	}
}

// Start 启动DPoS共识
func (dpos *DPoSConsensus) Start() error {
	dpos.mu.Lock()
	defer dpos.mu.Unlock()
	
	if dpos.isActive {
		return fmt.Errorf("DPoS consensus already active")
	}
	
	// 加载验证者集合
	err := dpos.loadValidatorSet()
	if err != nil {
		return fmt.Errorf("failed to load validator set: %w", err)
	}
	
	dpos.isActive = true
	
	// 启动健康检查循环
	go dpos.healthCheckLoop()
	
	return nil
}

// Stop 停止DPoS共识
func (dpos *DPoSConsensus) Stop() {
	dpos.mu.Lock()
	defer dpos.mu.Unlock()
	dpos.isActive = false
	
	// 清理节点健康信息
	dpos.nodeHealthMap = make(map[types.Address]*NodeHealthInfo)
	dpos.safetyMode = false
}

// loadValidatorSet 加载验证者集合
func (dpos *DPoSConsensus) loadValidatorSet() error {
	validators, err := dpos.storage.GetAllValidators()
	if err != nil {
		return fmt.Errorf("failed to get validators from storage: %w", err)
	}
	
	// 如果没有验证者，创建默认验证者（使用当前节点）
	if len(validators) == 0 {
		defaultValidator := &types.Validator{
			Address:      dpos.keyPair.Address,
			PublicKey:    dpos.keyPair.PublicKey,
			StakeAmount:  1000000, // 默认质押金额
			DelegatedAmt: 0,
			VotePower:    1000000,
			IsActive:     true,
			SlashCount:   0,
		}
		
		err = dpos.storage.SaveValidator(defaultValidator)
		if err != nil {
			return fmt.Errorf("failed to save default validator: %w", err)
		}
		
		validators = []*types.Validator{defaultValidator}
	}
	
	// 转换为ValidatorSet
	validatorList := make([]types.Validator, len(validators))
	for i, v := range validators {
		validatorList[i] = *v
	}
	
	dpos.validatorSet = &types.ValidatorSet{
		Validators: validatorList,
		Round:      0,
	}
	
	// 排序验证者（按投票权重降序）
	dpos.sortValidators()
	
	return nil
}

// sortValidators 排序验证者
func (dpos *DPoSConsensus) sortValidators() {
	sort.Slice(dpos.validatorSet.Validators, func(i, j int) bool {
		return dpos.validatorSet.Validators[i].VotePower > dpos.validatorSet.Validators[j].VotePower
	})
}

// GetCurrentValidator 获取当前应该出块的验证者
func (dpos *DPoSConsensus) GetCurrentValidator(blockHeight uint64) (*types.Validator, error) {
	dpos.mu.RLock()
	defer dpos.mu.RUnlock()
	
	activeValidators := dpos.getActiveValidators()
	if len(activeValidators) == 0 {
		return nil, fmt.Errorf("no active validators")
	}
	
	// 计算当前轮次的验证者索引
	index := blockHeight % uint64(len(activeValidators))
	return &activeValidators[index], nil
}

// getActiveValidators 获取活跃的验证者列表
func (dpos *DPoSConsensus) getActiveValidators() []types.Validator {
	var activeValidators []types.Validator
	for _, v := range dpos.validatorSet.Validators {
		if v.IsActive {
			activeValidators = append(activeValidators, v)
		}
	}
	return activeValidators
}

// IsValidator 检查地址是否为验证者
func (dpos *DPoSConsensus) IsValidator(address types.Address) bool {
	dpos.mu.RLock()
	defer dpos.mu.RUnlock()
	
	validator := dpos.validatorSet.GetValidator(address)
	return validator != nil && validator.IsActive
}

// IsCurrentValidator 检查地址是否为当前轮次的验证者
func (dpos *DPoSConsensus) IsCurrentValidator(address types.Address, blockHeight uint64) bool {
	currentValidator, err := dpos.GetCurrentValidator(blockHeight)
	if err != nil {
		return false
	}
	
	return currentValidator.Address == address
}

// ValidateBlock 验证区块
func (dpos *DPoSConsensus) ValidateBlock(block *types.Block) error {
	dpos.mu.RLock()
	defer dpos.mu.RUnlock()
	
	// 验证区块生产者是否为合法验证者
	validator := dpos.validatorSet.GetValidator(block.Header.Validator)
	if validator == nil {
		return fmt.Errorf("block producer is not a validator: %s", block.Header.Validator.String())
	}
	
	if !validator.IsActive {
		return fmt.Errorf("block producer is not active: %s", block.Header.Validator.String())
	}
	
	// 验证是否为当前轮次的验证者
	if !dpos.IsCurrentValidator(block.Header.Validator, block.Header.Number) {
		return fmt.Errorf("block producer is not current validator for height %d", block.Header.Number)
	}
	
	// 验证区块签名
	if !crypto.VerifyBlock(&block.Header, validator.PublicKey) {
		return fmt.Errorf("invalid block signature")
	}
	
	// 验证时间戳
	if dpos.timeController != nil {
		validator := NewStrictTimeValidator(dpos.timeController)
		if err := validator.ValidateBlock(block); err != nil {
			return fmt.Errorf("block time validation failed: %w", err)
		}
	}
	
	return nil
}

// AddValidator 添加验证者
func (dpos *DPoSConsensus) AddValidator(validator *types.Validator) error {
	dpos.mu.Lock()
	defer dpos.mu.Unlock()
	
	// 检查验证者是否已存在
	if dpos.validatorSet.GetValidator(validator.Address) != nil {
		return fmt.Errorf("validator already exists: %s", validator.Address.String())
	}
	
	// 验证最小质押金额
	if validator.StakeAmount < types.MinStakeAmount {
		return fmt.Errorf("insufficient stake amount: %d < %d", validator.StakeAmount, types.MinStakeAmount)
	}
	
	// 计算投票权重
	validator.VotePower = validator.StakeAmount + validator.DelegatedAmt
	validator.IsActive = true
	
	// 添加到验证者集合
	dpos.validatorSet.Validators = append(dpos.validatorSet.Validators, *validator)
	
	// 重新排序
	dpos.sortValidators()
	
	// 限制验证者数量为21个
	if len(dpos.validatorSet.Validators) > types.ValidatorCount {
		// 移除投票权重最小的验证者
		removed := dpos.validatorSet.Validators[types.ValidatorCount:]
		dpos.validatorSet.Validators = dpos.validatorSet.Validators[:types.ValidatorCount]
		
		// 将被移除的验证者设为非活跃状态
		for _, v := range removed {
			v.IsActive = false
			dpos.storage.SaveValidator(&v)
		}
	}
	
	// 保存验证者到存储
	err := dpos.storage.SaveValidator(validator)
	if err != nil {
		return fmt.Errorf("failed to save validator: %w", err)
	}
	
	return nil
}

// RemoveValidator 移除验证者
func (dpos *DPoSConsensus) RemoveValidator(address types.Address) error {
	dpos.mu.Lock()
	defer dpos.mu.Unlock()
	
	// 查找验证者索引
	index := -1
	for i, v := range dpos.validatorSet.Validators {
		if v.Address == address {
			index = i
			break
		}
	}
	
	if index == -1 {
		return fmt.Errorf("validator not found: %s", address.String())
	}
	
	// 设置为非活跃状态
	dpos.validatorSet.Validators[index].IsActive = false
	
	// 保存更新到存储
	err := dpos.storage.SaveValidator(&dpos.validatorSet.Validators[index])
	if err != nil {
		return fmt.Errorf("failed to update validator: %w", err)
	}
	
	return nil
}

// Delegate 委托质押
func (dpos *DPoSConsensus) Delegate(delegator types.Address, validator types.Address, amount uint64) error {
	dpos.mu.Lock()
	defer dpos.mu.Unlock()
	
	// 获取验证者
	v := dpos.validatorSet.GetValidator(validator)
	if v == nil {
		return fmt.Errorf("validator not found: %s", validator.String())
	}
	
	// 更新委托金额和投票权重
	v.DelegatedAmt += amount
	v.VotePower = v.StakeAmount + v.DelegatedAmt
	
	// 重新排序
	dpos.sortValidators()
	
	// 保存更新
	err := dpos.storage.SaveValidator(v)
	if err != nil {
		return fmt.Errorf("failed to update validator: %w", err)
	}
	
	return nil
}

// Undelegate 取消委托
func (dpos *DPoSConsensus) Undelegate(delegator types.Address, validator types.Address, amount uint64) error {
	dpos.mu.Lock()
	defer dpos.mu.Unlock()
	
	// 获取验证者
	v := dpos.validatorSet.GetValidator(validator)
	if v == nil {
		return fmt.Errorf("validator not found: %s", validator.String())
	}
	
	// 检查委托金额是否足够
	if v.DelegatedAmt < amount {
		return fmt.Errorf("insufficient delegated amount: %d < %d", v.DelegatedAmt, amount)
	}
	
	// 更新委托金额和投票权重
	v.DelegatedAmt -= amount
	v.VotePower = v.StakeAmount + v.DelegatedAmt
	
	// 重新排序
	dpos.sortValidators()
	
	// 保存更新
	err := dpos.storage.SaveValidator(v)
	if err != nil {
		return fmt.Errorf("failed to update validator: %w", err)
	}
	
	return nil
}

// SlashValidator 惩罚验证者
func (dpos *DPoSConsensus) SlashValidator(address types.Address, reason string) error {
	dpos.mu.Lock()
	defer dpos.mu.Unlock()
	
	// 获取验证者
	v := dpos.validatorSet.GetValidator(address)
	if v == nil {
		return fmt.Errorf("validator not found: %s", address.String())
	}
	
	// 增加惩罚计数
	v.SlashCount++
	
	// 根据惩罚次数决定处理方式
	if v.SlashCount >= 3 {
		// 多次违规，设为非活跃状态
		v.IsActive = false
	} else {
		// 减少部分质押作为惩罚
		slashAmount := v.StakeAmount / 10 // 惩罚10%
		if v.StakeAmount > slashAmount {
			v.StakeAmount -= slashAmount
			v.VotePower = v.StakeAmount + v.DelegatedAmt
		}
	}
	
	// 重新排序
	dpos.sortValidators()
	
	// 保存更新
	err := dpos.storage.SaveValidator(v)
	if err != nil {
		return fmt.Errorf("failed to update validator: %w", err)
	}
	
	return nil
}

// GetValidatorSet 获取验证者集合
func (dpos *DPoSConsensus) GetValidatorSet() *types.ValidatorSet {
	dpos.mu.RLock()
	defer dpos.mu.RUnlock()
	
	// 返回副本以避免并发修改
	validatorsCopy := make([]types.Validator, len(dpos.validatorSet.Validators))
	copy(validatorsCopy, dpos.validatorSet.Validators)
	
	return &types.ValidatorSet{
		Validators: validatorsCopy,
		Round:      dpos.validatorSet.Round,
	}
}

// GetValidatorByIndex 根据索引获取验证者
func (dpos *DPoSConsensus) GetValidatorByIndex(index int) *types.Validator {
	dpos.mu.RLock()
	defer dpos.mu.RUnlock()
	
	activeValidators := dpos.getActiveValidators()
	if index < 0 || index >= len(activeValidators) {
		return nil
	}
	
	return &activeValidators[index]
}

// GetActiveValidatorCount 获取活跃验证者数量
func (dpos *DPoSConsensus) GetActiveValidatorCount() int {
	dpos.mu.RLock()
	defer dpos.mu.RUnlock()
	
	return len(dpos.getActiveValidators())
}

// GetTotalVotePower 获取总投票权重
func (dpos *DPoSConsensus) GetTotalVotePower() uint64 {
	dpos.mu.RLock()
	defer dpos.mu.RUnlock()
	
	return dpos.validatorSet.GetTotalVotePower()
}

// GetConsensusInfo 获取共识信息
func (dpos *DPoSConsensus) GetConsensusInfo() map[string]interface{} {
	dpos.mu.RLock()
	defer dpos.mu.RUnlock()
	
	activeValidators := dpos.getActiveValidators()
	
	return map[string]interface{}{
		"consensus_type":        "DPoS",
		"is_active":             dpos.isActive,
		"total_validators":      len(dpos.validatorSet.Validators),
		"active_validators":     len(activeValidators),
		"current_round":         dpos.validatorSet.Round,
		"total_vote_power":      dpos.GetTotalVotePower(),
		"validator_addresses":   dpos.getValidatorAddresses(activeValidators),
	}
}

// getValidatorAddresses 获取验证者地址列表
func (dpos *DPoSConsensus) getValidatorAddresses(validators []types.Validator) []string {
	addresses := make([]string, len(validators))
	for i, v := range validators {
		addresses[i] = v.Address.String()
	}
	return addresses
}

// UpdateRound 更新轮次
func (dpos *DPoSConsensus) UpdateRound(round uint64) {
	dpos.mu.Lock()
	defer dpos.mu.Unlock()
	dpos.validatorSet.Round = round
}

// IsActive 检查共识是否活跃
func (dpos *DPoSConsensus) IsActive() bool {
	dpos.mu.RLock()
	defer dpos.mu.RUnlock()
	return dpos.isActive
}

// =============== 节点活跃度管理方法 ===============

// UpdateNodeHeartbeat 更新节点心跳
func (dpos *DPoSConsensus) UpdateNodeHeartbeat(address types.Address, latency int64) {
	dpos.mu.Lock()
	defer dpos.mu.Unlock()

	health, exists := dpos.nodeHealthMap[address]
	if !exists {
		health = &NodeHealthInfo{
			ActiveScore: 0.5, // 初始评分
		}
		dpos.nodeHealthMap[address] = health
	}

	health.LastHeartbeat = time.Now().Unix()
	health.NetworkLatency = latency
	health.IsResponsive = true

	// 更新活跃度评分
	dpos.updateNodeActiveScore(health)
}

// UpdateNodeBlockProduction 更新节点出块信息
func (dpos *DPoSConsensus) UpdateNodeBlockProduction(address types.Address, blockTime int64, produced bool) {
	dpos.mu.Lock()
	defer dpos.mu.Unlock()

	health, exists := dpos.nodeHealthMap[address]
	if !exists {
		health = &NodeHealthInfo{
			ActiveScore: 0.5,
		}
		dpos.nodeHealthMap[address] = health
	}

	health.LastBlockTime = blockTime
	if produced {
		health.BlocksProduced++
	} else {
		health.MissedBlocks++
	}

	// 更新活跃度评分
	dpos.updateNodeActiveScore(health)
}

// updateNodeActiveScore 更新节点活跃度评分
func (dpos *DPoSConsensus) updateNodeActiveScore(health *NodeHealthInfo) {
	now := time.Now().Unix()
	
	// 心跳评分 (40%权重)
	heartbeatScore := 1.0
	if now-health.LastHeartbeat > 30 { // 30秒超时
		heartbeatScore = 0.0
	} else if now-health.LastHeartbeat > 15 {
		heartbeatScore = 0.5
	}

	// 出块评分 (35%权重)
	blockScore := 1.0
	totalBlocks := health.BlocksProduced + health.MissedBlocks
	if totalBlocks > 0 {
		blockScore = float64(health.BlocksProduced) / float64(totalBlocks)
	}

	// 网络延迟评分 (25%权重)
	latencyScore := 1.0
	if health.NetworkLatency > 1000 { // 1秒以上
		latencyScore = 0.0
	} else if health.NetworkLatency > 500 { // 500ms以上
		latencyScore = 0.5
	}

	// 综合评分
	health.ActiveScore = heartbeatScore*0.4 + blockScore*0.35 + latencyScore*0.25
	health.IsResponsive = health.ActiveScore >= 0.6
}

// CheckNodeActivity 检查节点活跃度并更新安全模式
func (dpos *DPoSConsensus) CheckNodeActivity() {
	dpos.mu.Lock()
	defer dpos.mu.Unlock()

	activeCount := 0
	totalValidators := len(dpos.validatorSet.Validators)

	// 统计活跃验证者数量
	for _, validator := range dpos.validatorSet.Validators {
		if !validator.IsActive {
			continue
		}

		health, exists := dpos.nodeHealthMap[validator.Address]
		if exists && health.IsResponsive {
			activeCount++
		}
	}

	// 计算活跃度比例
	activeRatio := float64(activeCount) / float64(totalValidators)

	// 检查是否需要进入安全模式
	prevSafetyMode := dpos.safetyMode
	dpos.safetyMode = activeRatio < dpos.activeThreshold

	// 记录状态变化
	if prevSafetyMode != dpos.safetyMode {
		if dpos.safetyMode {
			fmt.Printf("⚠️  Entering SAFETY MODE: Active validators %d/%d (%.1f%% < %.1f%%)\n", 
				activeCount, totalValidators, activeRatio*100, dpos.activeThreshold*100)
		} else {
			fmt.Printf("✅ Exiting SAFETY MODE: Active validators %d/%d (%.1f%% >= %.1f%%)\n", 
				activeCount, totalValidators, activeRatio*100, dpos.activeThreshold*100)
		}
	}

	dpos.lastHealthCheck = time.Now().Unix()
}

// IsInSafetyMode 检查是否处于安全模式
func (dpos *DPoSConsensus) IsInSafetyMode() bool {
	dpos.mu.RLock()
	defer dpos.mu.RUnlock()
	return dpos.safetyMode
}

// GetActiveValidatorRatio 获取活跃验证者比例
func (dpos *DPoSConsensus) GetActiveValidatorRatio() float64 {
	dpos.mu.RLock()
	defer dpos.mu.RUnlock()

	activeCount := 0
	totalValidators := len(dpos.validatorSet.Validators)

	for _, validator := range dpos.validatorSet.Validators {
		if !validator.IsActive {
			continue
		}

		health, exists := dpos.nodeHealthMap[validator.Address]
		if exists && health.IsResponsive {
			activeCount++
		}
	}

	if totalValidators == 0 {
		return 0.0
	}

	return float64(activeCount) / float64(totalValidators)
}

// GetNodeHealthInfo 获取节点健康信息
func (dpos *DPoSConsensus) GetNodeHealthInfo(address types.Address) *NodeHealthInfo {
	dpos.mu.RLock()
	defer dpos.mu.RUnlock()

	if health, exists := dpos.nodeHealthMap[address]; exists {
		// 返回副本避免并发修改
		healthCopy := *health
		return &healthCopy
	}

	return nil
}

// GetHealthStats 获取健康统计信息
func (dpos *DPoSConsensus) GetHealthStats() map[string]interface{} {
	dpos.mu.RLock()
	defer dpos.mu.RUnlock()

	activeCount := 0
	totalCount := len(dpos.validatorSet.Validators)
	avgScore := 0.0

	for _, validator := range dpos.validatorSet.Validators {
		if !validator.IsActive {
			continue
		}

		health, exists := dpos.nodeHealthMap[validator.Address]
		if exists {
			avgScore += health.ActiveScore
			if health.IsResponsive {
				activeCount++
			}
		}
	}

	if totalCount > 0 {
		avgScore /= float64(totalCount)
	}

	return map[string]interface{}{
		"active_validators": activeCount,
		"total_validators":  totalCount,
		"active_ratio":      float64(activeCount) / float64(totalCount),
		"threshold":         dpos.activeThreshold,
		"safety_mode":       dpos.safetyMode,
		"avg_score":         avgScore,
		"last_check":        dpos.lastHealthCheck,
	}
}

// =============== 健康检查循环与自动管理 ===============

// healthCheckLoop 健康检查循环
func (dpos *DPoSConsensus) healthCheckLoop() {
	ticker := time.NewTicker(15 * time.Second) // 每15秒检查一次
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if !dpos.IsActive() {
				return
			}
			dpos.performHealthCheck()
		default:
			if !dpos.IsActive() {
				return
			}
			time.Sleep(100 * time.Millisecond)
		}
	}
}

// performHealthCheck 执行健康检查
func (dpos *DPoSConsensus) performHealthCheck() {
	// 检查节点活跃度
	dpos.CheckNodeActivity()
	
	// 清理过期的健康信息
	dpos.cleanupExpiredHealth()
	
	// 自动调整验证者状态
	dpos.autoAdjustValidators()
}

// cleanupExpiredHealth 清理过期的健康信息
func (dpos *DPoSConsensus) cleanupExpiredHealth() {
	dpos.mu.Lock()
	defer dpos.mu.Unlock()

	now := time.Now().Unix()
	expireThreshold := int64(300) // 5分钟过期

	for address, health := range dpos.nodeHealthMap {
		if now-health.LastHeartbeat > expireThreshold {
			// 清理过期节点
			delete(dpos.nodeHealthMap, address)
			fmt.Printf("🗑️ Removed expired node health info: %s\n", address.String())
		}
	}
}

// autoAdjustValidators 自动调整验证者状态
func (dpos *DPoSConsensus) autoAdjustValidators() {
	dpos.mu.Lock()
	defer dpos.mu.Unlock()

	now := time.Now().Unix()
	inactiveThreshold := int64(120) // 2分钟非活跃阈值
	recoveryThreshold := int64(60)  // 1分钟恢复阈值

	for i := range dpos.validatorSet.Validators {
		validator := &dpos.validatorSet.Validators[i]
		health, exists := dpos.nodeHealthMap[validator.Address]
		
		if !exists {
			continue
		}

		// 自动停用非活跃验证者
		if validator.IsActive && !health.IsResponsive {
			if now-health.LastHeartbeat > inactiveThreshold {
				validator.IsActive = false
				err := dpos.storage.SaveValidator(validator)
				if err == nil {
					fmt.Printf("⚠️ Auto-deactivated validator: %s (unresponsive for %ds)\n", 
						validator.Address.String(), now-health.LastHeartbeat)
				}
			}
		}

		// 自动激活恢复的验证者
		if !validator.IsActive && health.IsResponsive {
			if now-health.LastHeartbeat <= recoveryThreshold {
				validator.IsActive = true
				err := dpos.storage.SaveValidator(validator)
				if err == nil {
					fmt.Printf("✅ Auto-activated validator: %s (recovered)\n", 
						validator.Address.String())
				}
			}
		}
	}

	// 重新排序验证者
	dpos.sortValidators()
}

// =============== 高级共识管理方法 ===============

// GetConsensusMode 获取当前共识模式
func (dpos *DPoSConsensus) GetConsensusMode() string {
	dpos.mu.RLock()
	defer dpos.mu.RUnlock()

	if dpos.safetyMode {
		return "SAFETY_MODE"
	}

	activeRatio := dpos.GetActiveValidatorRatio()
	if activeRatio >= 0.8 {
		return "NORMAL"
	} else if activeRatio >= 0.6 {
		return "DEGRADED"
	} else {
		return "CRITICAL"
	}
}

// ForceValidatorSync 强制同步验证者集合
func (dpos *DPoSConsensus) ForceValidatorSync() error {
	dpos.mu.Lock()
	defer dpos.mu.Unlock()

	return dpos.loadValidatorSet()
}

// GetDetailedConsensusInfo 获取详细共识信息
func (dpos *DPoSConsensus) GetDetailedConsensusInfo() map[string]interface{} {
	dpos.mu.RLock()
	defer dpos.mu.RUnlock()

	activeValidators := dpos.getActiveValidators()
	validatorDetails := make([]map[string]interface{}, 0)

	for _, validator := range dpos.validatorSet.Validators {
		health := dpos.nodeHealthMap[validator.Address]
		detail := map[string]interface{}{
			"address":        validator.Address.String(),
			"is_active":      validator.IsActive,
			"stake_amount":   validator.StakeAmount,
			"delegated_amt":  validator.DelegatedAmt,
			"vote_power":     validator.VotePower,
			"slash_count":    validator.SlashCount,
		}

		if health != nil {
			detail["last_heartbeat"] = health.LastHeartbeat
			detail["last_block_time"] = health.LastBlockTime
			detail["active_score"] = health.ActiveScore
			detail["is_responsive"] = health.IsResponsive
			detail["blocks_produced"] = health.BlocksProduced
			detail["missed_blocks"] = health.MissedBlocks
			detail["network_latency"] = health.NetworkLatency
		} else {
			detail["health_status"] = "NO_DATA"
		}

		validatorDetails = append(validatorDetails, detail)
	}

	return map[string]interface{}{
		"consensus_type":        "DPoS",
		"is_active":             dpos.isActive,
		"consensus_mode":        dpos.GetConsensusMode(),
		"safety_mode":           dpos.safetyMode,
		"active_threshold":      dpos.activeThreshold,
		"total_validators":      len(dpos.validatorSet.Validators),
		"active_validators":     len(activeValidators),
		"active_ratio":          dpos.GetActiveValidatorRatio(),
		"current_round":         dpos.validatorSet.Round,
		"total_vote_power":      dpos.GetTotalVotePower(),
		"last_health_check":     dpos.lastHealthCheck,
		"validator_details":     validatorDetails,
	}
}