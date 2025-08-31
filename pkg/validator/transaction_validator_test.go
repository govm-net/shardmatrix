package validator

import (
	"testing"

	"github.com/govm-net/shardmatrix/pkg/storage"
	"github.com/govm-net/shardmatrix/pkg/types"
)

func TestTransactionValidator_ValidateTransactionFormat(t *testing.T) {
	config := DefaultValidationConfig()
	validator := NewTransactionValidator(config, nil)

	tests := []struct {
		name        string
		tx          *types.Transaction
		expectError bool
		errorType   string
	}{
		{
			name: "valid transaction",
			tx: &types.Transaction{
				From:   testAddress("sender"),
				To:     testAddress("receiver"),
				Amount: 100,
				Fee:    10,
				Nonce:  1,
			},
			expectError: false,
		},
		{
			name: "empty from address",
			tx: &types.Transaction{
				From:   types.Address{}, // empty
				To:     testAddress("receiver"),
				Amount: 100,
				Fee:    10,
				Nonce:  1,
			},
			expectError: true,
			errorType:   "INVALID_FROM_ADDRESS",
		},
		{
			name: "empty to address",
			tx: &types.Transaction{
				From:   testAddress("sender"),
				To:     types.Address{}, // empty
				Amount: 100,
				Fee:    10,
				Nonce:  1,
			},
			expectError: true,
			errorType:   "INVALID_TO_ADDRESS",
		},
		{
			name: "same from and to address",
			tx: &types.Transaction{
				From:   testAddress("same"),
				To:     testAddress("same"), // same address
				Amount: 100,
				Fee:    10,
				Nonce:  1,
			},
			expectError: true,
			errorType:   "INVALID_ADDRESSES",
		},
		{
			name: "zero fee",
			tx: &types.Transaction{
				From:   testAddress("sender"),
				To:     testAddress("receiver"),
				Amount: 100,
				Fee:    0, // zero fee
				Nonce:  1,
			},
			expectError: true,
			errorType:   "INVALID_FEE",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateTransactionFormat(tt.tx)

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

func TestTransactionValidator_ValidateTransactionSignature(t *testing.T) {
	config := DefaultValidationConfig()
	validator := NewTransactionValidator(config, nil)

	// Test transaction without signature
	txWithoutSig := &types.Transaction{
		From:   testAddress("sender"),
		To:     testAddress("receiver"),
		Amount: 100,
		Fee:    10,
		Nonce:  1,
		// No signature
	}

	err := validator.ValidateTransactionSignature(txWithoutSig)
	if err == nil {
		t.Error("transaction without signature should fail")
	}

	validationErr, ok := err.(*ValidationError)
	if !ok {
		t.Errorf("expected ValidationError, got %T", err)
	}

	if validationErr.Type != "MISSING_SIGNATURE" {
		t.Errorf("expected MISSING_SIGNATURE error, got %s", validationErr.Type)
	}

	// Test transaction with signature (should pass with simplified validation)
	txWithSig := &types.Transaction{
		From:      testAddress("sender"),
		To:        testAddress("receiver"),
		Amount:    100,
		Fee:       10,
		Nonce:     1,
		Signature: []byte("dummy_signature"),
	}

	err = validator.ValidateTransactionSignature(txWithSig)
	if err != nil {
		t.Errorf("transaction with signature should pass (simplified validation), got: %v", err)
	}
}

func TestTransactionValidator_ValidateAccountBalance(t *testing.T) {
	config := DefaultValidationConfig()

	// Create memory account store
	accountStore := storage.NewMemoryAccountStore()

	// Create sender account with balance
	senderAddr := testAddress("sender")
	senderAccount := types.NewAccountWithBalance(senderAddr, 1000)
	accountStore.PutAccount(senderAccount)

	validator := NewTransactionValidator(config, accountStore)

	// Test transaction with sufficient balance
	validTx := &types.Transaction{
		From:   senderAddr,
		To:     testAddress("receiver"),
		Amount: 100,
		Fee:    10,
		Nonce:  1,
	}

	err := validator.ValidateAccountBalance(validTx)
	if err != nil {
		t.Errorf("transaction with sufficient balance should pass, got: %v", err)
	}

	// Test transaction with insufficient balance
	invalidTx := &types.Transaction{
		From:   senderAddr,
		To:     testAddress("receiver"),
		Amount: 1000,
		Fee:    100, // total cost > balance
		Nonce:  1,
	}

	err = validator.ValidateAccountBalance(invalidTx)
	if err == nil {
		t.Error("transaction with insufficient balance should fail")
	}

	validationErr, ok := err.(*ValidationError)
	if !ok {
		t.Errorf("expected ValidationError, got %T", err)
	}

	if validationErr.Type != "INSUFFICIENT_BALANCE" {
		t.Errorf("expected INSUFFICIENT_BALANCE error, got %s", validationErr.Type)
	}

	// Test transaction from non-existent account
	nonExistentTx := &types.Transaction{
		From:   testAddress("nonexistent"),
		To:     testAddress("receiver"),
		Amount: 100,
		Fee:    10,
		Nonce:  1,
	}

	err = validator.ValidateAccountBalance(nonExistentTx)
	if err == nil {
		t.Error("transaction from non-existent account should fail")
	}

	validationErr, ok = err.(*ValidationError)
	if !ok {
		t.Errorf("expected ValidationError, got %T", err)
	}

	if validationErr.Type != "ACCOUNT_NOT_FOUND" {
		t.Errorf("expected ACCOUNT_NOT_FOUND error, got %s", validationErr.Type)
	}
}

func TestTransactionValidator_ValidateTransaction(t *testing.T) {
	config := DefaultValidationConfig()

	// Create memory account store
	accountStore := storage.NewMemoryAccountStore()

	// Create sender account with balance
	senderAddr := testAddress("sender")
	senderAccount := types.NewAccountWithBalance(senderAddr, 1000)
	accountStore.PutAccount(senderAccount)

	validator := NewTransactionValidator(config, accountStore)

	// Test valid transaction
	validTx := &types.Transaction{
		From:      senderAddr,
		To:        testAddress("receiver"),
		Amount:    100,
		Fee:       10,
		Nonce:     1,
		Signature: []byte("dummy_signature"),
	}

	err := validator.ValidateTransaction(validTx)
	if err != nil {
		t.Errorf("valid transaction should pass, got: %v", err)
	}

	// Test nil transaction
	err = validator.ValidateTransaction(nil)
	if err == nil {
		t.Error("nil transaction should fail")
	}

	validationErr, ok := err.(*ValidationError)
	if !ok {
		t.Errorf("expected ValidationError, got %T", err)
	}

	if validationErr.Type != "INVALID_TRANSACTION" {
		t.Errorf("expected INVALID_TRANSACTION error, got %s", validationErr.Type)
	}
}

func TestTransactionValidator_ValidateTransactionList(t *testing.T) {
	config := DefaultValidationConfig()

	// Create memory account store with accounts
	accountStore := storage.NewMemoryAccountStore()

	senderAddr1 := testAddress("sender1")
	senderAddr2 := testAddress("sender2")

	senderAccount1 := types.NewAccountWithBalance(senderAddr1, 1000)
	senderAccount2 := types.NewAccountWithBalance(senderAddr2, 1000)

	accountStore.PutAccount(senderAccount1)
	accountStore.PutAccount(senderAccount2)

	validator := NewTransactionValidator(config, accountStore)

	// Create valid transaction list
	tx1 := &types.Transaction{
		From:      senderAddr1,
		To:        testAddress("receiver1"),
		Amount:    100,
		Fee:       10,
		Nonce:     1,
		Signature: []byte("signature1"),
	}

	tx2 := &types.Transaction{
		From:      senderAddr2,
		To:        testAddress("receiver2"),
		Amount:    200,
		Fee:       20,
		Nonce:     1,
		Signature: []byte("signature2"),
	}

	validTxList := []*types.Transaction{tx1, tx2}

	err := validator.ValidateTransactionList(validTxList)
	if err != nil {
		t.Errorf("valid transaction list should pass, got: %v", err)
	}

	// Test list with duplicate transaction
	duplicateTxList := []*types.Transaction{tx1, tx1} // duplicate tx1

	err = validator.ValidateTransactionList(duplicateTxList)
	if err == nil {
		t.Error("transaction list with duplicates should fail")
	}

	// Test list with nil transaction
	nilTxList := []*types.Transaction{tx1, nil, tx2}

	err = validator.ValidateTransactionList(nilTxList)
	if err == nil {
		t.Error("transaction list with nil transaction should fail")
	}
}

func TestTransactionValidator_ValidateNonceOrder(t *testing.T) {
	validator := NewTransactionValidator(DefaultValidationConfig(), nil)

	senderAddr := testAddress("sender")

	// Test valid nonce order
	tx1 := &types.Transaction{From: senderAddr, Nonce: 1}
	tx2 := &types.Transaction{From: senderAddr, Nonce: 2}
	tx3 := &types.Transaction{From: senderAddr, Nonce: 3}

	validTxList := []*types.Transaction{tx1, tx2, tx3}

	err := validator.ValidateNonceOrder(validTxList)
	if err != nil {
		t.Errorf("valid nonce order should pass, got: %v", err)
	}

	// Test duplicate nonce
	tx4 := &types.Transaction{From: senderAddr, Nonce: 2} // duplicate nonce
	duplicateNonceTxList := []*types.Transaction{tx1, tx2, tx4}

	err = validator.ValidateNonceOrder(duplicateNonceTxList)
	if err == nil {
		t.Error("duplicate nonce should fail validation")
	}

	validationErr, ok := err.(*ValidationError)
	if !ok {
		t.Errorf("expected ValidationError, got %T", err)
	}

	if validationErr.Type != "DUPLICATE_NONCE" {
		t.Errorf("expected DUPLICATE_NONCE error, got %s", validationErr.Type)
	}

	// Test different senders (should pass even with same nonce)
	sender2 := testAddress("sender2")
	tx5 := &types.Transaction{From: sender2, Nonce: 1} // same nonce as tx1 but different sender

	multiSenderTxList := []*types.Transaction{tx1, tx5}

	err = validator.ValidateNonceOrder(multiSenderTxList)
	if err != nil {
		t.Errorf("same nonce from different senders should pass, got: %v", err)
	}
}
