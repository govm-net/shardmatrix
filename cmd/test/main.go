package main

import (
	"fmt"
	"time"

	"github.com/lengzhao/shardmatrix/pkg/consensus"
	"github.com/lengzhao/shardmatrix/pkg/crypto"
	"github.com/lengzhao/shardmatrix/pkg/types"
)

func main() {
	fmt.Println("🔗 ShardMatrix Core Feature Test")
	fmt.Println("===============================")
	
	// Test 1: Crypto operations
	fmt.Println("\n1. Testing Cryptographic Operations...")
	testCrypto()
	
	// Test 2: Time controller
	fmt.Println("\n2. Testing Strict Time Controller...")
	testTimeController()
	
	// Test 3: Block and transaction types
	fmt.Println("\n3. Testing Block and Transaction Types...")
	testTypes()
	
	fmt.Println("\n✅ All core tests passed successfully!")
}

func testCrypto() {
	// Generate keypair
	keyPair, err := crypto.GenerateKeyPair()
	if err != nil {
		panic(err)
	}
	
	fmt.Printf("   ✓ Generated keypair: %s\n", keyPair.Address.String()[:16]+"...")
	
	// Test signing
	data := []byte("test data")
	signature := keyPair.Sign(data)
	
	// Test verification
	isValid := crypto.VerifySignature(keyPair.PublicKey, data, signature)
	if !isValid {
		panic("signature verification failed")
	}
	
	fmt.Printf("   ✓ Signature verification passed\n")
	
	// Test hash consistency
	hash1 := crypto.Hash(data)
	hash2 := crypto.Hash(data)
	if hash1 != hash2 {
		panic("hash not deterministic")
	}
	
	fmt.Printf("   ✓ Hash calculation is deterministic\n")
}

func testTimeController() {
	// Create time controller
	genesisTime := time.Now().Unix()
	if genesisTime%2 != 0 {
		genesisTime--
	}
	
	timeController := consensus.NewTimeController(genesisTime)
	
	fmt.Printf("   ✓ Time controller created (genesis: %d)\n", genesisTime)
	
	// Test block time calculation
	blockTime1 := timeController.GetBlockTime(1)
	expectedTime1 := genesisTime + 2
	if blockTime1 != expectedTime1 {
		panic(fmt.Sprintf("block time calculation failed: expected %d, got %d", expectedTime1, blockTime1))
	}
	
	fmt.Printf("   ✓ Block time calculation correct (block 1: %d)\n", blockTime1)
	
	// Test time validation
	isValid := timeController.ValidateBlockTime(1, expectedTime1)
	if !isValid {
		panic("time validation failed for correct timestamp")
	}
	
	isInvalid := timeController.ValidateBlockTime(1, expectedTime1+1)
	if isInvalid {
		panic("time validation passed for incorrect timestamp")
	}
	
	fmt.Printf("   ✓ Strict time validation working correctly\n")
}

func testTypes() {
	// Test empty transaction root
	emptyRoot := types.EmptyTxRoot()
	calculatedRoot := types.CalculateTxRoot([]types.Hash{})
	
	if emptyRoot != calculatedRoot {
		panic("empty transaction root mismatch")
	}
	
	fmt.Printf("   ✓ Empty transaction root calculation correct\n")
	
	// Test block time calculation
	genesisTime := int64(1000000000)
	blockTime := types.CalculateBlockTime(genesisTime, 5)
	expectedTime := genesisTime + 10
	
	if blockTime != expectedTime {
		panic(fmt.Sprintf("block time calculation failed: expected %d, got %d", expectedTime, blockTime))
	}
	
	fmt.Printf("   ✓ Block time calculation utility correct\n")
	
	// Test time validation
	isValid := types.ValidateBlockTime(genesisTime, 3, genesisTime+6)
	if !isValid {
		panic("block time validation failed")
	}
	
	isInvalid := types.ValidateBlockTime(genesisTime, 3, genesisTime+7)
	if isInvalid {
		panic("block time validation should fail for wrong timestamp")
	}
	
	fmt.Printf("   ✓ Block time validation utility correct\n")
	
	// Create test block
	block := &types.Block{
		Header: types.BlockHeader{
			Number:    1,
			Timestamp: genesisTime + 2,
			TxRoot:    emptyRoot,
		},
		Transactions: []types.Hash{},
	}
	
	if !block.IsEmpty() {
		panic("block should be empty")
	}
	
	hash1 := block.Hash()
	hash2 := block.Hash()
	if hash1 != hash2 {
		panic("block hash not deterministic")
	}
	
	fmt.Printf("   ✓ Block creation and hashing working correctly\n")
	
	// Test constants
	if types.BlockTime.Seconds() != 2 {
		panic("block time constant incorrect")
	}
	
	if types.ShardID != 1 {
		panic("shard ID constant incorrect")
	}
	
	fmt.Printf("   ✓ All constants validated\n")
}