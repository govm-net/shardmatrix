package integration

import (
	"testing"

	"github.com/govm-net/shardmatrix/pkg/crypto"
	"github.com/govm-net/shardmatrix/pkg/types"
)

func TestTransactionSigning(t *testing.T) {
	// 生成密钥对
	keyPair, err := crypto.GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate key pair: %v", err)
	}

	// 创建交易
	from := keyPair.GetAddress()
	to := types.AddressFromPublicKey([]byte("receiver_public_key"))
	tx := types.NewTransaction(from, to, 100, 10, 1, []byte("test transaction"))

	// 创建签名函数
	signFunc := func(data []byte) ([]byte, error) {
		return keyPair.SignData(data)
	}

	// 创建验证函数
	verifyFunc := func(data []byte, signature []byte) bool {
		return keyPair.VerifySignature(data, signature)
	}

	// 签名交易
	err = tx.SignWithFunc(signFunc)
	if err != nil {
		t.Fatalf("Failed to sign transaction: %v", err)
	}

	// 验证签名不为空
	if len(tx.Signature) == 0 {
		t.Error("Transaction signature should not be empty after signing")
	}

	// 验证交易签名
	isValid := tx.VerifyWithFunc(verifyFunc)
	if !isValid {
		t.Error("Transaction signature verification should succeed")
	}

	// 创建错误的验证函数（使用不同的密钥对）
	wrongKeyPair, err := crypto.GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate wrong key pair: %v", err)
	}

	wrongVerifyFunc := func(data []byte, signature []byte) bool {
		return wrongKeyPair.VerifySignature(data, signature)
	}

	// 使用错误的公钥验证应该失败
	isValid = tx.VerifyWithFunc(wrongVerifyFunc)
	if isValid {
		t.Error("Transaction signature verification should fail with wrong public key")
	}
}

func TestMultipleTransactionSigning(t *testing.T) {
	// 创建多个用户
	alice, err := crypto.GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate Alice's key pair: %v", err)
	}

	bob, err := crypto.GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate Bob's key pair: %v", err)
	}

	charlie, err := crypto.GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate Charlie's key pair: %v", err)
	}

	// 创建交易：Alice -> Bob
	tx1 := types.NewTransaction(
		alice.GetAddress(),
		bob.GetAddress(),
		100, 5, 1,
		[]byte("Alice to Bob"),
	)

	// 创建交易：Bob -> Charlie
	tx2 := types.NewTransaction(
		bob.GetAddress(),
		charlie.GetAddress(),
		50, 3, 1,
		[]byte("Bob to Charlie"),
	)

	// Alice签名第一个交易
	aliceSignFunc := func(data []byte) ([]byte, error) {
		return alice.SignData(data)
	}
	err = tx1.SignWithFunc(aliceSignFunc)
	if err != nil {
		t.Fatalf("Failed to sign Alice's transaction: %v", err)
	}

	// Bob签名第二个交易
	bobSignFunc := func(data []byte) ([]byte, error) {
		return bob.SignData(data)
	}
	err = tx2.SignWithFunc(bobSignFunc)
	if err != nil {
		t.Fatalf("Failed to sign Bob's transaction: %v", err)
	}

	// 验证Alice的交易
	aliceVerifyFunc := func(data []byte, signature []byte) bool {
		return alice.VerifySignature(data, signature)
	}
	if !tx1.VerifyWithFunc(aliceVerifyFunc) {
		t.Error("Alice's transaction verification should succeed")
	}

	// 验证Bob的交易
	bobVerifyFunc := func(data []byte, signature []byte) bool {
		return bob.VerifySignature(data, signature)
	}
	if !tx2.VerifyWithFunc(bobVerifyFunc) {
		t.Error("Bob's transaction verification should succeed")
	}

	// 交叉验证应该失败
	if tx1.VerifyWithFunc(bobVerifyFunc) {
		t.Error("Alice's transaction should not verify with Bob's key")
	}
	if tx2.VerifyWithFunc(aliceVerifyFunc) {
		t.Error("Bob's transaction should not verify with Alice's key")
	}
}

func TestTransactionHashConsistency(t *testing.T) {
	// 创建相同的交易
	from := types.AddressFromPublicKey([]byte("sender_public_key"))
	to := types.AddressFromPublicKey([]byte("receiver_public_key"))

	tx1 := types.NewTransaction(from, to, 100, 10, 1, []byte("test data"))
	tx2 := types.NewTransaction(from, to, 100, 10, 1, []byte("test data"))

	// 签名前的哈希应该相同
	hash1 := tx1.Hash()
	hash2 := tx2.Hash()
	if hash1 != hash2 {
		t.Error("Identical transactions should have same hash before signing")
	}

	// 生成密钥对并签名
	keyPair, err := crypto.GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate key pair: %v", err)
	}

	signFunc := func(data []byte) ([]byte, error) {
		return keyPair.SignData(data)
	}

	err = tx1.SignWithFunc(signFunc)
	if err != nil {
		t.Fatalf("Failed to sign transaction 1: %v", err)
	}

	err = tx2.SignWithFunc(signFunc)
	if err != nil {
		t.Fatalf("Failed to sign transaction 2: %v", err)
	}

	// 签名后，交易哈希仍应该相同（因为签名不包含在哈希计算中）
	hashAfterSign1 := tx1.Hash()
	hashAfterSign2 := tx2.Hash()
	if hashAfterSign1 != hashAfterSign2 {
		t.Error("Transaction hashes should be same after signing (signature not included in hash)")
	}

	// 但哈希应该与签名前相同
	if hash1 != hashAfterSign1 {
		t.Error("Transaction hash should not change after signing")
	}
}

func TestBlockWithSignedTransactions(t *testing.T) {
	// 创建验证者
	validator, err := crypto.GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate validator key pair: %v", err)
	}

	// 创建用户
	user1, err := crypto.GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate user1 key pair: %v", err)
	}

	user2, err := crypto.GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate user2 key pair: %v", err)
	}

	// 创建区块
	prevHash := types.NewHash([]byte("previous_block_hash"))
	block := types.NewBlock(1, prevHash, validator.GetAddress())

	// 创建并签名交易
	tx := types.NewTransaction(
		user1.GetAddress(),
		user2.GetAddress(),
		100, 10, 1,
		[]byte("signed transaction"),
	)

	signFunc := func(data []byte) ([]byte, error) {
		return user1.SignData(data)
	}
	err = tx.SignWithFunc(signFunc)
	if err != nil {
		t.Fatalf("Failed to sign transaction: %v", err)
	}

	// 添加交易到区块
	block.AddTransaction(tx.Hash())

	// 验证区块
	if !block.IsValid() {
		t.Error("Block with signed transaction should be valid")
	}

	// 验证交易根
	if block.Header.TxRoot.IsZero() {
		t.Error("Block should have non-zero transaction root")
	}

	// 验证区块哈希
	blockHash := block.Hash()
	if blockHash.IsZero() {
		t.Error("Block should have non-zero hash")
	}
}
