package main

import (
	"fmt"
	"os"
	"time"

	"github.com/govm-net/shardmatrix/pkg/config"
	"github.com/govm-net/shardmatrix/pkg/node"
)

func main() {
	dataDir := "./data/persistence_test"

	// 创建配置
	cfg := &config.Config{}

	// 网络配置
	cfg.Network.Port = 8090
	cfg.Network.Host = "127.0.0.1"
	cfg.Network.BootstrapPeers = []string{}

	// 区块链配置
	cfg.Blockchain.ChainID = 1
	cfg.Blockchain.BlockInterval = 3
	cfg.Blockchain.MaxBlockSize = 2 * 1024 * 1024 // 2MB

	// 共识配置
	cfg.Consensus.Type = "dpos"
	cfg.Consensus.ValidatorCount = 3
	cfg.Consensus.Validators = []string{
		"0xeaecc6750ca9a0030dc7191093f93103b44e23ce",
		"0x291321edddaa4e75f7f5108e9f68c3dd5934859c",
		"0x4572bd2980ce0fd1c6fa28bf5e98c1ff9b04b679",
	}
	cfg.Consensus.MyValidator = "0xeaecc6750ca9a0030dc7191093f93103b44e23ce"

	// 存储配置 - 使用LevelDB
	cfg.Storage.DataDir = dataDir
	cfg.Storage.DBType = "leveldb"

	// API配置
	cfg.API.Port = 8091
	cfg.API.Host = "127.0.0.1"

	// 日志配置
	cfg.Log.Level = "debug"
	cfg.Log.File = "./logs/persistence_test.log"

	// 清理旧数据
	os.RemoveAll(dataDir)

	// 创建数据目录
	if err := os.MkdirAll(cfg.Storage.DataDir, 0755); err != nil {
		fmt.Printf("Failed to create data directory: %v\n", err)
		return
	}

	// 创建日志目录
	logDir := "./logs"
	if err := os.MkdirAll(logDir, 0755); err != nil {
		fmt.Printf("Failed to create log directory: %v\n", err)
		return
	}

	fmt.Println("=== LevelDB Persistence Test ===")

	// 第一次启动节点
	fmt.Println("1. Starting node for the first time...")
	n1, err := node.New(cfg)
	if err != nil {
		fmt.Printf("Failed to create node: %v\n", err)
		return
	}

	if err := n1.Start(); err != nil {
		fmt.Printf("Failed to start node: %v\n", err)
		return
	}

	// 等待几秒钟让节点初始化
	time.Sleep(5 * time.Second)

	// 停止第一个节点
	if err := n1.Stop(); err != nil {
		fmt.Printf("Failed to stop node: %v\n", err)
		return
	}

	fmt.Println("First node stopped. Data should be persisted to LevelDB.")

	// 等待几秒钟
	time.Sleep(2 * time.Second)

	// 第二次启动节点（验证数据持久化）
	fmt.Println("2. Starting node again to test data persistence...")
	n2, err := node.New(cfg)
	if err != nil {
		fmt.Printf("Failed to create second node: %v\n", err)
		return
	}

	if err := n2.Start(); err != nil {
		fmt.Printf("Failed to start second node: %v\n", err)
		return
	}

	// 等待几秒钟让节点初始化
	time.Sleep(5 * time.Second)

	// 停止第二个节点
	if err := n2.Stop(); err != nil {
		fmt.Printf("Failed to stop second node: %v\n", err)
		return
	}

	fmt.Println("Test completed successfully!")

	// 显示存储目录内容
	fmt.Println("\n=== Storage Directory Contents ===")
	showDirContents(dataDir, "")
}

func showDirContents(dirPath string, indent string) {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		fmt.Printf("%sError reading directory: %v\n", indent, err)
		return
	}

	for _, entry := range entries {
		if entry.IsDir() {
			fmt.Printf("%s[dir] %s/\n", indent, entry.Name())
			showDirContents(dirPath+"/"+entry.Name(), indent+"  ")
		} else {
			info, _ := entry.Info()
			fmt.Printf("%s[file] %s (%.1fKB)\n", indent, entry.Name(), float64(info.Size())/1024)
		}
	}
}
