package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/lengzhao/shardmatrix/pkg/api"
	"github.com/lengzhao/shardmatrix/pkg/config"
	"github.com/lengzhao/shardmatrix/pkg/crypto"
	"github.com/lengzhao/shardmatrix/pkg/node"
	"github.com/lengzhao/shardmatrix/pkg/types"
	"github.com/spf13/cobra"
)

// 全局配置
var (
	globalConfig *config.Config
	configFile   string
	dataDir      string
	logLevel     string
)

var rootCmd = &cobra.Command{
	Use:   "shardnode",
	Short: "ShardMatrix blockchain node",
	Long:  "A high-performance blockchain node implementation with DPoS consensus and 2-second block intervals",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// 加载配置
		var err error
		globalConfig, err = config.LoadConfig(configFile)
		if err != nil {
			log.Fatalf("Failed to load configuration: %v", err)
		}
		
		// 应用命令行参数覆盖
		applyCommandLineOverrides()
	},
}

// 各种子命令
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new blockchain node",
	Long:  "Initialize a new blockchain node with key generation, data directory setup, and genesis configuration",
	Run:   runInit,
}

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the blockchain node",
	Long:  "Start the blockchain node with all components including consensus, networking, and API server",
	Run:   runStart,
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show node status",
	Long:  "Display current status and health information of the blockchain node",
	Run:   runStatus,
}

var keysCmd = &cobra.Command{
	Use:   "keys",
	Short: "Manage validator keys",
	Long:  "Generate, import, export, and manage validator keypairs",
}

var keysGenerateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate a new validator keypair",
	Run:   runKeysGenerate,
}

var keysShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show validator address and public key",
	Run:   runKeysShow,
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage node configuration",
	Long:  "Show, validate, and manage node configuration files",
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show current configuration",
	Run:   runConfigShow,
}

var configValidateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate configuration file",
	Run:   runConfigValidate,
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version information",
	Long:  "Display version, build information, and supported features",
	Run:   runVersion,
}

func init() {
	// 添加子命令
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(keysCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(versionCmd)
	
	// keys 子命令
	keysCmd.AddCommand(keysGenerateCmd)
	keysCmd.AddCommand(keysShowCmd)
	
	// config 子命令
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configValidateCmd)
	
	// 全局标志
	rootCmd.PersistentFlags().StringVar(&configFile, "config", "", "Configuration file path")
	rootCmd.PersistentFlags().StringVarP(&dataDir, "datadir", "d", "", "Data directory for blockchain storage")
	rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", "", "Log level (debug, info, warn, error, fatal)")
	
	// init 命令标志
	initCmd.Flags().Bool("force", false, "Force initialization, overwrite existing data")
	initCmd.Flags().Int64("genesis-time", 0, "Genesis block timestamp (default: current time aligned to even seconds)")
	
	// start 命令标志
	startCmd.Flags().Bool("validator", false, "Run as validator node (overrides config)")
	startCmd.Flags().Bool("disable-api", false, "Disable API server")
	startCmd.Flags().StringSlice("bootstrap", []string{}, "Bootstrap nodes to connect to")
	
	// status 命令标志
	statusCmd.Flags().String("endpoint", "http://localhost:8545", "Node API endpoint")
	statusCmd.Flags().Bool("json", false, "Output in JSON format")
	
	// version 命令标志
	versionCmd.Flags().Bool("verbose", false, "Show detailed version information")
}

func runInit(cmd *cobra.Command, args []string) {
	force, _ := cmd.Flags().GetBool("force")
	genesisTime, _ := cmd.Flags().GetInt64("genesis-time")
	
	dataDir := globalConfig.Node.DataDir
	
	fmt.Printf("Initializing ShardMatrix node in %s...\n", dataDir)
	
	// 检查目录是否已存在
	if !force {
		if _, err := os.Stat(dataDir); !os.IsNotExist(err) {
			if _, err := os.Stat(filepath.Join(dataDir, "validator.key")); !os.IsNotExist(err) {
				log.Fatalf("Node already initialized in %s. Use --force to reinitialize.", dataDir)
			}
		}
	}
	
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
	
	// 设置创世时间
	if genesisTime == 0 {
		genesisTime = time.Now().Unix()
	}
	if genesisTime%2 != 0 {
		genesisTime-- // 确保是偶数秒
	}
	globalConfig.Consensus.GenesisTime = genesisTime
	
	// 保存配置文件
	configPath := config.GetConfigPath(dataDir)
	err = config.SaveConfig(globalConfig, configPath)
	if err != nil {
		log.Fatalf("Failed to save configuration: %v", err)
	}
	
	fmt.Printf("✓ Genesis time: %s (%d)\n", time.Unix(genesisTime, 0).Format(time.RFC3339), genesisTime)
	fmt.Printf("✓ Block interval: %v (strict)\n", types.BlockTime)
	fmt.Printf("✓ Validator count: %d\n", types.ValidatorCount)
	fmt.Printf("✓ Max block size: %d bytes\n", types.MaxBlockSize)
	fmt.Printf("✓ Shard ID: %d\n", types.ShardID)
	fmt.Printf("✓ Configuration saved: %s\n", configPath)
	
	fmt.Println("\nInitialization completed successfully!")
	fmt.Printf("To start the node, run: shardnode start --datadir %s\n", dataDir)
}

func runStart(cmd *cobra.Command, args []string) {
	isValidatorFlag, _ := cmd.Flags().GetBool("validator")
	disableAPI, _ := cmd.Flags().GetBool("disable-api")
	bootstrapNodes, _ := cmd.Flags().GetStringSlice("bootstrap")
	
	dataDir := globalConfig.Node.DataDir
	
	fmt.Printf("Starting ShardMatrix node...\n")
	fmt.Printf("Data directory: %s\n", dataDir)
	
	// 检查是否已初始化
	if _, err := os.Stat(dataDir); os.IsNotExist(err) {
		log.Fatalf("Data directory does not exist. Run 'shardnode init' first.")
	}
	
	if _, err := os.Stat(filepath.Join(dataDir, "validator.key")); os.IsNotExist(err) {
		log.Fatalf("Validator key not found. Run 'shardnode init' first.")
	}
	
	// 应用命令行覆盖
	if cmd.Flags().Changed("validator") {
		globalConfig.Consensus.IsValidator = isValidatorFlag
	}
	if disableAPI {
		globalConfig.API.Enabled = false
	}
	if len(bootstrapNodes) > 0 {
		globalConfig.Network.BootstrapNodes = bootstrapNodes
	}
	
	fmt.Printf("Validator mode: %v\n", globalConfig.Consensus.IsValidator)
	fmt.Printf("API enabled: %v\n", globalConfig.API.Enabled)
	
	// 确保创世时间对齐
	config.EnsureGenesisTimeAlignment(globalConfig)
	
	// 转换为旧格式配置（兼容性）
	nodeConfig := &node.Config{
		DataDir:     globalConfig.Node.DataDir,
		ListenAddr:  globalConfig.Network.ListenAddr,
		APIAddr:     globalConfig.API.ListenAddr,
		NodeID:      globalConfig.Node.NodeID,
		IsValidator: globalConfig.Consensus.IsValidator,
		GenesisTime: globalConfig.Consensus.GenesisTime,
	}
	
	// 创建节点
	n, err := node.NewNode(nodeConfig)
	if err != nil {
		log.Fatalf("Failed to create node: %v", err)
	}
	
	// 创建API服务器（如果启用）
	if globalConfig.API.Enabled {
		apiServer := api.NewServer(nil, nil, nil) // 简化实现
		n.SetAPIServer(apiServer)
	}
	
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
	if globalConfig.API.Enabled {
		fmt.Println("  • REST API endpoints")
		fmt.Printf("  • API server: http://localhost%s\n", globalConfig.API.ListenAddr)
	}
	fmt.Printf("  • P2P listen: %s\n", globalConfig.Network.ListenAddr)
	if len(globalConfig.Network.BootstrapNodes) > 0 {
		fmt.Printf("  • Bootstrap nodes: %s\n", strings.Join(globalConfig.Network.BootstrapNodes, ", "))
	}
	
	<-sigCh
	fmt.Println("\nShutting down...")
	
	// 停止节点
	err = n.Stop()
	if err != nil {
		log.Printf("Error stopping node: %v", err)
	}
	
	fmt.Println("Node stopped successfully.")
}

func runStatus(cmd *cobra.Command, args []string) {
	endpoint, _ := cmd.Flags().GetString("endpoint")
	jsonOutput, _ := cmd.Flags().GetBool("json")
	
	fmt.Printf("Checking node status at %s...\n", endpoint)
	
	// 尝试连接到节点API
	resp, err := http.Get(endpoint + "/status")
	if err != nil {
		fmt.Printf("❌ Node is not running or unreachable: %v\n", err)
		return
	}
	defer resp.Body.Close()
	
	if resp.StatusCode == http.StatusOK {
		fmt.Println("✅ Node is running")
		
		if jsonOutput {
			fmt.Println("{\"status\":\"running\"}")
		} else {
			fmt.Println("Status: Healthy")
			fmt.Println("API: Available")
		}
	} else {
		fmt.Printf("⚠️ Node returned status code: %d\n", resp.StatusCode)
	}
}

func runKeysGenerate(cmd *cobra.Command, args []string) {
	keyPair, err := crypto.GenerateKeyPair()
	if err != nil {
		log.Fatalf("Failed to generate keypair: %v", err)
	}
	
	fmt.Println("Generated new validator keypair:")
	fmt.Printf("Address: %s\n", keyPair.Address.String())
	fmt.Printf("Private Key: %s\n", keyPair.PrivateKeyHex())
	fmt.Println("\n⚠️ WARNING: Keep your private key secure and never share it!")
}

func runKeysShow(cmd *cobra.Command, args []string) {
	dataDir := globalConfig.Node.DataDir
	keyFile := filepath.Join(dataDir, "validator.key")
	
	if _, err := os.Stat(keyFile); os.IsNotExist(err) {
		log.Fatalf("Validator key not found. Run 'shardnode init' first.")
	}
	
	keyData, err := os.ReadFile(keyFile)
	if err != nil {
		log.Fatalf("Failed to read validator key: %v", err)
	}
	
	keyPair, err := crypto.NewKeyPairFromHex(string(keyData))
	if err != nil {
		log.Fatalf("Failed to parse validator key: %v", err)
	}
	
	fmt.Println("Validator Information:")
	fmt.Printf("Address: %s\n", keyPair.Address.String())
	fmt.Printf("Key file: %s\n", keyFile)
}

func runConfigShow(cmd *cobra.Command, args []string) {
	fmt.Println("Current Configuration:")
	fmt.Printf("Node ID: %s\n", globalConfig.Node.NodeID)
	fmt.Printf("Data Directory: %s\n", globalConfig.Node.DataDir)
	fmt.Printf("Log Level: %s\n", globalConfig.Node.LogLevel)
	fmt.Printf("Listen Address: %s\n", globalConfig.Network.ListenAddr)
	fmt.Printf("API Address: %s\n", globalConfig.API.ListenAddr)
	fmt.Printf("API Enabled: %v\n", globalConfig.API.Enabled)
	fmt.Printf("Validator Mode: %v\n", globalConfig.Consensus.IsValidator)
	fmt.Printf("Genesis Time: %d (%s)\n", globalConfig.Consensus.GenesisTime, 
		time.Unix(globalConfig.Consensus.GenesisTime, 0).Format(time.RFC3339))
	
	if len(globalConfig.Network.BootstrapNodes) > 0 {
		fmt.Printf("Bootstrap Nodes: %s\n", strings.Join(globalConfig.Network.BootstrapNodes, ", "))
	}
}

func runConfigValidate(cmd *cobra.Command, args []string) {
	fmt.Println("Validating configuration...")
	
	// 配置已在加载时验证，如果能到这里说明配置有效
	fmt.Println("✅ Configuration is valid")
	fmt.Printf("Config file: %s\n", configFile)
	fmt.Printf("Data directory: %s\n", globalConfig.Node.DataDir)
}

func runVersion(cmd *cobra.Command, args []string) {
	verboseFlag, _ := cmd.Flags().GetBool("verbose")
	
	fmt.Println("ShardMatrix Node v1.0.0")
	fmt.Println("High-performance blockchain with DPoS consensus")
	
	if verboseFlag {
		fmt.Println("\nFeatures:")
		fmt.Println("  • Strict 2-second block intervals")
		fmt.Println("  • DPoS consensus mechanism")
		fmt.Println("  • Multi-layer validation")
		fmt.Println("  • P2P networking")
		fmt.Println("  • REST API")
		fmt.Println("  • Health monitoring")
	}
}

func applyCommandLineOverrides() {
	// 应用命令行参数覆盖
	if dataDir != "" {
		globalConfig.Node.DataDir = dataDir
	}
	if logLevel != "" {
		globalConfig.Node.LogLevel = logLevel
	}
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}