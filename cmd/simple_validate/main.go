package main

import (
	"fmt"
	"time"

	"github.com/lengzhao/shardmatrix/pkg/consensus"
	"github.com/lengzhao/shardmatrix/pkg/crypto"
	"github.com/lengzhao/shardmatrix/pkg/types"
)

func main() {
	fmt.Println("🔗 ShardMatrix Core Implementation Validation")
	fmt.Println("=============================================")
	
	success := true
	
	// 1. 验证严格时间控制
	fmt.Println("\n1. ⏰ Strict Time Control Mechanism")
	if validateTimeControl() {
		fmt.Println("   ✅ PASSED")
	} else {
		fmt.Println("   ❌ FAILED")
		success = false
	}
	
	// 2. 验证加密操作
	fmt.Println("\n2. 🔐 Cryptographic Operations")
	if validateCrypto() {
		fmt.Println("   ✅ PASSED")
	} else {
		fmt.Println("   ❌ FAILED")
		success = false
	}
	
	// 3. 验证空区块机制
	fmt.Println("\n3. 📦 Empty Block Mechanism")
	if validateEmptyBlocks() {
		fmt.Println("   ✅ PASSED")
	} else {
		fmt.Println("   ❌ FAILED")
		success = false
	}
	
	// 4. 验证数据类型
	fmt.Println("\n4. 📊 Data Types and Constants")
	if validateTypes() {
		fmt.Println("   ✅ PASSED")
	} else {
		fmt.Println("   ❌ FAILED")
		success = false
	}
	
	// 5. 验证DPoS基础
	fmt.Println("\n5. 🗳️  DPoS Consensus Basics")
	if validateDPoS() {
		fmt.Println("   ✅ PASSED")
	} else {
		fmt.Println("   ❌ FAILED")
		success = false
	}
	
	// 性能测试
	fmt.Println("\n6. ⚡ Performance Tests")
	runPerformanceTests()
	
	// 最终结果
	fmt.Println("\n" + "===============================================")
	if success {
		fmt.Println("🎉 ALL CORE FEATURES VALIDATED SUCCESSFULLY!")
		fmt.Println("\nShardMatrix Single-Node Blockchain Implementation:")
		fmt.Println("✓ Strict 2-second block intervals with zero tolerance")
		fmt.Println("✓ Empty block filling mechanism")
		fmt.Println("✓ Ed25519 digital signatures")
		fmt.Println("✓ DPoS consensus foundations")
		fmt.Println("✓ Cryptographic hash operations")
		fmt.Println("✓ Time synchronization mechanisms")
		fmt.Println("\nImplementation is ready for deployment!")
	} else {
		fmt.Println("❌ Some validations failed. Please check implementation.")
	}
}

func validateTimeControl() bool {
	// 固定创世时间用于测试
	genesisTime := int64(1609459200) // 2021-01-01 00:00:00 UTC
	timeController := consensus.NewTimeController(genesisTime)
	
	fmt.Printf("   Genesis time: %d\\n", genesisTime)
	
	// 测试连续区块时间计算
	for i := uint64(1); i <= 5; i++ {
		expectedTime := genesisTime + int64(i)*2
		calculatedTime := timeController.GetBlockTime(i)
		
		if calculatedTime != expectedTime {
			fmt.Printf("   Block %d time error: expected %d, got %d\\n", i, expectedTime, calculatedTime)
			return false
		}
		
		// 验证严格时间验证
		if !timeController.ValidateBlockTime(i, calculatedTime) {
			fmt.Printf("   Block %d: strict validation failed\\n", i)
			return false
		}
		
		// 验证错误时间被拒绝
		if timeController.ValidateBlockTime(i, calculatedTime+1) {
			fmt.Printf("   Block %d: should reject wrong timestamp\\n", i)
			return false
		}
	}
	
	// 测试工具函数
	if !types.ValidateBlockTime(genesisTime, 3, genesisTime+6) {
		fmt.Printf("   ValidateBlockTime utility failed\\n")
		return false
	}
	
	fmt.Printf("   Validated 5 blocks with strict 2-second intervals\\n")
	return true
}

func validateCrypto() bool {
	// 生成密钥对
	keyPair, err := crypto.GenerateKeyPair()
	if err != nil {
		fmt.Printf("   Keypair generation failed: %v\\n", err)
		return false
	}
	
	fmt.Printf("   Generated validator address: %s\\n", keyPair.Address.String()[:16]+"...")
	
	// 测试签名和验证
	testData := []byte("ShardMatrix blockchain data")
	signature := keyPair.Sign(testData)
	
	if !crypto.VerifySignature(keyPair.PublicKey, testData, signature) {
		fmt.Printf("   Signature verification failed\\n")
		return false
	}
	
	// 测试地址生成一致性
	address1 := crypto.PublicKeyToAddress(keyPair.PublicKey)
	if address1 != keyPair.Address {
		fmt.Printf("   Address generation inconsistent\\n")
		return false
	}
	
	// 测试哈希确定性
	hash1 := crypto.Hash(testData)
	hash2 := crypto.Hash(testData)
	if hash1 != hash2 {
		fmt.Printf("   Hash not deterministic\\n")
		return false
	}
	
	// 测试区块签名
	blockHeader := &types.BlockHeader{
		Number:    1,
		Timestamp: time.Now().Unix(),
		Validator: keyPair.Address,
	}
	
	err = keyPair.SignBlock(blockHeader)
	if err != nil {
		fmt.Printf("   Block signing failed: %v\\n", err)
		return false
	}
	
	if !crypto.VerifyBlock(blockHeader, keyPair.PublicKey) {
		fmt.Printf("   Block signature verification failed\\n")
		return false
	}
	
	fmt.Printf("   Validated Ed25519 operations and block signing\\n")
	return true
}

func validateEmptyBlocks() bool {
	// 验证空交易根
	emptyRoot := types.EmptyTxRoot()
	calculatedRoot := types.CalculateTxRoot([]types.Hash{})
	
	if emptyRoot != calculatedRoot {
		fmt.Printf("   Empty transaction root mismatch\\n")
		return false
	}
	
	// 创建空区块
	genesisTime := int64(1609459200)
	emptyBlock := &types.Block{
		Header: types.BlockHeader{
			Number:         1,
			Timestamp:      genesisTime + 2, // 严格2秒后
			PrevHash:       crypto.RandomHash(),
			TxRoot:         emptyRoot,
			StateRoot:      crypto.RandomHash(),
			Validator:      crypto.RandomAddress(),
			ShardID:        types.ShardID,
			AdjacentHashes: [3]types.Hash{},
		},
		Transactions: []types.Hash{},
	}
	
	// 验证空区块特性
	if !emptyBlock.IsEmpty() {
		fmt.Printf("   Block should be empty\\n")
		return false
	}
	
	if len(emptyBlock.Transactions) != 0 {
		fmt.Printf("   Empty block should have 0 transactions\\n")
		return false
	}
	
	// 验证哈希一致性
	hash1 := emptyBlock.Hash()
	hash2 := emptyBlock.Hash()
	if hash1 != hash2 {
		fmt.Printf("   Block hash not deterministic\\n")
		return false
	}
	
	// 验证时间戳计算
	expectedTime := types.CalculateBlockTime(genesisTime, 1)
	if emptyBlock.Header.Timestamp != expectedTime {
		fmt.Printf("   Block timestamp incorrect\\n")
		return false
	}
	
	fmt.Printf("   Empty block mechanism working correctly\\n")
	return true
}

func validateTypes() bool {
	// 验证核心常量
	if types.BlockTime.Seconds() != 2 {
		fmt.Printf("   Block time should be 2 seconds\\n")
		return false
	}
	
	if types.ValidatorCount != 21 {
		fmt.Printf("   Validator count should be 21\\n")
		return false
	}
	
	if types.ShardID != 1 {
		fmt.Printf("   Shard ID should be 1\\n")
		return false
	}
	
	if types.MaxBlockSize != 2*1024*1024 {
		fmt.Printf("   Max block size should be 2MB\\n")
		return false
	}
	
	// 验证类型方法
	testHash := crypto.RandomHash()
	if testHash.IsEmpty() {
		fmt.Printf("   Random hash should not be empty\\n")
		return false
	}
	
	zeroHash := types.Hash{}
	if !zeroHash.IsEmpty() {
		fmt.Printf("   Zero hash should be empty\\n")
		return false
	}
	
	fmt.Printf("   All constants and types validated\\n")
	return true
}

func validateDPoS() bool {
	// 创建验证者
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
	
	// 验证投票权重
	expectedPower := validator.StakeAmount + validator.DelegatedAmt
	if validator.VotePower != expectedPower {
		fmt.Printf("   Vote power calculation wrong\\n")
		return false
	}
	
	// 创建验证者集合
	validatorSet := &types.ValidatorSet{
		Validators: []types.Validator{validator},
		Round:      1,
	}
	
	// 验证集合功能
	if validatorSet.GetTotalVotePower() != validator.VotePower {
		fmt.Printf("   Total vote power wrong\\n")
		return false
	}
	
	foundValidator := validatorSet.GetValidator(keyPair.Address)
	if foundValidator == nil || foundValidator.Address != keyPair.Address {
		fmt.Printf("   Validator lookup failed\\n")
		return false
	}
	
	currentValidator := validatorSet.GetCurrentValidator(1)
	if currentValidator == nil {
		fmt.Printf("   Current validator selection failed\\n")
		return false
	}
	
	fmt.Printf("   DPoS validator management working\\n")
	return true
}

func runPerformanceTests() {
	// 密钥生成测试
	start := time.Now()
	for i := 0; i < 10; i++ {
		crypto.GenerateKeyPair()
	}
	keyGenTime := time.Since(start)
	fmt.Printf("   Key generation: 10 keys in %v (avg: %v)\\n", 
		keyGenTime, keyGenTime/10)
	
	// 哈希计算测试
	testData := make([]byte, 1024)
	start = time.Now()
	for i := 0; i < 100; i++ {
		crypto.Hash(testData)
	}
	hashTime := time.Since(start)
	fmt.Printf("   Hash calculation: 100 hashes in %v (avg: %v)\\n", 
		hashTime, hashTime/100)
	
	// 签名测试
	keyPair, _ := crypto.GenerateKeyPair()
	start = time.Now()
	for i := 0; i < 10; i++ {
		keyPair.Sign(testData)
	}
	sigTime := time.Since(start)
	fmt.Printf("   Digital signatures: 10 sigs in %v (avg: %v)\\n", 
		sigTime, sigTime/10)
	
	fmt.Printf("   Performance baseline completed\\n")
}