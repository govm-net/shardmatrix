package blockchain

import (
	"fmt"
	"sort"
	"time"

	"github.com/govm-net/shardmatrix/pkg/types"
)

// SyncStatus 同步状态
type SyncStatus struct {
	IsSyncing     bool    `json:"is_syncing"`     // 是否正在同步
	SyncPeer      string  `json:"sync_peer"`      // 同步节点
	CurrentHeight uint64  `json:"current_height"` // 当前高度
	TargetHeight  uint64  `json:"target_height"`  // 目标高度
	Progress      float64 `json:"progress"`       // 同步进度
	StartTime     int64   `json:"start_time"`     // 开始时间
}

// BlockSyncRequest 区块同步请求
type BlockSyncRequest struct {
	FromHeight uint64 `json:"from_height"` // 起始高度
	ToHeight   uint64 `json:"to_height"`   // 结束高度
	MaxBlocks  int    `json:"max_blocks"`  // 最大区块数
}

// BlockSyncResponse 区块同步响应
type BlockSyncResponse struct {
	Blocks     []*types.Block `json:"blocks"`      // 区块列表
	HasMore    bool           `json:"has_more"`    // 是否还有更多
	NextHeight uint64         `json:"next_height"` // 下一个高度
}

// StartSync 开始同步
func (bc *Blockchain) StartSync(peerID string, targetHeight uint64) error {
	bc.mutex.Lock()
	defer bc.mutex.Unlock()

	if bc.isSyncing {
		return fmt.Errorf("already syncing with peer %s", bc.syncPeer)
	}

	currentHeight := bc.chainState.Height
	if targetHeight <= currentHeight {
		return fmt.Errorf("target height %d is not greater than current height %d", targetHeight, currentHeight)
	}

	bc.isSyncing = true
	bc.syncPeer = peerID

	fmt.Printf("Starting sync with peer %s from height %d to %d\n", peerID, currentHeight+1, targetHeight)

	return nil
}

// StopSync 停止同步
func (bc *Blockchain) StopSync() {
	bc.mutex.Lock()
	defer bc.mutex.Unlock()

	if !bc.isSyncing {
		return
	}

	fmt.Printf("Stopping sync with peer %s\n", bc.syncPeer)

	bc.isSyncing = false
	bc.syncPeer = ""

	if bc.onSyncComplete != nil {
		bc.onSyncComplete()
	}
}

// GetSyncStatus 获取同步状态
func (bc *Blockchain) GetSyncStatus() *SyncStatus {
	bc.mutex.RLock()
	defer bc.mutex.RUnlock()

	return &SyncStatus{
		IsSyncing:     bc.isSyncing,
		SyncPeer:      bc.syncPeer,
		CurrentHeight: bc.chainState.Height,
	}
}

// ProcessSyncBlocks 处理同步的区块
func (bc *Blockchain) ProcessSyncBlocks(blocks []*types.Block) error {
	bc.mutex.Lock()
	defer bc.mutex.Unlock()

	if !bc.isSyncing {
		return fmt.Errorf("not in sync mode")
	}

	// 按高度排序区块
	sort.Slice(blocks, func(i, j int) bool {
		return blocks[i].Header.Number < blocks[j].Header.Number
	})

	// 验证并添加区块
	for _, block := range blocks {
		// 验证区块连接性
		if err := bc.validateBlockForSync(block); err != nil {
			return fmt.Errorf("sync block validation failed for block %d: %v", block.Header.Number, err)
		}

		// 验证区块内容
		if err := bc.validator.ValidateNewBlock(block, nil); err != nil {
			return fmt.Errorf("sync block content validation failed for block %d: %v", block.Header.Number, err)
		}

		// 存储区块
		if err := bc.blockStore.PutBlock(block); err != nil {
			return fmt.Errorf("failed to store sync block %d: %v", block.Header.Number, err)
		}

		// 更新链状态
		bc.updateChainStateForSync(block)
	}

	fmt.Printf("Processed %d sync blocks, current height: %d\n", len(blocks), bc.chainState.Height)

	return nil
}

// validateBlockForSync 验证同步区块
func (bc *Blockchain) validateBlockForSync(block *types.Block) error {
	expectedHeight := bc.chainState.Height + 1

	// 检查区块高度
	if block.Header.Number != expectedHeight {
		return fmt.Errorf("invalid block height: expected %d, got %d", expectedHeight, block.Header.Number)
	}

	// 检查前一个区块哈希
	if !block.Header.PrevHash.Equal(bc.chainState.BestBlockHash) {
		return fmt.Errorf("invalid previous block hash")
	}

	return nil
}

// updateChainStateForSync 更新同步链状态
func (bc *Blockchain) updateChainStateForSync(block *types.Block) {
	bc.chainState.Height = block.Header.Number
	bc.chainState.BestBlockHash = block.Hash()
	bc.chainState.TotalWork += bc.calculateBlockWork(block)
	bc.chainState.LastUpdated = time.Now().Unix()
}

// GetBlocksForSync 获取用于同步的区块
func (bc *Blockchain) GetBlocksForSync(fromHeight, toHeight uint64, maxBlocks int) ([]*types.Block, error) {
	bc.mutex.RLock()
	defer bc.mutex.RUnlock()

	// 验证参数
	if fromHeight > toHeight {
		return nil, fmt.Errorf("fromHeight (%d) cannot be greater than toHeight (%d)", fromHeight, toHeight)
	}

	if fromHeight > bc.chainState.Height {
		return nil, fmt.Errorf("fromHeight (%d) is greater than chain height (%d)", fromHeight, bc.chainState.Height)
	}

	// 调整toHeight
	if toHeight > bc.chainState.Height {
		toHeight = bc.chainState.Height
	}

	// 限制返回的区块数量
	requestedCount := int(toHeight - fromHeight + 1)
	if maxBlocks > 0 && requestedCount > maxBlocks {
		toHeight = fromHeight + uint64(maxBlocks) - 1
	}

	// 获取区块
	var blocks []*types.Block
	for height := fromHeight; height <= toHeight; height++ {
		block, err := bc.blockStore.GetBlockByHeight(height)
		if err != nil {
			return nil, fmt.Errorf("failed to get block at height %d: %v", height, err)
		}
		blocks = append(blocks, block)
	}

	return blocks, nil
}

// GetBlockRange 获取区块范围
func (bc *Blockchain) GetBlockRange(fromHeight, toHeight uint64) ([]*types.Block, error) {
	bc.mutex.RLock()
	defer bc.mutex.RUnlock()

	if fromHeight > toHeight {
		return nil, fmt.Errorf("fromHeight cannot be greater than toHeight")
	}

	if toHeight > bc.chainState.Height {
		return nil, fmt.Errorf("toHeight is greater than chain height")
	}

	var blocks []*types.Block
	for height := fromHeight; height <= toHeight; height++ {
		block, err := bc.blockStore.GetBlockByHeight(height)
		if err != nil {
			return nil, fmt.Errorf("failed to get block at height %d: %v", height, err)
		}
		blocks = append(blocks, block)
	}

	return blocks, nil
}

// GetRecentBlocks 获取最近的区块
func (bc *Blockchain) GetRecentBlocks(count int) ([]*types.Block, error) {
	bc.mutex.RLock()
	currentHeight := bc.chainState.Height
	bc.mutex.RUnlock()

	if count <= 0 {
		return []*types.Block{}, nil
	}

	fromHeight := uint64(0)
	if currentHeight >= uint64(count-1) {
		fromHeight = currentHeight - uint64(count-1)
	}

	return bc.GetBlockRange(fromHeight, currentHeight)
}

// GetAncestors 获取区块的祖先区块
func (bc *Blockchain) GetAncestors(blockHash types.Hash, count int) ([]*types.Block, error) {
	bc.mutex.RLock()
	defer bc.mutex.RUnlock()

	if count <= 0 {
		return []*types.Block{}, nil
	}

	// 获取起始区块
	currentBlock, err := bc.blockStore.GetBlock(blockHash)
	if err != nil {
		return nil, fmt.Errorf("failed to get block %x: %v", blockHash, err)
	}

	var ancestors []*types.Block
	ancestors = append(ancestors, currentBlock)

	// 向前追溯
	for i := 1; i < count && !currentBlock.Header.PrevHash.IsZero(); i++ {
		parentBlock, err := bc.blockStore.GetBlock(currentBlock.Header.PrevHash)
		if err != nil {
			break // 无法获取父区块，停止追溯
		}

		ancestors = append(ancestors, parentBlock)
		currentBlock = parentBlock
	}

	return ancestors, nil
}

// LocateCommonAncestor 定位共同祖先
func (bc *Blockchain) LocateCommonAncestor(blockHashes []types.Hash) (types.Hash, error) {
	bc.mutex.RLock()
	defer bc.mutex.RUnlock()

	if len(blockHashes) == 0 {
		return types.EmptyHash(), fmt.Errorf("no block hashes provided")
	}

	// 获取所有区块的高度
	blockHeights := make(map[types.Hash]uint64)
	minHeight := uint64(^uint64(0)) // 最大值

	for _, hash := range blockHashes {
		block, err := bc.blockStore.GetBlock(hash)
		if err != nil {
			continue // 跳过不存在的区块
		}

		blockHeights[hash] = block.Header.Number
		if block.Header.Number < minHeight {
			minHeight = block.Header.Number
		}
	}

	if len(blockHeights) == 0 {
		return types.EmptyHash(), fmt.Errorf("no valid blocks found")
	}

	// 从最低高度开始向上查找
	for height := minHeight; ; height-- {
		// 获取该高度的区块
		block, err := bc.blockStore.GetBlockByHeight(height)
		if err != nil {
			continue
		}

		blockHash := block.Hash()

		// 检查所有链是否都包含这个区块
		allContain := true
		for _, hash := range blockHashes {
			if !bc.isAncestor(blockHash, hash) {
				allContain = false
				break
			}
		}

		if allContain {
			return blockHash, nil
		}

		if height == 0 {
			break
		}
	}

	return types.EmptyHash(), fmt.Errorf("no common ancestor found")
}

// isAncestor 检查block1是否是block2的祖先
func (bc *Blockchain) isAncestor(ancestorHash, descendantHash types.Hash) bool {
	if ancestorHash.Equal(descendantHash) {
		return true
	}

	currentHash := descendantHash
	for !currentHash.IsZero() {
		block, err := bc.blockStore.GetBlock(currentHash)
		if err != nil {
			return false
		}

		if block.Header.PrevHash.Equal(ancestorHash) {
			return true
		}

		currentHash = block.Header.PrevHash
	}

	return false
}

// ValidateChain 验证整个链的完整性
func (bc *Blockchain) ValidateChain() error {
	bc.mutex.RLock()
	defer bc.mutex.RUnlock()

	// 从创世区块开始验证
	currentHeight := uint64(0)
	maxHeight := bc.chainState.Height

	for currentHeight <= maxHeight {
		block, err := bc.blockStore.GetBlockByHeight(currentHeight)
		if err != nil {
			return fmt.Errorf("failed to get block at height %d: %v", currentHeight, err)
		}

		// 验证区块内容
		if currentHeight == 0 {
			// 验证创世区块
			if err := bc.validator.ValidateGenesisBlock(block); err != nil {
				return fmt.Errorf("genesis block validation failed: %v", err)
			}
		} else {
			// 验证普通区块
			if err := bc.validator.ValidateNewBlock(block, nil); err != nil {
				return fmt.Errorf("block %d validation failed: %v", currentHeight, err)
			}

			// 验证与前一个区块的连接
			prevBlock, err := bc.blockStore.GetBlockByHeight(currentHeight - 1)
			if err != nil {
				return fmt.Errorf("failed to get previous block at height %d: %v", currentHeight-1, err)
			}

			if !block.Header.PrevHash.Equal(prevBlock.Hash()) {
				return fmt.Errorf("block %d has invalid previous hash", currentHeight)
			}
		}

		currentHeight++
	}

	return nil
}
