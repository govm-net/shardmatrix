package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/lengzhao/shardmatrix/pkg/api"
	"github.com/lengzhao/shardmatrix/pkg/crypto"
	"github.com/lengzhao/shardmatrix/pkg/node"
	"github.com/lengzhao/shardmatrix/pkg/types"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "shardnode",
	Short: "ShardMatrix blockchain node",
	Long:  "A single-node blockchain implementation with DPoS consensus and 2-second block intervals",
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new blockchain node",
	Run:   runInit,
}

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the blockchain node",
	Run:   runStart,
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("ShardMatrix Node v1.0.0")
		fmt.Println("Single-node blockchain with strict 2-second block intervals")
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(versionCmd)
	
	// 添加标志
	initCmd.Flags().StringP("datadir", "d", "./data", "Data directory for blockchain storage")
	startCmd.Flags().StringP("datadir", "d", "./data", "Data directory for blockchain storage")
	startCmd.Flags().BoolP("validator", "v", true, "Run as validator node")
}

func runInit(cmd *cobra.Command, args []string) {
	dataDir, _ := cmd.Flags().GetString("datadir")
	
	fmt.Printf("Initializing ShardMatrix node in %s...\n", dataDir)
	
	// 创建数据目录
	err := os.MkdirAll(dataDir, 0755)
	if err != nil {
		log.Fatalf("Failed to create data directory: %v", err)
	}
	
	// 生成验证者密钥
	keyPair, err := crypto.GenerateKeyPair()
	if err != nil {
		log.Fatalf("Failed to generate validator keypair: %v", err)
	}
	
	// 保存密钥到文件
	keyFile := filepath.Join(dataDir, "validator.key")
	err = os.WriteFile(keyFile, []byte(keyPair.PrivateKeyHex()), 0600)
	if err != nil {
		log.Fatalf("Failed to save validator key: %v", err)
	}
	
	fmt.Printf("✓ Generated validator keypair\n")
	fmt.Printf("  Address: %s\n", keyPair.Address.String())
	fmt.Printf("  Key file: %s\n", keyFile)
	
	// 创建创世配置
	genesisTime := time.Now().Unix()
	if genesisTime%2 != 0 {
		genesisTime-- // 确保是偶数秒
	}
	
	fmt.Printf("✓ Genesis time: %s (%d)\n", time.Unix(genesisTime, 0).Format(time.RFC3339), genesisTime)
	fmt.Printf("✓ Block interval: %v (strict)\n", types.BlockTime)
	fmt.Printf("✓ Validator count: %d\n", types.ValidatorCount)
	fmt.Printf("✓ Max block size: %d bytes\n", types.MaxBlockSize)
	fmt.Printf("✓ Shard ID: %d\n", types.ShardID)
	
	fmt.Println("\nInitialization completed successfully!")
	fmt.Printf("To start the node, run: shardnode start --datadir %s\n", dataDir)
}

func runStart(cmd *cobra.Command, args []string) {
	dataDir, _ := cmd.Flags().GetString("datadir")
	isValidator, _ := cmd.Flags().GetBool("validator")
	
	fmt.Printf("Starting ShardMatrix node...\n")
	fmt.Printf("Data directory: %s\n", dataDir)
	fmt.Printf("Validator mode: %v\n", isValidator)
	
	// 检查数据目录是否存在
	if _, err := os.Stat(dataDir); os.IsNotExist(err) {
		log.Fatalf("Data directory does not exist. Run 'shardnode init' first.")
	}
	
	// 创建节点配置
	genesisTime := time.Now().Unix()
	if genesisTime%2 != 0 {
		genesisTime--
	}
	
	config := &node.Config{
		DataDir:     dataDir,
		ListenAddr:  ":8546",
		APIAddr:     ":8545",
		NodeID:      "shardmatrix-node-1",
		IsValidator: isValidator,
		GenesisTime: genesisTime,
	}
	
	// 创建节点
	n, err := node.NewNode(config)
	if err != nil {
		log.Fatalf("Failed to create node: %v", err)
	}
	
	// 创建API服务器
	apiServer := api.NewServer(nil, nil, nil) // 简化实现
	n.SetAPIServer(apiServer)
	
	// 启动节点
	err = n.Start()
	if err != nil {
		log.Fatalf("Failed to start node: %v", err)
	}
	
	// 等待中断信号
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	
	fmt.Println("\n✅ Node is running. Press Ctrl+C to stop.")
	fmt.Println("\nFeatures active:")
	fmt.Println("  • Strict 2-second block intervals")
	fmt.Println("  • Empty block filling mechanism")
	fmt.Println("  • DPoS consensus simulation")
	fmt.Println("  • P2P networking")
	fmt.Println("  • REST API endpoints")
	fmt.Printf("  • API server: http://localhost%s\n", config.APIAddr)
	fmt.Printf("  • P2P listen: %s\n", config.ListenAddr)
	
	<-sigCh
	fmt.Println("\nShutting down...")
	
	// 停止节点
	err = n.Stop()
	if err != nil {
		log.Printf("Error stopping node: %v", err)
	}
	
	fmt.Println("Node stopped successfully.")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}