package network

import (
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/govm-net/shardmatrix/pkg/types"
)

// MessageType 消息类型
type MessageType string

const (
	// 基础消息类型
	MessageTypeHandshake  MessageType = "handshake"
	MessageTypePing       MessageType = "ping"
	MessageTypePong       MessageType = "pong"
	MessageTypeDisconnect MessageType = "disconnect"

	// 区块链消息类型
	MessageTypeNewBlock      MessageType = "new_block"
	MessageTypeNewTx         MessageType = "new_tx"
	MessageTypeBlockRequest  MessageType = "block_request"
	MessageTypeBlockResponse MessageType = "block_response"
	MessageTypeTxRequest     MessageType = "tx_request"
	MessageTypeTxResponse    MessageType = "tx_response"

	// 同步消息类型
	MessageTypeSyncRequest  MessageType = "sync_request"
	MessageTypeSyncResponse MessageType = "sync_response"
)

// Message 网络消息结构
type Message struct {
	Type      MessageType `json:"type"`
	From      string      `json:"from"`      // 发送者节点ID
	To        string      `json:"to"`        // 接收者节点ID（空表示广播）
	Timestamp int64       `json:"timestamp"` // 时间戳
	Data      []byte      `json:"data"`      // 消息数据
	Signature []byte      `json:"signature"` // 消息签名
}

// NewMessage 创建新消息
func NewMessage(msgType MessageType, from, to string, data []byte) *Message {
	return &Message{
		Type:      msgType,
		From:      from,
		To:        to,
		Timestamp: time.Now().Unix(),
		Data:      data,
	}
}

// Serialize 序列化消息
func (m *Message) Serialize() ([]byte, error) {
	return json.Marshal(m)
}

// DeserializeMessage 反序列化消息
func DeserializeMessage(data []byte) (*Message, error) {
	var msg Message
	err := json.Unmarshal(data, &msg)
	return &msg, err
}

// Peer 代表网络中的一个对等节点
type Peer struct {
	ID       string   `json:"id"`
	Address  string   `json:"address"`
	Port     int      `json:"port"`
	Conn     net.Conn `json:"-"` // 连接对象，不参与序列化
	LastSeen int64    `json:"last_seen"`
	IsActive bool     `json:"is_active"`
	Version  string   `json:"version"`
}

// NewPeer 创建新的对等节点
func NewPeer(id, address string, port int) *Peer {
	return &Peer{
		ID:       id,
		Address:  address,
		Port:     port,
		LastSeen: time.Now().Unix(),
		IsActive: false,
		Version:  "1.0.0",
	}
}

// String 返回节点的字符串表示
func (p *Peer) String() string {
	return fmt.Sprintf("%s@%s:%d", p.ID, p.Address, p.Port)
}

// FullAddress 返回完整的地址
func (p *Peer) FullAddress() string {
	return fmt.Sprintf("%s:%d", p.Address, p.Port)
}

// IsConnected 检查是否已连接
func (p *Peer) IsConnected() bool {
	return p.Conn != nil && p.IsActive
}

// UpdateLastSeen 更新最后活跃时间
func (p *Peer) UpdateLastSeen() {
	p.LastSeen = time.Now().Unix()
}

// MessageHandler 消息处理器接口
type MessageHandler interface {
	HandleMessage(msg *Message, peer *Peer) error
}

// PeerManager 管理对等节点
type PeerManager struct {
	peers     map[string]*Peer // 节点ID -> 节点
	addresses map[string]*Peer // 地址 -> 节点
	mutex     sync.RWMutex
	maxPeers  int
}

// NewPeerManager 创建节点管理器
func NewPeerManager(maxPeers int) *PeerManager {
	return &PeerManager{
		peers:     make(map[string]*Peer),
		addresses: make(map[string]*Peer),
		maxPeers:  maxPeers,
	}
}

// AddPeer 添加节点
func (pm *PeerManager) AddPeer(peer *Peer) error {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	if len(pm.peers) >= pm.maxPeers {
		return fmt.Errorf("reached maximum peers limit: %d", pm.maxPeers)
	}

	// 检查是否已存在
	if existing, exists := pm.peers[peer.ID]; exists {
		if existing.IsActive {
			return fmt.Errorf("peer %s already exists and is active", peer.ID)
		}
		// 更新现有节点
		existing.Address = peer.Address
		existing.Port = peer.Port
		existing.UpdateLastSeen()
		return nil
	}

	pm.peers[peer.ID] = peer
	pm.addresses[peer.FullAddress()] = peer
	return nil
}

// RemovePeer 移除节点
func (pm *PeerManager) RemovePeer(peerID string) {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	if peer, exists := pm.peers[peerID]; exists {
		// 关闭连接
		if peer.Conn != nil {
			peer.Conn.Close()
		}

		delete(pm.peers, peerID)
		delete(pm.addresses, peer.FullAddress())
	}
}

// GetPeer 获取节点
func (pm *PeerManager) GetPeer(peerID string) (*Peer, bool) {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	peer, exists := pm.peers[peerID]
	return peer, exists
}

// GetPeerByAddress 通过地址获取节点
func (pm *PeerManager) GetPeerByAddress(address string) (*Peer, bool) {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	peer, exists := pm.addresses[address]
	return peer, exists
}

// GetActivePeers 获取所有活跃节点
func (pm *PeerManager) GetActivePeers() []*Peer {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	var activePeers []*Peer
	for _, peer := range pm.peers {
		if peer.IsActive && peer.IsConnected() {
			activePeers = append(activePeers, peer)
		}
	}
	return activePeers
}

// GetAllPeers 获取所有节点
func (pm *PeerManager) GetAllPeers() []*Peer {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	var allPeers []*Peer
	for _, peer := range pm.peers {
		allPeers = append(allPeers, peer)
	}
	return allPeers
}

// PeerCount 获取节点数量
func (pm *PeerManager) PeerCount() int {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	return len(pm.peers)
}

// ActivePeerCount 获取活跃节点数量
func (pm *PeerManager) ActivePeerCount() int {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	count := 0
	for _, peer := range pm.peers {
		if peer.IsActive && peer.IsConnected() {
			count++
		}
	}
	return count
}

// UpdatePeerStatus 更新节点状态
func (pm *PeerManager) UpdatePeerStatus(peerID string, isActive bool) {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	if peer, exists := pm.peers[peerID]; exists {
		peer.IsActive = isActive
		peer.UpdateLastSeen()
	}
}

// CleanupInactivePeers 清理不活跃的节点
func (pm *PeerManager) CleanupInactivePeers(timeout time.Duration) {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	now := time.Now().Unix()
	var toRemove []string

	for id, peer := range pm.peers {
		if !peer.IsActive || (now-peer.LastSeen) > int64(timeout.Seconds()) {
			toRemove = append(toRemove, id)
		}
	}

	for _, id := range toRemove {
		if peer := pm.peers[id]; peer != nil {
			if peer.Conn != nil {
				peer.Conn.Close()
			}
			delete(pm.peers, id)
			delete(pm.addresses, peer.FullAddress())
		}
	}
}

// HandshakeData 握手数据
type HandshakeData struct {
	NodeID     string     `json:"node_id"`
	Version    string     `json:"version"`
	ChainID    uint64     `json:"chain_id"`
	BestHeight uint64     `json:"best_height"`
	BestHash   types.Hash `json:"best_hash"`
	Timestamp  int64      `json:"timestamp"`
	ListenPort int        `json:"listen_port"`
}

// NewHandshakeData 创建握手数据
func NewHandshakeData(nodeID, version string, chainID, bestHeight uint64, bestHash types.Hash, listenPort int) *HandshakeData {
	return &HandshakeData{
		NodeID:     nodeID,
		Version:    version,
		ChainID:    chainID,
		BestHeight: bestHeight,
		BestHash:   bestHash,
		Timestamp:  time.Now().Unix(),
		ListenPort: listenPort,
	}
}

// Serialize 序列化握手数据
func (hd *HandshakeData) Serialize() ([]byte, error) {
	return json.Marshal(hd)
}

// DeserializeHandshakeData 反序列化握手数据
func DeserializeHandshakeData(data []byte) (*HandshakeData, error) {
	var hd HandshakeData
	err := json.Unmarshal(data, &hd)
	return &hd, err
}
