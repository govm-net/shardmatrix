package main

import (
	"fmt"
	"os"
	"time"

	"github.com/govm-net/shardmatrix/pkg/config"
	"github.com/govm-net/shardmatrix/pkg/node"
)

func main() {
	// 创建配置
	cfg := &config.Config{}

	// 网络配置
	cfg.Network.Port = 8080
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
	cfg.Storage.DataDir = "./data/leveldb_example"
	cfg.Storage.DBType = "leveldb"

	// API配置
	cfg.API.Port = 8081
	cfg.API.Host = "127.0.0.1"

	// 日志配置
	cfg.Log.Level = "debug"
	cfg.Log.File = "./logs/leveldb_example.log"

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

	fmt.Println("Creating node with LevelDB storage...")

	// 创建节点
	n, err := node.New(cfg)
	if err != nil {
		fmt.Printf("Failed to create node: %v\n", err)
		return
	}

	fmt.Println("Starting node...")

	// 启动节点
	if err := n.Start(); err != nil {
		fmt.Printf("Failed to start node: %v\n", err)
		return
	}

	fmt.Println("Node started successfully with LevelDB storage!")
	fmt.Println("Press Ctrl+C to stop the node...")

	// 运行一段时间以观察存储效果
	time.Sleep(30 * time.Second)

	// 停止节点
	if err := n.Stop(); err != nil {
		fmt.Printf("Failed to stop node: %v\n", err)
		return
	}

	fmt.Println("Node stopped successfully")
}
