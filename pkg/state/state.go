package state

import (
	"encoding/json"
	"fmt"

	db "github.com/cometbft/cometbft-db"
)

// Namespace 命名空间
const (
	NamespaceAccount  = "account"
	NamespaceContract = "contract"
	NamespaceBlock    = "block"
	NamespaceTx       = "tx"
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

// Get 获取状态
func (s *State) Get(namespace, key string, value interface{}) error {
	data, err := s.db.Get([]byte(fmt.Sprintf("%s:%s", namespace, key)))
	if err != nil {
		return err
	}

	if len(data) == 0 {
		return nil
	}
	return json.Unmarshal(data, value)
}

// Set 保存状态
func (s *State) Set(namespace, key string, value interface{}) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}

	return s.db.Set([]byte(fmt.Sprintf("%s:%s", namespace, key)), data)
}

// Delete 删除状态
func (s *State) Delete(namespace, key string) error {
	return s.db.Delete([]byte(fmt.Sprintf("%s:%s", namespace, key)))
}

// GetAccount retrieves an account from the state
func (s *State) GetAccount(address string) (*Account, error) {
	var account Account
	if err := s.Get(NamespaceAccount, address, &account); err != nil {
		return nil, err
	}
	if account.Address == "" {
		account.Address = address
	}
	return &account, nil
}

// SetAccount saves an account to the state
func (s *State) SetAccount(account *Account) error {
	return s.Set(NamespaceAccount, account.Address, account)
}

// GetContract retrieves a contract from the state
func (s *State) GetContract(address string) (*Contract, error) {
	var contract Contract
	if err := s.Get(NamespaceContract, address, &contract); err != nil {
		return nil, err
	}
	return &contract, nil
}

// SetContract saves a contract to the state
func (s *State) SetContract(contract *Contract) error {
	return s.Set(NamespaceContract, contract.Address, contract)
}

// DeleteContract 删除合约
func (s *State) DeleteContract(address string) error {
	return s.Delete(NamespaceContract, address)
}

// Hash returns the current state hash
func (s *State) Hash() ([]byte, error) {
	iter, err := s.db.Iterator(nil, nil)
	if err != nil {
		return nil, err
	}
	defer iter.Close()

	var pairs [][]byte
	for ; iter.Valid(); iter.Next() {
		pairs = append(pairs, iter.Key(), iter.Value())
	}

	hash := make([]byte, 32)
	for _, pair := range pairs {
		for i := range hash {
			hash[i] ^= pair[i%len(pair)]
		}
	}

	return hash, nil
}
