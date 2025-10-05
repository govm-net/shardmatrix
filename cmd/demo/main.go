package main

import (
	"fmt"
	"time"

	"github.com/lengzhao/shardmatrix/pkg/consensus"
	"github.com/lengzhao/shardmatrix/pkg/crypto"
	"github.com/lengzhao/shardmatrix/pkg/types"
)

func main() {
	fmt.Println("🔗 ShardMatrix Single-Node Blockchain Demo")
	fmt.Println("==========================================")
	
	// 1. 演示加密功能
	fmt.Println("\n1. 🔐 Cryptographic Operations")
	demonstrateCrypto()
	
	// 2. 演示时间控制器
	fmt.Println("\n2. ⏰ Strict Time Controller")
	demonstrateTimeController()
	
	// 3. 演示空区块机制
	fmt.Println("\n3. 📦 Empty Block Mechanism")
	demonstrateEmptyBlocks()
	
	// 4. 演示DPoS基础功能
	fmt.Println("\n4. 🗳️  DPoS Consensus Basics")
	demonstrateDPoS()
	
	fmt.Println("\n✅ Demo completed successfully!")
	fmt.Println("All core features of ShardMatrix single-node blockchain verified.")
}

func demonstrateCrypto() {
	// 生成密钥对
	keyPair, err := crypto.GenerateKeyPair()
	if err != nil {
		panic(err)
	}
	
	fmt.Printf("   ✓ Generated validator keypair\n")
	fmt.Printf("     Address: %s\n", keyPair.Address.String())
	fmt.Printf("     Public Key: %s\n", keyPair.PublicKeyHex()[:32]+"...")
	
	// 测试签名和验证
	data := []byte("test transaction data")
	signature := keyPair.Sign(data)
	
	isValid := crypto.VerifySignature(keyPair.PublicKey, data, signature)
	fmt.Printf("   ✓ Signature verification: %v\n", isValid)
	
	// 测试哈希
	hash1 := crypto.Hash(data)
	hash2 := crypto.Hash(data)
	fmt.Printf("   ✓ Hash deterministic: %v\n", hash1 == hash2)
	fmt.Printf("     Hash: %s\n", hash1.String()[:32]+"...")
}

func demonstrateTimeController() {
	// 创建时间控制器，使用当前时间作为创世时间
	genesisTime := time.Now().Unix()
	if genesisTime%2 != 0 {
		genesisTime-- // 确保是偶数秒
	}
	
	timeController := consensus.NewTimeController(genesisTime)
	
	fmt.Printf("   ✓ Time controller initialized\n")
	fmt.Printf("     Genesis time: %s\n", time.Unix(genesisTime, 0).Format("15:04:05"))
	fmt.Printf("     Block interval: %v (strict)\n", types.BlockTime)
	
	// 计算各个区块的时间
	for i := uint64(1); i <= 3; i++ {
		blockTime := timeController.GetBlockTime(i)
		fmt.Printf("     Block %d time: %s (%d)\n", 
			i, time.Unix(blockTime, 0).Format("15:04:05"), blockTime)
	}
	
	// 验证时间
	block2Time := timeController.GetBlockTime(2)
	isValid := timeController.ValidateBlockTime(2, block2Time)
	fmt.Printf("   ✓ Time validation (correct): %v\n", isValid)
	
	isValid = timeController.ValidateBlockTime(2, block2Time+1)
	fmt.Printf("   ✓ Time validation (wrong): %v\n", isValid)
}

func demonstrateEmptyBlocks() {
	// 演示空区块机制
	emptyTxRoot := types.EmptyTxRoot()
	fmt.Printf("   ✓ Empty transaction root: %s\n", emptyTxRoot.String()[:32]+"...")
	
	// 计算空交易列表的根
	calculatedRoot := types.CalculateTxRoot([]types.Hash{})
	fmt.Printf("   ✓ Calculated empty root matches: %v\n", calculatedRoot == emptyTxRoot)
	
	// 创建空区块示例
	genesisTime := time.Now().Unix()
	if genesisTime%2 != 0 {
		genesisTime--
	}
	
	emptyBlock := &types.Block{
		Header: types.BlockHeader{
			Number:         1,
			Timestamp:      genesisTime + 2, // 严格2秒后
			PrevHash:       crypto.RandomHash(),
			TxRoot:         emptyTxRoot,
			StateRoot:      crypto.RandomHash(),
			Validator:      crypto.RandomAddress(),
			ShardID:        types.ShardID,
			AdjacentHashes: [3]types.Hash{},
		},
		Transactions: []types.Hash{}, // 空交易列表
	}
	
	fmt.Printf("   ✓ Empty block created\n")
	fmt.Printf("     Block number: %d\n", emptyBlock.Header.Number)
	fmt.Printf("     Timestamp: %d\n", emptyBlock.Header.Timestamp)
	fmt.Printf("     Transaction count: %d\n", len(emptyBlock.Transactions))
	fmt.Printf("     Is empty: %v\n", emptyBlock.IsEmpty())
	fmt.Printf("     Size: %d bytes\n", emptyBlock.Size())
}

func demonstrateDPoS() {
	// 创建验证者示例
	keyPair, _ := crypto.GenerateKeyPair()
	
	validator := types.Validator{
		Address:      keyPair.Address,
		PublicKey:    keyPair.PublicKey,
		StakeAmount:  100000,
		DelegatedAmt: 50000,
		VotePower:    150000,
		IsActive:     true,
		SlashCount:   0,
	}
	
	fmt.Printf("   ✓ Validator created\n")
	fmt.Printf("     Address: %s\n", validator.Address.String())
	fmt.Printf("     Stake amount: %d\n", validator.StakeAmount)
	fmt.Printf("     Vote power: %d\n", validator.VotePower)
	fmt.Printf("     Is active: %v\n", validator.IsActive)
	
	// 创建验证者集合
	validatorSet := &types.ValidatorSet{
		Validators: []types.Validator{validator},
		Round:      1,
	}
	
	fmt.Printf("   ✓ Validator set created\n")
	fmt.Printf("     Total validators: %d\n", len(validatorSet.Validators))
	fmt.Printf("     Total vote power: %d\n", validatorSet.GetTotalVotePower())
	
	// 获取当前验证者
	currentValidator := validatorSet.GetCurrentValidator(1)
	if currentValidator != nil {
		fmt.Printf("     Current validator: %s\n", currentValidator.Address.String())
	}
}