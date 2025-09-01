package storage

import (
	"fmt"
	"path/filepath"

	"github.com/govm-net/shardmatrix/pkg/types"
	"github.com/syndtr/goleveldb/leveldb"
)

// LevelDBBlockStore LevelDB区块存储实现
type LevelDBBlockStore struct {
	db *leveldb.DB
}

// NewLevelDBBlockStore 创建LevelDB区块存储
func NewLevelDBBlockStore(dbPath string) (*LevelDBBlockStore, error) {
	// 确保数据库路径存在
	blockDBPath := filepath.Join(dbPath, "blocks")
	db, err := leveldb.OpenFile(blockDBPath, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to open block store: %v", err)
	}

	return &LevelDBBlockStore{
		db: db,
	}, nil
}

// PutBlock 存储区块
func (lbs *LevelDBBlockStore) PutBlock(block *types.Block) error {
	blockHash := block.Hash()

	// 使用JSON序列化区块
	data, err := block.Serialize()
	if err != nil {
		return fmt.Errorf("failed to serialize block: %v", err)
	}

	// 存储区块数据，使用区块哈希作为键
	blockKey := fmt.Sprintf("block:%s", blockHash.String())
	if err := lbs.db.Put([]byte(blockKey), data, nil); err != nil {
		return fmt.Errorf("failed to store block: %v", err)
	}

	// 存储高度到哈希的映射，便于按高度查询
	heightKey := fmt.Sprintf("height:%d", block.Header.Number)
	if err := lbs.db.Put([]byte(heightKey), []byte(blockHash.String()), nil); err != nil {
		return fmt.Errorf("failed to store height mapping: %v", err)
	}

	// 更新最新区块信息
	latestKey := "latest"
	latestData, err := lbs.db.Get([]byte(latestKey), nil)
	if err != nil || len(latestData) == 0 {
		// 如果没有最新区块信息或获取失败，直接设置
		if err := lbs.db.Put([]byte(latestKey), []byte(blockHash.String()), nil); err != nil {
			return fmt.Errorf("failed to update latest block: %v", err)
		}
	} else {
		// 检查是否需要更新最新区块（高度更高）
		latestHash, err := types.HashFromString(string(latestData))
		if err != nil {
			// 如果解析失败，直接更新为新区块
			if err := lbs.db.Put([]byte(latestKey), []byte(blockHash.String()), nil); err != nil {
				return fmt.Errorf("failed to update latest block: %v", err)
			}
		} else {
			// 获取当前最新区块的高度进行比较
			latestBlock, err := lbs.GetBlock(latestHash)
			if err == nil && block.Header.Number > latestBlock.Header.Number {
				if err := lbs.db.Put([]byte(latestKey), []byte(blockHash.String()), nil); err != nil {
					return fmt.Errorf("failed to update latest block: %v", err)
				}
			}
		}
	}

	return nil
}

// GetBlock 获取区块
func (lbs *LevelDBBlockStore) GetBlock(blockHash types.Hash) (*types.Block, error) {
	key := fmt.Sprintf("block:%s", blockHash.String())
	data, err := lbs.db.Get([]byte(key), nil)
	if err != nil {
		if err == leveldb.ErrNotFound {
			return nil, fmt.Errorf("block not found: %x", blockHash)
		}
		return nil, fmt.Errorf("failed to get block: %v", err)
	}

	// 使用JSON反序列化区块
	block, err := types.DeserializeBlock(data)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize block: %v", err)
	}

	return block, nil
}

// GetBlockByHeight 根据高度获取区块
func (lbs *LevelDBBlockStore) GetBlockByHeight(height uint64) (*types.Block, error) {
	// 先通过高度获取区块哈希
	heightKey := fmt.Sprintf("height:%d", height)
	hashData, err := lbs.db.Get([]byte(heightKey), nil)
	if err != nil {
		if err == leveldb.ErrNotFound {
			return nil, fmt.Errorf("block not found at height: %d", height)
		}
		return nil, fmt.Errorf("failed to get block hash by height: %v", err)
	}

	// 使用哈希获取区块
	blockHash, err := types.HashFromString(string(hashData))
	if err != nil {
		return nil, fmt.Errorf("failed to parse block hash: %v", err)
	}
	return lbs.GetBlock(blockHash)
}

// HasBlock 检查区块是否存在
func (lbs *LevelDBBlockStore) HasBlock(blockHash types.Hash) bool {
	key := fmt.Sprintf("block:%s", blockHash.String())
	_, err := lbs.db.Get([]byte(key), nil)
	return err == nil
}

// GetLatestBlock 获取最新区块
func (lbs *LevelDBBlockStore) GetLatestBlock() (*types.Block, error) {
	latestKey := "latest"
	hashData, err := lbs.db.Get([]byte(latestKey), nil)
	if err != nil {
		if err == leveldb.ErrNotFound {
			return nil, fmt.Errorf("no blocks in store")
		}
		return nil, fmt.Errorf("failed to get latest block info: %v", err)
	}

	blockHash, err := types.HashFromString(string(hashData))
	if err != nil {
		return nil, fmt.Errorf("failed to parse latest block hash: %v", err)
	}
	if blockHash.IsZero() {
		return nil, fmt.Errorf("no blocks in store")
	}

	return lbs.GetBlock(blockHash)
}

// Close 关闭存储
func (lbs *LevelDBBlockStore) Close() error {
	return lbs.db.Close()
}
