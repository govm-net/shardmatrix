package validator

import (
	"testing"
	"time"

	"github.com/govm-net/shardmatrix/pkg/storage"
	"github.com/govm-net/shardmatrix/pkg/types"
)

func TestValidationError(t *testing.T) {
	err := NewValidationError("TEST_ERROR", "test message")
	expectedMsg := "validation error [TEST_ERROR]: test message"
	if err.Error() != expectedMsg {
		t.Errorf("expected error message %s, got %s", expectedMsg, err.Error())
	}
}

func TestDefaultValidationConfig(t *testing.T) {
	config := DefaultValidationConfig()

	if config.MaxBlockSize <= 0 {
		t.Error("MaxBlockSize should be positive")
	}

	if config.MaxTransactions <= 0 {
		t.Error("MaxTransactions should be positive")
	}

	if config.BlockTimeWindow <= 0 {
		t.Error("BlockTimeWindow should be positive")
	}

	if !config.RequireSignature {
		t.Error("RequireSignature should be true by default")
	}
}

func TestBlockValidator_ValidateBlockHeader(t *testing.T) {
	config := DefaultValidationConfig()
	blockStore := storage.NewMemoryBlockStore()

	validator := NewBlockValidator(config, blockStore, nil, nil, nil)

	tests := []struct {
		name        string
		header      *types.BlockHeader
		expectError bool
		errorType   string
	}{
		{
			name:        "nil header",
			header:      nil,
			expectError: true,
			errorType:   "INVALID_HEADER",
		},
		{
			name: "empty validator address",
			header: &types.BlockHeader{
				Number:    1,
				Timestamp: time.Now().Unix(),
				Validator: types.Address{}, // empty address
			},
			expectError: true,
			errorType:   "INVALID_VALIDATOR",
		},
		{
			name: "invalid timestamp",
			header: &types.BlockHeader{
				Number:    1,
				Timestamp: -1,
				Validator: testAddress("validator1"),
			},
			expectError: true,
			errorType:   "INVALID_TIMESTAMP",
		},
		{
			name: "valid header",
			header: &types.BlockHeader{
				Number:    1,
				Timestamp: time.Now().Unix(),
				Validator: testAddress("validator1"),
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateBlockHeader(tt.header)

			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
					return
				}

				validationErr, ok := err.(*ValidationError)
				if !ok {
					t.Errorf("expected ValidationError, got %T", err)
					return
				}

				if validationErr.Type != tt.errorType {
					t.Errorf("expected error type %s, got %s", tt.errorType, validationErr.Type)
				}
			} else {
				if err != nil {
					t.Errorf("expected no error but got: %v", err)
				}
			}
		})
	}
}

func TestBlockValidator_ValidateBlockSize(t *testing.T) {
	config := DefaultValidationConfig()
	config.MaxBlockSize = 1000 // 1KB limit

	blockStore := storage.NewMemoryBlockStore()
	validator := NewBlockValidator(config, blockStore, nil, nil, nil)

	// Create a small block
	smallBlock := types.NewBlock(1, types.EmptyHash(), testAddress("validator1"))

	// Create a large block with many transactions
	largeBlock := types.NewBlock(1, types.EmptyHash(), testAddress("validator1"))
	for i := 0; i < 100; i++ {
		txHash := types.CalculateHash([]byte("large_transaction_data_to_make_block_large"))
		largeBlock.AddTransaction(txHash)
	}

	// Test small block (should pass)
	err := validator.ValidateBlockSize(smallBlock)
	if err != nil {
		t.Errorf("small block should pass validation, got error: %v", err)
	}

	// Test large block (should fail)
	err = validator.ValidateBlockSize(largeBlock)
	if err == nil {
		t.Error("large block should fail validation")
	}

	validationErr, ok := err.(*ValidationError)
	if !ok {
		t.Errorf("expected ValidationError, got %T", err)
	}

	if validationErr.Type != "BLOCK_TOO_LARGE" {
		t.Errorf("expected BLOCK_TOO_LARGE error, got %s", validationErr.Type)
	}
}

func TestBlockValidator_ValidateTransactionCount(t *testing.T) {
	config := DefaultValidationConfig()
	config.MaxTransactions = 5
	config.AllowEmptyBlocks = false

	blockStore := storage.NewMemoryBlockStore()
	validator := NewBlockValidator(config, blockStore, nil, nil, nil)

	// Test empty block when not allowed
	emptyBlock := types.NewBlock(1, types.EmptyHash(), testAddress("validator1"))
	err := validator.ValidateTransactionCount(emptyBlock)
	if err == nil {
		t.Error("empty block should fail when not allowed")
	}

	// Test block with too many transactions
	largeBlock := types.NewBlock(1, types.EmptyHash(), testAddress("validator1"))
	for i := 0; i < 10; i++ {
		txHash := types.CalculateHash([]byte("transaction"))
		largeBlock.AddTransaction(txHash)
	}

	err = validator.ValidateTransactionCount(largeBlock)
	if err == nil {
		t.Error("block with too many transactions should fail")
	}

	// Test valid block
	validBlock := types.NewBlock(1, types.EmptyHash(), testAddress("validator1"))
	for i := 0; i < 3; i++ {
		txHash := types.CalculateHash([]byte("transaction"))
		validBlock.AddTransaction(txHash)
	}

	err = validator.ValidateTransactionCount(validBlock)
	if err != nil {
		t.Errorf("valid block should pass, got error: %v", err)
	}
}

func TestBlockValidator_ValidateTransactionRoot(t *testing.T) {
	config := DefaultValidationConfig()
	blockStore := storage.NewMemoryBlockStore()
	validator := NewBlockValidator(config, blockStore, nil, nil, nil)

	// Create block with transactions
	block := types.NewBlock(1, types.EmptyHash(), testAddress("validator1"))
	txHash1 := types.CalculateHash([]byte("transaction1"))
	txHash2 := types.CalculateHash([]byte("transaction2"))

	block.AddTransaction(txHash1)
	block.AddTransaction(txHash2)

	// Should pass with correct root
	err := validator.ValidateTransactionRoot(block)
	if err != nil {
		t.Errorf("valid transaction root should pass, got error: %v", err)
	}

	// Manually corrupt the transaction root
	block.Header.TxRoot = types.CalculateHash([]byte("wrong_root"))

	// Should fail with incorrect root
	err = validator.ValidateTransactionRoot(block)
	if err == nil {
		t.Error("corrupted transaction root should fail validation")
	}

	validationErr, ok := err.(*ValidationError)
	if !ok {
		t.Errorf("expected ValidationError, got %T", err)
	}

	if validationErr.Type != "INVALID_TX_ROOT" {
		t.Errorf("expected INVALID_TX_ROOT error, got %s", validationErr.Type)
	}
}

func TestBlockValidator_ValidateTimestamp(t *testing.T) {
	config := DefaultValidationConfig()
	config.BlockTimeWindow = time.Minute * 5 // 5 minutes

	blockStore := storage.NewMemoryBlockStore()
	validator := NewBlockValidator(config, blockStore, nil, nil, nil)

	// Test genesis block (should pass)
	genesisBlock := types.NewGenesisBlock(testAddress("validator1"))
	genesisBlock.Header.Timestamp = time.Now().Unix()

	err := validator.ValidateTimestamp(genesisBlock)
	if err != nil {
		t.Errorf("genesis block should pass timestamp validation, got: %v", err)
	}

	// Test block with timestamp too far in the future
	futureBlock := types.NewBlock(1, types.EmptyHash(), testAddress("validator1"))
	futureBlock.Header.Timestamp = time.Now().Add(time.Hour).Unix() // 1 hour in future

	err = validator.ValidateTimestamp(futureBlock)
	if err == nil {
		t.Error("block with future timestamp should fail")
	}

	// Test valid timestamp
	validBlock := types.NewBlock(1, types.EmptyHash(), testAddress("validator1"))
	validBlock.Header.Timestamp = time.Now().Unix()

	// For non-genesis blocks, we need to store the previous block
	prevBlock := types.NewGenesisBlock(testAddress("validator1"))
	prevBlock.Header.Timestamp = validBlock.Header.Timestamp - 100 // Earlier timestamp
	blockStore.PutBlock(prevBlock)
	validBlock.Header.PrevHash = prevBlock.Hash()
	validBlock.Header.Number = 1

	err = validator.ValidateTimestamp(validBlock)
	if err != nil {
		t.Errorf("valid timestamp should pass, got: %v", err)
	}
}

func TestBlockValidator_ValidatePreviousBlock(t *testing.T) {
	config := DefaultValidationConfig()
	blockStore := storage.NewMemoryBlockStore()
	validator := NewBlockValidator(config, blockStore, nil, nil, nil)

	// Test genesis block (should pass)
	genesisBlock := types.NewGenesisBlock(testAddress("validator1"))

	err := validator.ValidatePreviousBlock(genesisBlock)
	if err != nil {
		t.Errorf("genesis block should pass previous block validation, got: %v", err)
	}

	// Store genesis block
	blockStore.PutBlock(genesisBlock)

	// Test block with valid previous block
	validBlock := types.NewBlock(1, genesisBlock.Hash(), testAddress("validator1"))

	err = validator.ValidatePreviousBlock(validBlock)
	if err != nil {
		t.Errorf("block with valid previous block should pass, got: %v", err)
	}

	// Test block with non-existent previous block
	invalidBlock := types.NewBlock(1, types.CalculateHash([]byte("nonexistent")), testAddress("validator1"))

	err = validator.ValidatePreviousBlock(invalidBlock)
	if err == nil {
		t.Error("block with non-existent previous block should fail")
	}
}

func TestBlockValidator_ValidateGenesisBlock(t *testing.T) {
	config := DefaultValidationConfig()
	blockStore := storage.NewMemoryBlockStore()
	validator := NewBlockValidator(config, blockStore, nil, nil, nil)

	// Test valid genesis block
	validGenesis := types.NewGenesisBlock(testAddress("validator1"))

	err := validator.ValidateGenesisBlock(validGenesis)
	if err != nil {
		t.Errorf("valid genesis block should pass, got: %v", err)
	}

	// Test non-genesis block
	nonGenesis := types.NewBlock(1, types.EmptyHash(), testAddress("validator1"))

	err = validator.ValidateGenesisBlock(nonGenesis)
	if err == nil {
		t.Error("non-genesis block should fail genesis validation")
	}

	// Test genesis block with non-empty previous hash
	invalidGenesis := types.NewBlock(0, types.CalculateHash([]byte("not_empty")), testAddress("validator1"))

	err = validator.ValidateGenesisBlock(invalidGenesis)
	if err == nil {
		t.Error("genesis block with non-empty previous hash should fail")
	}
}

func TestBlockValidator_ValidateBlock(t *testing.T) {
	config := DefaultValidationConfig()
	config.RequireSignature = false // Disable signature validation for this test

	blockStore := storage.NewMemoryBlockStore()
	validator := NewBlockValidator(config, blockStore, nil, nil, nil)

	// Create and store genesis block
	genesisBlock := types.NewGenesisBlock(testAddress("validator1"))
	genesisBlock.Header.Timestamp = time.Now().Unix() - 100 // Earlier timestamp
	blockStore.PutBlock(genesisBlock)

	// Test valid block
	validBlock := types.NewBlock(1, genesisBlock.Hash(), testAddress("validator1"))
	validBlock.Header.Timestamp = genesisBlock.Header.Timestamp + 10 // Later timestamp

	err := validator.ValidateBlock(validBlock)
	if err != nil {
		t.Errorf("valid block should pass validation, got: %v", err)
	}

	// Test invalid block (nil)
	err = validator.ValidateBlock(nil)
	if err == nil {
		t.Error("nil block should fail validation")
	}
}
