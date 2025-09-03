// Package wallet 提供ShardMatrix区块链的钱包功能
package wallet

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"path/filepath"

	"github.com/govm-net/shardmatrix/pkg/crypto"
	"github.com/govm-net/shardmatrix/pkg/types"
)

// Wallet 钱包结构
type Wallet struct {
	Address    types.Address `json:"address"`
	PrivateKey []byte        `json:"private_key"` // 实际应用中应该加密存储
	PublicKey  []byte        `json:"public_key"`
}

// WalletManager 钱包管理器
type WalletManager struct {
	wallets map[string]*Wallet
}

// NewWalletManager 创建新的钱包管理器
func NewWalletManager() *WalletManager {
	return &WalletManager{
		wallets: make(map[string]*Wallet),
	}
}

// CreateWallet 创建新钱包
func (wm *WalletManager) CreateWallet() (*Wallet, error) {
	// 生成密钥对
	keyPair, err := crypto.GenerateKeyPair()
	if err != nil {
		return nil, fmt.Errorf("failed to generate key pair: %v", err)
	}

	// 获取地址
	address := keyPair.GetAddress()

	// 创建钱包
	wallet := &Wallet{
		Address:    address,
		PrivateKey: keyPair.PrivateKey.D.Bytes(), // 简化存储，实际应用中应该更安全
		PublicKey:  append(keyPair.PublicKey.X.Bytes(), keyPair.PublicKey.Y.Bytes()...),
	}

	// 保存到管理器
	wm.wallets[address.String()] = wallet

	return wallet, nil
}

// GetWallet 获取钱包
func (wm *WalletManager) GetWallet(address types.Address) (*Wallet, error) {
	wallet, exists := wm.wallets[address.String()]
	if !exists {
		return nil, fmt.Errorf("wallet not found for address: %s", address.String())
	}
	return wallet, nil
}

// ListWallets 列出所有钱包
func (wm *WalletManager) ListWallets() []types.Address {
	addresses := make([]types.Address, 0, len(wm.wallets))
	for _, wallet := range wm.wallets {
		addresses = append(addresses, wallet.Address)
	}
	return addresses
}

// SaveToFile 保存钱包到文件
func (wm *WalletManager) SaveToFile(filename string) error {
	// 确保目录存在
	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %v", err)
	}

	// 序列化钱包数据
	data, err := json.MarshalIndent(wm.wallets, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal wallets: %v", err)
	}

	// 写入文件
	if err := os.WriteFile(filename, data, 0600); err != nil {
		return fmt.Errorf("failed to write file: %v", err)
	}

	return nil
}

// LoadFromFile 从文件加载钱包
func (wm *WalletManager) LoadFromFile(filename string) error {
	// 检查文件是否存在
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		// 文件不存在，创建空的钱包文件
		wm.wallets = make(map[string]*Wallet)
		return wm.SaveToFile(filename)
	}

	// 读取文件
	data, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read file: %v", err)
	}

	// 反序列化钱包数据
	if err := json.Unmarshal(data, &wm.wallets); err != nil {
		return fmt.Errorf("failed to unmarshal wallets: %v", err)
	}

	return nil
}

// SignTransaction 使用钱包签名交易
func (w *Wallet) SignTransaction(tx *types.Transaction) error {
	// 重构私钥
	privateKey := &ecdsa.PrivateKey{
		D: new(big.Int).SetBytes(w.PrivateKey),
		PublicKey: ecdsa.PublicKey{
			Curve: elliptic.P256(),
		},
	}

	// 从公钥字节恢复公钥坐标
	if len(w.PublicKey) >= 64 {
		privateKey.PublicKey.X = new(big.Int).SetBytes(w.PublicKey[:32])
		privateKey.PublicKey.Y = new(big.Int).SetBytes(w.PublicKey[32:64])
	}

	// 创建密钥对
	keyPair := &crypto.KeyPair{
		PrivateKey: privateKey,
		PublicKey:  &privateKey.PublicKey,
	}

	// 签名交易
	signature, err := keyPair.SignData(tx.Hash().Bytes())
	if err != nil {
		return fmt.Errorf("failed to sign transaction: %v", err)
	}

	// 设置交易签名
	tx.Signature = signature

	return nil
}

// GetBalance 获取余额（模拟）
func (w *Wallet) GetBalance() uint64 {
	// 这里应该从区块链查询实际余额
	// 目前返回模拟值
	return 1000000
}
