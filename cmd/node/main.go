package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/govm-net/shardmatrix/pkg/config"
	"github.com/govm-net/shardmatrix/pkg/consensus"
	"github.com/govm-net/shardmatrix/pkg/node"
	"github.com/govm-net/shardmatrix/pkg/types"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	configFile string
	logLevel   string
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "shardmatrix-node",
		Short: "ShardMatrix blockchain node",
		Long:  `A high-performance blockchain node with sharding capabilities`,
		RunE:  runNode,
	}

	// 添加节点管理命令
	rootCmd.AddCommand(nodeManagementCmd())

	rootCmd.Flags().StringVarP(&configFile, "config", "c", "config.yaml", "Configuration file path")
	rootCmd.Flags().StringVarP(&logLevel, "log-level", "l", "info", "Log level (debug, info, warn, error)")

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// nodeManagementCmd 节点管理命令
func nodeManagementCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "node",
		Short: "Node management commands",
		Long:  `Manage ShardMatrix blockchain node`,
	}

	// 添加子命令
	cmd.AddCommand(nodeStatusCmd())
	cmd.AddCommand(nodeConfigCmd())
	cmd.AddCommand(nodePeersCmd())
	cmd.AddCommand(nodeRestartCmd())

	return cmd
}

// nodeStatusCmd 节点状态命令
func nodeStatusCmd() *cobra.Command {
	var (
		configFile string
		format     string
	)

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Get node status",
		Long:  `Get the current status of the ShardMatrix node`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// 加载配置
			cfg, err := config.Load(configFile)
			if err != nil {
				return fmt.Errorf("failed to load config: %v", err)
			}

			// 创建节点（仅用于获取状态信息，不启动）
			n, err := node.New(cfg)
			if err != nil {
				return fmt.Errorf("failed to create node: %v", err)
			}

			// 获取节点信息
			nodeInfo := n.GetNodeInfo()

			// 根据格式输出
			if format == "json" {
				// JSON格式输出
				data, err := json.MarshalIndent(nodeInfo, "", "  ")
				if err != nil {
					return fmt.Errorf("failed to marshal node info: %v", err)
				}
				fmt.Println(string(data))
			} else {
				// 默认格式输出
				fmt.Println("=== ShardMatrix Node Status ===")
				fmt.Printf("Node ID: %s\n", nodeInfo["node_id"])
				fmt.Printf("Version: %s\n", nodeInfo["version"])
				fmt.Printf("Running: %v\n", nodeInfo["is_running"])
				fmt.Printf("Uptime: %s\n", nodeInfo["uptime"])
				fmt.Printf("Connected Peers: %d\n", nodeInfo["peer_count"])
				fmt.Printf("Blockchain Height: %d\n", nodeInfo["chain_height"])
				fmt.Printf("Best Block: %s\n", nodeInfo["best_block"])
				fmt.Printf("Chain Health: %s\n", nodeInfo["chain_health"])
				fmt.Printf("Is Syncing: %v\n", nodeInfo["is_syncing"])

				// 网络状态
				if networkStatus, ok := nodeInfo["network_status"].(map[string]interface{}); ok {
					fmt.Println("\n--- Network Status ---")
					fmt.Printf("Connected Peers: %d\n", networkStatus["connected_peers"])
					fmt.Printf("Partitioned: %v\n", networkStatus["is_partitioned"])
					if partitionSince, ok := networkStatus["partition_since"].(time.Time); ok && !partitionSince.IsZero() {
						fmt.Printf("Partition Since: %s\n", partitionSince.Format("2006-01-02 15:04:05"))
					}
					fmt.Printf("Reconnect Count: %d\n", networkStatus["reconnect_count"])
				}
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&configFile, "config", "c", "config.yaml", "Configuration file path")
	cmd.Flags().StringVarP(&format, "format", "f", "text", "Output format (text, json)")

	return cmd
}

// nodeConfigCmd 节点配置命令
func nodeConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Node configuration commands",
		Long:  `Manage node configuration`,
	}

	// 添加子命令
	cmd.AddCommand(configShowCmd())
	cmd.AddCommand(configValidateCmd())

	return cmd
}

// configShowCmd 显示配置命令
func configShowCmd() *cobra.Command {
	var (
		configFile string
		format     string
	)

	cmd := &cobra.Command{
		Use:   "show",
		Short: "Show node configuration",
		Long:  `Show the current node configuration`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// 加载配置
			cfg, err := config.Load(configFile)
			if err != nil {
				return fmt.Errorf("failed to load config: %v", err)
			}

			// 根据格式输出
			if format == "json" {
				// JSON格式输出
				data, err := json.MarshalIndent(cfg, "", "  ")
				if err != nil {
					return fmt.Errorf("failed to marshal config: %v", err)
				}
				fmt.Println(string(data))
			} else {
				// YAML格式输出
				// 注意：这里简化处理，实际应该使用yaml库
				fmt.Printf("Network:\n")
				fmt.Printf("  Port: %d\n", cfg.Network.Port)
				fmt.Printf("  Host: %s\n", cfg.Network.Host)
				fmt.Printf("  Bootstrap Peers: %v\n", cfg.Network.BootstrapPeers)

				fmt.Printf("\nBlockchain:\n")
				fmt.Printf("  Chain ID: %d\n", cfg.Blockchain.ChainID)
				fmt.Printf("  Block Interval: %d\n", cfg.Blockchain.BlockInterval)
				fmt.Printf("  Max Block Size: %d\n", cfg.Blockchain.MaxBlockSize)

				fmt.Printf("\nConsensus:\n")
				fmt.Printf("  Type: %s\n", cfg.Consensus.Type)
				fmt.Printf("  Validator Count: %d\n", cfg.Consensus.ValidatorCount)
				fmt.Printf("  Validators: %v\n", cfg.Consensus.Validators)
				fmt.Printf("  My Validator: %s\n", cfg.Consensus.MyValidator)

				fmt.Printf("\nStorage:\n")
				fmt.Printf("  Data Dir: %s\n", cfg.Storage.DataDir)
				fmt.Printf("  DB Type: %s\n", cfg.Storage.DBType)

				fmt.Printf("\nAPI:\n")
				fmt.Printf("  Port: %d\n", cfg.API.Port)
				fmt.Printf("  Host: %s\n", cfg.API.Host)

				fmt.Printf("\nLog:\n")
				fmt.Printf("  Level: %s\n", cfg.Log.Level)
				fmt.Printf("  File: %s\n", cfg.Log.File)
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&configFile, "config", "c", "config.yaml", "Configuration file path")
	cmd.Flags().StringVarP(&format, "format", "f", "yaml", "Output format (yaml, json)")

	return cmd
}

// configValidateCmd 验证配置命令
func configValidateCmd() *cobra.Command {
	var configFile string

	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate node configuration",
		Long:  `Validate the node configuration file`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// 加载配置
			cfg, err := config.Load(configFile)
			if err != nil {
				return fmt.Errorf("configuration validation failed: %v", err)
			}

			fmt.Println("✅ Configuration is valid!")
			fmt.Printf("Configuration file: %s\n", configFile)
			fmt.Printf("Chain ID: %d\n", cfg.Blockchain.ChainID)
			fmt.Printf("Network port: %d\n", cfg.Network.Port)
			fmt.Printf("Storage type: %s\n", cfg.Storage.DBType)
			fmt.Printf("Data directory: %s\n", cfg.Storage.DataDir)

			return nil
		},
	}

	cmd.Flags().StringVarP(&configFile, "config", "c", "config.yaml", "Configuration file path")

	return cmd
}

// nodePeersCmd 节点连接命令
func nodePeersCmd() *cobra.Command {
	var (
		configFile string
		format     string
	)

	cmd := &cobra.Command{
		Use:   "peers",
		Short: "Show connected peers",
		Long:  `Show the list of currently connected peers`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// 加载配置
			cfg, err := config.Load(configFile)
			if err != nil {
				return fmt.Errorf("failed to load config: %v", err)
			}

			// 创建节点（仅用于获取网络信息）
			n, err := node.New(cfg)
			if err != nil {
				return fmt.Errorf("failed to create node: %v", err)
			}

			// 获取连接的节点
			peers := n.GetNetworkManager().GetPeers()

			if format == "json" {
				// JSON格式输出
				data, err := json.MarshalIndent(peers, "", "  ")
				if err != nil {
					return fmt.Errorf("failed to marshal peers: %v", err)
				}
				fmt.Println(string(data))
			} else {
				// 默认格式输出
				fmt.Printf("Connected Peers (%d):\n", len(peers))
				if len(peers) == 0 {
					fmt.Println("  No connected peers")
				} else {
					for i, peer := range peers {
						fmt.Printf("  %d. %s\n", i+1, peer)
					}
				}
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&configFile, "config", "c", "config.yaml", "Configuration file path")
	cmd.Flags().StringVarP(&format, "format", "f", "text", "Output format (text, json)")

	return cmd
}

// nodeRestartCmd 重启节点命令
func nodeRestartCmd() *cobra.Command {
	var (
		configFile string
		force      bool
	)

	cmd := &cobra.Command{
		Use:   "restart",
		Short: "Restart the node",
		Long:  `Restart the ShardMatrix node`,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("🔄 Restarting ShardMatrix node...")

			if !force {
				fmt.Print("Are you sure you want to restart the node? [y/N]: ")
				var response string
				fmt.Scanln(&response)
				if response != "y" && response != "Y" {
					fmt.Println("Restart cancelled.")
					return nil
				}
			}

			// 这里应该实现实际的重启逻辑
			// 对于命令行工具，我们只能显示信息
			fmt.Println("⚠️  Note: This command only provides information about node restart.")
			fmt.Println("⚠️  To actually restart the node, you need to stop and start the node process manually.")

			// 加载配置
			cfg, err := config.Load(configFile)
			if err != nil {
				return fmt.Errorf("failed to load config: %v", err)
			}

			fmt.Printf("Configuration file: %s\n", configFile)
			fmt.Printf("Node will listen on: %s:%d\n", cfg.Network.Host, cfg.Network.Port)
			fmt.Printf("Data directory: %s\n", cfg.Storage.DataDir)

			fmt.Println("✅ Node restart information displayed successfully!")

			return nil
		},
	}

	cmd.Flags().StringVarP(&configFile, "config", "c", "config.yaml", "Configuration file path")
	cmd.Flags().BoolVarP(&force, "force", "f", false, "Force restart without confirmation")

	return cmd
}

func runNode(cmd *cobra.Command, args []string) error {
	// 设置日志级别
	level, err := logrus.ParseLevel(logLevel)
	if err != nil {
		return fmt.Errorf("invalid log level: %v", err)
	}
	logrus.SetLevel(level)

	// 加载配置
	cfg, err := config.Load(configFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %v", err)
	}

	// 创建节点
	n, err := node.New(cfg)
	if err != nil {
		return fmt.Errorf("failed to create node: %v", err)
	}

	// 如果配置了DPoS共识，设置共识机制
	if cfg.Consensus.Type == "dpos" || cfg.Consensus.Type == "pos" {
		logrus.Info("Setting up DPoS consensus...")

		// 创建DPoS配置
		dposConfig := consensus.DefaultDPoSConfig()
		dposConfig.MinStake = 1000 // 降低最小质押用于测试
		dposConfig.ValidatorCount = cfg.Consensus.ValidatorCount
		dposConfig.BlockInterval = time.Duration(cfg.Blockchain.BlockInterval) * time.Second

		// 创建DPoS共识
		dpos := consensus.NewDPoSConsensus(dposConfig)
		n.GetBlockchain().SetConsensus(dpos)

		// 启动共识
		if err := n.GetBlockchain().GetConsensus().Start(); err != nil {
			return fmt.Errorf("failed to start consensus: %v", err)
		}

		// 创建示例验证者用于单节点测试
		if len(cfg.Consensus.Validators) == 0 {
			logrus.Info("Creating demo validator for single node testing...")
			validatorAddr := types.GenerateAddress()
			if err := dpos.RegisterValidator(validatorAddr, 10000, 0.1); err != nil {
				logrus.Warnf("Failed to register demo validator: %v", err)
			} else {
				logrus.Infof("Demo validator registered: %s", validatorAddr.String())
			}
		} else {
			// 在多节点模式下，只注册当前节点负责的验证者
			if cfg.Consensus.MyValidator != "" {
				logrus.Info("Registering my validator for multi-node testing...")
				addr, err := types.AddressFromString(cfg.Consensus.MyValidator)
				if err != nil {
					logrus.Warnf("Invalid my_validator address: %s", cfg.Consensus.MyValidator)
				} else {
					stake := uint64(10000 + cfg.GetMyValidatorIndex()*1000) // 每个验证者不同的质押
					commission := 0.1 + float64(cfg.GetMyValidatorIndex())*0.05

					if err := dpos.RegisterValidator(addr, stake, commission); err != nil {
						logrus.Warnf("Failed to register my validator %s: %v", cfg.Consensus.MyValidator, err)
					} else {
						validatorIndex := cfg.GetMyValidatorIndex()
						logrus.Infof("My validator registered: %s (index: %d, stake: %d, commission: %.2f)", cfg.Consensus.MyValidator, validatorIndex, stake, commission)
					}
				}
			} else {
				// 回退到旧的行为：注册所有验证者（不推荐）
				logrus.Warn("No my_validator specified, registering all validators (not recommended for multi-node)")
				for i, validatorStr := range cfg.Consensus.Validators {
					addr, err := types.AddressFromString(validatorStr)
					if err != nil {
						logrus.Warnf("Invalid validator address: %s", validatorStr)
						continue
					}

					stake := uint64(5000 + i*1000)
					commission := 0.1 + float64(i)*0.05

					if err := dpos.RegisterValidator(addr, stake, commission); err != nil {
						logrus.Warnf("Failed to register validator %s: %v", validatorStr, err)
					} else {
						logrus.Infof("Validator registered: %s (stake: %d)", validatorStr, stake)
					}
				}
			}
		}
	}

	// 启动节点
	if err := n.Start(); err != nil {
		return fmt.Errorf("failed to start node: %v", err)
	}
	defer n.Stop()

	logrus.Info("ShardMatrix node started successfully")

	// 等待中断信号
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	logrus.Info("Shutting down node...")
	return nil
}
