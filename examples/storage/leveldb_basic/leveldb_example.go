package main

import (
	"fmt"
	"os"

	"github.com/govm-net/shardmatrix/pkg/storage"
	"github.com/govm-net/shardmatrix/pkg/types"
)

func main() {
	// 创建数据目录
	dataDir := "./data"
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		fmt.Printf("Failed to create data directory: %v\n", err)
		return
	}

	// 创建LevelDB存储管理器
	config := &storage.StorageConfig{
		DataDir:   dataDir,
		UseMemory: false,
	}

	storeManager, err := storage.NewStorageManager(config)
	if err != nil {
		fmt.Printf("Failed to create storage manager: %v\n", err)
		return
	}
	defer storeManager.Close()

	// 创建测试数据
	validatorAddr := types.AddressFromPublicKey([]byte("example_validator"))
	validator := types.NewValidator(validatorAddr, []byte("example_public_key"), 1000)
	account := types.NewAccountWithBalance(validatorAddr, 5000)

	// 初始化创世状态
	err = storeManager.InitializeGenesis([]*types.Validator{validator}, []*types.Account{account})
	if err != nil {
		fmt.Printf("Failed to initialize genesis: %v\n", err)
		return
	}

	fmt.Println("Genesis state initialized successfully")

	// 创建并存储创世区块
	genesisBlock := types.NewGenesisBlock(validatorAddr)
	err = storeManager.BlockStore.PutBlock(genesisBlock)
	if err != nil {
		fmt.Printf("Failed to store genesis block: %v\n", err)
		return
	}

	fmt.Printf("Genesis block stored with hash: %s\n", genesisBlock.Hash().String())

	// 验证区块存储
	storedGenesis, err := storeManager.BlockStore.GetBlock(genesisBlock.Hash())
	if err != nil {
		fmt.Printf("Failed to get stored genesis block: %v\n", err)
		return
	}

	fmt.Printf("Retrieved genesis block - Height: %d, Validator: %s\n",
		storedGenesis.Header.Number, storedGenesis.Header.Validator.String())

	// 获取存储统计信息
	stats, err := storeManager.GetStats()
	if err != nil {
		fmt.Printf("Failed to get storage stats: %v\n", err)
		return
	}

	fmt.Printf("Storage stats - Latest block height: %d, Validator count: %d\n",
		stats.LatestBlockHeight, stats.ValidatorCount)

	// 创建并存储一个新区块
	block1 := types.NewBlock(1, genesisBlock.Hash(), validatorAddr)
	err = storeManager.BlockStore.PutBlock(block1)
	if err != nil {
		fmt.Printf("Failed to store block1: %v\n", err)
		return
	}

	fmt.Printf("Block 1 stored with hash: %s\n", block1.Hash().String())

	// 按高度获取区块
	blockByHeight, err := storeManager.BlockStore.GetBlockByHeight(1)
	if err != nil {
		fmt.Printf("Failed to get block by height: %v\n", err)
		return
	}

	fmt.Printf("Retrieved block by height - Height: %d, Hash: %s\n",
		blockByHeight.Header.Number, blockByHeight.Hash().String())

	// 验证最新区块
	latestBlock, err := storeManager.BlockStore.GetLatestBlock()
	if err != nil {
		fmt.Printf("Failed to get latest block: %v\n", err)
		return
	}

	fmt.Printf("Latest block - Height: %d, Hash: %s\n",
		latestBlock.Header.Number, latestBlock.Hash().String())

	// 验证存储完整性
	err = storeManager.ValidateIntegrity()
	if err != nil {
		fmt.Printf("Storage integrity validation failed: %v\n", err)
		return
	}

	fmt.Println("Storage integrity validation passed")

	// 清理测试数据
	fmt.Println("Cleaning up test data...")
	os.RemoveAll(dataDir)
	fmt.Println("Example completed successfully")
}
