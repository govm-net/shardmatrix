package main

import (
	"fmt"
	"log"
	"path/filepath"

	"github.com/lengzhao/shardmatrix/pkg/storage"
)

func main() {
	// 打开存储
	storageDir := filepath.Join("./data", "storage")
	db, err := storage.NewLevelDBStorage(storageDir)
	if err != nil {
		log.Fatalf("Failed to open storage: %v", err)
	}
	defer db.Close()

	fmt.Println("🔍 检查存储内容...")

	// 检查创世区块
	genesis, err := db.GetGenesisBlock()
	if err != nil {
		fmt.Printf("❌ 无法获取创世区块: %v\n", err)
	} else {
		fmt.Printf("✅ 创世区块存在, 时间戳: %d\n", genesis.Timestamp)
	}

	// 检查最新区块高度
	height, err := db.GetLatestBlockHeight()
	if err != nil {
		fmt.Printf("❌ 无法获取最新高度: %v\n", err)
	} else {
		fmt.Printf("✅ 最新区块高度: %d\n", height)
	}

	// 尝试读取一些区块
	for i := uint64(0); i <= height && i < 5; i++ {
		block, err := db.GetBlock(i)
		if err != nil {
			fmt.Printf("❌ 区块 %d 读取失败: %v\n", i, err)
		} else {
			fmt.Printf("✅ 区块 %d: 时间戳=%d, 验证者=%s\n",
				block.Header.Number,
				block.Header.Timestamp,
				block.Header.Validator.String()[:8]+"...")
		}
	}

	// 获取存储统计
	stats := db.GetStats()
	fmt.Printf("\n📊 存储统计:\n")
	fmt.Printf("   读取次数: %v\n", stats["reads_total"])
	fmt.Printf("   写入次数: %v\n", stats["writes_total"])
	fmt.Printf("   批次总数: %v\n", stats["batches_total"])
}
