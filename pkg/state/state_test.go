package state

import (
	"testing"

	db "github.com/cometbft/cometbft-db"
	"github.com/stretchr/testify/require"
)

func TestState(t *testing.T) {
	// 创建内存数据库
	memDB := db.NewMemDB()
	state := NewState(memDB)

	// 测试账户操作
	t.Run("Account", func(t *testing.T) {
		// 创建测试账户
		account := &Account{
			Address: "test-address",
			Balance: 100,
			Nonce:   1,
		}

		// 保存账户
		err := state.SetAccount(account)
		require.NoError(t, err)

		// 获取账户
		got, err := state.GetAccount(account.Address)
		require.NoError(t, err)
		require.Equal(t, account, got)

		// 获取不存在的账户
		_, err = state.GetAccount("non-existent")
		require.NoError(t, err)
	})

	// 测试合约操作
	t.Run("Contract", func(t *testing.T) {
		// 创建测试合约
		contract := &Contract{
			Address:   "test-contract",
			Code:      []byte("test code"),
			State:     map[string]interface{}{"key": "value"},
			Creator:   "test-creator",
			CreatedAt: 1234567890,
		}

		// 保存合约
		err := state.SetContract(contract)
		require.NoError(t, err)

		// 获取合约
		got, err := state.GetContract(contract.Address)
		require.NoError(t, err)
		require.Equal(t, contract, got)

		// 获取不存在的合约
		_, err = state.GetContract("non-existent")
		require.Error(t, err)
	})

	// 测试状态哈希
	t.Run("Hash", func(t *testing.T) {
		// 初始状态
		hash1, err := state.Hash()
		require.NoError(t, err)
		require.NotEmpty(t, hash1)

		// 修改状态
		account := &Account{
			Address: "test",
			Balance: 100,
		}
		err = state.SetAccount(account)
		require.NoError(t, err)

		// 获取新哈希
		hash2, err := state.Hash()
		require.NoError(t, err)
		require.NotEmpty(t, hash2)

		// 验证哈希值不同
		require.NotEqual(t, hash1, hash2)
	})
}
