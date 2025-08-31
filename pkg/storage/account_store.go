package storage

import (
	"fmt"

	"github.com/govm-net/shardmatrix/pkg/types"
	"github.com/syndtr/goleveldb/leveldb"
)

// AccountStoreInterface 账户存储接口
type AccountStoreInterface interface {
	// PutAccount 存储账户
	PutAccount(account *types.Account) error

	// GetAccount 获取账户
	GetAccount(address types.Address) (*types.Account, error)

	// HasAccount 检查账户是否存在
	HasAccount(address types.Address) bool

	// DeleteAccount 删除账户
	DeleteAccount(address types.Address) error

	// UpdateBalance 更新账户余额
	UpdateBalance(address types.Address, balance uint64) error

	// GetBalance 获取账户余额
	GetBalance(address types.Address) (uint64, error)

	// Close 关闭存储
	Close() error
}

// AccountStore 账户存储实现
type AccountStore struct {
	db *leveldb.DB
}

// NewAccountStore 创建账户存储
func NewAccountStore(dbPath string) (*AccountStore, error) {
	db, err := leveldb.OpenFile(dbPath, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to open account store: %v", err)
	}

	return &AccountStore{
		db: db,
	}, nil
}

// PutAccount 存储账户
func (as *AccountStore) PutAccount(account *types.Account) error {
	// 使用JSON序列化账户
	data, err := account.Serialize()
	if err != nil {
		return fmt.Errorf("failed to serialize account: %v", err)
	}

	// 存储到数据库
	key := fmt.Sprintf("account:%s", account.Address.String())
	if err := as.db.Put([]byte(key), data, nil); err != nil {
		return fmt.Errorf("failed to store account: %v", err)
	}

	return nil
}

// GetAccount 获取账户
func (as *AccountStore) GetAccount(address types.Address) (*types.Account, error) {
	key := fmt.Sprintf("account:%s", address.String())
	data, err := as.db.Get([]byte(key), nil)
	if err != nil {
		if err == leveldb.ErrNotFound {
			return nil, fmt.Errorf("account not found: %x", address)
		}
		return nil, fmt.Errorf("failed to get account: %v", err)
	}

	// 使用JSON反序列化账户
	account, err := types.DeserializeAccount(data)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize account: %v", err)
	}

	return account, nil
}

// HasAccount 检查账户是否存在
func (as *AccountStore) HasAccount(address types.Address) bool {
	key := fmt.Sprintf("account:%s", address.String())
	_, err := as.db.Get([]byte(key), nil)
	return err == nil
}

// DeleteAccount 删除账户
func (as *AccountStore) DeleteAccount(address types.Address) error {
	key := fmt.Sprintf("account:%s", address.String())
	if err := as.db.Delete([]byte(key), nil); err != nil {
		return fmt.Errorf("failed to delete account: %v", err)
	}
	return nil
}

// UpdateBalance 更新账户余额
func (as *AccountStore) UpdateBalance(address types.Address, balance uint64) error {
	account, err := as.GetAccount(address)
	if err != nil {
		// 如果账户不存在，创建新账户
		account = types.NewAccountWithBalance(address, balance)
	} else {
		account.Balance = balance
	}

	return as.PutAccount(account)
}

// GetBalance 获取账户余额
func (as *AccountStore) GetBalance(address types.Address) (uint64, error) {
	account, err := as.GetAccount(address)
	if err != nil {
		return 0, err
	}
	return account.Balance, nil
}

// Close 关闭存储
func (as *AccountStore) Close() error {
	return as.db.Close()
}

// MemoryAccountStore 内存账户存储（用于测试）
type MemoryAccountStore struct {
	accounts map[string]*types.Account
}

// NewMemoryAccountStore 创建内存账户存储
func NewMemoryAccountStore() *MemoryAccountStore {
	return &MemoryAccountStore{
		accounts: make(map[string]*types.Account),
	}
}

// PutAccount 存储账户
func (mas *MemoryAccountStore) PutAccount(account *types.Account) error {
	addrStr := account.Address.String()
	// 克隆账户以避免外部修改
	mas.accounts[addrStr] = account.Clone()
	return nil
}

// GetAccount 获取账户
func (mas *MemoryAccountStore) GetAccount(address types.Address) (*types.Account, error) {
	addrStr := address.String()
	account, exists := mas.accounts[addrStr]
	if !exists {
		return nil, fmt.Errorf("account not found: %x", address)
	}
	// 返回克隆的账户以避免外部修改
	return account.Clone(), nil
}

// HasAccount 检查账户是否存在
func (mas *MemoryAccountStore) HasAccount(address types.Address) bool {
	addrStr := address.String()
	_, exists := mas.accounts[addrStr]
	return exists
}

// DeleteAccount 删除账户
func (mas *MemoryAccountStore) DeleteAccount(address types.Address) error {
	addrStr := address.String()
	delete(mas.accounts, addrStr)
	return nil
}

// UpdateBalance 更新账户余额
func (mas *MemoryAccountStore) UpdateBalance(address types.Address, balance uint64) error {
	account, err := mas.GetAccount(address)
	if err != nil {
		// 如果账户不存在，创建新账户
		account = types.NewAccountWithBalance(address, balance)
	} else {
		account.Balance = balance
	}

	return mas.PutAccount(account)
}

// GetBalance 获取账户余额
func (mas *MemoryAccountStore) GetBalance(address types.Address) (uint64, error) {
	account, err := mas.GetAccount(address)
	if err != nil {
		return 0, err
	}
	return account.Balance, nil
}

// Close 关闭存储
func (mas *MemoryAccountStore) Close() error {
	return nil
}

// GetAllAccounts 获取所有账户（用于测试）
func (mas *MemoryAccountStore) GetAllAccounts() []*types.Account {
	accounts := make([]*types.Account, 0, len(mas.accounts))
	for _, account := range mas.accounts {
		accounts = append(accounts, account.Clone())
	}
	return accounts
}
