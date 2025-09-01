package config

import (
	"fmt"
	"os"

	"github.com/spf13/viper"
)

// Config 配置结构
type Config struct {
	// 网络配置
	Network struct {
		Port           int      `mapstructure:"port"`
		Host           string   `mapstructure:"host"`
		BootstrapPeers []string `mapstructure:"bootstrap_peers"`
	} `mapstructure:"network"`

	// 区块链配置
	Blockchain struct {
		ChainID       uint64 `mapstructure:"chain_id"`
		BlockInterval int    `mapstructure:"block_interval"` // 区块生成间隔（秒）
		MaxBlockSize  int    `mapstructure:"max_block_size"` // 最大区块大小（字节）
	} `mapstructure:"blockchain"`

	// 共识配置
	Consensus struct {
		Type           string   `mapstructure:"type"`
		ValidatorCount int      `mapstructure:"validator_count"`
		Validators     []string `mapstructure:"validators"`
		MyValidator    string   `mapstructure:"my_validator"` // 当前节点负责的验证者
	} `mapstructure:"consensus"`

	// 存储配置
	Storage struct {
		DataDir string `mapstructure:"data_dir"`
		DBType  string `mapstructure:"db_type"`
	} `mapstructure:"storage"`

	// API配置
	API struct {
		Port int    `mapstructure:"port"`
		Host string `mapstructure:"host"`
	} `mapstructure:"api"`

	// 日志配置
	Log struct {
		Level string `mapstructure:"level"`
		File  string `mapstructure:"file"`
	} `mapstructure:"log"`
}

// Load 加载配置文件
func Load(configFile string) (*Config, error) {
	viper.SetConfigFile(configFile)
	viper.SetConfigType("yaml")

	// 设置默认值
	setDefaults()

	// 读取配置文件
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// 配置文件不存在，使用默认配置
			fmt.Printf("Config file not found: %s, using default configuration\n", configFile)
		} else {
			return nil, fmt.Errorf("failed to read config file: %v", err)
		}
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %v", err)
	}

	// 验证配置
	if err := validateConfig(&config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %v", err)
	}

	return &config, nil
}

// setDefaults 设置默认配置
func setDefaults() {
	// 网络默认配置
	viper.SetDefault("network.port", 8080)
	viper.SetDefault("network.host", "0.0.0.0")
	viper.SetDefault("network.bootstrap_peers", []string{})

	// 区块链默认配置
	viper.SetDefault("blockchain.chain_id", 1)
	viper.SetDefault("blockchain.block_interval", 2)            // 2秒
	viper.SetDefault("blockchain.max_block_size", 10*1024*1024) // 10MB

	// 共识默认配置
	viper.SetDefault("consensus.type", "pos")
	viper.SetDefault("consensus.validator_count", 21)
	viper.SetDefault("consensus.validators", []string{})
	viper.SetDefault("consensus.my_validator", "")

	// 存储默认配置
	viper.SetDefault("storage.data_dir", "./data")
	viper.SetDefault("storage.db_type", "leveldb")

	// API默认配置
	viper.SetDefault("api.port", 8081)
	viper.SetDefault("api.host", "127.0.0.1")

	// 日志默认配置
	viper.SetDefault("log.level", "info")
	viper.SetDefault("log.file", "")
}

// validateConfig 验证配置
func validateConfig(config *Config) error {
	// 验证网络配置
	if config.Network.Port <= 0 || config.Network.Port > 65535 {
		return fmt.Errorf("invalid network port: %d", config.Network.Port)
	}

	// 验证区块链配置
	if config.Blockchain.BlockInterval <= 0 {
		return fmt.Errorf("invalid block interval: %d", config.Blockchain.BlockInterval)
	}

	if config.Blockchain.MaxBlockSize <= 0 {
		return fmt.Errorf("invalid max block size: %d", config.Blockchain.MaxBlockSize)
	}

	// 验证共识配置
	if config.Consensus.ValidatorCount <= 0 {
		return fmt.Errorf("invalid validator count: %d", config.Consensus.ValidatorCount)
	}

	// 验证存储配置
	if config.Storage.DataDir == "" {
		return fmt.Errorf("data directory cannot be empty")
	}

	// 创建数据目录
	if err := os.MkdirAll(config.Storage.DataDir, 0755); err != nil {
		return fmt.Errorf("failed to create data directory: %v", err)
	}

	return nil
}

// Save 保存配置到文件
func (c *Config) Save(configFile string) error {
	viper.Set("network.port", c.Network.Port)
	viper.Set("network.host", c.Network.Host)
	viper.Set("network.bootstrap_peers", c.Network.BootstrapPeers)

	viper.Set("blockchain.chain_id", c.Blockchain.ChainID)
	viper.Set("blockchain.block_interval", c.Blockchain.BlockInterval)
	viper.Set("blockchain.max_block_size", c.Blockchain.MaxBlockSize)

	viper.Set("consensus.type", c.Consensus.Type)
	viper.Set("consensus.validator_count", c.Consensus.ValidatorCount)
	viper.Set("consensus.validators", c.Consensus.Validators)
	viper.Set("consensus.my_validator", c.Consensus.MyValidator)

	viper.Set("storage.data_dir", c.Storage.DataDir)
	viper.Set("storage.db_type", c.Storage.DBType)

	viper.Set("api.port", c.API.Port)
	viper.Set("api.host", c.API.Host)

	viper.Set("log.level", c.Log.Level)
	viper.Set("log.file", c.Log.File)

	return viper.WriteConfigAs(configFile)
}

// GetMyValidatorIndex 获取当前验证者在验证者列表中的索引
func (c *Config) GetMyValidatorIndex() int {
	if c.Consensus.MyValidator == "" {
		return -1
	}

	for i, validator := range c.Consensus.Validators {
		if validator == c.Consensus.MyValidator {
			return i
		}
	}

	return -1 // 未找到
}
