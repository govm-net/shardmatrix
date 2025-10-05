package storage

import (
	"os"
	"testing"
	"time"

	"github.com/lengzhao/shardmatrix/pkg/crypto"
	"github.com/lengzhao/shardmatrix/pkg/types"
)

// TestLevelDBStorage 测试LevelDB存储功能
func TestLevelDBStorage(t *testing.T) {
	// 创建临时目录
	tmpDir, err := os.MkdirTemp("", "shardmatrix_storage_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// 创建存储实例
	storage, err := NewLevelDBStorage(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer storage.Close()

	t.Run("BlockOperations", func(t *testing.T) {
		// 创建测试区块
		block := &types.Block{
			Header: types.BlockHeader{
				Number:    1,
				Timestamp: time.Now().Unix(),
				PrevHash:  crypto.RandomHash(),
				TxRoot:    types.EmptyTxRoot(),
				StateRoot: crypto.RandomHash(),
				Validator: crypto.RandomAddress(),
				ShardID:   types.ShardID,
			},
			Transactions: []types.Hash{crypto.RandomHash(), crypto.RandomHash()},
		}

		// 保存区块
		err := storage.SaveBlock(block)
		if err != nil {
			t.Errorf("Failed to save block: %v", err)
		}

		// 根据高度获取区块
		retrievedBlock, err := storage.GetBlock(1)
		if err != nil {
			t.Errorf("Failed to get block by height: %v", err)
		}

		if retrievedBlock.Header.Number != block.Header.Number {
			t.Errorf("Block number mismatch: expected %d, got %d", block.Header.Number, retrievedBlock.Header.Number)
		}

		// 根据哈希获取区块
		blockHash := block.Hash()
		retrievedByHash, err := storage.GetBlockByHash(blockHash)
		if err != nil {
			t.Errorf("Failed to get block by hash: %v", err)
		}

		if retrievedByHash.Header.Number != block.Header.Number {
			t.Errorf("Block number mismatch when retrieved by hash")
		}

		// 检查区块是否存在
		if !storage.HasBlock(blockHash) {
			t.Error("Block should exist in storage")
		}
	})

	t.Run("TransactionOperations", func(t *testing.T) {
		// 创建测试交易
		keyPair, _ := crypto.GenerateKeyPair()
		tx := &types.Transaction{
			ShardID:   types.ShardID,
			From:      keyPair.Address,
			To:        crypto.RandomAddress(),
			Amount:    1000,
			GasPrice:  100,
			GasLimit:  21000,
			Nonce:     1,
			Data:      []byte("test transaction"),
			Signature: crypto.RandomSignature(),
		}

		// 保存交易
		err := storage.SaveTransaction(tx)
		if err != nil {
			t.Errorf("Failed to save transaction: %v", err)
		}

		// 获取交易
		txHash := tx.Hash()
		retrievedTx, err := storage.GetTransaction(txHash)
		if err != nil {
			t.Errorf("Failed to get transaction: %v", err)
		}

		if retrievedTx.Amount != tx.Amount {
			t.Errorf("Transaction amount mismatch: expected %d, got %d", tx.Amount, retrievedTx.Amount)
		}

		// 检查交易是否存在
		if !storage.HasTransaction(txHash) {
			t.Error("Transaction should exist in storage")
		}
	})

	t.Run("AccountOperations", func(t *testing.T) {
		// 创建测试账户
		address := crypto.RandomAddress()
		account := &types.Account{
			Address: address,
			Balance: 10000,
			Nonce:   5,
			Staked:  1000,
		}

		// 保存账户
		err := storage.SaveAccount(account)
		if err != nil {
			t.Errorf("Failed to save account: %v", err)
		}

		// 获取账户
		retrievedAccount, err := storage.GetAccount(address)
		if err != nil {
			t.Errorf("Failed to get account: %v", err)
		}

		if retrievedAccount.Balance != account.Balance {
			t.Errorf("Account balance mismatch: expected %d, got %d", account.Balance, retrievedAccount.Balance)
		}

		// 检查账户是否存在
		if !storage.HasAccount(address) {
			t.Error("Account should exist in storage")
		}
	})

	t.Run("ValidatorOperations", func(t *testing.T) {
		// 创建测试验证者
		keyPair, _ := crypto.GenerateKeyPair()
		validator := &types.Validator{
			Address:      keyPair.Address,
			PublicKey:    keyPair.PublicKey,
			StakeAmount:  100000,
			DelegatedAmt: 50000,
			VotePower:    150000,
			IsActive:     true,
			SlashCount:   0,
		}

		// 保存验证者
		err := storage.SaveValidator(validator)
		if err != nil {
			t.Errorf("Failed to save validator: %v", err)
		}

		// 获取验证者
		retrievedValidator, err := storage.GetValidator(keyPair.Address)
		if err != nil {
			t.Errorf("Failed to get validator: %v", err)
		}

		if retrievedValidator.StakeAmount != validator.StakeAmount {
			t.Errorf("Validator stake amount mismatch: expected %d, got %d", validator.StakeAmount, retrievedValidator.StakeAmount)
		}

		// 获取所有验证者
		validators, err := storage.GetAllValidators()
		if err != nil {
			t.Errorf("Failed to get all validators: %v", err)
		}

		if len(validators) == 0 {
			t.Error("Should have at least one validator")
		}
	})

	t.Run("GenesisOperations", func(t *testing.T) {
		// 创建创世配置
		genesis := &types.GenesisBlock{
			Timestamp: time.Now().Unix(),
			InitAccounts: []types.Account{
				{
					Address: crypto.RandomAddress(),
					Balance: 1000000,
					Nonce:   0,
					Staked:  0,
				},
			},
			ChainID: "test-chain",
			Config: map[string]string{
				"consensus": "dpos",
				"version":   "1.0.0",
			},
		}

		// 保存创世配置
		err := storage.SaveGenesisBlock(genesis)
		if err != nil {
			t.Errorf("Failed to save genesis block: %v", err)
		}

		// 获取创世配置
		retrievedGenesis, err := storage.GetGenesisBlock()
		if err != nil {
			t.Errorf("Failed to get genesis block: %v", err)
		}

		if retrievedGenesis.ChainID != genesis.ChainID {
			t.Errorf("Genesis chain ID mismatch: expected %s, got %s", genesis.ChainID, retrievedGenesis.ChainID)
		}
	})

	t.Run("HeightOperations", func(t *testing.T) {
		// 测试高度操作
		initialHeight, err := storage.GetLatestBlockHeight()
		if err != nil {
			t.Errorf("Failed to get initial height: %v", err)
		}

		// 更新高度
		newHeight := uint64(100)
		err = storage.UpdateLatestBlockHeight(newHeight)
		if err != nil {
			t.Errorf("Failed to update height: %v", err)
		}

		// 验证高度更新
		updatedHeight, err := storage.GetLatestBlockHeight()
		if err != nil {
			t.Errorf("Failed to get updated height: %v", err)
		}

		if updatedHeight != newHeight {
			t.Errorf("Height mismatch: expected %d, got %d", newHeight, updatedHeight)
		}

		t.Logf("Height updated from %d to %d", initialHeight, updatedHeight)
	})
}

// TestStoragePerformance 测试存储性能
func TestStoragePerformance(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "shardmatrix_perf_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	storage, err := NewLevelDBStorage(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer storage.Close()

	t.Run("BlockWritePerformance", func(t *testing.T) {
		const numBlocks = 100
		blocks := make([]*types.Block, numBlocks)

		// 准备测试数据
		for i := 0; i < numBlocks; i++ {
			blocks[i] = &types.Block{
				Header: types.BlockHeader{
					Number:    uint64(i),
					Timestamp: time.Now().Unix(),
					PrevHash:  crypto.RandomHash(),
					TxRoot:    types.EmptyTxRoot(),
					StateRoot: crypto.RandomHash(),
					Validator: crypto.RandomAddress(),
					ShardID:   types.ShardID,
				},
				Transactions: []types.Hash{crypto.RandomHash()},
			}
		}

		// 测试写入性能
		start := time.Now()
		for _, block := range blocks {
			err := storage.SaveBlock(block)
			if err != nil {
				t.Errorf("Failed to save block %d: %v", block.Header.Number, err)
			}
		}
		writeDuration := time.Since(start)

		// 测试读取性能
		start = time.Now()
		for i := 0; i < numBlocks; i++ {
			_, err := storage.GetBlock(uint64(i))
			if err != nil {
				t.Errorf("Failed to get block %d: %v", i, err)
			}
		}
		readDuration := time.Since(start)

		t.Logf("Wrote %d blocks in %v (avg: %v per block)", numBlocks, writeDuration, writeDuration/numBlocks)
		t.Logf("Read %d blocks in %v (avg: %v per block)", numBlocks, readDuration, readDuration/numBlocks)

		// 性能要求：每个区块的写入和读取都应该在1ms内完成
		avgWrite := writeDuration / numBlocks
		avgRead := readDuration / numBlocks

		if avgWrite > time.Millisecond {
			t.Errorf("Block write too slow: %v > 1ms", avgWrite)
		}

		if avgRead > time.Millisecond {
			t.Errorf("Block read too slow: %v > 1ms", avgRead)
		}
	})
}

// TestMockStorage 测试模拟存储
func TestMockStorage(t *testing.T) {
	storage := NewMockStorage()

	t.Run("BasicOperations", func(t *testing.T) {
		// 测试区块操作
		block := &types.Block{
			Header: types.BlockHeader{
				Number: 1,
			},
		}

		err := storage.SaveBlock(block)
		if err != nil {
			t.Errorf("Mock storage should not return error: %v", err)
		}

		// 模拟存储的读取操作应该返回nil,nil或默认值
		retrievedBlock, err := storage.GetBlock(1)
		if err != nil {
			t.Errorf("Mock storage should not return error: %v", err)
		}

		// 模拟存储可能返回nil
		if retrievedBlock != nil {
			t.Logf("Mock storage returned block: %+v", retrievedBlock)
		}

		// 测试账户操作
		address := crypto.RandomAddress()
		account := &types.Account{
			Address: address,
			Balance: 1000,
		}

		err = storage.SaveAccount(account)
		if err != nil {
			t.Errorf("Mock storage should not return error: %v", err)
		}

		retrievedAccount, err := storage.GetAccount(address)
		if err != nil {
			t.Errorf("Mock storage should not return error: %v", err)
		}

		if retrievedAccount == nil {
			t.Error("Mock storage should return a default account")
		}
	})

	t.Run("CloseOperation", func(t *testing.T) {
		err := storage.Close()
		if err != nil {
			t.Errorf("Mock storage close should not return error: %v", err)
		}
	})
}