// Package blockchain implements the core blockchain management functionality
// including block validation, chain state management, fork handling, and reorganization.
package blockchain

import (
	"fmt"
	"sync"
	"time"

	"github.com/govm-net/shardmatrix/pkg/consensus"
	"github.com/govm-net/shardmatrix/pkg/storage"
	"github.com/govm-net/shardmatrix/pkg/types"
	"github.com/govm-net/shardmatrix/pkg/validator"
)

// ChainState 区块链状态
type ChainState struct {
	Height        uint64     `json:"height"`          // 当前区块高度
	BestBlockHash types.Hash `json:"best_block_hash"` // 最佳区块哈希
	TotalWork     uint64     `json:"total_work"`      // 总工作量
	LastUpdated   int64      `json:"last_updated"`    // 最后更新时间
}

// BlockchainConfig 区块链配置
type BlockchainConfig struct {
	GenesisBlock  *types.Block  `json:"genesis_block"`   // 创世区块
	ChainID       uint64        `json:"chain_id"`        // 链ID
	BlockInterval time.Duration `json:"block_interval"`  // 区块间隔
	MaxBlockSize  int           `json:"max_block_size"`  // 最大区块大小
	MaxReorgDepth uint64        `json:"max_reorg_depth"` // 最大重组深度
	ConfirmBlocks uint64        `json:"confirm_blocks"`  // 确认区块数
}

// DefaultBlockchainConfig 默认区块链配置
func DefaultBlockchainConfig() *BlockchainConfig {
	// 创建统一的创世区块配置
	genesisValidator := types.AddressFromPublicKey([]byte("shardmatrix_genesis_validator"))
	genesisBlock := types.NewGenesisBlock(genesisValidator)

	return &BlockchainConfig{
		GenesisBlock:  genesisBlock,
		ChainID:       1,
		BlockInterval: 3 * time.Second,
		MaxBlockSize:  2 * 1024 * 1024, // 2MB
		MaxReorgDepth: 100,              // 最多重组100个区块
		ConfirmBlocks: 6,                // 6个确认
	}
}

// Blockchain 区块链管理器
type Blockchain struct {
	config     *BlockchainConfig
	blockStore storage.BlockStore
	validator  *validator.Validator

	// 状态管理
	chainState *ChainState

	// 分叉管理
	forks map[string]*Fork // 分叉链，键为分叉起始区块哈希

	// 同步状态
	isSyncing bool
	syncPeer  string

	// DPoS共识集成
	consensus *ConsensusIntegration

	// 线程安全
	mutex sync.RWMutex

	// 回调函数
	onNewBlock     func(*types.Block)
	onChainReorg   func(oldBlocks []*types.Block, newBlocks []*types.Block)
	onSyncComplete func()
}

// Fork 分叉信息
type Fork struct {
	StartHeight uint64         `json:"start_height"` // 分叉起始高度
	StartHash   types.Hash     `json:"start_hash"`   // 分叉起始区块哈希
	Blocks      []*types.Block `json:"blocks"`       // 分叉中的区块
	TotalWork   uint64         `json:"total_work"`   // 总工作量
	LastUpdated int64          `json:"last_updated"` // 最后更新时间
}

// NewBlockchain 创建区块链管理器
func NewBlockchain(config *BlockchainConfig, blockStore storage.BlockStore, v *validator.Validator) (*Blockchain, error) {
	if config == nil {
		config = DefaultBlockchainConfig()
	}

	bc := &Blockchain{
		config:     config,
		blockStore: blockStore,
		validator:  v,
		forks:      make(map[string]*Fork),
		chainState: &ChainState{
			Height:      0,
			TotalWork:   0,
			LastUpdated: time.Now().Unix(),
		},
	}

	// 初始化区块链状态
	if err := bc.initialize(); err != nil {
		return nil, fmt.Errorf("failed to initialize blockchain: %v", err)
	}

	return bc, nil
}

// SetConsensus 设置DPoS共识
func (bc *Blockchain) SetConsensus(dpos *consensus.DPoSConsensus) {
	bc.mutex.Lock()
	defer bc.mutex.Unlock()

	bc.consensus = NewConsensusIntegration(bc, dpos)
}

// GetConsensus 获取共识集成
func (bc *Blockchain) GetConsensus() *ConsensusIntegration {
	bc.mutex.RLock()
	defer bc.mutex.RUnlock()

	return bc.consensus
}

// initialize 初始化区块链
func (bc *Blockchain) initialize() error {
	bc.mutex.Lock()
	defer bc.mutex.Unlock()

	// 检查是否已有区块
	latestBlock, err := bc.blockStore.GetLatestBlock()
	if err != nil {
		// 没有区块，存储创世区块
		if err := bc.storeGenesisBlock(); err != nil {
			return fmt.Errorf("failed to store genesis block: %v", err)
		}
		return nil
	}

	// 更新链状态
	bc.chainState.Height = latestBlock.Header.Number
	bc.chainState.BestBlockHash = latestBlock.Hash()
	bc.chainState.TotalWork = bc.calculateTotalWork(latestBlock)
	bc.chainState.LastUpdated = time.Now().Unix()

	return nil
}

// storeGenesisBlock 存储创世区块
func (bc *Blockchain) storeGenesisBlock() error {
	genesisBlock := bc.config.GenesisBlock

	// 验证创世区块
	if err := bc.validator.ValidateGenesisBlock(genesisBlock); err != nil {
		return fmt.Errorf("invalid genesis block: %v", err)
	}

	// 存储创世区块
	if err := bc.blockStore.PutBlock(genesisBlock); err != nil {
		return fmt.Errorf("failed to store genesis block: %v", err)
	}

	// 更新链状态
	bc.chainState.Height = 0
	bc.chainState.BestBlockHash = genesisBlock.Hash()
	bc.chainState.TotalWork = 1 // 创世区块工作量为1
	bc.chainState.LastUpdated = time.Now().Unix()

	return nil
}

// AddBlock 添加区块到区块链
func (bc *Blockchain) AddBlock(block *types.Block) error {
	bc.mutex.Lock()
	defer bc.mutex.Unlock()

	// 基本验证
	if block == nil {
		return fmt.Errorf("block cannot be nil")
	}

	// 检查区块是否已存在
	blockHash := block.Hash()
	if bc.blockStore.HasBlock(blockHash) {
		return fmt.Errorf("block already exists: %x", blockHash)
	}

	// 验证区块
	if err := bc.validator.ValidateNewBlock(block, nil); err != nil {
		return fmt.Errorf("block validation failed: %v", err)
	}

	// DPoS共识验证
	if bc.consensus != nil {
		if err := bc.consensus.ValidateBlockWithConsensus(block); err != nil {
			return fmt.Errorf("consensus validation failed: %v", err)
		}
	}

	// 检查区块是否连接到已知链
	if err := bc.validateBlockConnection(block); err != nil {
		return fmt.Errorf("block connection validation failed: %v", err)
	}

	// 存储区块
	if err := bc.blockStore.PutBlock(block); err != nil {
		return fmt.Errorf("failed to store block: %v", err)
	}

	// 处理链更新
	if err := bc.processBlockAddition(block); err != nil {
		return fmt.Errorf("failed to process block addition: %v", err)
	}

	return nil
}

// validateBlockConnection 验证区块连接
func (bc *Blockchain) validateBlockConnection(block *types.Block) error {
	prevHash := block.Header.PrevHash

	// 检查前一个区块是否存在
	if !bc.blockStore.HasBlock(prevHash) {
		return fmt.Errorf("previous block not found: %x", prevHash)
	}

	// 获取前一个区块
	prevBlock, err := bc.blockStore.GetBlock(prevHash)
	if err != nil {
		return fmt.Errorf("failed to get previous block: %v", err)
	}

	// 验证区块高度
	expectedHeight := prevBlock.Header.Number + 1
	if block.Header.Number != expectedHeight {
		return fmt.Errorf("invalid block height: expected %d, got %d", expectedHeight, block.Header.Number)
	}

	// 验证时间戳
	if block.Header.Timestamp <= prevBlock.Header.Timestamp {
		return fmt.Errorf("block timestamp must be greater than previous block")
	}

	return nil
}

// processBlockAddition 处理区块添加
func (bc *Blockchain) processBlockAddition(block *types.Block) error {
	// 检查是否是主链的下一个区块
	if block.Header.PrevHash.Equal(bc.chainState.BestBlockHash) {
		// 直接添加到主链
		return bc.extendMainChain(block)
	}

	// 检查是否创建了分叉
	return bc.handleFork(block)
}

// extendMainChain 扩展主链
func (bc *Blockchain) extendMainChain(block *types.Block) error {
	// 更新链状态
	bc.chainState.Height = block.Header.Number
	bc.chainState.BestBlockHash = block.Hash()
	bc.chainState.TotalWork += bc.calculateBlockWork(block)
	bc.chainState.LastUpdated = time.Now().Unix()

	// 处理区块奖励
	if bc.consensus != nil {
		if err := bc.consensus.ProcessBlockReward(block); err != nil {
			// 记录错误但不阻止区块添加
			fmt.Printf("Failed to process block reward: %v\n", err)
		}
	}

	// 触发新区块回调
	if bc.onNewBlock != nil {
		bc.onNewBlock(block)
	}

	return nil
}

// handleFork 处理分叉
func (bc *Blockchain) handleFork(block *types.Block) error {
	prevHash := block.Header.PrevHash
	forkKey := prevHash.String()

	// 检查是否是已知分叉的扩展
	if fork, exists := bc.forks[forkKey]; exists {
		// 扩展已有分叉
		fork.Blocks = append(fork.Blocks, block)
		fork.TotalWork += bc.calculateBlockWork(block)
		fork.LastUpdated = time.Now().Unix()
	} else {
		// 检查是否是分叉的扩展（从已有分叉区块开始）
		forkFromExisting := bc.findForkExtension(block)
		if forkFromExisting != nil {
			// 扩展现有分叉
			forkFromExisting.Blocks = append(forkFromExisting.Blocks, block)
			forkFromExisting.TotalWork += bc.calculateBlockWork(block)
			forkFromExisting.LastUpdated = time.Now().Unix()
		} else {
			// 创建新分叉
			fork := &Fork{
				StartHeight: block.Header.Number - 1,
				StartHash:   prevHash,
				Blocks:      []*types.Block{block},
				TotalWork:   bc.calculateBlockWork(block),
				LastUpdated: time.Now().Unix(),
			}
			bc.forks[forkKey] = fork
		}
	}

	// 检查是否需要切换到分叉链
	return bc.checkChainSwitch()
}

// findForkExtension 查找分叉扩展
func (bc *Blockchain) findForkExtension(block *types.Block) *Fork {
	for _, fork := range bc.forks {
		if len(fork.Blocks) > 0 {
			lastBlock := fork.Blocks[len(fork.Blocks)-1]
			if block.Header.PrevHash.Equal(lastBlock.Hash()) {
				return fork
			}
		}
	}
	return nil
}

// calculateBlockWork 计算区块工作量
func (bc *Blockchain) calculateBlockWork(_ *types.Block) uint64 {
	// 简单实现：每个区块工作量为1
	// 在真实实现中，这应该基于难度
	return 1
}

// calculateTotalWork 计算总工作量
func (bc *Blockchain) calculateTotalWork(block *types.Block) uint64 {
	// 简单实现：总工作量等于区块高度+1
	return block.Header.Number + 1
}

// checkChainSwitch 检查是否需要切换链
func (bc *Blockchain) checkChainSwitch() error {
	var bestFork *Fork
	var bestForkKey string
	maxWork := bc.chainState.TotalWork

	// 找到工作量最大的分叉
	for key, fork := range bc.forks {
		// 计算分叉的总工作量（包括主链到分叉点的工作量）
		forkTotalWork := bc.calculateForkTotalWork(fork)
		if forkTotalWork > maxWork {
			maxWork = forkTotalWork
			bestFork = fork
			bestForkKey = key
		}
	}

	// 如果找到更好的分叉，进行链重组
	if bestFork != nil {
		return bc.reorganizeChain(bestFork, bestForkKey)
	}

	return nil
}

// calculateForkTotalWork 计算分叉的总工作量
func (bc *Blockchain) calculateForkTotalWork(fork *Fork) uint64 {
	// 计算分叉总工作量：起始点工作量 + 分叉区块工作量
	baseWork := fork.StartHeight + 1 // 简化：工作量等于高度+1
	return baseWork + fork.TotalWork
}

// reorganizeChain 重组链
func (bc *Blockchain) reorganizeChain(newFork *Fork, forkKey string) error {
	// 获取当前主链的回滚区块
	revertBlocks, err := bc.getBlocksToRevert(newFork.StartHeight)
	if err != nil {
		return fmt.Errorf("failed to get blocks to revert: %v", err)
	}

	// 检查重组深度
	if uint64(len(revertBlocks)) > bc.config.MaxReorgDepth {
		return fmt.Errorf("reorganization too deep: %d > %d", len(revertBlocks), bc.config.MaxReorgDepth)
	}

	// 执行链重组
	lastBlock := newFork.Blocks[len(newFork.Blocks)-1]
	bc.chainState.Height = lastBlock.Header.Number
	bc.chainState.BestBlockHash = lastBlock.Hash()
	bc.chainState.TotalWork = bc.calculateForkTotalWork(newFork)
	bc.chainState.LastUpdated = time.Now().Unix()

	// 清理已处理的分叉
	delete(bc.forks, forkKey)

	// 触发链重组回调
	if bc.onChainReorg != nil {
		bc.onChainReorg(revertBlocks, newFork.Blocks)
	}

	fmt.Printf("Chain reorganized: new height %d, new best block %s\n", bc.chainState.Height, bc.chainState.BestBlockHash.String())

	return nil
}

// getBlocksToRevert 获取需要回滚的区块
func (bc *Blockchain) getBlocksToRevert(revertToHeight uint64) ([]*types.Block, error) {
	var blocksToRevert []*types.Block

	currentHeight := bc.chainState.Height
	for height := currentHeight; height > revertToHeight; height-- {
		block, err := bc.blockStore.GetBlockByHeight(height)
		if err != nil {
			return nil, fmt.Errorf("failed to get block at height %d: %v", height, err)
		}
		blocksToRevert = append(blocksToRevert, block)
	}

	return blocksToRevert, nil
}

// GetChainState 获取链状态
func (bc *Blockchain) GetChainState() *ChainState {
	bc.mutex.RLock()
	defer bc.mutex.RUnlock()

	// 返回状态副本
	return &ChainState{
		Height:        bc.chainState.Height,
		BestBlockHash: bc.chainState.BestBlockHash,
		TotalWork:     bc.chainState.TotalWork,
		LastUpdated:   bc.chainState.LastUpdated,
	}
}

// GetBestBlock 获取最佳区块
func (bc *Blockchain) GetBestBlock() (*types.Block, error) {
	bc.mutex.RLock()
	defer bc.mutex.RUnlock()

	return bc.blockStore.GetBlock(bc.chainState.BestBlockHash)
}

// GetBlock 获取区块
func (bc *Blockchain) GetBlock(blockHash types.Hash) (*types.Block, error) {
	return bc.blockStore.GetBlock(blockHash)
}

// GetBlockByHeight 根据高度获取区块
func (bc *Blockchain) GetBlockByHeight(height uint64) (*types.Block, error) {
	return bc.blockStore.GetBlockByHeight(height)
}

// HasBlock 检查区块是否存在
func (bc *Blockchain) HasBlock(blockHash types.Hash) bool {
	return bc.blockStore.HasBlock(blockHash)
}

// GetForks 获取分叉信息
func (bc *Blockchain) GetForks() map[string]*Fork {
	bc.mutex.RLock()
	defer bc.mutex.RUnlock()

	// 返回分叉副本
	forks := make(map[string]*Fork)
	for key, fork := range bc.forks {
		forks[key] = &Fork{
			StartHeight: fork.StartHeight,
			StartHash:   fork.StartHash,
			Blocks:      append([]*types.Block{}, fork.Blocks...),
			TotalWork:   fork.TotalWork,
			LastUpdated: fork.LastUpdated,
		}
	}

	return forks
}

// SetCallbacks 设置回调函数
func (bc *Blockchain) SetCallbacks(
	onNewBlock func(*types.Block),
	onChainReorg func([]*types.Block, []*types.Block),
	onSyncComplete func(),
) {
	bc.onNewBlock = onNewBlock
	bc.onChainReorg = onChainReorg
	bc.onSyncComplete = onSyncComplete
}

// IsConfirmed 检查区块是否已确认
func (bc *Blockchain) IsConfirmed(blockHash types.Hash) bool {
	bc.mutex.RLock()
	defer bc.mutex.RUnlock()

	block, err := bc.blockStore.GetBlock(blockHash)
	if err != nil {
		return false
	}

	// 计算确认数
	confirmations := bc.chainState.Height - block.Header.Number
	return confirmations >= bc.config.ConfirmBlocks
}

// GetConfirmations 获取区块确认数
func (bc *Blockchain) GetConfirmations(blockHash types.Hash) uint64 {
	bc.mutex.RLock()
	defer bc.mutex.RUnlock()

	block, err := bc.blockStore.GetBlock(blockHash)
	if err != nil {
		return 0
	}

	if block.Header.Number > bc.chainState.Height {
		return 0
	}

	return bc.chainState.Height - block.Header.Number
}
