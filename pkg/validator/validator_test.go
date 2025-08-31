package validator

import (
	"testing"
	"time"

	"github.com/govm-net/shardmatrix/pkg/storage"
	"github.com/govm-net/shardmatrix/pkg/types"
)

func TestValidator_NewValidator(t *testing.T) {
	config := DefaultValidationConfig()
	blockStore := storage.NewMemoryBlockStore()
	transactionStore := storage.NewMemoryTransactionStore()
	validatorStore := storage.NewMemoryValidatorStore()
	accountStore := storage.NewMemoryAccountStore()

	validator := NewValidator(
		config,
		blockStore,
		transactionStore,
		validatorStore,
		accountStore,
	)

	if validator == nil {
		t.Error("validator should not be nil")
	}

	if validator.config != config {
		t.Error("validator config should match input config")
	}

	if validator.GetConfig() != config {
		t.Error("GetConfig should return the same config")
	}
}

func TestValidator_ValidateTransaction(t *testing.T) {
	config := DefaultValidationConfig()
	accountStore := storage.NewMemoryAccountStore()

	// Create sender account
	senderAddr := testAddress("sender")
	senderAccount := types.NewAccountWithBalance(senderAddr, 1000)
	accountStore.PutAccount(senderAccount)

	validator := NewValidator(config, nil, nil, nil, accountStore)

	// Test valid transaction
	validTx := &types.Transaction{
		From:      senderAddr,
		To:        testAddress("receiver"),
		Amount:    100,
		Fee:       10,
		Nonce:     1,
		Signature: []byte("signature"),
	}

	err := validator.ValidateTransaction(validTx)
	if err != nil {
		t.Errorf("valid transaction should pass, got: %v", err)
	}

	// Test invalid transaction
	invalidTx := &types.Transaction{
		From:   types.Address{}, // empty from address
		To:     testAddress("receiver"),
		Amount: 100,
		Fee:    10,
		Nonce:  1,
	}

	err = validator.ValidateTransaction(invalidTx)
	if err == nil {
		t.Error("invalid transaction should fail")
	}
}

func TestValidator_ValidateGenesisBlock(t *testing.T) {
	config := DefaultValidationConfig()
	blockStore := storage.NewMemoryBlockStore()

	validator := NewValidator(config, blockStore, nil, nil, nil)

	// Test valid genesis block
	validGenesis := types.NewGenesisBlock(testAddress("validator"))

	err := validator.ValidateGenesisBlock(validGenesis)
	if err != nil {
		t.Errorf("valid genesis block should pass, got: %v", err)
	}

	// Test invalid genesis block
	invalidGenesis := types.NewBlock(1, types.EmptyHash(), testAddress("validator"))

	err = validator.ValidateGenesisBlock(invalidGenesis)
	if err == nil {
		t.Error("non-genesis block should fail genesis validation")
	}
}

func TestValidator_ValidateNewBlock(t *testing.T) {
	config := DefaultValidationConfig()
	config.RequireSignature = false // Disable signature validation

	blockStore := storage.NewMemoryBlockStore()
	transactionStore := storage.NewMemoryTransactionStore()
	accountStore := storage.NewMemoryAccountStore()

	validator := NewValidator(config, blockStore, transactionStore, nil, accountStore)

	// Create and store genesis block
	genesisBlock := types.NewGenesisBlock(testAddress("validator"))
	genesisBlock.Header.Timestamp = time.Now().Unix() - 100 // Earlier timestamp
	blockStore.PutBlock(genesisBlock)

	// Create sender account
	senderAddr := testAddress("sender")
	senderAccount := types.NewAccountWithBalance(senderAddr, 1000)
	accountStore.PutAccount(senderAccount)

	// Create and store valid transaction
	tx := &types.Transaction{
		From:      senderAddr,
		To:        testAddress("receiver"),
		Amount:    100,
		Fee:       10,
		Nonce:     1,
		Signature: []byte("signature"),
	}
	transactionStore.PutTransaction(tx)

	// Create valid block with transaction
	validBlock := types.NewBlock(1, genesisBlock.Hash(), testAddress("validator"))
	validBlock.Header.Timestamp = genesisBlock.Header.Timestamp + 10 // Later timestamp
	validBlock.AddTransaction(tx.Hash())

	err := validator.ValidateNewBlock(validBlock, transactionStore)
	if err != nil {
		t.Errorf("valid block should pass, got: %v", err)
	}

	// Test nil block
	err = validator.ValidateNewBlock(nil, transactionStore)
	if err == nil {
		t.Error("nil block should fail validation")
	}
}

func TestValidator_ValidateBlockSequence(t *testing.T) {
	config := DefaultValidationConfig()
	config.RequireSignature = false

	blockStore := storage.NewMemoryBlockStore()
	validator := NewValidator(config, blockStore, nil, nil, nil)

	// Create block sequence
	genesisBlock := types.NewGenesisBlock(testAddress("validator"))
	genesisBlock.Header.Timestamp = time.Now().Unix() - 100 // Earlier timestamp

	block1 := types.NewBlock(1, genesisBlock.Hash(), testAddress("validator"))
	block1.Header.Timestamp = genesisBlock.Header.Timestamp + 10

	block2 := types.NewBlock(2, block1.Hash(), testAddress("validator"))
	block2.Header.Timestamp = block1.Header.Timestamp + 10

	validSequence := []*types.Block{genesisBlock, block1, block2}

	// Store the genesis block first so the validator can find it
	blockStore.PutBlock(genesisBlock)
	// Store block1 so block2 validation can find it
	blockStore.PutBlock(block1)

	// Test valid sequence
	err := validator.ValidateBlockSequence(validSequence)
	if err != nil {
		t.Errorf("valid block sequence should pass, got: %v", err)
	}

	// Test invalid sequence (broken chain)
	invalidBlock := types.NewBlock(2, types.CalculateHash([]byte("wrong_hash")), testAddress("validator"))
	invalidBlock.Header.Timestamp = block1.Header.Timestamp + 10

	invalidSequence := []*types.Block{genesisBlock, block1, invalidBlock}

	err = validator.ValidateBlockSequence(invalidSequence)
	if err == nil {
		t.Error("invalid block sequence should fail")
	}

	// Test empty sequence
	err = validator.ValidateBlockSequence([]*types.Block{})
	if err != nil {
		t.Errorf("empty sequence should pass, got: %v", err)
	}
}

func TestValidator_ValidateChainIntegrity(t *testing.T) {
	config := DefaultValidationConfig()
	config.RequireSignature = false

	blockStore := storage.NewMemoryBlockStore()
	validator := NewValidator(config, blockStore, nil, nil, nil)

	// Create and store block chain
	genesisBlock := types.NewGenesisBlock(testAddress("validator"))
	blockStore.PutBlock(genesisBlock)

	block1 := types.NewBlock(1, genesisBlock.Hash(), testAddress("validator"))
	block1.Header.Timestamp = genesisBlock.Header.Timestamp + 10
	blockStore.PutBlock(block1)

	block2 := types.NewBlock(2, block1.Hash(), testAddress("validator"))
	block2.Header.Timestamp = block1.Header.Timestamp + 10
	blockStore.PutBlock(block2)

	// Test valid chain integrity
	err := validator.ValidateChainIntegrity(0, 2, blockStore)
	if err != nil {
		t.Errorf("valid chain should pass integrity check, got: %v", err)
	}

	// Test invalid range
	err = validator.ValidateChainIntegrity(2, 0, blockStore)
	if err == nil {
		t.Error("invalid range should fail")
	}

	// Test missing block
	err = validator.ValidateChainIntegrity(0, 5, blockStore)
	if err == nil {
		t.Error("missing block should fail integrity check")
	}
}

func TestValidator_UpdateConfig(t *testing.T) {
	config1 := DefaultValidationConfig()
	config1.MaxBlockSize = 1000

	validator := NewValidator(config1, nil, nil, nil, nil)

	if validator.GetConfig().MaxBlockSize != 1000 {
		t.Error("initial config should be set correctly")
	}

	// Update config
	config2 := DefaultValidationConfig()
	config2.MaxBlockSize = 2000

	validator.UpdateConfig(config2)

	if validator.GetConfig().MaxBlockSize != 2000 {
		t.Error("config should be updated")
	}

	// Test nil config (should not crash)
	validator.UpdateConfig(nil)

	if validator.GetConfig().MaxBlockSize != 2000 {
		t.Error("config should remain unchanged when nil is passed")
	}
}

func TestValidator_ValidateTransactionList(t *testing.T) {
	config := DefaultValidationConfig()
	accountStore := storage.NewMemoryAccountStore()

	// Create sender account
	senderAddr := testAddress("sender")
	senderAccount := types.NewAccountWithBalance(senderAddr, 1000)
	accountStore.PutAccount(senderAccount)

	validator := NewValidator(config, nil, nil, nil, accountStore)

	// Create valid transactions
	tx1 := &types.Transaction{
		From:      senderAddr,
		To:        testAddress("receiver1"),
		Amount:    100,
		Fee:       10,
		Nonce:     1,
		Signature: []byte("signature1"),
	}

	tx2 := &types.Transaction{
		From:      senderAddr,
		To:        testAddress("receiver2"),
		Amount:    200,
		Fee:       20,
		Nonce:     2,
		Signature: []byte("signature2"),
	}

	validTxList := []*types.Transaction{tx1, tx2}

	err := validator.ValidateTransactionList(validTxList)
	if err != nil {
		t.Errorf("valid transaction list should pass, got: %v", err)
	}

	// Test empty list
	err = validator.ValidateTransactionList([]*types.Transaction{})
	if err != nil {
		t.Errorf("empty transaction list should pass, got: %v", err)
	}
}
