package test

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	db "github.com/cometbft/cometbft-db"
	abcitypes "github.com/cometbft/cometbft/abci/types"
	"github.com/govm-net/shardmatrix/pkg/state"
	"github.com/stretchr/testify/require"
)

func TestE2E(t *testing.T) {
	// 创建测试目录
	testDir := filepath.Join(os.TempDir(), "shardmatrix-test")
	require.NoError(t, os.MkdirAll(testDir, 0755))
	defer os.RemoveAll(testDir)

	// 设置环境变量
	os.Setenv("HOME_DIR", testDir)
	os.Setenv("CHAIN_ID", "shardmatrix-test")
	os.Setenv("LOG_LEVEL", "debug")

	// 创建数据库和应用实例
	memDB := db.NewMemDB()
	app := state.NewApplication(memDB)

	// 初始化节点
	t.Run("InitNode", func(t *testing.T) {
		cmd := exec.Command("./scripts/init.sh")
		cmd.Dir = ".."
		output, err := cmd.CombinedOutput()
		require.NoError(t, err, "初始化失败: %s", string(output))
	})

	// 启动节点
	var nodeCmd *exec.Cmd
	t.Run("StartNode", func(t *testing.T) {
		nodeCmd = exec.Command("./scripts/start.sh")
		nodeCmd.Dir = ".."
		require.NoError(t, nodeCmd.Start())

		// 等待节点启动
		time.Sleep(5 * time.Second)
	})

	// 测试交易
	t.Run("Transaction", func(t *testing.T) {
		// 创建发送方账户并设置初始余额
		senderAccount := &state.Account{
			Address: "sender",
			Balance: 1000,
		}
		err := app.SetAccount("sender", senderAccount)
		require.NoError(t, err)

		// 创建转账交易
		transferData := struct {
			From   string `json:"from"`
			To     string `json:"to"`
			Amount uint64 `json:"amount"`
		}{
			From:   "sender",
			To:     "receiver",
			Amount: 100,
		}

		data, err := json.Marshal(transferData)
		require.NoError(t, err)

		// 创建交易字节数组
		tx := make([]byte, 1+len(data))
		tx[0] = 0x01 // 转账交易类型
		copy(tx[1:], data)

		// 检查交易
		checkResp, err := app.CheckTx(context.Background(), &abcitypes.CheckTxRequest{Tx: tx})
		require.NoError(t, err)
		require.Equal(t, uint32(0), checkResp.Code)

		// 处理交易
		finalizeResp, err := app.FinalizeBlock(context.Background(), &abcitypes.FinalizeBlockRequest{
			Txs: [][]byte{tx},
		})
		require.NoError(t, err)
		require.NotEmpty(t, finalizeResp.AppHash)

		// 查询账户余额
		queryResp, err := app.Query(context.Background(), &abcitypes.QueryRequest{
			Path: "balance",
			Data: []byte("sender"),
		})
		require.NoError(t, err)
		require.Equal(t, uint32(0), queryResp.Code)
		require.Equal(t, "900", string(queryResp.Value))

		queryResp, err = app.Query(context.Background(), &abcitypes.QueryRequest{
			Path: "balance",
			Data: []byte("receiver"),
		})
		require.NoError(t, err)
		require.Equal(t, uint32(0), queryResp.Code)
		require.Equal(t, "100", string(queryResp.Value))
	})

	// 停止节点
	t.Run("StopNode", func(t *testing.T) {
		require.NoError(t, nodeCmd.Process.Kill())
		_, err := nodeCmd.Process.Wait()
		require.NoError(t, err)
	})
}
