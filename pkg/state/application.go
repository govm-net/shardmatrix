package state

import (
	"context"
	"encoding/json"
	"fmt"

	db "github.com/cometbft/cometbft-db"
	abcitypes "github.com/cometbft/cometbft/abci/types"
)

// Application represents the ABCI application
type Application struct {
	db       db.DB
	registry *TxRegistry
}

var _ abcitypes.Application = (*Application)(nil)

// NewApplication creates a new ABCI application
func NewApplication(db db.DB) *Application {
	app := &Application{
		db:       db,
		registry: DefaultTxRegistry,
	}

	// 注册默认交易处理函数
	app.registry.Register(0x01, handleTransfer) // 转账交易
	app.registry.Register(0x02, handleContract) // 合约交易

	return app
}

// handleTransfer 处理转账交易
func handleTransfer(ctx context.Context, state *State, tx *Transaction) error {
	var transferData struct {
		From   string `json:"from"`
		To     string `json:"to"`
		Amount uint64 `json:"amount"`
	}

	if err := json.Unmarshal(tx.Data, &transferData); err != nil {
		return fmt.Errorf("invalid transfer data: %v", err)
	}

	// 获取发送方账户
	sender, err := state.GetAccount(transferData.From)
	if err != nil {
		return fmt.Errorf("failed to get sender account: %v", err)
	}

	// 检查余额
	if sender.Balance < transferData.Amount {
		return fmt.Errorf("insufficient balance")
	}

	// 获取接收方账户
	receiver, err := state.GetAccount(transferData.To)
	if err != nil {
		return fmt.Errorf("failed to get receiver account: %v", err)
	}

	// 更新余额
	sender.Balance -= transferData.Amount
	receiver.Balance += transferData.Amount

	// 保存账户
	if err := state.SetAccount(sender); err != nil {
		return fmt.Errorf("failed to save sender account: %v", err)
	}
	if err := state.SetAccount(receiver); err != nil {
		return fmt.Errorf("failed to save receiver account: %v", err)
	}

	return nil
}

// handleContract 处理合约交易
func handleContract(ctx context.Context, state *State, tx *Transaction) error {
	// TODO: 实现合约交易处理
	return nil
}

// Info returns information about the application state
func (app *Application) Info(_ context.Context, req *abcitypes.InfoRequest) (*abcitypes.InfoResponse, error) {
	return &abcitypes.InfoResponse{
		Data:             "shardmatrix",
		Version:          "0.1.0",
		AppVersion:       1,
		LastBlockHeight:  0,
		LastBlockAppHash: nil,
	}, nil
}

// CheckTx validates a transaction
func (app *Application) CheckTx(_ context.Context, req *abcitypes.CheckTxRequest) (*abcitypes.CheckTxResponse, error) {
	if len(req.Tx) == 0 {
		return &abcitypes.CheckTxResponse{Code: 1, Log: "Empty transaction"}, nil
	}

	// 检查交易类型是否已注册
	txType := req.Tx[0]
	if _, exists := app.registry.GetHandler(txType); !exists {
		return &abcitypes.CheckTxResponse{Code: 1, Log: "Unknown transaction type"}, nil
	}

	// TODO: 验证签名
	if !verifySignature(req.Tx) {
		return &abcitypes.CheckTxResponse{Code: 1, Log: "Invalid signature"}, nil
	}

	return &abcitypes.CheckTxResponse{Code: 0}, nil
}

// PrepareProposal prepares a block proposal
func (app *Application) PrepareProposal(ctx context.Context, req *abcitypes.PrepareProposalRequest) (*abcitypes.PrepareProposalResponse, error) {
	state := NewState(app.db)

	for _, txBytes := range req.Txs {
		if len(txBytes) == 0 {
			continue
		}

		txType := txBytes[0]
		handler, exists := app.registry.GetHandler(txType)
		if !exists {
			continue
		}

		// 处理交易
		tx := &Transaction{
			Type: txType,
			Data: txBytes[1:], // 剩余部分作为交易数据
		}

		if err := handler(ctx, state, tx); err != nil {
			continue
		}
	}

	return &abcitypes.PrepareProposalResponse{}, nil
}

// ProcessProposal processes a block proposal
func (app *Application) ProcessProposal(ctx context.Context, req *abcitypes.ProcessProposalRequest) (*abcitypes.ProcessProposalResponse, error) {
	state := NewState(app.db)

	for _, txBytes := range req.Txs {
		if len(txBytes) == 0 {
			return &abcitypes.ProcessProposalResponse{Status: abcitypes.PROCESS_PROPOSAL_STATUS_REJECT}, nil
		}

		txType := txBytes[0]
		handler, exists := app.registry.GetHandler(txType)
		if !exists {
			return &abcitypes.ProcessProposalResponse{Status: abcitypes.PROCESS_PROPOSAL_STATUS_REJECT}, nil
		}

		// 处理交易
		tx := &Transaction{
			Type: txType,
			Data: txBytes[1:], // 剩余部分作为交易数据
		}

		if err := handler(ctx, state, tx); err != nil {
			return &abcitypes.ProcessProposalResponse{Status: abcitypes.PROCESS_PROPOSAL_STATUS_REJECT}, nil
		}
	}

	return &abcitypes.ProcessProposalResponse{Status: abcitypes.PROCESS_PROPOSAL_STATUS_ACCEPT}, nil
}

// FinalizeBlock finalizes a block
func (app *Application) FinalizeBlock(ctx context.Context, req *abcitypes.FinalizeBlockRequest) (*abcitypes.FinalizeBlockResponse, error) {
	state := NewState(app.db)

	for _, txBytes := range req.Txs {
		if len(txBytes) == 0 {
			continue
		}

		txType := txBytes[0]
		handler, exists := app.registry.GetHandler(txType)
		if !exists {
			continue
		}

		// 处理交易
		tx := &Transaction{
			Type: txType,
			Data: txBytes[1:], // 剩余部分作为交易数据
		}

		if err := handler(ctx, state, tx); err != nil {
			continue
		}
	}

	// 获取当前状态的哈希
	hash, err := state.Hash()
	if err != nil {
		return nil, fmt.Errorf("failed to get state hash: %v", err)
	}

	return &abcitypes.FinalizeBlockResponse{
		AppHash: hash,
	}, nil
}

// Query handles queries about the application state
func (app *Application) Query(_ context.Context, req *abcitypes.QueryRequest) (*abcitypes.QueryResponse, error) {
	state := NewState(app.db)

	// Handle account balance query
	if req.Path == "balance" {
		account, err := state.GetAccount(string(req.Data))
		if err != nil {
			return &abcitypes.QueryResponse{Code: 1, Log: fmt.Sprintf("Failed to get account: %v", err)}, nil
		}
		return &abcitypes.QueryResponse{
			Code:  0,
			Value: []byte(fmt.Sprintf("%d", account.Balance)),
		}, nil
	}

	return &abcitypes.QueryResponse{
		Code: 1,
		Log:  "Unknown query path",
	}, nil
}

// verifySignature verifies the transaction signature
func verifySignature(_ []byte) bool {
	// TODO: Implement signature verification
	return true
}

// GetAccount retrieves an account from the database
func (app *Application) GetAccount(address string) (*Account, error) {
	key := []byte(fmt.Sprintf("account:%s", address))
	value, err := app.db.Get(key)
	if err != nil {
		return nil, err
	}
	if value == nil {
		return &Account{
			Address: address,
			Balance: 0,
		}, nil
	}

	var account Account
	if err := json.Unmarshal(value, &account); err != nil {
		return nil, err
	}
	return &account, nil
}

// SetAccount saves an account to the database
func (app *Application) SetAccount(address string, account *Account) error {
	key := []byte(fmt.Sprintf("account:%s", address))
	value, err := json.Marshal(account)
	if err != nil {
		return err
	}
	return app.db.Set(key, value)
}

// InitChain initializes the blockchain
func (app *Application) InitChain(_ context.Context, req *abcitypes.InitChainRequest) (*abcitypes.InitChainResponse, error) {
	return &abcitypes.InitChainResponse{
		Validators: req.Validators,
	}, nil
}

// Commit commits the current state
func (app *Application) Commit(_ context.Context, req *abcitypes.CommitRequest) (*abcitypes.CommitResponse, error) {
	return &abcitypes.CommitResponse{
		RetainHeight: 0,
	}, nil
}

// ListSnapshots lists available snapshots
func (app *Application) ListSnapshots(_ context.Context, req *abcitypes.ListSnapshotsRequest) (*abcitypes.ListSnapshotsResponse, error) {
	return &abcitypes.ListSnapshotsResponse{}, nil
}

// OfferSnapshot offers a snapshot to the application
func (app *Application) OfferSnapshot(_ context.Context, req *abcitypes.OfferSnapshotRequest) (*abcitypes.OfferSnapshotResponse, error) {
	return &abcitypes.OfferSnapshotResponse{}, nil
}

// LoadSnapshotChunk loads a snapshot chunk
func (app *Application) LoadSnapshotChunk(_ context.Context, req *abcitypes.LoadSnapshotChunkRequest) (*abcitypes.LoadSnapshotChunkResponse, error) {
	return &abcitypes.LoadSnapshotChunkResponse{}, nil
}

// ApplySnapshotChunk applies a snapshot chunk
func (app *Application) ApplySnapshotChunk(_ context.Context, req *abcitypes.ApplySnapshotChunkRequest) (*abcitypes.ApplySnapshotChunkResponse, error) {
	return &abcitypes.ApplySnapshotChunkResponse{Result: abcitypes.APPLY_SNAPSHOT_CHUNK_RESULT_ACCEPT}, nil
}

// ExtendVote extends a vote
func (app *Application) ExtendVote(_ context.Context, req *abcitypes.ExtendVoteRequest) (*abcitypes.ExtendVoteResponse, error) {
	return &abcitypes.ExtendVoteResponse{}, nil
}

// VerifyVoteExtension verifies a vote extension
func (app *Application) VerifyVoteExtension(_ context.Context, req *abcitypes.VerifyVoteExtensionRequest) (*abcitypes.VerifyVoteExtensionResponse, error) {
	return &abcitypes.VerifyVoteExtensionResponse{}, nil
}
