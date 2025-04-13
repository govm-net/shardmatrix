package tx

import (
	"context"
	"encoding/json"
	"fmt"

	db "github.com/cometbft/cometbft-db"
)

// TransferProcess 转账交易处理器
type TransferProcess struct{}

// Type 返回交易类型
func (p *TransferProcess) Type() byte {
	return TxTypeTransfer
}

// Validate 验证转账交易
func (p *TransferProcess) Validate(ctx context.Context, db db.DB, txBytes []byte) error {
	var tx Transaction
	if err := json.Unmarshal(txBytes, &tx); err != nil {
		return fmt.Errorf("invalid transaction: %w", err)
	}

	var data struct {
		From   string `json:"from"`
		To     string `json:"to"`
		Amount uint64 `json:"amount"`
	}
	if err := json.Unmarshal(tx.Data, &data); err != nil {
		return fmt.Errorf("invalid transfer data: %w", err)
	}

	// 获取发送方账户
	accountData, err := db.Get([]byte(fmt.Sprintf("account:%s", data.From)))
	if err != nil {
		return fmt.Errorf("failed to get account: %w", err)
	}
	if len(accountData) == 0 {
		return fmt.Errorf("account not found: %s", data.From)
	}

	var account struct {
		Balance uint64 `json:"balance"`
		Nonce   uint64 `json:"nonce"`
	}
	if err := json.Unmarshal(accountData, &account); err != nil {
		return fmt.Errorf("failed to unmarshal account: %w", err)
	}

	// 检查余额
	if account.Balance < data.Amount {
		return fmt.Errorf("insufficient balance: %d < %d", account.Balance, data.Amount)
	}

	return nil
}

// Execute 执行转账交易
func (p *TransferProcess) Execute(ctx context.Context, db db.DB, txBytes []byte) error {
	var tx Transaction
	if err := json.Unmarshal(txBytes, &tx); err != nil {
		return fmt.Errorf("invalid transaction: %w", err)
	}

	var data struct {
		From   string `json:"from"`
		To     string `json:"to"`
		Amount uint64 `json:"amount"`
	}
	if err := json.Unmarshal(tx.Data, &data); err != nil {
		return fmt.Errorf("invalid transfer data: %w", err)
	}

	// 获取发送方账户
	fromAccountData, err := db.Get([]byte(fmt.Sprintf("account:%s", data.From)))
	if err != nil {
		return fmt.Errorf("failed to get from account: %w", err)
	}
	if len(fromAccountData) == 0 {
		return fmt.Errorf("from account not found: %s", data.From)
	}

	var fromAccount struct {
		Balance uint64 `json:"balance"`
		Nonce   uint64 `json:"nonce"`
	}
	if err := json.Unmarshal(fromAccountData, &fromAccount); err != nil {
		return fmt.Errorf("failed to unmarshal from account: %w", err)
	}

	// 获取接收方账户
	toAccountData, err := db.Get([]byte(fmt.Sprintf("account:%s", data.To)))
	if err != nil {
		return fmt.Errorf("failed to get to account: %w", err)
	}

	var toAccount struct {
		Balance uint64 `json:"balance"`
		Nonce   uint64 `json:"nonce"`
	}
	if len(toAccountData) > 0 {
		if err := json.Unmarshal(toAccountData, &toAccount); err != nil {
			return fmt.Errorf("failed to unmarshal to account: %w", err)
		}
	}

	// 更新余额
	fromAccount.Balance -= data.Amount
	toAccount.Balance += data.Amount
	fromAccount.Nonce++

	// 保存发送方账户
	fromData, err := json.Marshal(fromAccount)
	if err != nil {
		return fmt.Errorf("failed to marshal from account: %w", err)
	}
	if err := db.Set([]byte(fmt.Sprintf("account:%s", data.From)), fromData); err != nil {
		return fmt.Errorf("failed to save from account: %w", err)
	}

	// 保存接收方账户
	toData, err := json.Marshal(toAccount)
	if err != nil {
		return fmt.Errorf("failed to marshal to account: %w", err)
	}
	if err := db.Set([]byte(fmt.Sprintf("account:%s", data.To)), toData); err != nil {
		return fmt.Errorf("failed to save to account: %w", err)
	}

	return nil
}

// ContractProcess 合约交易处理器
type ContractProcess struct{}

// Type 返回交易类型
func (p *ContractProcess) Type() byte {
	return TxTypeContract
}

// Validate 验证合约交易
func (p *ContractProcess) Validate(ctx context.Context, db db.DB, txBytes []byte) error {
	var tx Transaction
	if err := json.Unmarshal(txBytes, &tx); err != nil {
		return fmt.Errorf("invalid transaction: %w", err)
	}

	var data struct {
		From    string `json:"from"`
		Address string `json:"address"`
		Code    []byte `json:"code"`
	}
	if err := json.Unmarshal(tx.Data, &data); err != nil {
		return fmt.Errorf("invalid contract data: %w", err)
	}

	// 获取发送方账户
	accountData, err := db.Get([]byte(fmt.Sprintf("account:%s", data.From)))
	if err != nil {
		return fmt.Errorf("failed to get account: %w", err)
	}
	if len(accountData) == 0 {
		return fmt.Errorf("account not found: %s", data.From)
	}

	return nil
}

// Execute 执行合约交易
func (p *ContractProcess) Execute(ctx context.Context, db db.DB, txBytes []byte) error {
	var tx Transaction
	if err := json.Unmarshal(txBytes, &tx); err != nil {
		return fmt.Errorf("invalid transaction: %w", err)
	}

	var data struct {
		From    string `json:"from"`
		Address string `json:"address"`
		Code    []byte `json:"code"`
	}
	if err := json.Unmarshal(tx.Data, &data); err != nil {
		return fmt.Errorf("invalid contract data: %w", err)
	}

	// 创建合约
	contract := struct {
		Address   string                 `json:"address"`
		Code      []byte                 `json:"code"`
		State     map[string]interface{} `json:"state"`
		Creator   string                 `json:"creator"`
		CreatedAt int64                  `json:"created_at"`
	}{
		Address:   data.Address,
		Code:      data.Code,
		State:     make(map[string]interface{}),
		Creator:   data.From,
		CreatedAt: 0, // TODO: 使用实际时间戳
	}

	// 保存合约
	contractData, err := json.Marshal(contract)
	if err != nil {
		return fmt.Errorf("failed to marshal contract: %w", err)
	}
	if err := db.Set([]byte(fmt.Sprintf("contract:%s", data.Address)), contractData); err != nil {
		return fmt.Errorf("failed to save contract: %w", err)
	}

	return nil
}
