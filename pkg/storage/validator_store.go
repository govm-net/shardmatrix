package storage

import (
	"bytes"
	"fmt"

	"github.com/govm-net/shardmatrix/pkg/types"
	"github.com/syndtr/goleveldb/leveldb"
)

// ValidatorStoreInterface 验证者存储接口
type ValidatorStoreInterface interface {
	// PutValidator 存储验证者
	PutValidator(validator *types.Validator) error

	// GetValidator 获取验证者
	GetValidator(address types.Address) (*types.Validator, error)

	// GetAllValidators 获取所有验证者
	GetAllValidators() ([]*types.Validator, error)

	// HasValidator 检查验证者是否存在
	HasValidator(address types.Address) bool

	// DeleteValidator 删除验证者
	DeleteValidator(address types.Address) error

	// UpdateValidatorStake 更新验证者权益
	UpdateValidatorStake(address types.Address, stake uint64) error

	// Close 关闭存储
	Close() error
}

// ValidatorStore 验证者存储实现
type ValidatorStore struct {
	db *leveldb.DB
}

// NewValidatorStore 创建验证者存储
func NewValidatorStore(dbPath string) (*ValidatorStore, error) {
	db, err := leveldb.OpenFile(dbPath, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to open validator store: %v", err)
	}

	return &ValidatorStore{
		db: db,
	}, nil
}

// PutValidator 存储验证者
func (vs *ValidatorStore) PutValidator(validator *types.Validator) error {
	// 使用JSON序列化验证者
	data, err := validator.Serialize()
	if err != nil {
		return fmt.Errorf("failed to serialize validator: %v", err)
	}

	// 存储到数据库
	key := fmt.Sprintf("validator:%s", validator.Address.String())
	if err := vs.db.Put([]byte(key), data, nil); err != nil {
		return fmt.Errorf("failed to store validator: %v", err)
	}

	return nil
}

// GetValidator 获取验证者
func (vs *ValidatorStore) GetValidator(address types.Address) (*types.Validator, error) {
	key := fmt.Sprintf("validator:%s", address.String())
	data, err := vs.db.Get([]byte(key), nil)
	if err != nil {
		if err == leveldb.ErrNotFound {
			return nil, fmt.Errorf("validator not found: %x", address)
		}
		return nil, fmt.Errorf("failed to get validator: %v", err)
	}

	// 使用JSON反序列化验证者
	validator, err := types.DeserializeValidator(data)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize validator: %v", err)
	}

	return validator, nil
}

// GetAllValidators 获取所有验证者
func (vs *ValidatorStore) GetAllValidators() ([]*types.Validator, error) {
	iter := vs.db.NewIterator(nil, nil)
	defer iter.Release()

	var validators []*types.Validator
	prefix := []byte("validator:")

	for iter.Seek(prefix); iter.Valid(); iter.Next() {
		key := iter.Key()
		if !bytes.HasPrefix(key, prefix) {
			break
		}

		data := iter.Value()
		validator, err := types.DeserializeValidator(data)
		if err != nil {
			return nil, fmt.Errorf("failed to deserialize validator: %v", err)
		}

		validators = append(validators, validator)
	}

	if err := iter.Error(); err != nil {
		return nil, fmt.Errorf("iterator error: %v", err)
	}

	return validators, nil
}

// HasValidator 检查验证者是否存在
func (vs *ValidatorStore) HasValidator(address types.Address) bool {
	key := fmt.Sprintf("validator:%s", address.String())
	_, err := vs.db.Get([]byte(key), nil)
	return err == nil
}

// DeleteValidator 删除验证者
func (vs *ValidatorStore) DeleteValidator(address types.Address) error {
	key := fmt.Sprintf("validator:%s", address.String())
	if err := vs.db.Delete([]byte(key), nil); err != nil {
		return fmt.Errorf("failed to delete validator: %v", err)
	}
	return nil
}

// UpdateValidatorStake 更新验证者权益
func (vs *ValidatorStore) UpdateValidatorStake(address types.Address, stake uint64) error {
	validator, err := vs.GetValidator(address)
	if err != nil {
		return fmt.Errorf("validator not found: %v", err)
	}

	validator.Stake = stake
	return vs.PutValidator(validator)
}

// Close 关闭存储
func (vs *ValidatorStore) Close() error {
	return vs.db.Close()
}

// MemoryValidatorStore 内存验证者存储（用于测试）
type MemoryValidatorStore struct {
	validators map[string]*types.Validator
}

// NewMemoryValidatorStore 创建内存验证者存储
func NewMemoryValidatorStore() *MemoryValidatorStore {
	return &MemoryValidatorStore{
		validators: make(map[string]*types.Validator),
	}
}

// PutValidator 存储验证者
func (mvs *MemoryValidatorStore) PutValidator(validator *types.Validator) error {
	addrStr := validator.Address.String()
	// 克隆验证者以避免外部修改
	mvs.validators[addrStr] = validator.Clone()
	return nil
}

// GetValidator 获取验证者
func (mvs *MemoryValidatorStore) GetValidator(address types.Address) (*types.Validator, error) {
	addrStr := address.String()
	validator, exists := mvs.validators[addrStr]
	if !exists {
		return nil, fmt.Errorf("validator not found: %x", address)
	}
	// 返回克隆的验证者以避免外部修改
	return validator.Clone(), nil
}

// GetAllValidators 获取所有验证者
func (mvs *MemoryValidatorStore) GetAllValidators() ([]*types.Validator, error) {
	validators := make([]*types.Validator, 0, len(mvs.validators))
	for _, validator := range mvs.validators {
		validators = append(validators, validator.Clone())
	}
	return validators, nil
}

// HasValidator 检查验证者是否存在
func (mvs *MemoryValidatorStore) HasValidator(address types.Address) bool {
	addrStr := address.String()
	_, exists := mvs.validators[addrStr]
	return exists
}

// DeleteValidator 删除验证者
func (mvs *MemoryValidatorStore) DeleteValidator(address types.Address) error {
	addrStr := address.String()
	delete(mvs.validators, addrStr)
	return nil
}

// UpdateValidatorStake 更新验证者权益
func (mvs *MemoryValidatorStore) UpdateValidatorStake(address types.Address, stake uint64) error {
	validator, err := mvs.GetValidator(address)
	if err != nil {
		return fmt.Errorf("validator not found: %v", err)
	}

	validator.Stake = stake
	return mvs.PutValidator(validator)
}

// Close 关闭存储
func (mvs *MemoryValidatorStore) Close() error {
	return nil
}
