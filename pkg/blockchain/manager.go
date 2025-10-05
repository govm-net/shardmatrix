package blockchain

import (
	"fmt"
	"sync"

	"github.com/lengzhao/shardmatrix/pkg/crypto"
	"github.com/lengzhao/shardmatrix/pkg/storage"
	"github.com/lengzhao/shardmatrix/pkg/types"
)

// Manager 区块链管理器
type Manager struct {
	mu           sync.RWMutex
	storage      storage.Storage
	currentBlock *types.Block
	currentState map[types.Address]*types.Account // 简化的状态管理
	genesisBlock *types.Block
	isInitialized bool
}

// NewManager 创建新的区块链管理器
func NewManager(storage storage.Storage) *Manager {
	return &Manager{
		storage:      storage,
		currentState: make(map[types.Address]*types.Account),
	}
}

// Initialize 初始化区块链
func (m *Manager) Initialize(genesisConfig *types.GenesisBlock) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if m.isInitialized {
		return fmt.Errorf("blockchain already initialized")
	}
	
	// 检查是否已有创世区块
	existingGenesis, err := m.storage.GetGenesisBlock()
	if err == nil {
		// 已有创世区块，加载现有区块链
		return m.loadExistingChain(existingGenesis)
	}
	
	// 创建新的区块链
	return m.createNewChain(genesisConfig)
}

// loadExistingChain 加载现有区块链
func (m *Manager) loadExistingChain(genesis *types.GenesisBlock) error {
	// 加载创世区块
	genesisBlock, err := m.storage.GetBlock(0)
	if err != nil {
		return fmt.Errorf("failed to load genesis block: %w", err)
	}
	m.genesisBlock = genesisBlock
	
	// 加载最新区块高度
	lastHeight, err := m.storage.GetLatestBlockHeight()
	if err != nil {
		return fmt.Errorf("failed to get last height: %w", err)
	}
	
	// 加载最新区块
	if lastHeight > 0 {
		currentBlock, err := m.storage.GetBlock(lastHeight)
		if err != nil {
			return fmt.Errorf("failed to load current block: %w", err)
		}
		m.currentBlock = currentBlock
	} else {
		m.currentBlock = genesisBlock
	}
	
	// 重建状态（简化实现）
	err = m.rebuildState()
	if err != nil {
		return fmt.Errorf("failed to rebuild state: %w", err)
	}
	
	m.isInitialized = true
	return nil
}

// createNewChain 创建新区块链
func (m *Manager) createNewChain(genesisConfig *types.GenesisBlock) error {
	// 创建创世区块（简化实现）
	genesisBlock := &types.Block{
		Header: types.BlockHeader{
			Number:         0,
			Timestamp:      genesisConfig.Timestamp,
			PrevHash:       types.Hash{},
			TxRoot:         types.EmptyTxRoot(),
			StateRoot:      types.Hash{}, // 需要从初始状态计算
			Validator:      types.Address{}, // 创世区块无验证者
			Signature:      types.Signature{},
			ShardID:        types.ShardID,
			AdjacentHashes: [3]types.Hash{},
		},
		Transactions: []types.Hash{},
	}
	
	m.genesisBlock = genesisBlock
	m.currentBlock = genesisBlock
	
	// 初始化账户状态
	for _, account := range genesisConfig.InitAccounts {
		m.currentState[account.Address] = &account
	}
	
	// 计算初始状态根
	stateRoot := m.calculateStateRoot()
	m.genesisBlock.Header.StateRoot = stateRoot
	
	// 保存创世区块和配置
	err := m.storage.SaveGenesisBlock(genesisConfig)
	if err != nil {
		return fmt.Errorf("failed to save genesis config: %w", err)
	}
	
	err = m.storage.SaveBlock(genesisBlock)
	if err != nil {
		return fmt.Errorf("failed to save genesis block: %w", err)
	}
	
	// 保存初始账户状态
	for _, account := range genesisConfig.InitAccounts {
		err = m.storage.SaveAccount(&account)
		if err != nil {
			return fmt.Errorf("failed to save initial account: %w", err)
		}
	}
	
	m.isInitialized = true
	return nil
}

// AddBlock 添加新区块
func (m *Manager) AddBlock(block *types.Block) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if !m.isInitialized {
		return fmt.Errorf("blockchain not initialized")
	}
	
	// 验证区块
	err := m.validateBlock(block)
	if err != nil {
		return fmt.Errorf("block validation failed: %w", err)
	}
	
	// 处理区块中的交易
	err = m.processBlockTransactions(block)
	if err != nil {
		return fmt.Errorf("failed to process block transactions: %w", err)
	}
	
	// 保存区块
	err = m.storage.SaveBlock(block)
	if err != nil {
		return fmt.Errorf("failed to save block: %w", err)
	}
	
	// 更新当前区块
	m.currentBlock = block
	
	// 保存更新的账户状态
	err = m.saveCurrentState()
	if err != nil {
		return fmt.Errorf("failed to save state: %w", err)
	}
	
	return nil
}

// validateBlock 验证区块
func (m *Manager) validateBlock(block *types.Block) error {
	// 验证区块高度
	expectedHeight := m.currentBlock.Header.Number + 1
	if block.Header.Number != expectedHeight {
		return fmt.Errorf("invalid block height: expected %d, got %d", 
			expectedHeight, block.Header.Number)
	}
	
	// 验证前一区块哈希
	expectedPrevHash := m.currentBlock.Hash()
	if block.Header.PrevHash != expectedPrevHash {
		return fmt.Errorf("invalid previous hash: expected %s, got %s", 
			expectedPrevHash.String(), block.Header.PrevHash.String())
	}
	
	// 验证交易根
	expectedTxRoot := types.CalculateTxRoot(block.Transactions)
	if block.Header.TxRoot != expectedTxRoot {
		return fmt.Errorf("invalid transaction root")
	}
	
	// 验证分片ID
	if block.Header.ShardID != types.ShardID {
		return fmt.Errorf("invalid shard ID")
	}
	
	return nil
}

// processBlockTransactions 处理区块中的交易
func (m *Manager) processBlockTransactions(block *types.Block) error {
	for _, txHash := range block.Transactions {
		// 获取交易详情
		tx, err := m.storage.GetTransaction(txHash)
		if err != nil {
			// 如果交易不在存储中，跳过（可能是空区块或同步问题）
			continue
		}
		
		// 处理交易
		err = m.processTransaction(tx)
		if err != nil {
			return fmt.Errorf("failed to process transaction %s: %w", 
				txHash.String(), err)
		}
	}
	
	return nil
}

// processTransaction 处理单个交易
func (m *Manager) processTransaction(tx *types.Transaction) error {
	// 获取发送方账户
	fromAccount, err := m.getAccount(tx.From)
	if err != nil {
		return fmt.Errorf("failed to get from account: %w", err)
	}
	
	// 获取接收方账户
	toAccount, err := m.getAccount(tx.To)
	if err != nil {
		return fmt.Errorf("failed to get to account: %w", err)
	}
	
	// 计算总费用
	totalCost := tx.Amount + (tx.GasPrice * tx.GasLimit)
	
	// 检查余额
	if fromAccount.Balance < totalCost {
		return fmt.Errorf("insufficient balance: %d < %d", 
			fromAccount.Balance, totalCost)
	}
	
	// 检查Nonce
	if tx.Nonce != fromAccount.Nonce+1 {
		return fmt.Errorf("invalid nonce: expected %d, got %d", 
			fromAccount.Nonce+1, tx.Nonce)
	}
	
	// 执行转账
	fromAccount.Balance -= totalCost
	fromAccount.Nonce++
	
	toAccount.Balance += tx.Amount
	
	// 更新状态
	m.currentState[tx.From] = fromAccount
	m.currentState[tx.To] = toAccount
	
	return nil
}

// getAccount 获取账户（从状态或存储）
func (m *Manager) getAccount(address types.Address) (*types.Account, error) {
	// 先从当前状态查找
	if account, exists := m.currentState[address]; exists {
		return account, nil
	}
	
	// 从存储中加载
	account, err := m.storage.GetAccount(address)
	if err != nil {
		// 如果账户不存在，创建默认账户
		account = &types.Account{
			Address: address,
			Balance: 0,
			Nonce:   0,
			Staked:  0,
		}
	}
	
	// 添加到当前状态
	m.currentState[address] = account
	return account, nil
}

// GetCurrentBlock 获取当前最新区块
func (m *Manager) GetCurrentBlock() *types.Block {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.currentBlock
}

// GetLastBlock 获取最新区块
func (m *Manager) GetLastBlock() (*types.Block, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if m.currentBlock == nil {
		return nil, fmt.Errorf("no blocks in chain")
	}
	
	return m.currentBlock, nil
}

// GetBlockByHeight 根据高度获取区块
func (m *Manager) GetBlockByHeight(height uint64) (*types.Block, error) {
	return m.storage.GetBlock(height)
}

// GetCurrentStateRoot 获取当前状态根
func (m *Manager) GetCurrentStateRoot() types.Hash {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	return m.calculateStateRoot()
}

// GetAccount 获取账户信息
func (m *Manager) GetAccount(address types.Address) (*types.Account, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	return m.getAccount(address)
}

// ProcessTransactions 处理交易列表并返回新的状态根
func (m *Manager) ProcessTransactions(txs []*types.Transaction) (types.Hash, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	// 创建状态副本
	stateCopy := make(map[types.Address]*types.Account)
	for addr, account := range m.currentState {
		accountCopy := *account
		stateCopy[addr] = &accountCopy
	}
	
	// 处理交易
	for _, tx := range txs {
		// 在副本上处理交易
		err := m.processTransactionOnState(tx, stateCopy)
		if err != nil {
			return types.Hash{}, fmt.Errorf("failed to process transaction: %w", err)
		}
	}
	
	// 计算新的状态根
	return m.calculateStateRootFromState(stateCopy), nil
}

// processTransactionOnState 在指定状态上处理交易
func (m *Manager) processTransactionOnState(tx *types.Transaction, state map[types.Address]*types.Account) error {
	// 获取或创建发送方账户
	fromAccount, exists := state[tx.From]
	if !exists {
		fromAccount = &types.Account{
			Address: tx.From,
			Balance: 0,
			Nonce:   0,
			Staked:  0,
		}
		state[tx.From] = fromAccount
	}
	
	// 获取或创建接收方账户
	toAccount, exists := state[tx.To]
	if !exists {
		toAccount = &types.Account{
			Address: tx.To,
			Balance: 0,
			Nonce:   0,
			Staked:  0,
		}
		state[tx.To] = toAccount
	}
	
	// 计算总费用
	totalCost := tx.Amount + (tx.GasPrice * tx.GasLimit)
	
	// 检查余额
	if fromAccount.Balance < totalCost {
		return fmt.Errorf("insufficient balance")
	}
	
	// 执行转账
	fromAccount.Balance -= totalCost
	fromAccount.Nonce++
	toAccount.Balance += tx.Amount
	
	return nil
}

// calculateStateRoot 计算状态根
func (m *Manager) calculateStateRoot() types.Hash {
	return m.calculateStateRootFromState(m.currentState)
}

// calculateStateRootFromState 从指定状态计算状态根
func (m *Manager) calculateStateRootFromState(state map[types.Address]*types.Account) types.Hash {
	// 简化的状态根计算（实际应使用Merkle树）
	var data []byte
	for addr, account := range state {
		data = append(data, addr[:]...)
		data = append(data, byte(account.Balance))
		data = append(data, byte(account.Nonce))
		data = append(data, byte(account.Staked))
	}
	
	return crypto.Hash(data)
}

// saveCurrentState 保存当前状态到存储
func (m *Manager) saveCurrentState() error {
	for _, account := range m.currentState {
		err := m.storage.SaveAccount(account)
		if err != nil {
			return fmt.Errorf("failed to save account %s: %w", 
				account.Address.String(), err)
		}
	}
	return nil
}

// rebuildState 重建状态（从存储中恢复）
func (m *Manager) rebuildState() error {
	// 简化实现：重新处理所有区块来重建状态
	// 实际项目中应该有更高效的方法
	
	// 清空当前状态
	m.currentState = make(map[types.Address]*types.Account)
	
	// 从创世区块开始重建
	genesis, err := m.storage.GetGenesisBlock()
	if err != nil {
		return fmt.Errorf("failed to get genesis block: %w", err)
	}
	
	// 检查创世区块是否为空（模拟存储情况）
	if genesis == nil {
		// 模拟存储情况，跳过状态重建
		return nil
	}
	
	// 加载创世账户状态
	for _, account := range genesis.InitAccounts {
		accountCopy := account // 创建副本以避免地址共享
		m.currentState[account.Address] = &accountCopy
	}
	
	// 处理所有区块（除了创世区块）
	lastHeight, err := m.storage.GetLatestBlockHeight()
	if err != nil {
		return fmt.Errorf("failed to get last height: %w", err)
	}
	
	for height := uint64(1); height <= lastHeight; height++ {
		block, err := m.storage.GetBlock(height)
		if err != nil {
			return fmt.Errorf("failed to get block at height %d: %w", height, err)
		}
		
		err = m.processBlockTransactions(block)
		if err != nil {
			return fmt.Errorf("failed to process block transactions: %w", err)
		}
	}
	
	return nil
}

// GetStats 获取区块链统计信息
func (m *Manager) GetStats() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	var currentHeight uint64 = 0
	var currentHash string = ""
	
	if m.currentBlock != nil {
		currentHeight = m.currentBlock.Header.Number
		currentHash = m.currentBlock.Hash().String()
	}
	
	return map[string]interface{}{
		"is_initialized":   m.isInitialized,
		"current_height":   currentHeight,
		"current_hash":     currentHash,
		"account_count":    len(m.currentState),
		"state_root":       m.calculateStateRoot().String(),
	}
}