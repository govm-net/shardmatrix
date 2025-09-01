package storage

import (
	"os"
	"testing"

	"github.com/govm-net/shardmatrix/pkg/types"
)

func TestLevelDBBlockStore(t *testing.T) {
	// 创建临时目录用于测试
	tempDir, err := os.MkdirTemp("", "leveldb_block_store_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 创建LevelDB区块存储
	store, err := NewLevelDBBlockStore(tempDir)
	if err != nil {
		t.Fatalf("Failed to create LevelDB block store: %v", err)
	}
	defer store.Close()

	// 创建测试数据
	validator := types.AddressFromPublicKey([]byte("validator_public_key"))
	prevHash := types.NewHash([]byte("prev_hash"))
	block := types.NewBlock(1, prevHash, validator)

	// 测试存储区块
	err = store.PutBlock(block)
	if err != nil {
		t.Fatalf("Failed to store block: %v", err)
	}

	// 测试按哈希获取区块
	blockHash := block.Hash()
	retrievedBlock, err := store.GetBlock(blockHash)
	if err != nil {
		t.Fatalf("Failed to get block: %v", err)
	}

	// 验证区块内容
	if retrievedBlock.Header.Number != block.Header.Number {
		t.Errorf("Expected block number %d, got %d", block.Header.Number, retrievedBlock.Header.Number)
	}
	if retrievedBlock.Header.Validator != block.Header.Validator {
		t.Errorf("Expected validator %x, got %x", block.Header.Validator, retrievedBlock.Header.Validator)
	}

	// 测试按高度获取区块
	blockByHeight, err := store.GetBlockByHeight(1)
	if err != nil {
		t.Fatalf("Failed to get block by height: %v", err)
	}
	if blockByHeight.Header.Number != block.Header.Number {
		t.Errorf("Expected block number %d, got %d", block.Header.Number, blockByHeight.Header.Number)
	}

	// 测试HasBlock
	if !store.HasBlock(blockHash) {
		t.Error("Block should exist")
	}

	// 测试获取最新区块
	latestBlock, err := store.GetLatestBlock()
	if err != nil {
		t.Fatalf("Failed to get latest block: %v", err)
	}
	if latestBlock.Header.Number != block.Header.Number {
		t.Errorf("Expected latest block number %d, got %d", block.Header.Number, latestBlock.Header.Number)
	}

	// 测试存储多个区块并验证最新区块
	block2 := types.NewBlock(2, blockHash, validator)
	err = store.PutBlock(block2)
	if err != nil {
		t.Fatalf("Failed to store second block: %v", err)
	}

	// 验证最新区块已更新
	latestBlock, err = store.GetLatestBlock()
	if err != nil {
		t.Fatalf("Failed to get latest block: %v", err)
	}
	if latestBlock.Header.Number != 2 {
		t.Errorf("Expected latest block number 2, got %d", latestBlock.Header.Number)
	}
}

func TestLevelDBBlockStoreGenesisBlock(t *testing.T) {
	// 创建临时目录用于测试
	tempDir, err := os.MkdirTemp("", "leveldb_block_store_genesis_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 创建LevelDB区块存储
	store, err := NewLevelDBBlockStore(tempDir)
	if err != nil {
		t.Fatalf("Failed to create LevelDB block store: %v", err)
	}
	defer store.Close()

	// 创建创世区块
	validator := types.AddressFromPublicKey([]byte("genesis_validator"))
	genesisBlock := types.NewGenesisBlock(validator)

	// 测试存储创世区块
	err = store.PutBlock(genesisBlock)
	if err != nil {
		t.Fatalf("Failed to store genesis block: %v", err)
	}

	// 测试获取创世区块
	retrievedBlock, err := store.GetBlock(genesisBlock.Hash())
	if err != nil {
		t.Fatalf("Failed to get genesis block: %v", err)
	}

	// 验证创世区块内容
	if retrievedBlock.Header.Number != 0 {
		t.Errorf("Expected genesis block number 0, got %d", retrievedBlock.Header.Number)
	}
	if retrievedBlock.Header.Validator != validator {
		t.Errorf("Expected validator %x, got %x", validator, retrievedBlock.Header.Validator)
	}

	// 验证创世区块没有签名
	if len(retrievedBlock.Header.Signature) != 0 {
		t.Errorf("Genesis block should not have signature, got %d bytes", len(retrievedBlock.Header.Signature))
	}

	// 测试按高度0获取创世区块
	genesisByHeight, err := store.GetBlockByHeight(0)
	if err != nil {
		t.Fatalf("Failed to get genesis block by height: %v", err)
	}
	if genesisByHeight.Header.Number != 0 {
		t.Errorf("Expected genesis block number 0, got %d", genesisByHeight.Header.Number)
	}

	// 测试获取最新区块（应该是创世区块）
	latestBlock, err := store.GetLatestBlock()
	if err != nil {
		t.Fatalf("Failed to get latest block: %v", err)
	}
	if latestBlock.Header.Number != 0 {
		t.Errorf("Expected latest block number 0, got %d", latestBlock.Header.Number)
	}
}

func TestLevelDBBlockStoreNonExistentBlock(t *testing.T) {
	// 创建临时目录用于测试
	tempDir, err := os.MkdirTemp("", "leveldb_block_store_nonexistent_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 创建LevelDB区块存储
	store, err := NewLevelDBBlockStore(tempDir)
	if err != nil {
		t.Fatalf("Failed to create LevelDB block store: %v", err)
	}
	defer store.Close()

	// 测试获取不存在的区块
	nonExistentHash := types.NewHash([]byte("non_existent"))
	_, err = store.GetBlock(nonExistentHash)
	if err == nil {
		t.Error("Expected error when getting non-existent block")
	}

	// 测试检查不存在的区块
	if store.HasBlock(nonExistentHash) {
		t.Error("Non-existent block should not exist")
	}

	// 测试获取不存在高度的区块
	_, err = store.GetBlockByHeight(100)
	if err == nil {
		t.Error("Expected error when getting block by non-existent height")
	}

	// 测试获取最新区块（应该返回错误，因为没有区块）
	_, err = store.GetLatestBlock()
	if err == nil {
		t.Error("Expected error when getting latest block from empty store")
	}
}
