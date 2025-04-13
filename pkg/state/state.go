package state

import (
	"encoding/json"
	"fmt"

	db "github.com/cometbft/cometbft-db"
)

// Account represents a user account in the system
type Account struct {
	Address string
	Balance uint64
	Nonce   uint64
}

// Contract represents a smart contract in the system
type Contract struct {
	Address   string
	Code      []byte
	State     map[string]interface{}
	Creator   string
	CreatedAt int64
}

// State represents the application state
type State struct {
	db db.DB
}

// NewState creates a new state instance
func NewState(db db.DB) *State {
	return &State{
		db: db,
	}
}

// GetAccount retrieves an account from the state
func (s *State) GetAccount(address string) (*Account, error) {
	data, err := s.db.Get([]byte(fmt.Sprintf("account:%s", address)))
	if err != nil {
		return nil, err
	}

	var account Account
	if err := json.Unmarshal(data, &account); err != nil {
		return nil, err
	}

	return &account, nil
}

// SetAccount saves an account to the state
func (s *State) SetAccount(account *Account) error {
	data, err := json.Marshal(account)
	if err != nil {
		return err
	}

	return s.db.Set([]byte(fmt.Sprintf("account:%s", account.Address)), data)
}

// GetContract retrieves a contract from the state
func (s *State) GetContract(address string) (*Contract, error) {
	data, err := s.db.Get([]byte(fmt.Sprintf("contract:%s", address)))
	if err != nil {
		return nil, err
	}

	var contract Contract
	if err := json.Unmarshal(data, &contract); err != nil {
		return nil, err
	}

	return &contract, nil
}

// SetContract saves a contract to the state
func (s *State) SetContract(contract *Contract) error {
	data, err := json.Marshal(contract)
	if err != nil {
		return err
	}

	return s.db.Set([]byte(fmt.Sprintf("contract:%s", contract.Address)), data)
}

// Hash returns the current state hash
func (s *State) Hash() ([]byte, error) {
	// 创建一个迭代器来遍历所有键值对
	iter, err := s.db.Iterator(nil, nil)
	if err != nil {
		return nil, err
	}
	defer iter.Close()

	// 收集所有键值对
	var pairs [][]byte
	for ; iter.Valid(); iter.Next() {
		pairs = append(pairs, iter.Key(), iter.Value())
	}

	// 计算默克尔根哈希
	// TODO: 使用更高效的默克尔树实现
	hash := make([]byte, 32) // 32字节的哈希值
	for _, pair := range pairs {
		// 简单的哈希计算，实际应该使用更安全的哈希算法
		for i := range hash {
			hash[i] ^= pair[i%len(pair)]
		}
	}

	return hash, nil
}
