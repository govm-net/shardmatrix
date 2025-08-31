package blockchain

import (
	"testing"
	"time"

	"github.com/govm-net/shardmatrix/pkg/storage"
	"github.com/govm-net/shardmatrix/pkg/types"
	"github.com/govm-net/shardmatrix/pkg/validator"
)

// 测试辅助函数
func createTestBlockchain(t *testing.T) *Blockchain {
	config := DefaultBlockchainConfig()
	blockStore := storage.NewMemoryBlockStore()
	validatorConfig := validator.DefaultValidationConfig()

	// 创建简单的验证器（不依赖具体存储）
	v := validator.NewValidator(validatorConfig, blockStore, nil, nil, nil)

	bc, err := NewBlockchain(config, blockStore, v)
	if err != nil {
		t.Fatalf("Failed to create blockchain: %v", err)
	}

	return bc
}

func createTestBlock(height uint64, prevHash types.Hash, validator types.Address) *types.Block {
	block := types.NewBlock(height, prevHash, validator)
	// 为了通过时间戳验证，设置一个更晚的时间
	block.Header.Timestamp = time.Now().Unix() + int64(height*10) // 每个区块间隄10秒
	// 添加一个假的签名用于测试
	block.Header.Signature = []byte("test_signature")
	return block
}

func TestNewBlockchain(t *testing.T) {
	bc := createTestBlockchain(t)

	// 验证初始状态
	state := bc.GetChainState()
	if state.Height != 0 {
		t.Errorf("Expected initial height 0, got %d", state.Height)
	}

	if state.TotalWork != 1 {
		t.Errorf("Expected initial total work 1, got %d", state.TotalWork)
	}

	// 验证创世区块存在
	genesisBlock, err := bc.GetBlockByHeight(0)
	if err != nil {
		t.Errorf("Failed to get genesis block: %v", err)
	}

	if genesisBlock.Header.Number != 0 {
		t.Errorf("Expected genesis block height 0, got %d", genesisBlock.Header.Number)
	}
}

func TestAddBlock(t *testing.T) {
	bc := createTestBlockchain(t)

	// 获取创世区块
	genesisBlock, err := bc.GetBlockByHeight(0)
	if err != nil {
		t.Fatalf("Failed to get genesis block: %v", err)
	}

	// 创建第一个区块
	validator := types.AddressFromPublicKey([]byte("test_validator"))
	block1 := createTestBlock(1, genesisBlock.Hash(), validator)

	// 添加区块
	err = bc.AddBlock(block1)
	if err != nil {
		t.Errorf("Failed to add block: %v", err)
	}

	// 验证链状态更新
	state := bc.GetChainState()
	if state.Height != 1 {
		t.Errorf("Expected height 1, got %d", state.Height)
	}

	if !state.BestBlockHash.Equal(block1.Hash()) {
		t.Errorf("Best block hash not updated correctly")
	}

	// 验证区块可以检索
	retrievedBlock, err := bc.GetBlockByHeight(1)
	if err != nil {
		t.Errorf("Failed to retrieve block: %v", err)
		return
	}

	if retrievedBlock == nil {
		t.Error("Retrieved block is nil")
		return
	}

	if !retrievedBlock.Hash().Equal(block1.Hash()) {
		t.Errorf("Retrieved block hash doesn't match")
	}
}

func TestAddBlockValidation(t *testing.T) {
	bc := createTestBlockchain(t)

	// 测试添加nil区块
	err := bc.AddBlock(nil)
	if err == nil {
		t.Error("Expected error when adding nil block")
	}

	// 测试添加无效高度的区块
	validator := types.AddressFromPublicKey([]byte("test_validator"))
	invalidBlock := createTestBlock(10, types.EmptyHash(), validator) // 跳过高度

	err = bc.AddBlock(invalidBlock)
	if err == nil {
		t.Error("Expected error when adding block with invalid height")
	}

	// 测试添加重复区块
	genesisBlock, _ := bc.GetBlockByHeight(0)
	validBlock := createTestBlock(1, genesisBlock.Hash(), validator)

	// 第一次添加应该成功
	err = bc.AddBlock(validBlock)
	if err != nil {
		t.Errorf("First add should succeed: %v", err)
	}

	// 第二次添加应该失败
	err = bc.AddBlock(validBlock)
	if err == nil {
		t.Error("Expected error when adding duplicate block")
	}
}

func TestForkHandling(t *testing.T) {
	bc := createTestBlockchain(t)

	// 获取创世区块
	genesisBlock, _ := bc.GetBlockByHeight(0)
	validator := types.AddressFromPublicKey([]byte("test_validator"))

	// 添加第一个区块到主链
	block1 := createTestBlock(1, genesisBlock.Hash(), validator)
	err := bc.AddBlock(block1)
	if err != nil {
		t.Fatalf("Failed to add block1: %v", err)
	}

	// 创建分叉：从创世区块开始的另一个区块
	forkBlock1 := createTestBlock(1, genesisBlock.Hash(), validator)
	forkBlock1.Header.Timestamp = block1.Header.Timestamp + 100 // 足够大的时间差异

	err = bc.AddBlock(forkBlock1)
	if err != nil {
		t.Errorf("Failed to add fork block: %v", err)
	}

	// 验证分叉被记录
	forks := bc.GetForks()
	if len(forks) == 0 {
		t.Error("Expected fork to be recorded")
	}

	// 验证主链仍然是原来的
	state := bc.GetChainState()
	if !state.BestBlockHash.Equal(block1.Hash()) {
		t.Error("Main chain should not have changed")
	}
}

func TestChainReorganization(t *testing.T) {
	bc := createTestBlockchain(t)

	// 获取创世区块
	genesisBlock, _ := bc.GetBlockByHeight(0)
	validator := types.AddressFromPublicKey([]byte("test_validator"))

	// 构建主链：genesis -> block1
	block1 := createTestBlock(1, genesisBlock.Hash(), validator)
	err := bc.AddBlock(block1)
	if err != nil {
		t.Fatalf("Failed to add block1: %v", err)
	}
	t.Logf("Added block1, current height: %d", bc.GetChainState().Height)

	// 构建分叉：genesis -> forkBlock1
	forkBlock1 := createTestBlock(1, genesisBlock.Hash(), validator)
	forkBlock1.Header.Timestamp = block1.Header.Timestamp + 100
	err = bc.AddBlock(forkBlock1)
	if err != nil {
		t.Fatalf("Failed to add forkBlock1: %v", err)
	}
	t.Logf("Added forkBlock1, current height: %d, forks: %d", bc.GetChainState().Height, len(bc.GetForks()))

	// 扩展分叉：forkBlock1 -> forkBlock2
	forkBlock2 := createTestBlock(2, forkBlock1.Hash(), validator)
	forkBlock2.Header.Timestamp = forkBlock1.Header.Timestamp + 10 // 确保时间戳递增
	err = bc.AddBlock(forkBlock2)
	if err != nil {
		t.Fatalf("Failed to add forkBlock2: %v", err)
	}

	// 调试信息
	state := bc.GetChainState()
	forks := bc.GetForks()
	t.Logf("After forkBlock2: height=%d, forks=%d", state.Height, len(forks))
	for key, fork := range forks {
		t.Logf("Fork %s: blocks=%d, totalWork=%d", key, len(fork.Blocks), fork.TotalWork)
		for i, block := range fork.Blocks {
			t.Logf("  Block %d: height=%d, hash=%s", i, block.Header.Number, block.Hash().String())
		}
	}

	// 验证链重组发生
	if state.Height != 2 {
		t.Errorf("Expected height 2 after reorganization, got %d", state.Height)
	}

	if !state.BestBlockHash.Equal(forkBlock2.Hash()) {
		t.Error("Chain should have reorganized to longer fork")
	}
}

func TestGetChainStats(t *testing.T) {
	bc := createTestBlockchain(t)

	// 获取初始统计
	stats, err := bc.GetChainStats()
	if err != nil {
		t.Errorf("Failed to get chain stats: %v", err)
	}

	if stats.Height != 0 {
		t.Errorf("Expected height 0, got %d", stats.Height)
	}

	if stats.TotalBlocks != 1 { // 包括创世区块
		t.Errorf("Expected 1 total block, got %d", stats.TotalBlocks)
	}

	// 添加一些区块
	genesisBlock, _ := bc.GetBlockByHeight(0)
	validator := types.AddressFromPublicKey([]byte("test_validator"))

	for i := uint64(1); i <= 3; i++ {
		var prevHash types.Hash
		if i == 1 {
			prevHash = genesisBlock.Hash()
		} else {
			prevBlock, _ := bc.GetBlockByHeight(i - 1)
			prevHash = prevBlock.Hash()
		}

		block := createTestBlock(i, prevHash, validator)
		bc.AddBlock(block)
	}

	// 重新获取统计
	stats, err = bc.GetChainStats()
	if err != nil {
		t.Errorf("Failed to get chain stats: %v", err)
	}

	if stats.Height != 3 {
		t.Errorf("Expected height 3, got %d", stats.Height)
	}

	if stats.TotalBlocks != 4 { // 包括创世区块
		t.Errorf("Expected 4 total blocks, got %d", stats.TotalBlocks)
	}
}

func TestGetChainHealth(t *testing.T) {
	bc := createTestBlockchain(t)

	// 获取健康状况
	health := bc.GetChainHealth()

	if health.Status != "healthy" {
		t.Errorf("Expected healthy status, got %s", health.Status)
	}

	if health.IsSyncing {
		t.Error("Should not be syncing initially")
	}

	if health.ForkCount != 0 {
		t.Errorf("Expected 0 forks, got %d", health.ForkCount)
	}
}

func TestBlockConfirmations(t *testing.T) {
	bc := createTestBlockchain(t)

	// 添加几个区块
	genesisBlock, _ := bc.GetBlockByHeight(0)
	validator := types.AddressFromPublicKey([]byte("test_validator"))

	var blocks []*types.Block
	blocks = append(blocks, genesisBlock)

	for i := uint64(1); i <= 5; i++ {
		prevBlock := blocks[len(blocks)-1]
		block := createTestBlock(i, prevBlock.Hash(), validator)
		bc.AddBlock(block)
		blocks = append(blocks, block)
	}

	// 测试确认数
	confirmations := bc.GetConfirmations(genesisBlock.Hash())
	if confirmations != 5 { // 当前高度5，创世区块在高度0
		t.Errorf("Expected 5 confirmations for genesis block, got %d", confirmations)
	}

	// 测试最新区块的确认数
	latestBlock := blocks[len(blocks)-1]
	confirmations = bc.GetConfirmations(latestBlock.Hash())
	if confirmations != 0 {
		t.Errorf("Expected 0 confirmations for latest block, got %d", confirmations)
	}

	// 测试确认状态（默认需要6个确认）
	if bc.IsConfirmed(genesisBlock.Hash()) {
		t.Error("Genesis block should not be confirmed with only 5 blocks")
	}
}

func TestSyncFunctionality(t *testing.T) {
	bc := createTestBlockchain(t)

	// 测试开始同步
	err := bc.StartSync("peer1", 10)
	if err != nil {
		t.Errorf("Failed to start sync: %v", err)
	}

	// 验证同步状态
	syncStatus := bc.GetSyncStatus()
	if !syncStatus.IsSyncing {
		t.Error("Should be syncing")
	}

	if syncStatus.SyncPeer != "peer1" {
		t.Errorf("Expected sync peer 'peer1', got '%s'", syncStatus.SyncPeer)
	}

	// 测试重复开始同步应该失败
	err = bc.StartSync("peer2", 15)
	if err == nil {
		t.Error("Expected error when starting sync while already syncing")
	}

	// 测试停止同步
	bc.StopSync()

	syncStatus = bc.GetSyncStatus()
	if syncStatus.IsSyncing {
		t.Error("Should not be syncing after stop")
	}
}

func TestGetBlockRange(t *testing.T) {
	bc := createTestBlockchain(t)

	// 添加几个区块
	genesisBlock, _ := bc.GetBlockByHeight(0)
	validator := types.AddressFromPublicKey([]byte("test_validator"))

	for i := uint64(1); i <= 3; i++ {
		var prevHash types.Hash
		if i == 1 {
			prevHash = genesisBlock.Hash()
		} else {
			prevBlock, _ := bc.GetBlockByHeight(i - 1)
			prevHash = prevBlock.Hash()
		}

		block := createTestBlock(i, prevHash, validator)
		bc.AddBlock(block)
	}

	// 测试获取区块范围
	blocks, err := bc.GetBlockRange(1, 3)
	if err != nil {
		t.Errorf("Failed to get block range: %v", err)
	}

	if len(blocks) != 3 {
		t.Errorf("Expected 3 blocks, got %d", len(blocks))
	}

	// 验证区块顺序
	for i, block := range blocks {
		expectedHeight := uint64(i + 1)
		if block.Header.Number != expectedHeight {
			t.Errorf("Block %d has wrong height: expected %d, got %d", i, expectedHeight, block.Header.Number)
		}
	}

	// 测试无效范围
	_, err = bc.GetBlockRange(3, 1) // fromHeight > toHeight
	if err == nil {
		t.Error("Expected error for invalid range")
	}

	_, err = bc.GetBlockRange(0, 10) // toHeight > chain height
	if err == nil {
		t.Error("Expected error for range beyond chain height")
	}
}

func TestValidateChain(t *testing.T) {
	bc := createTestBlockchain(t)

	// 验证初始链（只有创世区块）
	err := bc.ValidateChain()
	if err != nil {
		t.Errorf("Failed to validate initial chain: %v", err)
	}

	// 添加几个有效区块
	genesisBlock, _ := bc.GetBlockByHeight(0)
	validator := types.AddressFromPublicKey([]byte("test_validator"))

	for i := uint64(1); i <= 3; i++ {
		var prevHash types.Hash
		if i == 1 {
			prevHash = genesisBlock.Hash()
		} else {
			prevBlock, _ := bc.GetBlockByHeight(i - 1)
			prevHash = prevBlock.Hash()
		}

		block := createTestBlock(i, prevHash, validator)
		bc.AddBlock(block)
	}

	// 验证完整链
	err = bc.ValidateChain()
	if err != nil {
		t.Errorf("Failed to validate chain with valid blocks: %v", err)
	}
}

func TestPerformanceMetrics(t *testing.T) {
	bc := createTestBlockchain(t)

	// 获取性能指标（即使没有数据）
	metrics, err := bc.GetPerformanceMetrics(60) // 最近60分钟
	if err != nil {
		t.Errorf("Failed to get performance metrics: %v", err)
	}

	// 检查网络高度（应该是0，只有创世区块）
	chainState := bc.GetChainState()
	if chainState.Height != 0 {
		t.Errorf("Expected chain height 0, got %d", chainState.Height)
	}

	if metrics.NetworkHeight != chainState.Height {
		t.Errorf("Expected network height %d, got %d", chainState.Height, metrics.NetworkHeight)
	}

	// 验证基本的统计结构
	if metrics.SampleSize < 0 {
		t.Errorf("Sample size should not be negative: %d", metrics.SampleSize)
	}
}

// 基准测试
func BenchmarkAddBlock(b *testing.B) {
	bc := createTestBlockchain(&testing.T{})
	validator := types.AddressFromPublicKey([]byte("test_validator"))

	// 准备区块
	blocks := make([]*types.Block, b.N)
	genesisBlock, _ := bc.GetBlockByHeight(0)

	blocks[0] = createTestBlock(1, genesisBlock.Hash(), validator)
	for i := 1; i < b.N; i++ {
		blocks[i] = createTestBlock(uint64(i+1), blocks[i-1].Hash(), validator)
	}

	b.ResetTimer()

	// 运行基准测试
	for i := 0; i < b.N; i++ {
		bc.AddBlock(blocks[i])
	}
}

func BenchmarkGetBlock(b *testing.B) {
	bc := createTestBlockchain(&testing.T{})
	validator := types.AddressFromPublicKey([]byte("test_validator"))

	// 添加一些区块
	genesisBlock, _ := bc.GetBlockByHeight(0)
	for i := uint64(1); i <= 100; i++ {
		var prevHash types.Hash
		if i == 1 {
			prevHash = genesisBlock.Hash()
		} else {
			prevBlock, _ := bc.GetBlockByHeight(i - 1)
			prevHash = prevBlock.Hash()
		}

		block := createTestBlock(i, prevHash, validator)
		bc.AddBlock(block)
	}

	b.ResetTimer()

	// 运行基准测试
	for i := 0; i < b.N; i++ {
		height := uint64(i%100 + 1) // 循环访问区块
		bc.GetBlockByHeight(height)
	}
}
