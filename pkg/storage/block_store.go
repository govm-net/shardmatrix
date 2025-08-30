package storage

import (
	"fmt"

	"github.com/govm-net/shardmatrix/pkg/types"
)

// BlockStore 区块存储接口
type BlockStore interface {
	// PutBlock 存储区块
	PutBlock(block *types.Block) error

	// GetBlock 获取区块
	GetBlock(blockHash []byte) (*types.Block, error)

	// GetBlockByHeight 根据高度获取区块
	GetBlockByHeight(height uint64) (*types.Block, error)

	// HasBlock 检查区块是否存在
	HasBlock(blockHash []byte) bool

	// GetLatestBlock 获取最新区块
	GetLatestBlock() (*types.Block, error)

	// Close 关闭存储
	Close() error
}

// MemoryBlockStore 内存区块存储（用于测试）
type MemoryBlockStore struct {
	blocks     map[string]*types.Block // 按哈希存储
	heights    map[uint64][]byte       // 按高度存储
	latestHash []byte
}

// NewMemoryBlockStore 创建内存区块存储
func NewMemoryBlockStore() *MemoryBlockStore {
	return &MemoryBlockStore{
		blocks:  make(map[string]*types.Block),
		heights: make(map[uint64][]byte),
	}
}

// PutBlock 存储区块
func (mbs *MemoryBlockStore) PutBlock(block *types.Block) error {
	blockHash := block.Hash()
	hashStr := fmt.Sprintf("%x", blockHash)

	// 存储区块
	mbs.blocks[hashStr] = block

	// 存储高度映射
	mbs.heights[block.Header.Number] = blockHash

	// 更新最新区块
	if mbs.latestHash == nil || block.Header.Number > mbs.getLatestHeight() {
		mbs.latestHash = blockHash
	}

	return nil
}

// GetBlock 获取区块
func (mbs *MemoryBlockStore) GetBlock(blockHash []byte) (*types.Block, error) {
	hashStr := fmt.Sprintf("%x", blockHash)
	block, exists := mbs.blocks[hashStr]
	if !exists {
		return nil, fmt.Errorf("block not found: %x", blockHash)
	}
	return block, nil
}

// GetBlockByHeight 根据高度获取区块
func (mbs *MemoryBlockStore) GetBlockByHeight(height uint64) (*types.Block, error) {
	blockHash, exists := mbs.heights[height]
	if !exists {
		return nil, fmt.Errorf("block not found at height: %d", height)
	}
	return mbs.GetBlock(blockHash)
}

// HasBlock 检查区块是否存在
func (mbs *MemoryBlockStore) HasBlock(blockHash []byte) bool {
	hashStr := fmt.Sprintf("%x", blockHash)
	_, exists := mbs.blocks[hashStr]
	return exists
}

// GetLatestBlock 获取最新区块
func (mbs *MemoryBlockStore) GetLatestBlock() (*types.Block, error) {
	if mbs.latestHash == nil {
		return nil, fmt.Errorf("no blocks in store")
	}
	return mbs.GetBlock(mbs.latestHash)
}

// getLatestHeight 获取最新区块高度
func (mbs *MemoryBlockStore) getLatestHeight() uint64 {
	if mbs.latestHash == nil {
		return 0
	}

	block, err := mbs.GetBlock(mbs.latestHash)
	if err != nil {
		return 0
	}
	return block.Header.Number
}

// Close 关闭存储
func (mbs *MemoryBlockStore) Close() error {
	// 内存存储不需要特殊清理
	return nil
}
