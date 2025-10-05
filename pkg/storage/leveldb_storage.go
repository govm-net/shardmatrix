package storage

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"time"

	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
	"github.com/lengzhao/shardmatrix/pkg/types"
)

// LevelDBStorage LevelDB存储实现
type LevelDBStorage struct {
	db              *leveldb.DB
	// 新增：性能统计和检查点管理
	stats           *StorageStats
	checkpointMgr   *CheckpointManager
	batchQueue      chan *BatchJob
	isRunning       bool
}

// StorageStats 存储统计信息
type StorageStats struct {
	ReadsTotal     int64 // 总读取次数
	WritesTotal    int64 // 总写入次数
	BytesRead      int64 // 读取字节数
	BytesWritten   int64 // 写入字节数
	BatchesTotal   int64 // 批量操作次数
	CompactionsTotal int64 // 压缩次数
	LastCompaction int64 // 最后压缩时间
}

// CheckpointManager 检查点管理器
type CheckpointManager struct {
	storage         *LevelDBStorage
	checkpointHeight uint64
	checkpointHash   types.Hash
	lastCheckpoint   int64
	checkpointInterval uint64 // 检查点间隔（区块数）
}

// BatchJob 批量作业
type BatchJob struct {
	Operations []BatchOperation
	Callback   func(error)
}

// Checkpoint 检查点数据结构
type Checkpoint struct {
	Height       uint64    `json:"height"`        // 检查点高度
	BlockHash    types.Hash `json:"block_hash"`    // 区块哈希
	StateRoot    types.Hash `json:"state_root"`    // 状态根哈希
	ValidatorSetHash types.Hash `json:"validator_set_hash"` // 验证者集合哈希
	Timestamp    int64     `json:"timestamp"`     // 创建时间
	Signature    types.Signature `json:"signature"`    // 多重签名（简化）
}

// NewLevelDBStorage 创建新的LevelDB存储
func NewLevelDBStorage(dataDir string) (*LevelDBStorage, error) {
	db, err := leveldb.OpenFile(dataDir, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to open leveldb: %w", err)
	}

	// 初始化存储统计
	stats := &StorageStats{}

	// 初始化检查点管理器
	checkpointMgr := &CheckpointManager{
		checkpointInterval: 100, // 每100个区块一个检查点
	}

	storage := &LevelDBStorage{
		db:            db,
		stats:         stats,
		checkpointMgr: checkpointMgr,
		batchQueue:    make(chan *BatchJob, 100),
		isRunning:     true,
	}

	// 设置检查点管理器的存储引用
	checkpointMgr.storage = storage

	// 启动批量处理协程
	go storage.batchProcessor()

	return storage, nil
}

// batchProcessor 批量处理器
func (s *LevelDBStorage) batchProcessor() {
	for {
		select {
		case job := <-s.batchQueue:
			if job == nil {
				return // 停止信号
			}
			err := s.executeBatch(job.Operations)
			if job.Callback != nil {
				job.Callback(err)
			}
		}
	}
}

// executeBatch 执行批量操作
func (s *LevelDBStorage) executeBatch(operations []BatchOperation) error {
	batch := new(leveldb.Batch)
	
	for _, op := range operations {
		switch op.Type {
		case BatchOpPut:
			batch.Put([]byte(op.Key), op.Value)
			s.stats.BytesWritten += int64(len(op.Key)) + int64(len(op.Value))
		case BatchOpDelete:
			batch.Delete([]byte(op.Key))
		}
	}
	
	err := s.db.Write(batch, nil)
	if err == nil {
		s.stats.BatchesTotal++
		s.stats.WritesTotal += int64(len(operations))
	}
	
	return err
}

// =============== 检查点管理方法 ===============

// ShouldCreateCheckpoint 判断是否应该创建检查点
func (cm *CheckpointManager) ShouldCreateCheckpoint(blockHeight uint64) bool {
	return blockHeight > 0 && blockHeight%cm.checkpointInterval == 0
}

// createCheckpoint 创建检查点
func (s *LevelDBStorage) createCheckpoint(block *types.Block) {
	checkpoint := &Checkpoint{
		Height:           block.Header.Number,
		BlockHash:        block.Hash(),
		StateRoot:        block.Header.StateRoot,
		ValidatorSetHash: types.Hash{}, // 简化实现
		Timestamp:        time.Now().Unix(),
		Signature:        types.Signature{}, // 简化实现
	}

	err := s.saveCheckpoint(checkpoint)
	if err != nil {
		fmt.Printf("Failed to save checkpoint at height %d: %v\n", block.Header.Number, err)
		return
	}

	// 更新检查点管理器状态
	s.checkpointMgr.checkpointHeight = block.Header.Number
	s.checkpointMgr.checkpointHash = block.Hash()
	s.checkpointMgr.lastCheckpoint = time.Now().Unix()

	fmt.Printf("✓ Created checkpoint at height %d (hash: %s)\n", 
		block.Header.Number, block.Hash().String()[:8])
}

// saveCheckpoint 保存检查点
func (s *LevelDBStorage) saveCheckpoint(checkpoint *Checkpoint) error {
	key := fmt.Sprintf("%s%d", checkpointPrefix, checkpoint.Height)
	
	data, err := json.Marshal(checkpoint)
	if err != nil {
		return fmt.Errorf("failed to serialize checkpoint: %w", err)
	}
	
	err = s.db.Put([]byte(key), data, nil)
	if err != nil {
		return fmt.Errorf("failed to save checkpoint: %w", err)
	}
	
	// 更新最新检查点高度
	latestKey := metaPrefix + "latest_checkpoint"
	heightData := make([]byte, 8)
	binary.BigEndian.PutUint64(heightData, checkpoint.Height)
	
	return s.db.Put([]byte(latestKey), heightData, nil)
}

// GetCheckpoint 获取检查点
func (s *LevelDBStorage) GetCheckpoint(height uint64) (*Checkpoint, error) {
	key := fmt.Sprintf("%s%d", checkpointPrefix, height)
	
	data, err := s.db.Get([]byte(key), nil)
	if err != nil {
		if err == leveldb.ErrNotFound {
			return nil, fmt.Errorf("checkpoint not found at height %d", height)
		}
		return nil, fmt.Errorf("failed to get checkpoint: %w", err)
	}
	
	checkpoint := &Checkpoint{}
	err = json.Unmarshal(data, checkpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize checkpoint: %w", err)
	}
	
	return checkpoint, nil
}

// GetLatestCheckpoint 获取最新检查点
func (s *LevelDBStorage) GetLatestCheckpoint() (*Checkpoint, error) {
	latestKey := metaPrefix + "latest_checkpoint"
	
	data, err := s.db.Get([]byte(latestKey), nil)
	if err != nil {
		if err == leveldb.ErrNotFound {
			return nil, fmt.Errorf("no checkpoint found")
		}
		return nil, fmt.Errorf("failed to get latest checkpoint height: %w", err)
	}
	
	height := binary.BigEndian.Uint64(data)
	return s.GetCheckpoint(height)
}

// Close 关闭存储
func (s *LevelDBStorage) Close() error {
	s.isRunning = false
	
	// 停止批量处理器
	close(s.batchQueue)
	
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
	checkpointPrefix = "checkpoint:" // 新增：检查点前缀
	healthPrefix    = "health:"     // 新增：健康信息前缀
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
	
	// 更新统计
	s.stats.WritesTotal++
	s.stats.BytesWritten += int64(len(data))
	
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
	
	// 检查是否需要创建检查点
	if s.checkpointMgr.ShouldCreateCheckpoint(block.Header.Number) {
		go s.createCheckpoint(block)
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
		"type":                "leveldb",
		"compactions":         "enabled",
		"compression":         "snappy",
		"reads_total":         s.stats.ReadsTotal,
		"writes_total":        s.stats.WritesTotal,
		"bytes_read":          s.stats.BytesRead,
		"bytes_written":       s.stats.BytesWritten,
		"batches_total":       s.stats.BatchesTotal,
		"compactions_total":   s.stats.CompactionsTotal,
		"last_compaction":     s.stats.LastCompaction,
		"checkpoint_height":   s.checkpointMgr.checkpointHeight,
		"last_checkpoint":     s.checkpointMgr.lastCheckpoint,
		"checkpoint_interval": s.checkpointMgr.checkpointInterval,
		"batch_queue_size":    len(s.batchQueue),
		"is_running":          s.isRunning,
	}
}

// =============== 异步批量写入方法 ===============

// AsyncBatchWrite 异步批量写入
func (s *LevelDBStorage) AsyncBatchWrite(operations []BatchOperation, callback func(error)) error {
	if !s.isRunning {
		return fmt.Errorf("storage is not running")
	}

	job := &BatchJob{
		Operations: operations,
		Callback:   callback,
	}

	select {
	case s.batchQueue <- job:
		return nil
	default:
		return fmt.Errorf("batch queue is full")
	}
}

// CompactStorage 手动触发存储压缩
func (s *LevelDBStorage) CompactStorage() error {
	// 使用正确的leveldb压缩方法
	err := s.db.CompactRange(util.Range{})
	if err == nil {
		s.stats.CompactionsTotal++
		s.stats.LastCompaction = time.Now().Unix()
	}
	return err
}

// SetCheckpointInterval 设置检查点间隔
func (s *LevelDBStorage) SetCheckpointInterval(interval uint64) {
	s.checkpointMgr.checkpointInterval = interval
}

// GetStorageHealth 获取存储健康状态
func (s *LevelDBStorage) GetStorageHealth() map[string]interface{} {
	// 简化的健康检查
	health := map[string]interface{}{
		"status":              "healthy",
		"read_write_ok":       true,
		"checkpoint_current":  s.checkpointMgr.checkpointHeight > 0,
		"batch_queue_healthy": len(s.batchQueue) < 80, // 80%阈值
	}

	// 测试读写操作
	testKey := "health_check_test"
	testValue := []byte("test")

	err := s.db.Put([]byte(testKey), testValue, nil)
	if err != nil {
		health["status"] = "unhealthy"
		health["read_write_ok"] = false
		health["error"] = err.Error()
		return health
	}

	_, err = s.db.Get([]byte(testKey), nil)
	if err != nil {
		health["status"] = "unhealthy"
		health["read_write_ok"] = false
		health["error"] = err.Error()
	}

	// 清理测试数据
	s.db.Delete([]byte(testKey), nil)

	return health
}