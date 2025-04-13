package state

import (
	"context"
	"encoding/json"
	"fmt"
)

// TransferProcess 处理转账交易
type TransferProcess struct{}

func (p *TransferProcess) Type() byte {
	return TxTypeTransfer
}

func (p *TransferProcess) Validate(ctx context.Context, state *State, tx *Transaction) error {
	var data struct {
		From   string `json:"from"`
		To     string `json:"to"`
		Amount uint64 `json:"amount"`
	}

	if err := json.Unmarshal(tx.Data, &data); err != nil {
		return fmt.Errorf("invalid transfer data: %v", err)
	}

	if data.From == "" || data.To == "" {
		return fmt.Errorf("invalid addresses")
	}

	if data.Amount == 0 {
		return fmt.Errorf("invalid amount")
	}

	// 检查发送方账户是否存在
	fromAccount, err := state.GetAccount(data.From)
	if err != nil {
		return fmt.Errorf("failed to get sender account: %v", err)
	}

	// 检查余额是否足够
	if fromAccount.Balance < data.Amount {
		return fmt.Errorf("insufficient balance")
	}

	// 检查接收方账户是否存在
	_, err = state.GetAccount(data.To)
	if err != nil {
		return fmt.Errorf("failed to get receiver account: %v", err)
	}

	return nil
}

func (p *TransferProcess) Execute(ctx context.Context, state *State, tx *Transaction) error {
	var data struct {
		From   string `json:"from"`
		To     string `json:"to"`
		Amount uint64 `json:"amount"`
	}

	if err := json.Unmarshal(tx.Data, &data); err != nil {
		return fmt.Errorf("invalid transfer data: %v", err)
	}

	// 获取发送方账户
	fromAccount, err := state.GetAccount(data.From)
	if err != nil {
		return fmt.Errorf("failed to get sender account: %v", err)
	}

	// 获取接收方账户
	toAccount, err := state.GetAccount(data.To)
	if err != nil {
		return fmt.Errorf("failed to get receiver account: %v", err)
	}

	// 更新余额
	fromAccount.Balance -= data.Amount
	toAccount.Balance += data.Amount

	// 保存账户
	if err := state.SetAccount(fromAccount); err != nil {
		return fmt.Errorf("failed to save sender account: %v", err)
	}
	if err := state.SetAccount(toAccount); err != nil {
		return fmt.Errorf("failed to save receiver account: %v", err)
	}

	return nil
}

// ContractProcess 处理合约交易
type ContractProcess struct{}

func (p *ContractProcess) Type() byte {
	return TxTypeContract
}

func (p *ContractProcess) Validate(ctx context.Context, state *State, tx *Transaction) error {
	var data struct {
		Code    string `json:"code"`
		Address string `json:"address"`
	}

	if err := json.Unmarshal(tx.Data, &data); err != nil {
		return fmt.Errorf("invalid contract data: %v", err)
	}

	if data.Code == "" {
		return fmt.Errorf("empty contract code")
	}

	// 检查合约地址是否已存在
	_, err := state.GetContract(data.Address)
	if err == nil {
		return fmt.Errorf("contract address already exists")
	}

	return nil
}

func (p *ContractProcess) Execute(ctx context.Context, state *State, tx *Transaction) error {
	var data struct {
		Code    string `json:"code"`
		Address string `json:"address"`
	}

	if err := json.Unmarshal(tx.Data, &data); err != nil {
		return fmt.Errorf("invalid contract data: %v", err)
	}

	// 创建合约
	contract := &Contract{
		Address: data.Address,
		Code:    []byte(data.Code),
	}

	// 保存合约
	if err := state.SetContract(contract); err != nil {
		return fmt.Errorf("failed to save contract: %v", err)
	}

	return nil
}

func init() {
	DefaultTxRegistry.Register(&TransferProcess{})
	DefaultTxRegistry.Register(&ContractProcess{})
}
