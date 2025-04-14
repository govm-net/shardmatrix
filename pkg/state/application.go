package state

import (
	"context"
	"encoding/json"
	"fmt"

	db "github.com/cometbft/cometbft-db"
	abcitypes "github.com/cometbft/cometbft/abci/types"
	"github.com/govm-net/shardmatrix/pkg/tx"
)

// Application represents the ABCI application
type Application struct {
	db       db.DB
	registry *tx.TxRegistry
	state    *State
}

var _ abcitypes.Application = (*Application)(nil)

// NewApplication creates a new ABCI application
func NewApplication(db db.DB) *Application {
	app := &Application{
		db:       db,
		registry: tx.DefaultTxRegistry,
		state:    NewState(db),
	}

	return app
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
func (app *Application) CheckTx(ctx context.Context, req *abcitypes.CheckTxRequest) (*abcitypes.CheckTxResponse, error) {
	if len(req.Tx) == 0 {
		return &abcitypes.CheckTxResponse{Code: 1, Log: "Empty transaction"}, nil
	}

	// 检查交易类型是否已注册
	process, exists := app.registry.GetProcessor(req.Tx[0])
	if !exists {
		return &abcitypes.CheckTxResponse{Code: 1, Log: "Unknown transaction type"}, nil
	}

	// 验证交易
	if err := process.Validate(ctx, app.db, req.Tx); err != nil {
		return &abcitypes.CheckTxResponse{Code: 1, Log: err.Error()}, nil
	}

	return &abcitypes.CheckTxResponse{Code: 0}, nil
}

// PrepareProposal prepares a block proposal
func (app *Application) PrepareProposal(_ context.Context, req *abcitypes.PrepareProposalRequest) (*abcitypes.PrepareProposalResponse, error) {
	var txs [][]byte
	for _, txData := range req.Txs {
		if len(txData) == 0 {
			continue
		}

		process, exists := app.registry.GetProcessor(txData[0])
		if !exists {
			continue
		}

		// 验证交易
		if err := process.Validate(context.Background(), app.db, txData); err != nil {
			continue
		}

		txs = append(txs, txData)
	}

	return &abcitypes.PrepareProposalResponse{
		Txs: txs,
	}, nil
}

// ProcessProposal processes a block proposal
func (app *Application) ProcessProposal(ctx context.Context, req *abcitypes.ProcessProposalRequest) (*abcitypes.ProcessProposalResponse, error) {
	for _, txData := range req.Txs {
		if len(txData) == 0 {
			continue
		}

		txType := txData[0]
		process, exists := app.registry.GetProcessor(txType)
		if !exists {
			continue
		}

		// 验证交易
		if err := process.Validate(ctx, app.db, txData); err != nil {
			return &abcitypes.ProcessProposalResponse{
				Status: abcitypes.PROCESS_PROPOSAL_STATUS_REJECT,
			}, nil
		}
	}

	return &abcitypes.ProcessProposalResponse{
		Status: abcitypes.PROCESS_PROPOSAL_STATUS_ACCEPT,
	}, nil
}

// FinalizeBlock finalizes a block
func (app *Application) FinalizeBlock(ctx context.Context, req *abcitypes.FinalizeBlockRequest) (*abcitypes.FinalizeBlockResponse, error) {
	for _, txData := range req.Txs {
		if len(txData) == 0 {
			continue
		}

		process, exists := app.registry.GetProcessor(txData[0])
		if !exists {
			continue
		}

		// 验证交易
		if err := process.Validate(ctx, app.db, txData); err != nil {
			continue
		}

		// 执行交易
		if err := process.Execute(ctx, app.db, txData); err != nil {
			continue
		}
	}

	// 计算状态哈希
	hash, err := app.state.Hash()
	if err != nil {
		return nil, fmt.Errorf("failed to calculate state hash: %v", err)
	}

	return &abcitypes.FinalizeBlockResponse{
		AppHash: hash,
	}, nil
}

// Query handles queries about the application state
func (app *Application) Query(_ context.Context, req *abcitypes.QueryRequest) (*abcitypes.QueryResponse, error) {
	val, err := app.db.Get([]byte(req.Path))
	if err != nil {
		return &abcitypes.QueryResponse{Code: 1, Log: fmt.Sprintf("Failed to get account: %v", err)}, nil
	}
	return &abcitypes.QueryResponse{
		Code:  0,
		Value: val,
	}, nil
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
