package storage

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"

	"github.com/syndtr/goleveldb/leveldb"
	"github.com/lengzhao/shardmatrix/pkg/types"
)

// LevelDBStorage LevelDB存储实现
type LevelDBStorage struct {
	db *leveldb.DB
}

// NewLevelDBStorage 创建新的LevelDB存储
func NewLevelDBStorage(dataDir string) (*LevelDBStorage, error) {
	db, err := leveldb.OpenFile(dataDir, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to open leveldb: %w", err)
	}
	
	return &LevelDBStorage{
		db: db,
	}, nil
}

// Close 关闭存储
func (s *LevelDBStorage) Close() error {
	return s.db.Close()
}

// 键前缀定义
const (
	blockPrefix     = "block:"
	txPrefix        = "tx:"
	accountPrefix   = "account:"
	validatorPrefix = "validator:"
	genesisPrefix   = "genesis:"
	metaPrefix      = "meta:"
)

// SaveBlock 保存区块
func (s *LevelDBStorage) SaveBlock(block *types.Block) error {
	key := fmt.Sprintf("%s%d", blockPrefix, block.Header.Number)
	
	// 序列化区块
	data, err := json.Marshal(block)
	if err != nil {
		return fmt.Errorf("failed to serialize block: %w", err)
	}
	
	// 保存区块数据
	err = s.db.Put([]byte(key), data, nil)
	if err != nil {
		return fmt.Errorf("failed to save block: %w", err)
	}
	
	// 保存区块哈希到高度的映射
	hashKey := fmt.Sprintf("%shash:%s", blockPrefix, block.Hash().String())
	heightData := make([]byte, 8)
	binary.BigEndian.PutUint64(heightData, block.Header.Number)
	
	err = s.db.Put([]byte(hashKey), heightData, nil)
	if err != nil {
		return fmt.Errorf("failed to save block hash mapping: %w", err)
	}
	
	// 更新最新区块高度
	err = s.UpdateLatestBlockHeight(block.Header.Number)
	if err != nil {
		return fmt.Errorf("failed to update latest height: %w", err)
	}
	
	return nil
}

// GetBlock 根据高度获取区块
func (s *LevelDBStorage) GetBlock(height uint64) (*types.Block, error) {
	key := fmt.Sprintf("%s%d", blockPrefix, height)
	
	data, err := s.db.Get([]byte(key), nil)
	if err != nil {
		if err == leveldb.ErrNotFound {
			return nil, fmt.Errorf("block not found at height %d", height)
		}
		return nil, fmt.Errorf("failed to get block: %w", err)
	}
	
	// 反序列化区块
	block := &types.Block{}
	err = json.Unmarshal(data, block)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize block: %w", err)
	}
	
	return block, nil
}

// GetBlockByHash 根据哈希获取区块
func (s *LevelDBStorage) GetBlockByHash(hash types.Hash) (*types.Block, error) {
	// 先获取高度
	hashKey := fmt.Sprintf("%shash:%s", blockPrefix, hash.String())
	heightData, err := s.db.Get([]byte(hashKey), nil)
	if err != nil {
		if err == leveldb.ErrNotFound {
			return nil, fmt.Errorf("block not found with hash %s", hash.String())
		}
		return nil, fmt.Errorf("failed to get block height: %w", err)
	}
	
	height := binary.BigEndian.Uint64(heightData)
	return s.GetBlock(height)
}

// SaveTransaction 保存交易
func (s *LevelDBStorage) SaveTransaction(tx *types.Transaction) error {
	key := fmt.Sprintf("%s%s", txPrefix, tx.Hash().String())
	
	// 序列化交易
	data, err := json.Marshal(tx)
	if err != nil {
		return fmt.Errorf("failed to serialize transaction: %w", err)
	}
	
	// 保存交易数据
	err = s.db.Put([]byte(key), data, nil)
	if err != nil {
		return fmt.Errorf("failed to save transaction: %w", err)
	}
	
	return nil
}

// GetTransaction 获取交易
func (s *LevelDBStorage) GetTransaction(hash types.Hash) (*types.Transaction, error) {
	key := fmt.Sprintf("%s%s", txPrefix, hash.String())
	
	data, err := s.db.Get([]byte(key), nil)
	if err != nil {
		if err == leveldb.ErrNotFound {
			return nil, fmt.Errorf("transaction not found: %s", hash.String())
		}
		return nil, fmt.Errorf("failed to get transaction: %w", err)
	}
	
	// 反序列化交易
	tx := &types.Transaction{}
	err = json.Unmarshal(data, tx)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize transaction: %w", err)
	}
	
	return tx, nil
}

// SaveAccount 保存账户
func (s *LevelDBStorage) SaveAccount(account *types.Account) error {
	key := fmt.Sprintf("%s%s", accountPrefix, account.Address.String())
	
	// 序列化账户
	data, err := json.Marshal(account)
	if err != nil {
		return fmt.Errorf("failed to serialize account: %w", err)
	}
	
	// 保存账户数据
	err = s.db.Put([]byte(key), data, nil)
	if err != nil {
		return fmt.Errorf("failed to save account: %w", err)
	}
	
	return nil
}

// GetAccount 获取账户
func (s *LevelDBStorage) GetAccount(address types.Address) (*types.Account, error) {
	key := fmt.Sprintf("%s%s", accountPrefix, address.String())
	
	data, err := s.db.Get([]byte(key), nil)
	if err != nil {
		if err == leveldb.ErrNotFound {
			return nil, fmt.Errorf("account not found: %s", address.String())
		}
		return nil, fmt.Errorf("failed to get account: %w", err)
	}
	
	// 反序列化账户
	account := &types.Account{}
	err = json.Unmarshal(data, account)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize account: %w", err)
	}
	
	return account, nil
}

// SaveValidator 保存验证者
func (s *LevelDBStorage) SaveValidator(validator *types.Validator) error {
	key := fmt.Sprintf("%s%s", validatorPrefix, validator.Address.String())
	
	// 序列化验证者
	data, err := json.Marshal(validator)
	if err != nil {
		return fmt.Errorf("failed to serialize validator: %w", err)
	}
	
	// 保存验证者数据
	err = s.db.Put([]byte(key), data, nil)
	if err != nil {
		return fmt.Errorf("failed to save validator: %w", err)
	}
	
	return nil
}

// GetValidator 获取验证者
func (s *LevelDBStorage) GetValidator(address types.Address) (*types.Validator, error) {
	key := fmt.Sprintf("%s%s", validatorPrefix, address.String())
	
	data, err := s.db.Get([]byte(key), nil)
	if err != nil {
		if err == leveldb.ErrNotFound {
			return nil, fmt.Errorf("validator not found: %s", address.String())
		}
		return nil, fmt.Errorf("failed to get validator: %w", err)
	}
	
	// 反序列化验证者
	validator := &types.Validator{}
	err = json.Unmarshal(data, validator)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize validator: %w", err)
	}
	
	return validator, nil
}

// GetAllValidators 获取所有验证者
func (s *LevelDBStorage) GetAllValidators() ([]*types.Validator, error) {
	var validators []*types.Validator
	
	// 遍历所有验证者
	iter := s.db.NewIterator(nil, nil)
	defer iter.Release()
	
	prefix := []byte(validatorPrefix)
	for iter.Seek(prefix); iter.Valid() && bytes.HasPrefix(iter.Key(), prefix); iter.Next() {
		// 反序列化验证者
		validator := &types.Validator{}
		err := json.Unmarshal(iter.Value(), validator)
		if err != nil {
			continue // 跳过损坏的数据
		}
		validators = append(validators, validator)
	}
	
	if err := iter.Error(); err != nil {
		return nil, fmt.Errorf("iterator error: %w", err)
	}
	
	return validators, nil
}

// SaveGenesisBlock 保存创世区块配置
func (s *LevelDBStorage) SaveGenesisBlock(genesis *types.GenesisBlock) error {
	key := genesisPrefix + "config"
	
	// 序列化创世配置
	data, err := json.Marshal(genesis)
	if err != nil {
		return fmt.Errorf("failed to serialize genesis: %w", err)
	}
	
	// 保存创世配置
	err = s.db.Put([]byte(key), data, nil)
	if err != nil {
		return fmt.Errorf("failed to save genesis: %w", err)
	}
	
	return nil
}

// GetGenesisBlock 获取创世区块配置
func (s *LevelDBStorage) GetGenesisBlock() (*types.GenesisBlock, error) {
	key := genesisPrefix + "config"
	
	data, err := s.db.Get([]byte(key), nil)
	if err != nil {
		if err == leveldb.ErrNotFound {
			return nil, fmt.Errorf("genesis block not found")
		}
		return nil, fmt.Errorf("failed to get genesis: %w", err)
	}
	
	// 反序列化创世配置
	genesis := &types.GenesisBlock{}
	err = json.Unmarshal(data, genesis)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize genesis: %w", err)
	}
	
	return genesis, nil
}

// GetLatestBlockHeight 获取最新区块高度
func (s *LevelDBStorage) GetLatestBlockHeight() (uint64, error) {
	key := metaPrefix + "latest_height"
	
	data, err := s.db.Get([]byte(key), nil)
	if err != nil {
		if err == leveldb.ErrNotFound {
			return 0, nil // 没有区块时返回0
		}
		return 0, fmt.Errorf("failed to get latest height: %w", err)
	}
	
	height := binary.BigEndian.Uint64(data)
	return height, nil
}

// UpdateLatestBlockHeight 更新最新区块高度
func (s *LevelDBStorage) UpdateLatestBlockHeight(height uint64) error {
	key := metaPrefix + "latest_height"
	
	data := make([]byte, 8)
	binary.BigEndian.PutUint64(data, height)
	
	err := s.db.Put([]byte(key), data, nil)
	if err != nil {
		return fmt.Errorf("failed to update latest height: %w", err)
	}
	
	return nil
}

// GetBlockByHeight 根据高度获取区块（为了兼容接口）
func (s *LevelDBStorage) GetBlockByHeight(height uint64) (*types.Block, error) {
	return s.GetBlock(height)
}

// HasBlock 检查区块是否存在
func (s *LevelDBStorage) HasBlock(hash types.Hash) bool {
	hashKey := fmt.Sprintf("%shash:%s", blockPrefix, hash.String())
	has, err := s.db.Has([]byte(hashKey), nil)
	if err != nil {
		return false
	}
	return has
}

// HasTransaction 检查交易是否存在
func (s *LevelDBStorage) HasTransaction(hash types.Hash) bool {
	key := fmt.Sprintf("%s%s", txPrefix, hash.String())
	has, err := s.db.Has([]byte(key), nil)
	if err != nil {
		return false
	}
	return has
}

// HasAccount 检查账户是否存在
func (s *LevelDBStorage) HasAccount(address types.Address) bool {
	key := fmt.Sprintf("%s%s", accountPrefix, address.String())
	has, err := s.db.Has([]byte(key), nil)
	if err != nil {
		return false
	}
	return has
}

// BatchWrite 批量写入操作
func (s *LevelDBStorage) BatchWrite(operations []BatchOperation) error {
	batch := new(leveldb.Batch)
	
	for _, op := range operations {
		switch op.Type {
		case BatchOpPut:
			batch.Put([]byte(op.Key), op.Value)
		case BatchOpDelete:
			batch.Delete([]byte(op.Key))
		}
	}
	
	err := s.db.Write(batch, nil)
	if err != nil {
		return fmt.Errorf("batch write failed: %w", err)
	}
	
	return nil
}

// BatchOperationType 批量操作类型
type BatchOperationType int

const (
	BatchOpPut BatchOperationType = iota
	BatchOpDelete
)

// BatchOperation 批量操作
type BatchOperation struct {
	Type  BatchOperationType
	Key   string
	Value []byte
}

// Has 检查键是否存在
func (s *LevelDBStorage) Has(key string) (bool, error) {
	return s.db.Has([]byte(key), nil)
}

// Delete 删除键
func (s *LevelDBStorage) Delete(key string) error {
	return s.db.Delete([]byte(key), nil)
}

// GetStats 获取存储统计信息
func (s *LevelDBStorage) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"type":        "leveldb",
		"compactions": "enabled",
		"compression": "snappy",
	}
}