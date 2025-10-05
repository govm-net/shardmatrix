package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Config 节点配置结构
type Config struct {
	// 节点基础配置
	Node struct {
		NodeID   string `yaml:"node_id" json:"node_id"`
		DataDir  string `yaml:"data_dir" json:"data_dir"`
		LogLevel string `yaml:"log_level" json:"log_level"`
	} `yaml:"node" json:"node"`

	// 网络配置
	Network struct {
		ListenAddr     string   `yaml:"listen_addr" json:"listen_addr"`
		MaxPeers       int      `yaml:"max_peers" json:"max_peers"`
		BootstrapNodes []string `yaml:"bootstrap_nodes" json:"bootstrap_nodes"`
	} `yaml:"network" json:"network"`

	// API配置
	API struct {
		Enabled     bool     `yaml:"enabled" json:"enabled"`
		ListenAddr  string   `yaml:"listen_addr" json:"listen_addr"`
		CorsOrigins []string `yaml:"cors_origins" json:"cors_origins"`
		RateLimit   int      `yaml:"rate_limit" json:"rate_limit"`
	} `yaml:"api" json:"api"`

	// 共识配置
	Consensus struct {
		Type        string `yaml:"type" json:"type"`
		IsValidator bool   `yaml:"is_validator" json:"is_validator"`
		BlockTime   string `yaml:"block_time" json:"block_time"`
		GenesisTime int64  `yaml:"genesis_time" json:"genesis_time"`
	} `yaml:"consensus" json:"consensus"`

	// 存储配置
	Storage struct {
		Type      string `yaml:"type" json:"type"`
		CacheSize string `yaml:"cache_size" json:"cache_size"`
	} `yaml:"storage" json:"storage"`

	// 监控配置
	Monitoring struct {
		Enabled               bool   `yaml:"enabled" json:"enabled"`
		MetricsAddr           string `yaml:"metrics_addr" json:"metrics_addr"`
		HealthCheckInterval   string `yaml:"health_check_interval" json:"health_check_interval"`
	} `yaml:"monitoring" json:"monitoring"`
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	config := &Config{}
	
	// 节点基础配置
	config.Node.NodeID = "shardmatrix-node-1"
	config.Node.DataDir = "./data"
	config.Node.LogLevel = "info"
	
	// 网络配置
	config.Network.ListenAddr = ":8546"
	config.Network.MaxPeers = 50
	config.Network.BootstrapNodes = []string{}
	
	// API配置
	config.API.Enabled = true
	config.API.ListenAddr = ":8545"
	config.API.CorsOrigins = []string{"*"}
	config.API.RateLimit = 100
	
	// 共识配置
	config.Consensus.Type = "dpos"
	config.Consensus.IsValidator = true
	config.Consensus.BlockTime = "2s"
	config.Consensus.GenesisTime = time.Now().Unix()
	
	// 存储配置
	config.Storage.Type = "leveldb"
	config.Storage.CacheSize = "128mb"
	
	// 监控配置
	config.Monitoring.Enabled = true
	config.Monitoring.MetricsAddr = ":9090"
	config.Monitoring.HealthCheckInterval = "30s"
	
	return config
}

// LoadConfig 从多个来源加载配置，按优先级合并
func LoadConfig(configFile string) (*Config, error) {
	// 1. 从默认值开始
	config := DefaultConfig()
	
	// 2. 加载配置文件（如果存在）
	if configFile != "" {
		if err := loadFromFile(config, configFile); err != nil {
			return nil, fmt.Errorf("failed to load config file: %w", err)
		}
	}
	
	// 3. 覆盖环境变量
	loadFromEnv(config)
	
	// 4. 验证配置
	if err := validateConfig(config); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}
	
	return config, nil
}

// loadFromFile 从YAML文件加载配置
func loadFromFile(config *Config, filename string) error {
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return nil // 文件不存在，使用默认配置
	}
	
	data, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}
	
	if err := yaml.Unmarshal(data, config); err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}
	
	return nil
}

// loadFromEnv 从环境变量加载配置
func loadFromEnv(config *Config) {
	// 节点配置
	if nodeID := os.Getenv("SHARD_NODE_ID"); nodeID != "" {
		config.Node.NodeID = nodeID
	}
	if dataDir := os.Getenv("SHARD_DATA_DIR"); dataDir != "" {
		config.Node.DataDir = dataDir
	}
	if logLevel := os.Getenv("SHARD_LOG_LEVEL"); logLevel != "" {
		config.Node.LogLevel = logLevel
	}
	
	// 网络配置
	if listenAddr := os.Getenv("SHARD_LISTEN_ADDR"); listenAddr != "" {
		config.Network.ListenAddr = listenAddr
	}
	if maxPeers := os.Getenv("SHARD_MAX_PEERS"); maxPeers != "" {
		if val, err := strconv.Atoi(maxPeers); err == nil {
			config.Network.MaxPeers = val
		}
	}
	if bootstrapNodes := os.Getenv("SHARD_BOOTSTRAP_NODES"); bootstrapNodes != "" {
		config.Network.BootstrapNodes = strings.Split(bootstrapNodes, ",")
	}
	
	// API配置
	if apiAddr := os.Getenv("SHARD_API_ADDR"); apiAddr != "" {
		config.API.ListenAddr = apiAddr
	}
	if apiEnabled := os.Getenv("SHARD_API_ENABLED"); apiEnabled != "" {
		config.API.Enabled = apiEnabled == "true"
	}
	
	// 共识配置
	if isValidator := os.Getenv("SHARD_IS_VALIDATOR"); isValidator != "" {
		config.Consensus.IsValidator = isValidator == "true"
	}
	if genesisTime := os.Getenv("SHARD_GENESIS_TIME"); genesisTime != "" {
		if val, err := strconv.ParseInt(genesisTime, 10, 64); err == nil {
			config.Consensus.GenesisTime = val
		}
	}
	
	// 存储配置
	if storageType := os.Getenv("SHARD_STORAGE_TYPE"); storageType != "" {
		config.Storage.Type = storageType
	}
	
	// 监控配置
	if monitoringEnabled := os.Getenv("SHARD_MONITORING_ENABLED"); monitoringEnabled != "" {
		config.Monitoring.Enabled = monitoringEnabled == "true"
	}
	if metricsAddr := os.Getenv("SHARD_METRICS_ADDR"); metricsAddr != "" {
		config.Monitoring.MetricsAddr = metricsAddr
	}
}

// validateConfig 验证配置的合理性
func validateConfig(config *Config) error {
	// 验证数据目录
	if config.Node.DataDir == "" {
		return fmt.Errorf("data directory cannot be empty")
	}
	
	// 验证监听地址
	if config.Network.ListenAddr == "" {
		return fmt.Errorf("network listen address cannot be empty")
	}
	
	// 验证API地址
	if config.API.Enabled && config.API.ListenAddr == "" {
		return fmt.Errorf("API listen address cannot be empty when API is enabled")
	}
	
	// 验证区块时间
	if _, err := time.ParseDuration(config.Consensus.BlockTime); err != nil {
		return fmt.Errorf("invalid block time format: %w", err)
	}
	
	// 验证健康检查间隔
	if _, err := time.ParseDuration(config.Monitoring.HealthCheckInterval); err != nil {
		return fmt.Errorf("invalid health check interval format: %w", err)
	}
	
	// 验证日志级别
	validLogLevels := []string{"debug", "info", "warn", "error", "fatal"}
	validLogLevel := false
	for _, level := range validLogLevels {
		if config.Node.LogLevel == level {
			validLogLevel = true
			break
		}
	}
	if !validLogLevel {
		return fmt.Errorf("invalid log level: %s", config.Node.LogLevel)
	}
	
	return nil
}

// SaveConfig 保存配置到文件
func SaveConfig(config *Config, filename string) error {
	// 确保目录存在
	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}
	
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	
	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}
	
	return nil
}

// GetConfigPath 获取配置文件路径
func GetConfigPath(dataDir string) string {
	return filepath.Join(dataDir, "config.yaml")
}

// EnsureGenesisTimeAlignment 确保创世时间对齐到偶数秒
func EnsureGenesisTimeAlignment(config *Config) {
	if config.Consensus.GenesisTime%2 != 0 {
		config.Consensus.GenesisTime--
	}
}

// GetBlockTimeDuration 获取区块时间作为Duration
func (c *Config) GetBlockTimeDuration() (time.Duration, error) {
	return time.ParseDuration(c.Consensus.BlockTime)
}

// GetHealthCheckInterval 获取健康检查间隔作为Duration
func (c *Config) GetHealthCheckInterval() (time.Duration, error) {
	return time.ParseDuration(c.Monitoring.HealthCheckInterval)
}

// ToLegacyNodeConfig 转换为旧版本的节点配置（兼容性）
func (c *Config) ToLegacyNodeConfig() map[string]interface{} {
	return map[string]interface{}{
		"DataDir":     c.Node.DataDir,
		"ListenAddr":  c.Network.ListenAddr,
		"APIAddr":     c.API.ListenAddr,
		"NodeID":      c.Node.NodeID,
		"IsValidator": c.Consensus.IsValidator,
		"GenesisTime": c.Consensus.GenesisTime,
	}
}