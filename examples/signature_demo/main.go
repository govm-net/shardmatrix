package main

import (
	"fmt"
	"log"

	"github.com/govm-net/shardmatrix/pkg/crypto"
	"github.com/govm-net/shardmatrix/pkg/types"
)

func main() {
	fmt.Println("=== 区块和交易签名验签示例 ===")

	// 1. 生成密钥对
	fmt.Println("\n1. 生成密钥对...")
	aliceKeyPair, err := crypto.GenerateKeyPair()
	if err != nil {
		log.Fatal("生成密钥对失败:", err)
	}
	fmt.Println("✓ Alice密钥对生成成功")

	bobKeyPair, err := crypto.GenerateKeyPair()
	if err != nil {
		log.Fatal("生成密钥对失败:", err)
	}
	fmt.Println("✓ Bob密钥对生成成功")

	validatorKeyPair, err := crypto.GenerateKeyPair()
	if err != nil {
		log.Fatal("生成密钥对失败:", err)
	}
	fmt.Println("✓ 验证者密钥对生成成功")

	// 2. 创建交易
	fmt.Println("\n2. 创建并签名交易...")
	aliceAddr := aliceKeyPair.GetAddress()
	bobAddr := bobKeyPair.GetAddress()
	validatorAddr := validatorKeyPair.GetAddress()

	// 创建交易
	tx := types.NewTransaction(
		aliceAddr,             // 发送方
		bobAddr,               // 接收方
		100,                   // 金额
		10,                    // 手续费
		1,                     // nonce
		[]byte("转账100代币给Bob"), // 数据
	)

	// 签名交易
	err = tx.Sign(aliceKeyPair.PrivateKey)
	if err != nil {
		log.Fatal("交易签名失败:", err)
	}
	fmt.Printf("✓ 交易签名成功，签名长度: %d 字节\n", len(tx.Signature))

	// 验证交易签名
	isValid := tx.Verify(&aliceKeyPair.PrivateKey.PublicKey)
	if isValid {
		fmt.Println("✓ 交易签名验证通过")
	} else {
		fmt.Println("✗ 交易签名验证失败")
	}

	// 3. 创建区块
	fmt.Println("\n3. 创建并签名区块...")
	prevHash := types.EmptyHash()
	block := types.NewBlock(
		1,             // 高度
		prevHash,      // 前一个区块哈希
		validatorAddr, // 验证者地址
	)

	// 添加交易到区块
	block.AddTransaction(tx.Hash())
	fmt.Printf("✓ 区块创建成功，包含 %d 笔交易\n", len(block.Transactions))

	// 签名区块
	err = block.SignBlock(validatorKeyPair.PrivateKey)
	if err != nil {
		log.Fatal("区块签名失败:", err)
	}
	fmt.Printf("✓ 区块签名成功，签名长度: %d 字节\n", len(block.Header.Signature))

	// 验证区块签名
	isValid = block.VerifyBlockSignature(&validatorKeyPair.PrivateKey.PublicKey)
	if isValid {
		fmt.Println("✓ 区块签名验证通过")
	} else {
		fmt.Println("✗ 区块签名验证失败")
	}

	// 4. 验证区块和交易的有效性
	fmt.Println("\n4. 验证区块和交易的有效性...")
	if tx.IsValid() {
		fmt.Println("✓ 交易格式有效")
	} else {
		fmt.Println("✗ 交易格式无效")
	}

	if block.IsValid() {
		fmt.Println("✓ 区块格式有效")
	} else {
		fmt.Println("✗ 区块格式无效")
	}

	// 5. 显示哈希值
	fmt.Println("\n5. 显示哈希值...")
	fmt.Printf("交易哈希: %x\n", tx.Hash())
	fmt.Printf("区块哈希: %x\n", block.Hash())
	fmt.Printf("交易Merkle根: %x\n", block.Header.TxRoot)

	fmt.Println("\n=== 示例完成 ===")
}
