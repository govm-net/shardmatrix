package validator

import (
	"testing"

	"github.com/govm-net/shardmatrix/pkg/storage"
	"github.com/govm-net/shardmatrix/pkg/types"
)

func TestGenesisBlockWithoutSignature(t *testing.T) {
	// 创建创世区块
	validator := types.AddressFromPublicKey([]byte("test_validator"))
	genesisBlock := types.NewGenesisBlock(validator)

	// 验证创世区块没有签名
	if len(genesisBlock.Header.Signature) != 0 {
		t.Errorf("Genesis block should not have signature, got %d bytes", len(genesisBlock.Header.Signature))
	}

	// 创建验证器配置，要求签名
	config := DefaultValidationConfig()
	config.RequireSignature = true

	// 创建存储实例
	blockStore := storage.NewMemoryBlockStore()
	transactionStore := storage.NewMemoryTransactionStore()
	validatorStore := storage.NewMemoryValidatorStore()
	accountStore := storage.NewMemoryAccountStore()

	// 创建验证器
	validatorInstance := NewValidator(config, blockStore, transactionStore, validatorStore, accountStore)

	// 验证创世区块应该通过，即使RequireSignature为true
	err := validatorInstance.ValidateGenesisBlock(genesisBlock)
	if err != nil {
		t.Errorf("Genesis block validation should pass without signature: %v", err)
	}

	// 创建一个普通区块（高度>0）进行对比测试
	normalBlock := types.NewBlock(1, genesisBlock.Hash(), validator)
	normalBlock.Header.Signature = []byte{} // 空签名

	// 普通区块应该因为缺少签名而验证失败
	err = validatorInstance.ValidateNewBlock(normalBlock, transactionStore)
	if err == nil {
		t.Error("Normal block should fail validation without signature")
	}
}
