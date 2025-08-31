package types

import (
	"encoding/json"
	"time"
)

// ValidatorStatus 验证者状态
type ValidatorStatus int

const (
	ValidatorActive ValidatorStatus = iota
	ValidatorInactive
	ValidatorJailed
)

// String 返回验证者状态的字符串表示
func (vs ValidatorStatus) String() string {
	switch vs {
	case ValidatorActive:
		return "active"
	case ValidatorInactive:
		return "inactive"
	case ValidatorJailed:
		return "jailed"
	default:
		return "unknown"
	}
}

// Validator 验证者结构
type Validator struct {
	Address    Address         `json:"address"`     // 验证者地址
	Stake      uint64          `json:"stake"`       // 质押数量
	Status     ValidatorStatus `json:"status"`      // 验证者状态
	LastBlock  uint64          `json:"last_block"`  // 最后出块高度
	TotalVotes uint64          `json:"total_votes"` // 总投票数
	CreatedAt  int64           `json:"created_at"`  // 创建时间
	UpdatedAt  int64           `json:"updated_at"`  // 最后更新时间
}

// NewValidator 创建新验证者
func NewValidator(address Address, stake uint64) *Validator {
	now := time.Now().Unix()
	return &Validator{
		Address:    address,
		Stake:      stake,
		Status:     ValidatorActive,
		LastBlock:  0,
		TotalVotes: stake, // 初始投票数等于质押数
		CreatedAt:  now,
		UpdatedAt:  now,
	}
}

// AddStake 增加质押
func (v *Validator) AddStake(amount uint64) {
	v.Stake += amount
	v.TotalVotes += amount
	v.UpdatedAt = time.Now().Unix()
}

// SubStake 减少质押
func (v *Validator) SubStake(amount uint64) bool {
	if v.Stake < amount {
		return false
	}
	v.Stake -= amount
	v.TotalVotes -= amount
	v.UpdatedAt = time.Now().Unix()
	return true
}

// SetStatus 设置验证者状态
func (v *Validator) SetStatus(status ValidatorStatus) {
	v.Status = status
	v.UpdatedAt = time.Now().Unix()
}

// UpdateLastBlock 更新最后出块高度
func (v *Validator) UpdateLastBlock(height uint64) {
	v.LastBlock = height
	v.UpdatedAt = time.Now().Unix()
}

// IsActive 检查验证者是否活跃
func (v *Validator) IsActive() bool {
	return v.Status == ValidatorActive
}

// CanValidate 检查是否可以参与验证
func (v *Validator) CanValidate() bool {
	return v.IsActive() && v.Stake > 0
}

// Clone 克隆验证者
func (v *Validator) Clone() *Validator {
	return &Validator{
		Address:    v.Address,
		Stake:      v.Stake,
		Status:     v.Status,
		LastBlock:  v.LastBlock,
		TotalVotes: v.TotalVotes,
		CreatedAt:  v.CreatedAt,
		UpdatedAt:  v.UpdatedAt,
	}
}

// Serialize 序列化验证者为JSON
func (v *Validator) Serialize() ([]byte, error) {
	return json.Marshal(v)
}

// Deserialize 从 JSON 反序列化验证者
func (v *Validator) Deserialize(data []byte) error {
	return json.Unmarshal(data, v)
}

// DeserializeValidator 从 JSON 创建验证者
func DeserializeValidator(data []byte) (*Validator, error) {
	var validator Validator
	err := json.Unmarshal(data, &validator)
	if err != nil {
		return nil, err
	}
	return &validator, nil
}

// Delegator 委托人结构
type Delegator struct {
	Address   Address `json:"address"`    // 委托人地址
	Validator Address `json:"validator"`  // 委托的验证者地址
	Stake     uint64  `json:"stake"`      // 委托数量
	CreatedAt int64   `json:"created_at"` // 创建时间
	UpdatedAt int64   `json:"updated_at"` // 最后更新时间
}

// NewDelegator 创建新委托人
func NewDelegator(address, validator Address, stake uint64) *Delegator {
	now := time.Now().Unix()
	return &Delegator{
		Address:   address,
		Validator: validator,
		Stake:     stake,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// AddStake 增加委托
func (d *Delegator) AddStake(amount uint64) {
	d.Stake += amount
	d.UpdatedAt = time.Now().Unix()
}

// SubStake 减少委托
func (d *Delegator) SubStake(amount uint64) bool {
	if d.Stake < amount {
		return false
	}
	d.Stake -= amount
	d.UpdatedAt = time.Now().Unix()
	return true
}

// Clone 克隆委托人
func (d *Delegator) Clone() *Delegator {
	return &Delegator{
		Address:   d.Address,
		Validator: d.Validator,
		Stake:     d.Stake,
		CreatedAt: d.CreatedAt,
		UpdatedAt: d.UpdatedAt,
	}
}

// Serialize 序列化委托人为JSON
func (d *Delegator) Serialize() ([]byte, error) {
	return json.Marshal(d)
}

// Deserialize 从 JSON 反序列化委托人
func (d *Delegator) Deserialize(data []byte) error {
	return json.Unmarshal(data, d)
}

// DeserializeDelegator 从 JSON 创建委托人
func DeserializeDelegator(data []byte) (*Delegator, error) {
	var delegator Delegator
	err := json.Unmarshal(data, &delegator)
	if err != nil {
		return nil, err
	}
	return &delegator, nil
}
