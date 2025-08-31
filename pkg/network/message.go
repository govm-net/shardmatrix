package network

import (
	"bytes"
	"encoding/gob"
	"time"

	"github.com/govm-net/shardmatrix/pkg/types"
)

// NetMessage 网络消息
type NetMessage struct {
	Topic string
	From  string
	Data  []byte
}

// Request 请求消息
type Request struct {
	Type      string    `json:"type"`
	Data      []byte    `json:"data"`
	Timestamp time.Time `json:"timestamp"`
}

// Response 响应消息
type Response struct {
	Type      string    `json:"type"`
	Data      []byte    `json:"data"`
	Error     string    `json:"error,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// Serialize 序列化请求
func (r *Request) Serialize() ([]byte, error) {
	var buf bytes.Buffer
	encoder := gob.NewEncoder(&buf)
	err := encoder.Encode(r)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Deserialize 反序列化请求
func (r *Request) Deserialize(data []byte) error {
	buf := bytes.NewBuffer(data)
	decoder := gob.NewDecoder(buf)
	return decoder.Decode(r)
}

// Serialize 序列化响应
func (resp *Response) Serialize() ([]byte, error) {
	var buf bytes.Buffer
	encoder := gob.NewEncoder(&buf)
	err := encoder.Encode(resp)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Deserialize 反序列化响应
func (resp *Response) Deserialize(data []byte) error {
	buf := bytes.NewBuffer(data)
	decoder := gob.NewDecoder(buf)
	return decoder.Decode(resp)
}

// DPOS 相关消息类型

// VoteMessage 投票消息
type VoteMessage struct {
	Voter      string `json:"voter"`       // 投票者地址
	Candidate  string `json:"candidate"`   // 候选人地址
	VoteAmount int64  `json:"vote_amount"` // 投票数量
	Round      uint64 `json:"round"`       // 投票轮次
}

// ValidatorRegistrationMessage 验证者注册消息
type ValidatorRegistrationMessage struct {
	Candidate   string `json:"candidate"`    // 候选人地址
	StakeAmount int64  `json:"stake_amount"` // 质押数量
	PublicKey   []byte `json:"public_key"`   // 公钥
}

// BlockProposalMessage 区块提议消息
type BlockProposalMessage struct {
	Validator string       `json:"validator"` // 验证者地址
	Round     uint64       `json:"round"`     // 轮次
	Slot      int          `json:"slot"`      // 槽位
	Block     *types.Block `json:"block"`     // 区块
}

// BlockConfirmationMessage 区块确认消息
type BlockConfirmationMessage struct {
	Validator string     `json:"validator"`  // 验证者地址
	BlockHash types.Hash `json:"block_hash"` // 区块哈希
	Round     uint64     `json:"round"`      // 轮次
	Slot      int        `json:"slot"`       // 槽位
}

// ValidatorSetMessage 验证者集合消息
type ValidatorSetMessage struct {
	Round      uint64   `json:"round"`      // 轮次
	Validators []string `json:"validators"` // 验证者列表
	Weights    []int64  `json:"weights"`    // 权重列表
}

// RoundChangeMessage 轮次变更消息
type RoundChangeMessage struct {
	OldRound uint64 `json:"old_round"` // 旧轮次
	NewRound uint64 `json:"new_round"` // 新轮次
	Height   uint64 `json:"height"`    // 区块高度
}

// Serialize 序列化投票消息
func (vm *VoteMessage) Serialize() ([]byte, error) {
	var buf bytes.Buffer
	encoder := gob.NewEncoder(&buf)
	err := encoder.Encode(vm)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Deserialize 反序列化投票消息
func (vm *VoteMessage) Deserialize(data []byte) error {
	buf := bytes.NewBuffer(data)
	decoder := gob.NewDecoder(buf)
	return decoder.Decode(vm)
}

// Serialize 序列化验证者注册消息
func (vrm *ValidatorRegistrationMessage) Serialize() ([]byte, error) {
	var buf bytes.Buffer
	encoder := gob.NewEncoder(&buf)
	err := encoder.Encode(vrm)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Deserialize 反序列化验证者注册消息
func (vrm *ValidatorRegistrationMessage) Deserialize(data []byte) error {
	buf := bytes.NewBuffer(data)
	decoder := gob.NewDecoder(buf)
	return decoder.Decode(vrm)
}

// Serialize 序列化区块提议消息
func (bpm *BlockProposalMessage) Serialize() ([]byte, error) {
	var buf bytes.Buffer
	encoder := gob.NewEncoder(&buf)
	err := encoder.Encode(bpm)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Deserialize 反序列化区块提议消息
func (bpm *BlockProposalMessage) Deserialize(data []byte) error {
	buf := bytes.NewBuffer(data)
	decoder := gob.NewDecoder(buf)
	return decoder.Decode(bpm)
}

// Serialize 序列化区块确认消息
func (bcm *BlockConfirmationMessage) Serialize() ([]byte, error) {
	var buf bytes.Buffer
	encoder := gob.NewEncoder(&buf)
	err := encoder.Encode(bcm)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Deserialize 反序列化区块确认消息
func (bcm *BlockConfirmationMessage) Deserialize(data []byte) error {
	buf := bytes.NewBuffer(data)
	decoder := gob.NewDecoder(buf)
	return decoder.Decode(bcm)
}

// Serialize 序列化验证者集合消息
func (vsm *ValidatorSetMessage) Serialize() ([]byte, error) {
	var buf bytes.Buffer
	encoder := gob.NewEncoder(&buf)
	err := encoder.Encode(vsm)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Deserialize 反序列化验证者集合消息
func (vsm *ValidatorSetMessage) Deserialize(data []byte) error {
	buf := bytes.NewBuffer(data)
	decoder := gob.NewDecoder(buf)
	return decoder.Decode(vsm)
}

// Serialize 序列化轮次变更消息
func (rcm *RoundChangeMessage) Serialize() ([]byte, error) {
	var buf bytes.Buffer
	encoder := gob.NewEncoder(&buf)
	err := encoder.Encode(rcm)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Deserialize 反序列化轮次变更消息
func (rcm *RoundChangeMessage) Deserialize(data []byte) error {
	buf := bytes.NewBuffer(data)
	decoder := gob.NewDecoder(buf)
	return decoder.Decode(rcm)
}
