package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/cometbft/cometbft/config"
	"github.com/cometbft/cometbft/libs/log"
	"github.com/cometbft/cometbft/node"
	"github.com/cometbft/cometbft/p2p"
	"github.com/cometbft/cometbft/privval"
	"github.com/cometbft/cometbft/proxy"
	"github.com/govm-net/shardmatrix/pkg/state"
)

func main() {
	// 解析命令行参数
	homeDir := flag.String("home", ".shardmatrix", "home directory")
	flag.Parse()

	// 创建节点
	node, err := createNode(*homeDir)
	if err != nil {
		fmt.Printf("Failed to create node: %v\n", err)
		os.Exit(1)
	}

	// 启动节点
	if err := node.Start(); err != nil {
		fmt.Printf("Failed to start node: %v\n", err)
		os.Exit(1)
	}

	// 等待中断信号
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	// 停止节点
	if err := node.Stop(); err != nil {
		fmt.Printf("Failed to stop node: %v\n", err)
		os.Exit(1)
	}
}

func createNode(homeDir string) (*node.Node, error) {
	// 加载私钥
	pv := privval.LoadFilePV(
		homeDir+"/config/priv_validator_key.json",
		homeDir+"/data/priv_validator_state.json",
	)

	// 创建节点配置
	nodeKey, err := p2p.LoadOrGenNodeKey(homeDir + "/config/node_key.json")
	if err != nil {
		return nil, err
	}

	// 创建数据库
	db, err := config.DefaultDBProvider(&config.DBContext{
		ID:     "application",
		Config: config.DefaultConfig(),
	})
	if err != nil {
		return nil, err
	}

	// 创建ABCI应用
	app := state.NewApplication(db)

	// 创建日志
	logger := log.NewTMLogger(log.NewSyncWriter(os.Stdout))

	// 创建节点
	return node.NewNode(
		context.Background(),
		config.DefaultConfig(),
		pv,
		nodeKey,
		proxy.NewLocalClientCreator(app),
		node.DefaultGenesisDocProviderFunc(config.DefaultConfig()),
		config.DefaultDBProvider,
		node.DefaultMetricsProvider(config.DefaultConfig().Instrumentation),
		logger,
	)
}
