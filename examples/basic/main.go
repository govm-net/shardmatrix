package main

import (
	"fmt"

	"github.com/govm-net/shardmatrix/pkg/crypto"
	"github.com/govm-net/shardmatrix/pkg/types"
)

func main() {
	fmt.Println("=== ShardMatrix 基本使用示例 ===")

	// 1. 创建交易
	fmt.Println("\n1. 创建交易...")

	// 生成密钥对
	aliceKeyPair, _ := crypto.GenerateKeyPair()
	bobKeyPair, _ := crypto.GenerateKeyPair()
	validatorKeyPair, _ := crypto.GenerateKeyPair()

	// 创建地址
	aliceAddr := aliceKeyPair.GetAddress()
	bobAddr := bobKeyPair.GetAddress()
	charlieAddr := types.AddressFromPublicKey([]byte("charlie_public_key"))
	validatorAddr := validatorKeyPair.GetAddress()

	tx1 := types.NewTransaction(
		aliceAddr,                    // 发送方地址
		bobAddr,                      // 接收方地址
		50,                           // 转账金额
		10,                           // 手续费
		1,                            // nonce
		[]byte("transfer 50 tokens"), // 交易数据
	)

	tx2 := types.NewTransaction(
		bobAddr,                      // 发送方地址
		charlieAddr,                  // 接收方地址
		25,                           // 转账金额
		5,                            // 手续费
		1,                            // nonce
		[]byte("transfer 25 tokens"), // 交易数据
	)

	// 签名交易
	tx1.Sign(aliceKeyPair.PrivateKey)
	tx2.Sign(bobKeyPair.PrivateKey)

	fmt.Printf("交易1哈希: %x\n", tx1.Hash())
	fmt.Printf("交易2哈希: %x\n", tx2.Hash())

	// 2. 创建区块
	fmt.Println("\n2. 创建区块...")

	// 创建前一个区块哈希和验证者地址
	prevHash := types.NewHash([]byte("genesis_block_hash"))

	block := types.NewBlock(
		1,             // 区块高度
		prevHash,      // 前一个区块哈希
		validatorAddr, // 验证者地址
	)

	// 添加交易到区块
	block.AddTransaction(tx1.Hash())
	block.AddTransaction(tx2.Hash())

	// 签名区块
	block.SignBlock(validatorKeyPair.PrivateKey)

	fmt.Printf("区块哈希: %x\n", block.Hash())
	fmt.Printf("交易Merkle根: %x\n", block.Header.TxRoot)
	fmt.Printf("区块包含 %d 个交易\n", len(block.Transactions))

	// 3. 验证交易
	fmt.Println("\n3. 验证交易...")
	if tx1.IsValid() {
		fmt.Println("交易1验证通过")
	} else {
		fmt.Println("交易1验证失败")
	}

	if tx2.IsValid() {
		fmt.Println("交易2验证通过")
	} else {
		fmt.Println("交易2验证失败")
	}

	// 4. 验证区块
	fmt.Println("\n4. 验证区块...")
	blockHash := block.Hash()
	if len(blockHash.Bytes()) == 32 {
		fmt.Println("区块哈希计算正确")
	} else {
		fmt.Println("区块哈希计算错误")
	}

	// 验证区块签名
	if block.VerifyBlockSignature(&validatorKeyPair.PrivateKey.PublicKey) {
		fmt.Println("区块签名验证通过")
	} else {
		fmt.Println("区块签名验证失败")
	}

	// 5. 显示区块信息
	fmt.Println("\n5. 区块信息:")
	fmt.Printf("  区块高度: %d\n", block.Header.Number)
	fmt.Printf("  时间戳: %d\n", block.Header.Timestamp)
	fmt.Printf("  前一个区块: %x\n", block.Header.PrevHash)
	fmt.Printf("  验证者: %x\n", block.Header.Validator)
	fmt.Printf("  交易根: %x\n", block.Header.TxRoot)

	// 6. 演示区块结构
	fmt.Println("\n6. 区块结构演示:")
	fmt.Println("  区块只包含交易哈希，不包含完整交易数据")
	fmt.Println("  这样可以:")
	fmt.Println("    - 减少区块大小")
	fmt.Println("    - 提高网络传输效率")
	fmt.Println("    - 符合区块链设计模式")
	fmt.Println("  完整交易数据存储在交易池或数据库中")

	fmt.Println("\n=== 示例完成 ===")
}
