package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/govm-net/shardmatrix/pkg/config"
	"github.com/govm-net/shardmatrix/pkg/node"
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

	rootCmd.Flags().StringVarP(&configFile, "config", "c", "config.yaml", "Configuration file path")
	rootCmd.Flags().StringVarP(&logLevel, "log-level", "l", "info", "Log level (debug, info, warn, error)")

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
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
