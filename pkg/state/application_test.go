package state

import (
	"context"
	"encoding/json"
	"testing"

	db "github.com/cometbft/cometbft-db"
	abcitypes "github.com/cometbft/cometbft/abci/types"
	"github.com/stretchr/testify/require"
)

func TestApplication(t *testing.T) {
	// 创建内存数据库
	memDB := db.NewMemDB()
	app := NewApplication(memDB)

	// 测试交易处理
	t.Run("Transaction", func(t *testing.T) {
		// 创建状态实例
		state := NewState(memDB)

		// 创建发送方账户并设置初始余额
		senderAccount := &Account{
			Address: "sender",
			Balance: 1000,
		}
		err := state.SetAccount(senderAccount)
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
		tx[0] = TxTypeTransfer // 转账交易类型
		copy(tx[1:], data)

		// 检查交易
		checkResp, err := app.CheckTx(context.Background(), &abcitypes.CheckTxRequest{Tx: tx})
		require.NoError(t, err)
		require.Equal(t, uint32(0), checkResp.Code, "CheckTx failed: %s", checkResp.Log)

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

	// 测试区块处理
	t.Run("Block", func(t *testing.T) {
		// 准备区块
		resp, err := app.PrepareProposal(context.Background(), &abcitypes.PrepareProposalRequest{
			Txs: [][]byte{},
		})
		require.NoError(t, err)
		require.Empty(t, resp.Txs)

		// 处理区块
		resp2, err := app.ProcessProposal(context.Background(), &abcitypes.ProcessProposalRequest{
			Txs: [][]byte{},
		})
		require.NoError(t, err)
		require.Equal(t, abcitypes.PROCESS_PROPOSAL_STATUS_ACCEPT, resp2.Status)

		// 最终化区块
		resp3, err := app.FinalizeBlock(context.Background(), &abcitypes.FinalizeBlockRequest{
			Txs: [][]byte{},
		})
		require.NoError(t, err)
		require.NotEmpty(t, resp3.AppHash)

		// 提交区块
		resp4, err := app.Commit(context.Background(), &abcitypes.CommitRequest{})
		require.NoError(t, err)
		require.Equal(t, int64(0), resp4.RetainHeight)
	})
}
