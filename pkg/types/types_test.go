package types

import (
	"testing"
	"time"
)

// TestHashOperations 测试哈希操作
func TestHashOperations(t *testing.T) {
	t.Run("HashString", func(t *testing.T) {
		hash := RandomHash()
		hashStr := hash.String()

		if len(hashStr) != 64 { // 32 bytes * 2 characters per byte
			t.Errorf("Hash string length should be 64, got %d", len(hashStr))
		}
	})

	t.Run("HashEmpty", func(t *testing.T) {
		var emptyHash Hash
		if !emptyHash.IsEmpty() {
			t.Error("Zero hash should be empty")
		}

		nonEmptyHash := RandomHash()
		if nonEmptyHash.IsEmpty() {
			t.Error("Random hash should not be empty")
		}
	})
}

// TestAddressOperations 测试地址操作
func TestAddressOperations(t *testing.T) {
	t.Run("AddressString", func(t *testing.T) {
		address := RandomAddress()
		addressStr := address.String()

		if len(addressStr) != 40 { // 20 bytes * 2 characters per byte
			t.Errorf("Address string length should be 40, got %d", len(addressStr))
		}
	})

	t.Run("AddressEmpty", func(t *testing.T) {
		var emptyAddress Address
		if !emptyAddress.IsEmpty() {
			t.Error("Zero address should be empty")
		}

		nonEmptyAddress := RandomAddress()
		if nonEmptyAddress.IsEmpty() {
			t.Error("Random address should not be empty")
		}
	})
}

// TestSignatureOperations 测试签名操作
func TestSignatureOperations(t *testing.T) {
	t.Run("SignatureString", func(t *testing.T) {
		signature := RandomSignature()
		signatureStr := signature.String()

		if len(signatureStr) != 128 { // 64 bytes * 2 characters per byte
			t.Errorf("Signature string length should be 128, got %d", len(signatureStr))
		}
	})

	t.Run("SignatureEmpty", func(t *testing.T) {
		var emptySignature Signature
		if !emptySignature.IsEmpty() {
			t.Error("Zero signature should be empty")
		}

		nonEmptySignature := RandomSignature()
		if nonEmptySignature.IsEmpty() {
			t.Error("Random signature should not be empty")
		}
	})
}

// TestBlockOperations 测试区块操作
func TestBlockOperations(t *testing.T) {
	t.Run("BlockHash", func(t *testing.T) {
		block := &Block{
			Header: BlockHeader{
				Number:    1,
				Timestamp: time.Now().Unix(),
				PrevHash:  RandomHash(),
				TxRoot:    EmptyTxRoot(),
				StateRoot: RandomHash(),
				Validator: RandomAddress(),
				ShardID:   ShardID,
			},
			Transactions: []Hash{},
		}

		hash1 := block.Hash()
		hash2 := block.Hash()

		if hash1 != hash2 {
			t.Error("Block hash should be deterministic")
		}

		if hash1.IsEmpty() {
			t.Error("Block hash should not be empty")
		}
	})

	t.Run("BlockEmpty", func(t *testing.T) {
		emptyBlock := &Block{
			Header: BlockHeader{
				Number: 1,
			},
			Transactions: []Hash{},
		}

		if !emptyBlock.IsEmpty() {
			t.Error("Block with no transactions should be empty")
		}

		nonEmptyBlock := &Block{
			Header: BlockHeader{
				Number: 1,
			},
			Transactions: []Hash{RandomHash()},
		}

		if nonEmptyBlock.IsEmpty() {
			t.Error("Block with transactions should not be empty")
		}
	})

	t.Run("BlockSize", func(t *testing.T) {
		block := &Block{
			Header: BlockHeader{
				Number: 1,
			},
			Transactions: []Hash{RandomHash(), RandomHash()},
		}

		size := block.Size()
		if size <= 0 {
			t.Error("Block size should be positive")
		}

		// 空区块应该比有交易的区块小
		emptyBlock := &Block{
			Header: BlockHeader{
				Number: 1,
			},
			Transactions: []Hash{},
		}

		emptySize := emptyBlock.Size()
		if emptySize >= size {
			t.Error("Empty block should be smaller than block with transactions")
		}
	})
}

// TestTransactionOperations 测试交易操作
func TestTransactionOperations(t *testing.T) {
	t.Run("TransactionHash", func(t *testing.T) {
		tx := &Transaction{
			ShardID:   ShardID,
			From:      RandomAddress(),
			To:        RandomAddress(),
			Amount:    1000,
			GasPrice:  100,
			GasLimit:  21000,
			Nonce:     1,
			Data:      []byte("test data"),
			Signature: RandomSignature(),
		}

		hash1 := tx.Hash()
		hash2 := tx.Hash()

		if hash1 != hash2 {
			t.Error("Transaction hash should be deterministic")
		}

		if hash1.IsEmpty() {
			t.Error("Transaction hash should not be empty")
		}
	})

	t.Run("TransactionVerify", func(t *testing.T) {
		// 有签名的交易
		txWithSignature := &Transaction{
			Signature: RandomSignature(),
		}

		if !txWithSignature.Verify() {
			t.Error("Transaction with signature should verify")
		}

		// 无签名的交易
		txWithoutSignature := &Transaction{
			Signature: Signature{}, // 空签名
		}

		if txWithoutSignature.Verify() {
			t.Error("Transaction without signature should not verify")
		}
	})
}

// TestValidatorOperations 测试验证者操作
func TestValidatorOperations(t *testing.T) {
	t.Run("ValidatorSetTotalVotePower", func(t *testing.T) {
		validators := []Validator{
			{
				Address:   RandomAddress(),
				VotePower: 1000,
				IsActive:  true,
			},
			{
				Address:   RandomAddress(),
				VotePower: 2000,
				IsActive:  true,
			},
			{
				Address:   RandomAddress(),
				VotePower: 1500,
				IsActive:  false, // 不活跃的验证者
			},
		}

		validatorSet := &ValidatorSet{
			Validators: validators,
			Round:      1,
		}

		totalPower := validatorSet.GetTotalVotePower()
		expected := uint64(3000) // 只计算活跃验证者的投票权重

		if totalPower != expected {
			t.Errorf("Total vote power should be %d, got %d", expected, totalPower)
		}
	})
}

// TestUtilityFunctions 测试工具函数
func TestUtilityFunctions(t *testing.T) {
	t.Run("EmptyTxRoot", func(t *testing.T) {
		emptyRoot := EmptyTxRoot()
		if emptyRoot.IsEmpty() {
			t.Error("Empty tx root should not be zero hash")
		}

		// 计算空交易列表的根应该与EmptyTxRoot()相同
		calculatedRoot := CalculateTxRoot([]Hash{})
		if calculatedRoot != emptyRoot {
			t.Error("Calculated empty root should match EmptyTxRoot()")
		}
	})

	t.Run("CalculateTxRoot", func(t *testing.T) {
		// 测试非空交易列表
		txHashes := []Hash{
			RandomHash(),
			RandomHash(),
			RandomHash(),
		}

		root := CalculateTxRoot(txHashes)
		if root.IsEmpty() {
			t.Error("Transaction root should not be empty")
		}

		// 相同输入应该产生相同输出
		root2 := CalculateTxRoot(txHashes)
		if root != root2 {
			t.Error("Transaction root calculation should be deterministic")
		}

		// 不同输入应该产生不同输出
		differentTxHashes := []Hash{
			RandomHash(),
			RandomHash(),
		}
		differentRoot := CalculateTxRoot(differentTxHashes)
		if root == differentRoot {
			t.Error("Different transaction lists should produce different roots")
		}
	})

	t.Run("BlockTimeUtilities", func(t *testing.T) {
		genesisTime := int64(1609459200)

		// 测试CalculateBlockTime
		blockTime := CalculateBlockTime(genesisTime, 5)
		expected := genesisTime + 10 // 5 blocks * 2 seconds
		if blockTime != expected {
			t.Errorf("Block time calculation wrong: expected %d, got %d", expected, blockTime)
		}

		// 测试ValidateBlockTime
		if !ValidateBlockTime(genesisTime, 3, genesisTime+6) {
			t.Error("Should validate correct block time")
		}

		if ValidateBlockTime(genesisTime, 3, genesisTime+7) {
			t.Error("Should reject incorrect block time")
		}
	})
}

// TestConstants 测试常量定义
func TestConstants(t *testing.T) {
	t.Run("BlockTimeConstant", func(t *testing.T) {
		if BlockTime.Seconds() != 2 {
			t.Errorf("Block time should be 2 seconds, got %v", BlockTime.Seconds())
		}
	})

	t.Run("MaxBlockSize", func(t *testing.T) {
		if MaxBlockSize != 2*1024*1024 {
			t.Errorf("Max block size should be 2MB, got %d", MaxBlockSize)
		}
	})

	t.Run("MaxTxPerBlock", func(t *testing.T) {
		if MaxTxPerBlock != 10000 {
			t.Errorf("Max transactions per block should be 10000, got %d", MaxTxPerBlock)
		}
	})

	t.Run("ValidatorCount", func(t *testing.T) {
		if ValidatorCount != 21 {
			t.Errorf("Validator count should be 21, got %d", ValidatorCount)
		}
	})

	t.Run("ShardID", func(t *testing.T) {
		if ShardID != 1 {
			t.Errorf("Shard ID should be 1, got %d", ShardID)
		}
	})
}
