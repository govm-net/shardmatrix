package blockchain

import (
	"fmt"
	"sync"
	"time"

	"github.com/lengzhao/shardmatrix/pkg/crypto"
	"github.com/lengzhao/shardmatrix/pkg/storage"
	"github.com/lengzhao/shardmatrix/pkg/types"
)

// BlockProducer 区块生产器
type BlockProducer struct {
	mu           sync.RWMutex
	keyPair      *crypto.KeyPair     // 验证者密钥对
	storage      storage.Storage     // 存储接口
	txPool       TxPool              // 交易池接口
	stateManager StateManager        // 状态管理器接口
	isRunning    bool               // 是否正在运行
	stopCh       chan struct{}      // 停止信号
	blockCh      chan *types.Block  // 新区块通道
	// 新增：空区块生成策略
	emptyBlockConfig *EmptyBlockConfig // 空区块配置
	networkChecker   NetworkChecker    // 网络检查器
}

// EmptyBlockConfig 空区块配置
type EmptyBlockConfig struct {
	Enabled              bool          // 是否启用空区块
	MaxEmptyInterval     time.Duration // 最大空区块间隔
	NetworkPartitionMode bool          // 网络分区模式
	SystemValidator      types.Address // 系统默认占位地址
}

// NetworkChecker 网络检查器接口
type NetworkChecker interface {
	IsInSafetyMode() bool                    // 是否处于安全模式
	GetActiveValidatorRatio() float64        // 获取活跃验证者比例
	IsCurrentValidator(types.Address, uint64) bool // 是否为当前验证者
}

// EmptyBlockTrigger 空区块触发原因
type EmptyBlockTrigger int

const (
	TriggerNoTransactions EmptyBlockTrigger = iota // 无交易
	TriggerSafetyMode                               // 安全模式
	TriggerNetworkPartition                         // 网络分区
	TriggerValidatorInactive                        // 验证者非活跃
	TriggerTransactionFailure                       // 交易处理失败
)

// String 返回触发原因的字符串表示
func (t EmptyBlockTrigger) String() string {
	switch t {
	case TriggerNoTransactions:
		return "NO_TRANSACTIONS"
	case TriggerSafetyMode:
		return "SAFETY_MODE"
	case TriggerNetworkPartition:
		return "NETWORK_PARTITION"
	case TriggerValidatorInactive:
		return "VALIDATOR_INACTIVE"
	case TriggerTransactionFailure:
		return "TRANSACTION_FAILURE"
	default:
		return "UNKNOWN"
	}
}

// TxPool 交易池接口
type TxPool interface {
	GetPendingTransactions(maxCount int) ([]*types.Transaction, error)
	RemoveTransactions(txHashes []types.Hash) error
	GetTransactionCount() int
}

// StateManager 状态管理器接口
type StateManager interface {
	GetCurrentStateRoot() types.Hash
	GetLastBlock() (*types.Block, error)
	GetBlockByHeight(height uint64) (*types.Block, error)
	GetAccount(address types.Address) (*types.Account, error)
	ProcessTransactions(txs []*types.Transaction) (types.Hash, error)
}

// BlockProducerConfig 区块生产器配置
type BlockProducerConfig struct {
	MaxTxPerBlock     int           // 每个区块最大交易数
	MinBlockInterval  time.Duration // 最小区块间隔
	MaxBlockSize      int           // 最大区块大小
	EnableEmptyBlocks bool          // 是否启用空区块
}

// NewBlockProducer 创建新的区块生产器
func NewBlockProducer(
	keyPair *crypto.KeyPair,
	storage storage.Storage,
	txPool *TransactionPool,
	stateManager StateManager,
) *BlockProducer {
	return &BlockProducer{
		keyPair:      keyPair,
		storage:      storage,
		txPool:       txPool,
		stateManager: stateManager,
		stopCh:       make(chan struct{}),
		blockCh:      make(chan *types.Block, 10),
		// 初始化空区块配置
		emptyBlockConfig: &EmptyBlockConfig{
			Enabled:              true,
			MaxEmptyInterval:     30 * time.Second, // 30秒最大空区块间隔
			NetworkPartitionMode: false,
			SystemValidator:      types.Address{}, // 系统默认地址
		},
	}
}

// SetNetworkChecker 设置网络检查器
func (bp *BlockProducer) SetNetworkChecker(checker NetworkChecker) {
	bp.mu.Lock()
	defer bp.mu.Unlock()
	bp.networkChecker = checker
}

// SetEmptyBlockConfig 设置空区块配置
func (bp *BlockProducer) SetEmptyBlockConfig(config *EmptyBlockConfig) {
	bp.mu.Lock()
	defer bp.mu.Unlock()
	bp.emptyBlockConfig = config
}

// Start 启动区块生产器
func (bp *BlockProducer) Start() error {
	bp.mu.Lock()
	defer bp.mu.Unlock()
	
	if bp.isRunning {
		return fmt.Errorf("block producer already running")
	}
	
	bp.isRunning = true
	bp.stopCh = make(chan struct{})
	
	return nil
}

// Stop 停止区块生产器
func (bp *BlockProducer) Stop() {
	bp.mu.Lock()
	defer bp.mu.Unlock()
	
	if !bp.isRunning {
		return
	}
	
	bp.isRunning = false
	close(bp.stopCh)
}

// ProduceBlock 生产区块（由时间控制器触发）
func (bp *BlockProducer) ProduceBlock(blockTime int64, blockNumber uint64) (*types.Block, error) {
	bp.mu.Lock()
	defer bp.mu.Unlock()
	
	if !bp.isRunning {
		return nil, fmt.Errorf("block producer not running")
	}
	
	// 获取前一个区块
	var prevHash types.Hash
	if blockNumber > 0 {
		prevBlock, err := bp.stateManager.GetBlockByHeight(blockNumber - 1)
		if err != nil {
			return nil, fmt.Errorf("failed to get previous block: %w", err)
		}
		prevHash = prevBlock.Hash()
	}
	
	// 获取当前状态根
	stateRoot := bp.stateManager.GetCurrentStateRoot()
	
	// 检查是否需要生成空区块
	emptyTrigger, shouldCreateEmpty := bp.shouldCreateEmptyBlock(blockNumber)
	if shouldCreateEmpty {
		return bp.createEmptyBlockWithReason(blockTime, blockNumber, prevHash, stateRoot, emptyTrigger)
	}
	
	// 尝试获取待打包的交易
	pendingTxs, err := bp.txPool.GetPendingTransactions(types.MaxTxPerBlock)
	if err != nil {
		// 如果获取交易失败，生成空区块
		return bp.createEmptyBlockWithReason(blockTime, blockNumber, prevHash, stateRoot, TriggerTransactionFailure)
	}
	
	// 如果没有待处理的交易，生成空区块
	if len(pendingTxs) == 0 {
		return bp.createEmptyBlockWithReason(blockTime, blockNumber, prevHash, stateRoot, TriggerNoTransactions)
	}
	
	// 验证并过滤交易
	validTxs := bp.filterValidTransactions(pendingTxs)
	
	// 如果没有有效交易，生成空区块
	if len(validTxs) == 0 {
		return bp.createEmptyBlockWithReason(blockTime, blockNumber, prevHash, stateRoot, TriggerTransactionFailure)
	}
	
	// 创建包含交易的区块
	return bp.createBlockWithTransactions(blockTime, blockNumber, prevHash, stateRoot, validTxs)
}

// shouldCreateEmptyBlock 判断是否应该创建空区块
func (bp *BlockProducer) shouldCreateEmptyBlock(blockNumber uint64) (EmptyBlockTrigger, bool) {
	// 检查空区块配置
	if !bp.emptyBlockConfig.Enabled {
		return TriggerNoTransactions, false
	}

	// 检查网络检查器是否可用
	if bp.networkChecker == nil {
		return TriggerNoTransactions, false
	}

	// 检查是否处于安全模式（活跃验证者不足60%）
	if bp.networkChecker.IsInSafetyMode() {
		return TriggerSafetyMode, true
	}

	// 检查当前验证者是否为本节点，但节点非活跃状态
	if !bp.networkChecker.IsCurrentValidator(bp.keyPair.Address, blockNumber) {
		// 如果不是当前验证者，但活跃验证者比例过低，仍生成空区块占位
		activeRatio := bp.networkChecker.GetActiveValidatorRatio()
		if activeRatio < 0.8 { // 80%以下时考虑生成空区块维持时间链
			return TriggerValidatorInactive, true
		}
	}

	// 检查网络分区模式
	if bp.emptyBlockConfig.NetworkPartitionMode {
		return TriggerNetworkPartition, true
	}

	return TriggerNoTransactions, false
}

// createEmptyBlockWithReason 根据原因创建空区块
func (bp *BlockProducer) createEmptyBlockWithReason(
	blockTime int64,
	blockNumber uint64,
	prevHash types.Hash,
	stateRoot types.Hash,
	trigger EmptyBlockTrigger,
) (*types.Block, error) {
	
	// 根据触发原因选择验证者地址
	validatorAddr := bp.getValidatorForEmptyBlock(trigger)
	
	// 创建空区块头
	header := &types.BlockHeader{
		Number:         blockNumber,
		Timestamp:      blockTime,
		PrevHash:       prevHash,
		TxRoot:         types.EmptyTxRoot(), // 确定性空交易根
		StateRoot:      stateRoot,           // 延续前块状态
		Validator:      validatorAddr,
		ShardID:        types.ShardID,
		AdjacentHashes: [3]types.Hash{}, // 第一阶段为空
	}
	
	// 根据触发原因决定是否签名
	if trigger == TriggerSafetyMode || trigger == TriggerNetworkPartition {
		// 安全模式或网络分区时使用空签名
		header.Signature = types.Signature{}
	} else {
		// 其他情况正常签名
		err := bp.keyPair.SignBlock(header)
		if err != nil {
			return nil, fmt.Errorf("failed to sign empty block: %w", err)
		}
	}
	
	// 创建空区块
	block := &types.Block{
		Header:       *header,
		Transactions: []types.Hash{}, // 确定性空交易列表
	}
	
	// 记录空区块创建日志
	fmt.Printf("📦 Created empty block %d (reason: %s) at %s\n",
		blockNumber, trigger.String(), time.Unix(blockTime, 0).Format("15:04:05"))
	
	return block, nil
}

// getValidatorForEmptyBlock 根据触发原因获取验证者地址
func (bp *BlockProducer) getValidatorForEmptyBlock(trigger EmptyBlockTrigger) types.Address {
	switch trigger {
	case TriggerSafetyMode, TriggerNetworkPartition:
		// 安全模式或网络分区时使用系统默认地址
		if !bp.emptyBlockConfig.SystemValidator.IsEmpty() {
			return bp.emptyBlockConfig.SystemValidator
		}
		// 如果没有配置系统验证者，使用空地址
		return types.Address{}
	default:
		// 其他情况使用当前验证者地址
		return bp.keyPair.Address
	}
}

// createEmptyBlock 创建空区块（兼容方法）
func (bp *BlockProducer) createEmptyBlock(blockTime int64, blockNumber uint64, prevHash types.Hash, stateRoot types.Hash) (*types.Block, error) {
	return bp.createEmptyBlockWithReason(blockTime, blockNumber, prevHash, stateRoot, TriggerNoTransactions)
}

// =============== 空区块管理与统计方法 ===============

// GetEmptyBlockStats 获取空区块统计信息
func (bp *BlockProducer) GetEmptyBlockStats() map[string]interface{} {
	bp.mu.RLock()
	defer bp.mu.RUnlock()

	return map[string]interface{}{
		"empty_block_enabled":      bp.emptyBlockConfig.Enabled,
		"max_empty_interval":       bp.emptyBlockConfig.MaxEmptyInterval.Seconds(),
		"network_partition_mode":   bp.emptyBlockConfig.NetworkPartitionMode,
		"system_validator":         bp.emptyBlockConfig.SystemValidator.String(),
		"network_checker_available": bp.networkChecker != nil,
	}
}

// EnableEmptyBlocks 启用空区块生成
func (bp *BlockProducer) EnableEmptyBlocks(enabled bool) {
	bp.mu.Lock()
	defer bp.mu.Unlock()
	bp.emptyBlockConfig.Enabled = enabled
}

// SetNetworkPartitionMode 设置网络分区模式
func (bp *BlockProducer) SetNetworkPartitionMode(enabled bool) {
	bp.mu.Lock()
	defer bp.mu.Unlock()
	bp.emptyBlockConfig.NetworkPartitionMode = enabled
}

// SetSystemValidator 设置系统默认验证者地址
func (bp *BlockProducer) SetSystemValidator(address types.Address) {
	bp.mu.Lock()
	defer bp.mu.Unlock()
	bp.emptyBlockConfig.SystemValidator = address
}

// ValidateEmptyBlock 验证空区块的有效性
func (bp *BlockProducer) ValidateEmptyBlock(block *types.Block) error {
	// 验证是否为空区块
	if !block.IsEmpty() {
		return fmt.Errorf("block is not empty")
	}

	// 验证交易根是否为空交易根
	if block.Header.TxRoot != types.EmptyTxRoot() {
		return fmt.Errorf("empty block should have empty tx root")
	}

	// 验证分片ID
	if block.Header.ShardID != types.ShardID {
		return fmt.Errorf("invalid shard ID in empty block")
	}

	// 验证验证者地址（允许空地址用于系统生成的空区块）
	if !block.Header.Validator.IsEmpty() && block.Header.Validator != bp.keyPair.Address {
		// 检查是否为配置的系统验证者
		if block.Header.Validator != bp.emptyBlockConfig.SystemValidator {
			return fmt.Errorf("invalid validator address in empty block")
		}
	}

	return nil
}

// createBlockWithTransactions 创建包含交易的区块
func (bp *BlockProducer) createBlockWithTransactions(
	blockTime int64,
	blockNumber uint64,
	prevHash types.Hash,
	stateRoot types.Hash,
	txs []*types.Transaction,
) (*types.Block, error) {
	
	// 计算交易哈希列表
	txHashes := make([]types.Hash, len(txs))
	for i, tx := range txs {
		txHashes[i] = tx.Hash()
	}
	
	// 计算交易Merkle根
	txRoot := types.CalculateTxRoot(txHashes)
	
	// 处理交易，更新状态根
	newStateRoot, err := bp.stateManager.ProcessTransactions(txs)
	if err != nil {
		// 如果交易处理失败，创建空区块
		return bp.createEmptyBlock(blockTime, blockNumber, prevHash, stateRoot)
	}
	
	// 创建区块头
	header := &types.BlockHeader{
		Number:         blockNumber,
		Timestamp:      blockTime,
		PrevHash:       prevHash,
		TxRoot:         txRoot,
		StateRoot:      newStateRoot,
		Validator:      bp.keyPair.Address,
		ShardID:        types.ShardID,
		AdjacentHashes: [3]types.Hash{}, // 第一阶段为空
	}
	
	// 签名区块头
	err = bp.keyPair.SignBlock(header)
	if err != nil {
		return nil, fmt.Errorf("failed to sign block: %w", err)
	}
	
	// 创建区块
	block := &types.Block{
		Header:       *header,
		Transactions: txHashes,
	}
	
	// 验证区块大小
	if block.Size() > types.MaxBlockSize {
		// 如果区块过大，减少交易数量重新创建
		maxTxCount := len(txs) / 2
		if maxTxCount > 0 {
			return bp.createBlockWithTransactions(blockTime, blockNumber, prevHash, stateRoot, txs[:maxTxCount])
		} else {
			// 如果单个交易就超过限制，创建空区块
			return bp.createEmptyBlock(blockTime, blockNumber, prevHash, stateRoot)
		}
	}
	
	return block, nil
}

// filterValidTransactions 过滤有效交易
func (bp *BlockProducer) filterValidTransactions(txs []*types.Transaction) []*types.Transaction {
	var validTxs []*types.Transaction
	
	for _, tx := range txs {
		if bp.validateTransaction(tx) {
			validTxs = append(validTxs, tx)
		}
	}
	
	return validTxs
}

// validateTransaction 验证交易
func (bp *BlockProducer) validateTransaction(tx *types.Transaction) bool {
	// 基本字段验证
	if tx.From.IsEmpty() || tx.To.IsEmpty() {
		return false
	}
	
	if tx.Amount == 0 {
		return false
	}
	
	if tx.GasPrice == 0 || tx.GasLimit == 0 {
		return false
	}
	
	// 获取发送方账户
	account, err := bp.stateManager.GetAccount(tx.From)
	if err != nil {
		return false
	}
	
	// 检查余额
	totalCost := tx.Amount + (tx.GasPrice * tx.GasLimit)
	if account.Balance < totalCost {
		return false
	}
	
	// 检查Nonce
	if tx.Nonce != account.Nonce+1 {
		return false
	}
	
	// 验证签名（简化版本，实际需要公钥恢复）
	if tx.Signature.IsEmpty() {
		return false
	}
	
	return true
}

// SaveBlock 保存生产的区块
func (bp *BlockProducer) SaveBlock(block *types.Block) error {
	// 保存区块到存储
	err := bp.storage.SaveBlock(block)
	if err != nil {
		return fmt.Errorf("failed to save block: %w", err)
	}
	
	// 保存区块中的交易
	for _, txHash := range block.Transactions {
		// 这里需要从交易池获取完整的交易数据来保存
		// 简化实现中先跳过
		_ = txHash // 避免未使用变量警告
	}
	
	// 从交易池中移除已打包的交易
	if len(block.Transactions) > 0 {
		err = bp.txPool.RemoveTransactions(block.Transactions)
		if err != nil {
			// 记录警告，但不返回错误
			fmt.Printf("Warning: failed to remove transactions from pool: %v\n", err)
		}
	}
	
	return nil
}

// GetBlockChannel 获取新区块通道
func (bp *BlockProducer) GetBlockChannel() <-chan *types.Block {
	return bp.blockCh
}

// ProcessNewBlock 处理新生产的区块
func (bp *BlockProducer) ProcessNewBlock(block *types.Block) error {
	// 保存区块
	err := bp.SaveBlock(block)
	if err != nil {
		return fmt.Errorf("failed to save new block: %w", err)
	}
	
	// 通知其他组件
	select {
	case bp.blockCh <- block:
	default:
		// 如果通道满了，非阻塞发送
		select {
		case <-bp.blockCh: // 移除旧的区块
			bp.blockCh <- block // 发送新区块
		default:
		}
	}
	
	return nil
}

// GetValidatorAddress 获取验证者地址
func (bp *BlockProducer) GetValidatorAddress() types.Address {
	return bp.keyPair.Address
}

// IsRunning 检查是否正在运行
func (bp *BlockProducer) IsRunning() bool {
	bp.mu.RLock()
	defer bp.mu.RUnlock()
	return bp.isRunning
}

// GetStats 获取区块生产器统计信息
func (bp *BlockProducer) GetStats() map[string]interface{} {
	bp.mu.RLock()
	defer bp.mu.RUnlock()
	
	stats := map[string]interface{}{
		"validator_address": bp.keyPair.Address.String(),
		"is_running":        bp.isRunning,
		"pending_tx_count":  0,
	}
	
	if bp.txPool != nil {
		stats["pending_tx_count"] = bp.txPool.GetTransactionCount()
	}
	
	return stats
}

// ValidateProducedBlock 验证生产的区块
func (bp *BlockProducer) ValidateProducedBlock(block *types.Block) error {
	// 验证区块头签名
	if !crypto.VerifyBlock(&block.Header, bp.keyPair.PublicKey) {
		return fmt.Errorf("invalid block signature")
	}
	
	// 验证验证者地址
	if block.Header.Validator != bp.keyPair.Address {
		return fmt.Errorf("invalid validator address")
	}
	
	// 验证分片ID
	if block.Header.ShardID != types.ShardID {
		return fmt.Errorf("invalid shard ID")
	}
	
	// 验证交易根
	expectedTxRoot := types.CalculateTxRoot(block.Transactions)
	if block.Header.TxRoot != expectedTxRoot {
		return fmt.Errorf("invalid transaction root")
	}
	
	// 验证区块大小
	if block.Size() > types.MaxBlockSize {
		return fmt.Errorf("block size exceeds limit")
	}
	
	return nil
}

// CreateGenesisBlock 创建创世区块
func (bp *BlockProducer) CreateGenesisBlock(genesisConfig *types.GenesisBlock) (*types.Block, error) {
	// 创建创世区块头
	header := &types.BlockHeader{
		Number:         0,
		Timestamp:      genesisConfig.Timestamp,
		PrevHash:       types.Hash{}, // 创世区块没有前置区块
		TxRoot:         types.EmptyTxRoot(),
		StateRoot:      types.Hash{}, // 初始状态根，需要从初始状态计算
		Validator:      bp.keyPair.Address,
		ShardID:        types.ShardID,
		AdjacentHashes: [3]types.Hash{},
	}
	
	// 签名创世区块
	err := bp.keyPair.SignBlock(header)
	if err != nil {
		return nil, fmt.Errorf("failed to sign genesis block: %w", err)
	}
	
	// 创建创世区块
	block := &types.Block{
		Header:       *header,
		Transactions: []types.Hash{},
	}
	
	return block, nil
}