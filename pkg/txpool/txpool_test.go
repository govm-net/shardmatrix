package txpool

import (
	"testing"

	"github.com/govm-net/shardmatrix/pkg/crypto"
	"github.com/govm-net/shardmatrix/pkg/types"
)

func TestMemoryTxPool_AddTransaction(t *testing.T) {
	pool := NewMemoryTxPool(DefaultTxPoolConfig())
	defer pool.Stop()

	// 创建测试交易
	from := types.AddressFromPublicKey([]byte("from_public_key"))
	to := types.AddressFromPublicKey([]byte("to_public_key"))
	tx := types.NewTransaction(from, to, 100, 10, 1, []byte("test data"))

	// 签名交易
	keyPair, err := crypto.GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate key pair: %v", err)
	}

	signFunc := func(data []byte) ([]byte, error) {
		return keyPair.SignData(data)
	}
	err = tx.SignWithFunc(signFunc)
	if err != nil {
		t.Fatalf("Failed to sign transaction: %v", err)
	}

	// 添加交易
	err = pool.AddTransaction(tx)
	if err != nil {
		t.Fatalf("Failed to add transaction: %v", err)
	}

	// 验证交易已添加
	if pool.Size() != 1 {
		t.Errorf("Expected pool size 1, got %d", pool.Size())
	}

	// 验证可以获取交易
	retrievedTx, exists := pool.GetTransaction(tx.Hash())
	if !exists {
		t.Error("Transaction should exist in pool")
	}
	if retrievedTx.Hash() != tx.Hash() {
		t.Error("Retrieved transaction hash mismatch")
	}
}

func TestMemoryTxPool_DuplicateTransaction(t *testing.T) {
	pool := NewMemoryTxPool(DefaultTxPoolConfig())
	defer pool.Stop()

	// 创建测试交易
	from := types.AddressFromPublicKey([]byte("from_public_key"))
	to := types.AddressFromPublicKey([]byte("to_public_key"))
	tx := types.NewTransaction(from, to, 100, 10, 1, []byte("test data"))

	// 签名交易
	keyPair, err := crypto.GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate key pair: %v", err)
	}

	signFunc := func(data []byte) ([]byte, error) {
		return keyPair.SignData(data)
	}
	err = tx.SignWithFunc(signFunc)
	if err != nil {
		t.Fatalf("Failed to sign transaction: %v", err)
	}

	// 第一次添加应该成功
	err = pool.AddTransaction(tx)
	if err != nil {
		t.Fatalf("Failed to add transaction first time: %v", err)
	}

	// 第二次添加应该失败
	err = pool.AddTransaction(tx)
	if err == nil {
		t.Error("Adding duplicate transaction should fail")
	}

	// 池大小应该仍为1
	if pool.Size() != 1 {
		t.Errorf("Expected pool size 1, got %d", pool.Size())
	}
}

func TestMemoryTxPool_RemoveTransaction(t *testing.T) {
	pool := NewMemoryTxPool(DefaultTxPoolConfig())
	defer pool.Stop()

	// 创建并添加交易
	from := types.AddressFromPublicKey([]byte("from_public_key"))
	to := types.AddressFromPublicKey([]byte("to_public_key"))
	tx := types.NewTransaction(from, to, 100, 10, 1, []byte("test data"))

	// 签名交易
	keyPair, err := crypto.GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate key pair: %v", err)
	}

	signFunc := func(data []byte) ([]byte, error) {
		return keyPair.SignData(data)
	}
	err = tx.SignWithFunc(signFunc)
	if err != nil {
		t.Fatalf("Failed to sign transaction: %v", err)
	}

	err = pool.AddTransaction(tx)
	if err != nil {
		t.Fatalf("Failed to add transaction: %v", err)
	}

	// 移除交易
	err = pool.RemoveTransaction(tx.Hash())
	if err != nil {
		t.Fatalf("Failed to remove transaction: %v", err)
	}

	// 验证交易已移除
	if pool.Size() != 0 {
		t.Errorf("Expected pool size 0, got %d", pool.Size())
	}

	// 验证无法获取交易
	_, exists := pool.GetTransaction(tx.Hash())
	if exists {
		t.Error("Transaction should not exist in pool after removal")
	}
}

func TestMemoryTxPool_GetTransactionsByAddress(t *testing.T) {
	pool := NewMemoryTxPool(DefaultTxPoolConfig())
	defer pool.Stop()

	// 创建两个不同地址的交易
	from1 := types.AddressFromPublicKey([]byte("from1_public_key"))
	from2 := types.AddressFromPublicKey([]byte("from2_public_key"))
	to := types.AddressFromPublicKey([]byte("to_public_key"))

	tx1 := types.NewTransaction(from1, to, 100, 10, 1, []byte("test data 1"))
	tx2 := types.NewTransaction(from1, to, 200, 20, 2, []byte("test data 2"))
	tx3 := types.NewTransaction(from2, to, 300, 30, 1, []byte("test data 3"))

	// 签名交易
	keyPair, err := crypto.GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate key pair: %v", err)
	}

	signFunc := func(data []byte) ([]byte, error) {
		return keyPair.SignData(data)
	}

	for _, tx := range []*types.Transaction{tx1, tx2, tx3} {
		err = tx.SignWithFunc(signFunc)
		if err != nil {
			t.Fatalf("Failed to sign transaction: %v", err)
		}
		err = pool.AddTransaction(tx)
		if err != nil {
			t.Fatalf("Failed to add transaction: %v", err)
		}
	}

	// 获取from1的交易
	from1Txs := pool.GetTransactionsByAddress(from1)
	if len(from1Txs) != 2 {
		t.Errorf("Expected 2 transactions for from1, got %d", len(from1Txs))
	}

	// 获取from2的交易
	from2Txs := pool.GetTransactionsByAddress(from2)
	if len(from2Txs) != 1 {
		t.Errorf("Expected 1 transaction for from2, got %d", len(from2Txs))
	}

	// 获取不存在地址的交易
	nonExistentAddr := types.AddressFromPublicKey([]byte("nonexistent_key"))
	nonExistentTxs := pool.GetTransactionsByAddress(nonExistentAddr)
	if len(nonExistentTxs) != 0 {
		t.Errorf("Expected 0 transactions for nonexistent address, got %d", len(nonExistentTxs))
	}
}

func TestMemoryTxPool_PendingQueue(t *testing.T) {
	pool := NewMemoryTxPool(DefaultTxPoolConfig())
	defer pool.Stop()

	// 创建不同手续费的交易
	from := types.AddressFromPublicKey([]byte("from_public_key"))
	to := types.AddressFromPublicKey([]byte("to_public_key"))

	// 创建手续费为 5, 15, 10 的交易
	tx1 := types.NewTransaction(from, to, 100, 5, 1, []byte("low fee"))
	tx2 := types.NewTransaction(from, to, 100, 15, 2, []byte("high fee"))
	tx3 := types.NewTransaction(from, to, 100, 10, 3, []byte("medium fee"))

	// 签名交易
	keyPair, err := crypto.GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate key pair: %v", err)
	}

	signFunc := func(data []byte) ([]byte, error) {
		return keyPair.SignData(data)
	}

	// 按顺序添加交易
	for _, tx := range []*types.Transaction{tx1, tx2, tx3} {
		err = tx.SignWithFunc(signFunc)
		if err != nil {
			t.Fatalf("Failed to sign transaction: %v", err)
		}
		err = pool.AddTransaction(tx)
		if err != nil {
			t.Fatalf("Failed to add transaction: %v", err)
		}
	}

	// 获取待处理交易（应该按手续费降序排列）
	pendingTxs := pool.GetPendingTransactions()
	if len(pendingTxs) != 3 {
		t.Fatalf("Expected 3 pending transactions, got %d", len(pendingTxs))
	}

	// 验证排序：15, 10, 5
	expectedFees := []uint64{15, 10, 5}
	for i, tx := range pendingTxs {
		if tx.Fee != expectedFees[i] {
			t.Errorf("Expected fee %d at position %d, got %d", expectedFees[i], i, tx.Fee)
		}
	}
}

func TestMemoryTxPool_Stats(t *testing.T) {
	pool := NewMemoryTxPool(DefaultTxPoolConfig())
	defer pool.Stop()

	// 空池的统计
	stats := pool.GetStats()
	if stats.PendingCount != 0 {
		t.Errorf("Expected 0 pending transactions, got %d", stats.PendingCount)
	}

	// 添加一些交易
	from := types.AddressFromPublicKey([]byte("from_public_key"))
	to := types.AddressFromPublicKey([]byte("to_public_key"))

	tx1 := types.NewTransaction(from, to, 100, 10, 1, []byte("test data 1"))
	tx2 := types.NewTransaction(from, to, 200, 20, 2, []byte("test data 2"))

	// 签名交易
	keyPair, err := crypto.GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate key pair: %v", err)
	}

	signFunc := func(data []byte) ([]byte, error) {
		return keyPair.SignData(data)
	}

	for _, tx := range []*types.Transaction{tx1, tx2} {
		err = tx.SignWithFunc(signFunc)
		if err != nil {
			t.Fatalf("Failed to sign transaction: %v", err)
		}
		err = pool.AddTransaction(tx)
		if err != nil {
			t.Fatalf("Failed to add transaction: %v", err)
		}
	}

	// 检查统计信息
	stats = pool.GetStats()
	if stats.PendingCount != 2 {
		t.Errorf("Expected 2 pending transactions, got %d", stats.PendingCount)
	}
	if stats.TotalFee != 30 {
		t.Errorf("Expected total fee 30, got %d", stats.TotalFee)
	}
	if stats.AverageFee != 15 {
		t.Errorf("Expected average fee 15, got %d", stats.AverageFee)
	}
	if stats.HighestFee != 20 {
		t.Errorf("Expected highest fee 20, got %d", stats.HighestFee)
	}
	if stats.LowestFee != 10 {
		t.Errorf("Expected lowest fee 10, got %d", stats.LowestFee)
	}
}

func TestMemoryTxPool_MaxSize(t *testing.T) {
	// 创建小容量的交易池
	config := DefaultTxPoolConfig()
	config.MaxSize = 2
	pool := NewMemoryTxPool(config)
	defer pool.Stop()

	// 创建测试交易
	from := types.AddressFromPublicKey([]byte("from_public_key"))
	to := types.AddressFromPublicKey([]byte("to_public_key"))

	// 签名函数
	keyPair, err := crypto.GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate key pair: %v", err)
	}

	signFunc := func(data []byte) ([]byte, error) {
		return keyPair.SignData(data)
	}

	// 添加2个交易（达到容量）
	for i := 0; i < 2; i++ {
		tx := types.NewTransaction(from, to, 100, uint64(10+i), uint64(i+1), []byte("test data"))
		err = tx.SignWithFunc(signFunc)
		if err != nil {
			t.Fatalf("Failed to sign transaction: %v", err)
		}
		err = pool.AddTransaction(tx)
		if err != nil {
			t.Fatalf("Failed to add transaction %d: %v", i, err)
		}
	}

	// 添加第3个交易（高手续费，应该挤出低手续费的交易）
	tx3 := types.NewTransaction(from, to, 100, 20, 3, []byte("high fee tx"))
	err = tx3.SignWithFunc(signFunc)
	if err != nil {
		t.Fatalf("Failed to sign transaction: %v", err)
	}
	err = pool.AddTransaction(tx3)
	if err != nil {
		t.Fatalf("Failed to add high fee transaction: %v", err)
	}

	// 池大小应该仍为2
	if pool.Size() != 2 {
		t.Errorf("Expected pool size 2, got %d", pool.Size())
	}

	// 获取待处理交易，应该包含高手续费的交易
	pendingTxs := pool.GetPendingTransactions()
	found := false
	for _, tx := range pendingTxs {
		if tx.Fee == 20 {
			found = true
			break
		}
	}
	if !found {
		t.Error("High fee transaction should be in the pool")
	}
}

func TestMemoryTxPool_Clear(t *testing.T) {
	pool := NewMemoryTxPool(DefaultTxPoolConfig())
	defer pool.Stop()

	// 添加一些交易
	from := types.AddressFromPublicKey([]byte("from_public_key"))
	to := types.AddressFromPublicKey([]byte("to_public_key"))

	// 签名函数
	keyPair, err := crypto.GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate key pair: %v", err)
	}

	signFunc := func(data []byte) ([]byte, error) {
		return keyPair.SignData(data)
	}

	for i := 0; i < 3; i++ {
		tx := types.NewTransaction(from, to, 100, uint64(10+i), uint64(i+1), []byte("test data"))
		err = tx.SignWithFunc(signFunc)
		if err != nil {
			t.Fatalf("Failed to sign transaction: %v", err)
		}
		err = pool.AddTransaction(tx)
		if err != nil {
			t.Fatalf("Failed to add transaction %d: %v", i, err)
		}
	}

	// 验证池不为空
	if pool.Size() == 0 {
		t.Error("Pool should not be empty before clear")
	}

	// 清空池
	pool.Clear()

	// 验证池已清空
	if pool.Size() != 0 {
		t.Errorf("Expected pool size 0 after clear, got %d", pool.Size())
	}

	// 验证统计信息也被重置
	stats := pool.GetStats()
	if stats.PendingCount != 0 {
		t.Errorf("Expected 0 pending transactions after clear, got %d", stats.PendingCount)
	}
}

func TestMemoryTxPool_ValidationRules(t *testing.T) {
	config := DefaultTxPoolConfig()
	config.MinFee = 5
	config.MaxTxSize = 1000 // 增大最大交易大小
	pool := NewMemoryTxPool(config)
	defer pool.Stop()

	from := types.AddressFromPublicKey([]byte("from_public_key"))
	to := types.AddressFromPublicKey([]byte("to_public_key"))

	// 签名函数
	keyPair, err := crypto.GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate key pair: %v", err)
	}

	signFunc := func(data []byte) ([]byte, error) {
		return keyPair.SignData(data)
	}

	// 测试手续费过低的交易
	lowFeeTx := types.NewTransaction(from, to, 100, 3, 1, []byte("low fee"))
	err = lowFeeTx.SignWithFunc(signFunc)
	if err != nil {
		t.Fatalf("Failed to sign transaction: %v", err)
	}
	err = pool.AddTransaction(lowFeeTx)
	if err == nil {
		t.Error("Adding transaction with low fee should fail")
	}

	// 测试无效交易（零地址）
	invalidTx := types.NewTransaction(types.Address{}, to, 100, 10, 1, []byte("invalid"))
	err = invalidTx.SignWithFunc(signFunc)
	if err != nil {
		t.Fatalf("Failed to sign transaction: %v", err)
	}
	err = pool.AddTransaction(invalidTx)
	if err == nil {
		t.Error("Adding invalid transaction should fail")
	}

	// 测试有效交易
	validTx := types.NewTransaction(from, to, 100, 10, 1, []byte("valid"))
	err = validTx.SignWithFunc(signFunc)
	if err != nil {
		t.Fatalf("Failed to sign transaction: %v", err)
	}
	err = pool.AddTransaction(validTx)
	if err != nil {
		t.Errorf("Adding valid transaction should succeed: %v", err)
	}
}
