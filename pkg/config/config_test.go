package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()
	
	// 验证默认值
	if config.Node.NodeID != "shardmatrix-node-1" {
		t.Errorf("Expected default node ID 'shardmatrix-node-1', got %s", config.Node.NodeID)
	}
	
	if config.Node.DataDir != "./data" {
		t.Errorf("Expected default data dir './data', got %s", config.Node.DataDir)
	}
	
	if config.Network.ListenAddr != ":8546" {
		t.Errorf("Expected default listen addr ':8546', got %s", config.Network.ListenAddr)
	}
	
	if !config.API.Enabled {
		t.Error("Expected API to be enabled by default")
	}
	
	if !config.Consensus.IsValidator {
		t.Error("Expected validator mode to be enabled by default")
	}
}

func TestLoadConfigFromFile(t *testing.T) {
	// 创建临时配置文件
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.yaml")
	
	configContent := `
node:
  node_id: "test-node"
  data_dir: "/test/data"
  log_level: "debug"

network:
  listen_addr: ":9546"
  max_peers: 100

api:
  enabled: false
  listen_addr: ":9545"

consensus:
  is_validator: false
  genesis_time: 1234567890
`
	
	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write test config file: %v", err)
	}
	
	// 加载配置
	config, err := LoadConfig(configFile)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}
	
	// 验证加载的值
	if config.Node.NodeID != "test-node" {
		t.Errorf("Expected node ID 'test-node', got %s", config.Node.NodeID)
	}
	
	if config.Node.DataDir != "/test/data" {
		t.Errorf("Expected data dir '/test/data', got %s", config.Node.DataDir)
	}
	
	if config.Network.ListenAddr != ":9546" {
		t.Errorf("Expected listen addr ':9546', got %s", config.Network.ListenAddr)
	}
	
	if config.API.Enabled {
		t.Error("Expected API to be disabled")
	}
	
	if config.Consensus.IsValidator {
		t.Error("Expected validator mode to be disabled")
	}
}

func TestLoadConfigFromEnv(t *testing.T) {
	// 设置环境变量
	originalEnv := map[string]string{
		"SHARD_NODE_ID":     os.Getenv("SHARD_NODE_ID"),
		"SHARD_DATA_DIR":    os.Getenv("SHARD_DATA_DIR"),
		"SHARD_LISTEN_ADDR": os.Getenv("SHARD_LISTEN_ADDR"),
		"SHARD_IS_VALIDATOR": os.Getenv("SHARD_IS_VALIDATOR"),
	}
	
	// 设置测试环境变量
	os.Setenv("SHARD_NODE_ID", "env-test-node")
	os.Setenv("SHARD_DATA_DIR", "/env/test/data")
	os.Setenv("SHARD_LISTEN_ADDR", ":7546")
	os.Setenv("SHARD_IS_VALIDATOR", "false")
	
	// 恢复原始环境变量
	defer func() {
		for key, value := range originalEnv {
			if value == "" {
				os.Unsetenv(key)
			} else {
				os.Setenv(key, value)
			}
		}
	}()
	
	// 加载配置
	config, err := LoadConfig("")
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}
	
	// 验证环境变量覆盖了默认值
	if config.Node.NodeID != "env-test-node" {
		t.Errorf("Expected node ID 'env-test-node', got %s", config.Node.NodeID)
	}
	
	if config.Node.DataDir != "/env/test/data" {
		t.Errorf("Expected data dir '/env/test/data', got %s", config.Node.DataDir)
	}
	
	if config.Network.ListenAddr != ":7546" {
		t.Errorf("Expected listen addr ':7546', got %s", config.Network.ListenAddr)
	}
	
	if config.Consensus.IsValidator {
		t.Error("Expected validator mode to be disabled by env var")
	}
}

func TestConfigValidation(t *testing.T) {
	// 测试无效配置
	config := DefaultConfig()
	config.Node.DataDir = "" // 空数据目录
	
	err := validateConfig(config)
	if err == nil {
		t.Error("Expected validation error for empty data directory")
	}
	
	// 测试无效日志级别
	config = DefaultConfig()
	config.Node.LogLevel = "invalid"
	
	err = validateConfig(config)
	if err == nil {
		t.Error("Expected validation error for invalid log level")
	}
	
	// 测试无效区块时间
	config = DefaultConfig()
	config.Consensus.BlockTime = "invalid"
	
	err = validateConfig(config)
	if err == nil {
		t.Error("Expected validation error for invalid block time")
	}
}

func TestSaveAndLoadConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "test-config.yaml")
	
	// 创建测试配置
	originalConfig := DefaultConfig()
	originalConfig.Node.NodeID = "save-test-node"
	originalConfig.Node.DataDir = "/save/test/data"
	originalConfig.Network.MaxPeers = 99
	
	// 保存配置
	if err := SaveConfig(originalConfig, configFile); err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}
	
	// 重新加载配置
	loadedConfig, err := LoadConfig(configFile)
	if err != nil {
		t.Fatalf("Failed to load saved config: %v", err)
	}
	
	// 验证配置一致性
	if loadedConfig.Node.NodeID != originalConfig.Node.NodeID {
		t.Errorf("Node ID mismatch: expected %s, got %s", 
			originalConfig.Node.NodeID, loadedConfig.Node.NodeID)
	}
	
	if loadedConfig.Node.DataDir != originalConfig.Node.DataDir {
		t.Errorf("Data dir mismatch: expected %s, got %s", 
			originalConfig.Node.DataDir, loadedConfig.Node.DataDir)
	}
	
	if loadedConfig.Network.MaxPeers != originalConfig.Network.MaxPeers {
		t.Errorf("Max peers mismatch: expected %d, got %d", 
			originalConfig.Network.MaxPeers, loadedConfig.Network.MaxPeers)
	}
}

func TestEnsureGenesisTimeAlignment(t *testing.T) {
	config := DefaultConfig()
	
	// 测试奇数时间戳
	config.Consensus.GenesisTime = 1234567891 // 奇数
	EnsureGenesisTimeAlignment(config)
	
	if config.Consensus.GenesisTime%2 != 0 {
		t.Error("Genesis time should be aligned to even seconds")
	}
	
	// 测试偶数时间戳（不应该改变）
	originalTime := int64(1234567890) // 偶数
	config.Consensus.GenesisTime = originalTime
	EnsureGenesisTimeAlignment(config)
	
	if config.Consensus.GenesisTime != originalTime {
		t.Error("Even genesis time should not be modified")
	}
}

func TestGetBlockTimeDuration(t *testing.T) {
	config := DefaultConfig()
	
	duration, err := config.GetBlockTimeDuration()
	if err != nil {
		t.Fatalf("Failed to parse block time duration: %v", err)
	}
	
	expected := 2 * time.Second
	if duration != expected {
		t.Errorf("Expected block time %v, got %v", expected, duration)
	}
}

func TestGetHealthCheckInterval(t *testing.T) {
	config := DefaultConfig()
	
	interval, err := config.GetHealthCheckInterval()
	if err != nil {
		t.Fatalf("Failed to parse health check interval: %v", err)
	}
	
	expected := 30 * time.Second
	if interval != expected {
		t.Errorf("Expected health check interval %v, got %v", expected, interval)
	}
}

func TestToLegacyNodeConfig(t *testing.T) {
	config := DefaultConfig()
	config.Node.NodeID = "legacy-test"
	config.Node.DataDir = "/legacy/data"
	
	legacy := config.ToLegacyNodeConfig()
	
	if legacy["NodeID"] != config.Node.NodeID {
		t.Errorf("Legacy NodeID mismatch: expected %s, got %v", 
			config.Node.NodeID, legacy["NodeID"])
	}
	
	if legacy["DataDir"] != config.Node.DataDir {
		t.Errorf("Legacy DataDir mismatch: expected %s, got %v", 
			config.Node.DataDir, legacy["DataDir"])
	}
	
	if legacy["IsValidator"] != config.Consensus.IsValidator {
		t.Errorf("Legacy IsValidator mismatch: expected %v, got %v", 
			config.Consensus.IsValidator, legacy["IsValidator"])
	}
}