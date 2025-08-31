package types

import (
	"encoding/json"
	"time"
)

// Account 账户结构
type Account struct {
	Address   Address `json:"address"`    // 账户地址
	Balance   uint64  `json:"balance"`    // 账户余额
	Nonce     uint64  `json:"nonce"`      // 交易计数器
	UpdatedAt int64   `json:"updated_at"` // 最后更新时间
}

// NewAccount 创建新账户
func NewAccount(address Address) *Account {
	return &Account{
		Address:   address,
		Balance:   0,
		Nonce:     0,
		UpdatedAt: time.Now().Unix(),
	}
}

// NewAccountWithBalance 创建带余额的账户
func NewAccountWithBalance(address Address, balance uint64) *Account {
	return &Account{
		Address:   address,
		Balance:   balance,
		Nonce:     0,
		UpdatedAt: time.Now().Unix(),
	}
}

// AddBalance 增加余额
func (a *Account) AddBalance(amount uint64) {
	a.Balance += amount
	a.UpdatedAt = time.Now().Unix()
}

// SubBalance 减少余额
func (a *Account) SubBalance(amount uint64) bool {
	if a.Balance < amount {
		return false
	}
	a.Balance -= amount
	a.UpdatedAt = time.Now().Unix()
	return true
}

// HasSufficientBalance 检查是否有足够余额
func (a *Account) HasSufficientBalance(amount uint64) bool {
	return a.Balance >= amount
}

// IncrementNonce 增加nonce
func (a *Account) IncrementNonce() {
	a.Nonce++
	a.UpdatedAt = time.Now().Unix()
}

// IsEmpty 检查账户是否为空
func (a *Account) IsEmpty() bool {
	return a.Balance == 0 && a.Nonce == 0
}

// Clone 克隆账户
func (a *Account) Clone() *Account {
	return &Account{
		Address:   a.Address,
		Balance:   a.Balance,
		Nonce:     a.Nonce,
		UpdatedAt: a.UpdatedAt,
	}
}

// Serialize 序列化账户为JSON
func (a *Account) Serialize() ([]byte, error) {
	return json.Marshal(a)
}

// Deserialize 从 JSON 反序列化账户
func (a *Account) Deserialize(data []byte) error {
	return json.Unmarshal(data, a)
}

// DeserializeAccount 从 JSON 创建账户
func DeserializeAccount(data []byte) (*Account, error) {
	var account Account
	err := json.Unmarshal(data, &account)
	if err != nil {
		return nil, err
	}
	return &account, nil
}
