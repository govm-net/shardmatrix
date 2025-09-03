package wallet

import (
	"os"
	"testing"

	"github.com/govm-net/shardmatrix/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWalletManager(t *testing.T) {
	t.Run("CreateWallet", func(t *testing.T) {
		// 创建钱包管理器
		wm := NewWalletManager()

		// 创建钱包
		wallet, err := wm.CreateWallet()
		require.NoError(t, err)
		assert.NotEmpty(t, wallet.Address)
		assert.NotEmpty(t, wallet.PrivateKey)
		assert.NotEmpty(t, wallet.PublicKey)
	})

	t.Run("GetWallet", func(t *testing.T) {
		// 创建钱包管理器
		wm := NewWalletManager()

		// 创建钱包
		wallet, err := wm.CreateWallet()
		require.NoError(t, err)

		// 获取钱包
		retrievedWallet, err := wm.GetWallet(wallet.Address)
		require.NoError(t, err)
		assert.Equal(t, wallet.Address, retrievedWallet.Address)
	})

	t.Run("ListWallets", func(t *testing.T) {
		// 创建钱包管理器
		wm := NewWalletManager()

		// 创建多个钱包
		wallet1, err := wm.CreateWallet()
		require.NoError(t, err)

		wallet2, err := wm.CreateWallet()
		require.NoError(t, err)

		// 列出钱包
		wallets := wm.ListWallets()
		assert.Len(t, wallets, 2)

		// 检查地址是否在列表中
		found1 := false
		found2 := false
		for _, addr := range wallets {
			if addr.Equal(wallet1.Address) {
				found1 = true
			}
			if addr.Equal(wallet2.Address) {
				found2 = true
			}
		}
		assert.True(t, found1)
		assert.True(t, found2)
	})

	t.Run("SaveAndLoadFromFile", func(t *testing.T) {
		// 创建临时文件
		tmpFile := "/tmp/test_wallets.json"
		defer os.Remove(tmpFile)

		// 创建钱包管理器并添加钱包
		wm1 := NewWalletManager()
		wallet, err := wm1.CreateWallet()
		require.NoError(t, err)

		// 保存到文件
		err = wm1.SaveToFile(tmpFile)
		require.NoError(t, err)

		// 创建新的钱包管理器并从文件加载
		wm2 := NewWalletManager()
		err = wm2.LoadFromFile(tmpFile)
		require.NoError(t, err)

		// 检查钱包是否正确加载
		retrievedWallet, err := wm2.GetWallet(wallet.Address)
		require.NoError(t, err)
		assert.Equal(t, wallet.Address, retrievedWallet.Address)
	})

	t.Run("LoadNonExistentFile", func(t *testing.T) {
		// 创建临时文件路径（文件不存在）
		tmpFile := "/tmp/non_existent_wallets.json"
		defer os.Remove(tmpFile)

		// 加载不存在的文件应该创建空文件
		wm := NewWalletManager()
		err := wm.LoadFromFile(tmpFile)
		require.NoError(t, err)

		// 检查文件是否创建
		_, err = os.Stat(tmpFile)
		assert.NoError(t, err)

		// 检查钱包列表为空
		wallets := wm.ListWallets()
		assert.Empty(t, wallets)
	})
}

func TestWallet(t *testing.T) {
	t.Run("GetBalance", func(t *testing.T) {
		// 创建钱包管理器和钱包
		wm := NewWalletManager()
		wallet, err := wm.CreateWallet()
		require.NoError(t, err)

		// 获取余额（模拟）
		balance := wallet.GetBalance()
		assert.Equal(t, uint64(1000000), balance)
	})

	t.Run("AddressString", func(t *testing.T) {
		// 创建钱包管理器和钱包
		wm := NewWalletManager()
		wallet, err := wm.CreateWallet()
		require.NoError(t, err)

		// 检查地址字符串格式
		addrStr := wallet.Address.String()
		assert.Contains(t, addrStr, "0x")
		assert.Len(t, addrStr, 42) // "0x" + 40个十六进制字符
	})
}

func TestWalletManagerGetNonExistentWallet(t *testing.T) {
	wm := NewWalletManager()

	// 尝试获取不存在的钱包
	addr := types.GenerateAddress()
	_, err := wm.GetWallet(addr)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "wallet not found")
}
