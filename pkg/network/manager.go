package network

import (
	"fmt"
	"sync"
	"time"

	"github.com/govm-net/shardmatrix/pkg/types"
	"github.com/libp2p/go-libp2p/core/peer"
)

// NetworkManager 网络管理器，整合所有网络功能
type NetworkManager struct {
	p2p       *LibP2PNetwork
	broadcast *BroadcastManager
	config    *LibP2PConfig

	// 状态
	isRunning bool
	mutex     sync.RWMutex

	// 回调函数
	onPeerConnected    func(peerID peer.ID)
	onPeerDisconnected func(peerID peer.ID)
	onBlockReceived    func(block *types.Block, fromPeer peer.ID)
	onTxReceived       func(tx *types.Transaction, fromPeer peer.ID)
}

// NewNetworkManager 创建网络管理器
func NewNetworkManager(config *LibP2PConfig) (*NetworkManager, error) {
	// 创建P2P网络
	p2p, err := NewLibP2PNetwork(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create libp2p network: %v", err)
	}

	// 创建广播管理器
	broadcast := NewBroadcastManager(p2p)

	nm := &NetworkManager{
		p2p:       p2p,
		broadcast: broadcast,
		config:    config,
		isRunning: false,
	}

	// 设置P2P回调
	p2p.SetCallbacks(nm.handlePeerConnected, nm.handlePeerDisconnected, nm.handleBlockReceived, nm.handleTxReceived, nm.handleMessage)

	// 设置广播处理器
	broadcast.SetBlockHandler(nm)
	broadcast.SetTransactionHandler(nm)

	return nm, nil
}

// Start 启动网络管理器
func (nm *NetworkManager) Start() error {
	nm.mutex.Lock()
	defer nm.mutex.Unlock()

	if nm.isRunning {
		return fmt.Errorf("network manager is already running")
	}

	// 启动P2P网络
	if err := nm.p2p.Start(); err != nil {
		return fmt.Errorf("failed to start P2P network: %v", err)
	}

	nm.isRunning = true
	return nil
}

// Stop 停止网络管理器
func (nm *NetworkManager) Stop() error {
	nm.mutex.Lock()
	defer nm.mutex.Unlock()

	if !nm.isRunning {
		return nil
	}

	// 停止P2P网络
	if err := nm.p2p.Stop(); err != nil {
		return fmt.Errorf("failed to stop P2P network: %v", err)
	}

	nm.isRunning = false
	return nil
}

// SetCallbacks 设置回调函数
func (nm *NetworkManager) SetCallbacks(
	onPeerConnected func(peer.ID),
	onPeerDisconnected func(peer.ID),
	onBlockReceived func(*types.Block, peer.ID),
	onTxReceived func(*types.Transaction, peer.ID),
) {
	nm.onPeerConnected = onPeerConnected
	nm.onPeerDisconnected = onPeerDisconnected
	nm.onBlockReceived = onBlockReceived
	nm.onTxReceived = onTxReceived
}

// ConnectToPeer 连接到指定节点
func (nm *NetworkManager) ConnectToPeer(multiAddr string) error {
	// TODO: 实现libp2p连接
	return fmt.Errorf("libp2p connect not implemented yet")
}

// BroadcastBlock 广播区块
func (nm *NetworkManager) BroadcastBlock(block *types.Block) error {
	return nm.p2p.BroadcastBlock(block)
}

// BroadcastTransaction 广播交易
func (nm *NetworkManager) BroadcastTransaction(tx *types.Transaction) error {
	return nm.p2p.BroadcastTransaction(tx)
}

// RequestBlock 请求区块
func (nm *NetworkManager) RequestBlock(peerID peer.ID, blockHash types.Hash) error {
	// TODO: 实现libp2p区块请求
	return fmt.Errorf("libp2p block request not implemented yet")
}

// RequestTransaction 请求交易
func (nm *NetworkManager) RequestTransaction(peerID peer.ID, txHash types.Hash) error {
	// TODO: 实现libp2p交易请求
	return fmt.Errorf("libp2p transaction request not implemented yet")
}

// GetPeers 获取所有节点信息
func (nm *NetworkManager) GetPeers() []peer.ID {
	return nm.p2p.GetConnectedPeers()
}

// GetActivePeers 获取活跃节点
func (nm *NetworkManager) GetActivePeers() []peer.ID {
	return nm.p2p.GetConnectedPeers()
}

// GetPeerCount 获取节点数量
func (nm *NetworkManager) GetPeerCount() int {
	return nm.p2p.GetPeerCount()
}

// GetActivePeerCount 获取活跃节点数量
func (nm *NetworkManager) GetActivePeerCount() int {
	return nm.p2p.GetPeerCount()
}

// IsRunning 检查是否运行中
func (nm *NetworkManager) IsRunning() bool {
	nm.mutex.RLock()
	defer nm.mutex.RUnlock()
	return nm.isRunning
}

// GetConfig 获取网络配置
func (nm *NetworkManager) GetConfig() *LibP2PConfig {
	return nm.config
}

// GetNodeID 获取节点ID
func (nm *NetworkManager) GetNodeID() peer.ID {
	return nm.p2p.GetHostID()
}

// GetListenAddress 获取监听地址
func (nm *NetworkManager) GetListenAddress() []string {
	addrs := nm.p2p.GetListenAddrs()
	strAddrs := make([]string, len(addrs))
	for i, addr := range addrs {
		strAddrs[i] = addr.String()
	}
	return strAddrs
}

// 内部处理方法

// handlePeerConnected 处理节点连接
func (nm *NetworkManager) handlePeerConnected(peerID peer.ID) {
	fmt.Printf("Peer connected: %s\n", peerID.String())

	if nm.onPeerConnected != nil {
		nm.onPeerConnected(peerID)
	}
}

// handlePeerDisconnected 处理节点断开
func (nm *NetworkManager) handlePeerDisconnected(peerID peer.ID) {
	fmt.Printf("Peer disconnected: %s\n", peerID.String())

	if nm.onPeerDisconnected != nil {
		nm.onPeerDisconnected(peerID)
	}
}

// handleBlockReceived 处理接收到的区块
func (nm *NetworkManager) handleBlockReceived(block *types.Block, peerID peer.ID) {
	fmt.Printf("Received block %s from peer %s\n", block.Hash().String(), peerID.String())

	if nm.onBlockReceived != nil {
		nm.onBlockReceived(block, peerID)
	}
}

// handleTxReceived 处理接收到的交易
func (nm *NetworkManager) handleTxReceived(tx *types.Transaction, peerID peer.ID) {
	fmt.Printf("Received transaction %s from peer %s\n", tx.Hash().String(), peerID.String())

	if nm.onTxReceived != nil {
		nm.onTxReceived(tx, peerID)
	}
}

// handleMessage 处理消息
func (nm *NetworkManager) handleMessage(msg *Message, peerID peer.ID) {
	// 这里可以添加通用的消息处理逻辑
	// 具体的区块和交易消息由BroadcastManager处理
}

// 实现BlockMessageHandler接口

// HandleNewBlock 处理新区块
func (nm *NetworkManager) HandleNewBlock(block *types.Block, fromPeer peer.ID) error {
	fmt.Printf("Received new block %s from peer %s\n", block.Hash().String(), fromPeer.String())

	if nm.onBlockReceived != nil {
		nm.onBlockReceived(block, fromPeer)
	}

	return nil
}

// HandleReceivedBlock 处理接收到的区块
func (nm *NetworkManager) HandleReceivedBlock(block *types.Block, fromPeer peer.ID) error {
	fmt.Printf("Received block response %s from peer %s\n", block.Hash().String(), fromPeer.String())

	if nm.onBlockReceived != nil {
		nm.onBlockReceived(block, fromPeer)
	}

	return nil
}

// GetBlock 获取区块（这里需要外部提供区块存储）
func (nm *NetworkManager) GetBlock(blockHash types.Hash) (*types.Block, error) {
	// TODO: 这里需要集成区块存储，暂时返回nil
	return nil, fmt.Errorf("block storage not implemented")
}

// 实现TransactionMessageHandler接口

// HandleNewTransaction 处理新交易
func (nm *NetworkManager) HandleNewTransaction(tx *types.Transaction, fromPeer peer.ID) error {
	fmt.Printf("Received new transaction %s from peer %s\n", tx.Hash().String(), fromPeer.String())

	if nm.onTxReceived != nil {
		nm.onTxReceived(tx, fromPeer)
	}

	return nil
}

// HandleReceivedTransaction 处理接收到的交易
func (nm *NetworkManager) HandleReceivedTransaction(tx *types.Transaction, fromPeer peer.ID) error {
	fmt.Printf("Received transaction response %s from peer %s\n", tx.Hash().String(), fromPeer.String())

	if nm.onTxReceived != nil {
		nm.onTxReceived(tx, fromPeer)
	}

	return nil
}

// GetTransaction 获取交易（这里需要外部提供交易存储）
func (nm *NetworkManager) GetTransaction(txHash types.Hash) (*types.Transaction, error) {
	// TODO: 这里需要集成交易存储，暂时返回nil
	return nil, fmt.Errorf("transaction storage not implemented")
}

// 网络统计和监控

// NetworkStats 网络统计信息
type NetworkStats struct {
	TotalPeers       int           `json:"total_peers"`
	ActivePeers      int           `json:"active_peers"`
	ConnectedPeers   int           `json:"connected_peers"`
	UpTime           time.Duration `json:"uptime"`
	BytesSent        uint64        `json:"bytes_sent"`
	BytesReceived    uint64        `json:"bytes_received"`
	MessagesSent     uint64        `json:"messages_sent"`
	MessagesReceived uint64        `json:"messages_received"`
}

// GetNetworkStats 获取网络统计信息
func (nm *NetworkManager) GetNetworkStats() *NetworkStats {
	// TODO: 实现详细的网络统计
	return &NetworkStats{
		TotalPeers:     nm.GetPeerCount(),
		ActivePeers:    nm.GetActivePeerCount(),
		ConnectedPeers: nm.GetActivePeerCount(), // 暂时等同于活跃节点
	}
}

// 网络健康检查

// HealthCheck 网络健康检查
func (nm *NetworkManager) HealthCheck() map[string]interface{} {
	health := make(map[string]interface{})

	health["is_running"] = nm.IsRunning()
	health["node_id"] = nm.GetNodeID()
	health["listen_address"] = nm.GetListenAddress()
	health["peer_count"] = nm.GetPeerCount()
	health["active_peer_count"] = nm.GetActivePeerCount()

	// 检查连接状态
	activePeers := nm.GetActivePeers()
	peerStatus := make([]map[string]interface{}, len(activePeers))
	for i, peerID := range activePeers {
		peerStatus[i] = map[string]interface{}{
			"id":        peerID.String(),
			"connected": nm.p2p.IsConnectedToPeer(peerID),
		}
	}
	health["peers"] = peerStatus

	return health
}
