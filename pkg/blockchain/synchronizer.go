package blockchain

import (
	"fmt"
	"sync"
	"time"

	"github.com/lengzhao/shardmatrix/pkg/storage"
	"github.com/lengzhao/shardmatrix/pkg/types"
)

// SyncState 同步状态
type SyncState int

const (
	SyncStateIdle          SyncState = iota // 空闲状态
	SyncStateFindingPeers                   // 寻找对等节点
	SyncStateNegotiating                    // 协商最高区块
	SyncStateDownloading                    // 批量下载区块
	SyncStateValidating                     // 验证区块
	SyncStateApplying                       // 应用区块
	SyncStateCompleted                      // 同步完成
	SyncStateError                          // 同步错误
)

// String 返回同步状态的字符串表示
func (s SyncState) String() string {
	switch s {
	case SyncStateIdle:
		return "IDLE"
	case SyncStateFindingPeers:
		return "FINDING_PEERS"
	case SyncStateNegotiating:
		return "NEGOTIATING"
	case SyncStateDownloading:
		return "DOWNLOADING"
	case SyncStateValidating:
		return "VALIDATING"
	case SyncStateApplying:
		return "APPLYING"
	case SyncStateCompleted:
		return "COMPLETED"
	case SyncStateError:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// SyncPeer 同步对等节点信息
type SyncPeer struct {
	ID           string       // 节点ID
	Address      string       // 节点地址
	LatestHeight uint64       // 最新区块高度
	LatestHash   types.Hash   // 最新区块哈希
	Latency      int64        // 网络延迟(ms)
	LastSeen     int64        // 最后联系时间
	IsReliable   bool         // 是否可靠
}

// SyncConfig 同步配置
type SyncConfig struct {
	MaxPeers           int           // 最大对等节点数
	BatchSize          int           // 批量下载区块数
	RequestTimeout     time.Duration // 请求超时时间
	MaxRetries         int           // 最大重试次数
	ConflictResolution string        // 冲突解决策略
	ValidationEnabled  bool          // 是否启用区块验证
	CheckpointSync     bool          // 是否启用检查点同步
}

// BlockSynchronizer 区块同步器
type BlockSynchronizer struct {
	mu              sync.RWMutex
	storage         storage.Storage
	blockchain      *Manager
	validator       BlockValidator
	state           SyncState
	config          *SyncConfig
	peers           map[string]*SyncPeer
	targetHeight    uint64
	currentHeight   uint64
	syncProgress    float64
	lastSyncTime    int64
	errorCount      int
	isRunning       bool
	stopCh          chan struct{}
	syncCallbacks   []SyncCallback
}

// BlockValidator 区块验证器接口
type BlockValidator interface {
	ValidateBlock(block *types.Block) error
	ValidateSequence(blocks []*types.Block) error
}

// SyncCallback 同步回调函数
type SyncCallback func(state SyncState, progress float64, height uint64)

// NetworkProvider 网络提供者接口
type NetworkProvider interface {
	GetPeers() []*SyncPeer
	RequestBlock(peerID string, height uint64) (*types.Block, error)
	RequestBlocks(peerID string, startHeight, endHeight uint64) ([]*types.Block, error)
	BroadcastHeight(height uint64, hash types.Hash) error
}

// NewBlockSynchronizer 创建新的区块同步器
func NewBlockSynchronizer(
	storage storage.Storage,
	blockchain *Manager,
	validator BlockValidator,
) *BlockSynchronizer {
	return &BlockSynchronizer{
		storage:   storage,
		blockchain: blockchain,
		validator: validator,
		state:     SyncStateIdle,
		config: &SyncConfig{
			MaxPeers:           10,
			BatchSize:          50,
			RequestTimeout:     30 * time.Second,
			MaxRetries:         3,
			ConflictResolution: "longest_chain",
			ValidationEnabled:  true,
			CheckpointSync:     true,
		},
		peers:         make(map[string]*SyncPeer),
		stopCh:        make(chan struct{}),
		syncCallbacks: make([]SyncCallback, 0),
	}
}

// Start 启动同步器
func (bs *BlockSynchronizer) Start() error {
	bs.mu.Lock()
	defer bs.mu.Unlock()

	if bs.isRunning {
		return fmt.Errorf("synchronizer already running")
	}

	bs.isRunning = true
	bs.stopCh = make(chan struct{})

	// 获取当前本地区块高度
	height, err := bs.storage.GetLatestBlockHeight()
	if err != nil {
		bs.currentHeight = 0
	} else {
		bs.currentHeight = height
	}

	// 启动同步循环
	go bs.syncLoop()

	fmt.Println("✓ Block synchronizer started")
	return nil
}

// Stop 停止同步器
func (bs *BlockSynchronizer) Stop() {
	bs.mu.Lock()
	defer bs.mu.Unlock()

	if !bs.isRunning {
		return
	}

	bs.isRunning = false
	close(bs.stopCh)
	bs.state = SyncStateIdle
}

// syncLoop 同步循环
func (bs *BlockSynchronizer) syncLoop() {
	ticker := time.NewTicker(10 * time.Second) // 每10秒检查一次
	defer ticker.Stop()

	for {
		select {
		case <-bs.stopCh:
			return
		case <-ticker.C:
			if bs.shouldSync() {
				bs.performSync()
			}
		}
	}
}

// shouldSync 判断是否需要同步
func (bs *BlockSynchronizer) shouldSync() bool {
	bs.mu.RLock()
	defer bs.mu.RUnlock()

	// 如果已经在同步中，跳过
	if bs.state != SyncStateIdle && bs.state != SyncStateCompleted {
		return false
	}

	// 检查是否有更高的对等节点
	return bs.hasHigherPeers()
}

// hasHigherPeers 检查是否有更高的对等节点
func (bs *BlockSynchronizer) hasHigherPeers() bool {
	for _, peer := range bs.peers {
		if peer.LatestHeight > bs.currentHeight && peer.IsReliable {
			return true
		}
	}
	return false
}

// performSync 执行同步
func (bs *BlockSynchronizer) performSync() {
	bs.mu.Lock()
	bs.state = SyncStateFindingPeers
	bs.mu.Unlock()

	bs.notifyStateChange()

	// 步骤1：寻找可靠的对等节点
	err := bs.findReliablePeers()
	if err != nil {
		bs.handleSyncError(fmt.Errorf("failed to find peers: %w", err))
		return
	}

	// 步骤2：协商最高区块
	err = bs.negotiateTargetHeight()
	if err != nil {
		bs.handleSyncError(fmt.Errorf("failed to negotiate target: %w", err))
		return
	}

	// 步骤3：批量下载区块
	err = bs.downloadBlocks()
	if err != nil {
		bs.handleSyncError(fmt.Errorf("failed to download blocks: %w", err))
		return
	}

	// 步骤4：同步完成
	bs.mu.Lock()
	bs.state = SyncStateCompleted
	bs.lastSyncTime = time.Now().Unix()
	bs.errorCount = 0
	bs.mu.Unlock()

	bs.notifyStateChange()
	fmt.Printf("🔄 Sync completed: %d -> %d blocks\n", bs.currentHeight, bs.targetHeight)
}

// findReliablePeers 寻找可靠的对等节点
func (bs *BlockSynchronizer) findReliablePeers() error {
	// 简化实现：假设已有对等节点
	bs.mu.Lock()
	defer bs.mu.Unlock()

	reliableCount := 0
	for _, peer := range bs.peers {
		if peer.IsReliable {
			reliableCount++
		}
	}

	if reliableCount == 0 {
		return fmt.Errorf("no reliable peers found")
	}

	return nil
}

// negotiateTargetHeight 协商目标高度
func (bs *BlockSynchronizer) negotiateTargetHeight() error {
	bs.mu.Lock()
	bs.state = SyncStateNegotiating
	defer func() {
		bs.mu.Unlock()
		bs.notifyStateChange()
	}()

	maxHeight := bs.currentHeight
	var targetPeer *SyncPeer

	// 找到最高的可靠节点
	for _, peer := range bs.peers {
		if peer.IsReliable && peer.LatestHeight > maxHeight {
			maxHeight = peer.LatestHeight
			targetPeer = peer
		}
	}

	if targetPeer == nil {
		return fmt.Errorf("no target peer found")
	}

	bs.targetHeight = maxHeight
	fmt.Printf("🎯 Target height set to %d (peer: %s)\n", bs.targetHeight, targetPeer.ID)
	return nil
}

// downloadBlocks 批量下载区块
func (bs *BlockSynchronizer) downloadBlocks() error {
	bs.mu.Lock()
	bs.state = SyncStateDownloading
	bs.mu.Unlock()

	bs.notifyStateChange()

	startHeight := bs.currentHeight + 1
	totalBlocks := bs.targetHeight - bs.currentHeight

	fmt.Printf("📥 Downloading blocks %d -> %d (%d blocks)\n", 
		startHeight, bs.targetHeight, totalBlocks)

	for height := startHeight; height <= bs.targetHeight; height += uint64(bs.config.BatchSize) {
		endHeight := height + uint64(bs.config.BatchSize) - 1
		if endHeight > bs.targetHeight {
			endHeight = bs.targetHeight
		}

		// 下载一批区块
		blocks, err := bs.downloadBatch(height, endHeight)
		if err != nil {
			return fmt.Errorf("failed to download batch %d-%d: %w", height, endHeight, err)
		}

		// 验证区块
		if bs.config.ValidationEnabled {
			bs.mu.Lock()
			bs.state = SyncStateValidating
			bs.mu.Unlock()

			err = bs.validateBlocks(blocks)
			if err != nil {
				return fmt.Errorf("validation failed for batch %d-%d: %w", height, endHeight, err)
			}
		}

		// 应用区块
		bs.mu.Lock()
		bs.state = SyncStateApplying
		bs.mu.Unlock()

		err = bs.applyBlocks(blocks)
		if err != nil {
			return fmt.Errorf("failed to apply batch %d-%d: %w", height, endHeight, err)
		}

		// 更新进度
		bs.updateProgress()

		fmt.Printf("✓ Applied blocks %d-%d\n", height, endHeight)
	}

	return nil
}

// downloadBatch 下载一批区块
func (bs *BlockSynchronizer) downloadBatch(startHeight, endHeight uint64) ([]*types.Block, error) {
	// 简化实现：创建模拟区块
	var blocks []*types.Block

	for height := startHeight; height <= endHeight; height++ {
		block := &types.Block{
			Header: types.BlockHeader{
				Number:    height,
				Timestamp: time.Now().Unix(),
				PrevHash:  types.Hash{}, // 简化
				TxRoot:    types.EmptyTxRoot(),
				StateRoot: types.Hash{}, // 简化
				Validator: types.Address{},
				ShardID:   types.ShardID,
			},
			Transactions: []types.Hash{},
		}
		blocks = append(blocks, block)
	}

	// 模拟网络延迟
	time.Sleep(100 * time.Millisecond)

	return blocks, nil
}

// validateBlocks 验证区块序列
func (bs *BlockSynchronizer) validateBlocks(blocks []*types.Block) error {
	if bs.validator == nil {
		return nil // 跳过验证
	}

	// 验证区块序列
	return bs.validator.ValidateSequence(blocks)
}

// applyBlocks 应用区块到本地链
func (bs *BlockSynchronizer) applyBlocks(blocks []*types.Block) error {
	for _, block := range blocks {
		err := bs.blockchain.AddBlock(block)
		if err != nil {
			return fmt.Errorf("failed to add block %d: %w", block.Header.Number, err)
		}

		bs.mu.Lock()
		bs.currentHeight = block.Header.Number
		bs.mu.Unlock()
	}

	return nil
}

// updateProgress 更新同步进度
func (bs *BlockSynchronizer) updateProgress() {
	bs.mu.Lock()
	defer bs.mu.Unlock()

	if bs.targetHeight > 0 {
		bs.syncProgress = float64(bs.currentHeight) / float64(bs.targetHeight) * 100
	}

	bs.notifyStateChange()
}

// handleSyncError 处理同步错误
func (bs *BlockSynchronizer) handleSyncError(err error) {
	bs.mu.Lock()
	bs.state = SyncStateError
	bs.errorCount++
	bs.mu.Unlock()

	fmt.Printf("❌ Sync error: %v (count: %d)\n", err, bs.errorCount)
	bs.notifyStateChange()

	// 错误恢复策略
	if bs.errorCount < bs.config.MaxRetries {
		time.Sleep(time.Duration(bs.errorCount) * 10 * time.Second)
		bs.mu.Lock()
		bs.state = SyncStateIdle
		bs.mu.Unlock()
	}
}

// notifyStateChange 通知状态变化
func (bs *BlockSynchronizer) notifyStateChange() {
	bs.mu.RLock()
	callbacks := make([]SyncCallback, len(bs.syncCallbacks))
	copy(callbacks, bs.syncCallbacks)
	state := bs.state
	progress := bs.syncProgress
	height := bs.currentHeight
	bs.mu.RUnlock()

	for _, callback := range callbacks {
		go callback(state, progress, height)
	}
}

// RegisterSyncCallback 注册同步回调
func (bs *BlockSynchronizer) RegisterSyncCallback(callback SyncCallback) {
	bs.mu.Lock()
	defer bs.mu.Unlock()
	bs.syncCallbacks = append(bs.syncCallbacks, callback)
}

// AddPeer 添加对等节点
func (bs *BlockSynchronizer) AddPeer(peer *SyncPeer) {
	bs.mu.Lock()
	defer bs.mu.Unlock()
	bs.peers[peer.ID] = peer
}

// RemovePeer 移除对等节点
func (bs *BlockSynchronizer) RemovePeer(peerID string) {
	bs.mu.Lock()
	defer bs.mu.Unlock()
	delete(bs.peers, peerID)
}

// GetSyncStatus 获取同步状态
func (bs *BlockSynchronizer) GetSyncStatus() map[string]interface{} {
	bs.mu.RLock()
	defer bs.mu.RUnlock()

	return map[string]interface{}{
		"state":           bs.state.String(),
		"current_height":  bs.currentHeight,
		"target_height":   bs.targetHeight,
		"progress":        bs.syncProgress,
		"peer_count":      len(bs.peers),
		"last_sync_time":  bs.lastSyncTime,
		"error_count":     bs.errorCount,
		"is_running":      bs.isRunning,
	}
}

// SetConfig 设置同步配置
func (bs *BlockSynchronizer) SetConfig(config *SyncConfig) {
	bs.mu.Lock()
	defer bs.mu.Unlock()
	bs.config = config
}

// ForceSyncFrom 强制从指定高度开始同步
func (bs *BlockSynchronizer) ForceSyncFrom(startHeight uint64) {
	bs.mu.Lock()
	defer bs.mu.Unlock()

	bs.currentHeight = startHeight
	bs.state = SyncStateIdle
	bs.errorCount = 0
}