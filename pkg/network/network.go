package network

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/lengzhao/shardmatrix/pkg/types"
)

// MessageType 消息类型
type MessageType uint8

const (
	MsgNewBlock MessageType = iota
	MsgNewTransaction
	MsgBlockRequest
	MsgBlockResponse
	MsgPeerDiscovery
	MsgHeartbeat
	MsgPing
	MsgPong
	MsgValidatorUpdate
	MsgEmptyBlock
	MsgSyncRequest
	MsgSyncResponse
)

// MessagePriority 消息优先级
type MessagePriority uint8

const (
	PriorityHigh   MessagePriority = iota // 高优先级（立即处理）
	PriorityMedium                         // 中优先级（批量处理）
	PriorityLow                            // 低优先级（空闲处理）
)

// GetMessagePriority 获取消息类型对应的优先级
func GetMessagePriority(msgType MessageType) MessagePriority {
	switch msgType {
	case MsgNewBlock, MsgValidatorUpdate, MsgEmptyBlock:
		return PriorityHigh // 区块和验证者消息高优先级
	case MsgNewTransaction, MsgHeartbeat, MsgPing, MsgPong:
		return PriorityMedium // 交易和心跳中优先级
	default:
		return PriorityLow // 其他消息低优先级
	}
}

// Message 网络消息结构
type Message struct {
	Type      MessageType     `json:"type"`
	Priority  MessagePriority `json:"priority"`  // 新增：消息优先级
	Data      []byte          `json:"data"`
	Timestamp int64           `json:"timestamp"`
	NodeID    string          `json:"node_id"`
	MessageID string          `json:"message_id"` // 消息唯一ID，用于去重
	TTL       int             `json:"ttl"`        // 生存时间，防止无限传播
	Size      int             `json:"size"`       // 消息大小
	Retries   int             `json:"retries"`    // 重试次数
}

// Peer 网络节点
type Peer struct {
	ID           string        `json:"id"`
	Address      string        `json:"address"`
	LastSeen     time.Time     `json:"last_seen"`
	Conn         net.Conn      `json:"-"`
	IsHealthy    bool          `json:"is_healthy"`    // 节点健康状态
	Latency      time.Duration `json:"latency"`       // 网络延迟
	FailCount    int           `json:"fail_count"`    // 连续失败次数
	LastPing     time.Time     `json:"last_ping"`     // 上次ping时间
	BytesSent    uint64        `json:"bytes_sent"`    // 发送字节数
	BytesRecv    uint64        `json:"bytes_recv"`    // 接收字节数
	MessagesSent uint64        `json:"messages_sent"` // 发送消息数
	MessagesRecv uint64        `json:"messages_recv"` // 接收消息数
}

// NetworkStats 网络统计信息
type NetworkStats struct {
	TotalMessagesSent     uint64    `json:"total_messages_sent"`
	TotalMessagesReceived uint64    `json:"total_messages_received"`
	TotalBytesSent        uint64    `json:"total_bytes_sent"`
	TotalBytesReceived    uint64    `json:"total_bytes_received"`
	DuplicateMessages     uint64    `json:"duplicate_messages"`
	FailedSends           uint64    `json:"failed_sends"`
	ActiveConnections     int       `json:"active_connections"`
	StartTime             time.Time `json:"start_time"`
	// 新增：优先级统计
	HighPriorityMessages  uint64    `json:"high_priority_messages"`
	MediumPriorityMessages uint64   `json:"medium_priority_messages"`
	LowPriorityMessages   uint64    `json:"low_priority_messages"`
	DroppedMessages       uint64    `json:"dropped_messages"`
	CongestionEvents      uint64    `json:"congestion_events"`
}

// PriorityQueue 优先级消息队列
type PriorityQueue struct {
	mu           sync.RWMutex
	highPriority chan *Message // 高优先级队列
	mediumPriority chan *Message // 中优先级队列
	lowPriority  chan *Message // 低优先级队列
	maxSize      int           // 最大队列大小
	dropCount    uint64        // 丢弃消息数
	isRunning    bool          // 是否运行中
	stopCh       chan struct{} // 停止信号
}

// CongestionController 拥塞控制器
type CongestionController struct {
	mu               sync.RWMutex
	congestionLevel  float64           // 拥塞级别(0-1)
	windowSize       int               // 滑动窗口大小
	lastUpdate       time.Time         // 最后更新时间
	throughputHistory []float64         // 吞吐量历史
	latencyHistory   []time.Duration   // 延迟历史
	adaptiveMode     bool              // 自适应模式
	thresholds       CongestionThresholds // 拥塞阈值
}

// CongestionThresholds 拥塞控制阈值
type CongestionThresholds struct {
	LatencyThreshold  time.Duration // 延迟阈值
	ThroughputMin     float64       // 最小吞吐量
	QueueSizeMax      int           // 最大队列大小
	DropRateMax       float64       // 最大丢包率
}

// BandwidthLimiter 带宽限制器
type BandwidthLimiter struct {
	mu            sync.RWMutex
	limit         int64     // 带宽限制(bytes/s)
	used          int64     // 已使用带宽
	lastReset     time.Time // 最后重置时间
	bucketSize    int64     // 令牌桶大小
	tokens        int64     // 当前令牌数
	bytesPerToken int64     // 每个令牌代表的字节数
}

// Network P2P网络管理器
type Network struct {
	mu          sync.RWMutex
	nodeID      string
	listenAddr  string
	peers       map[string]*Peer
	listener    net.Listener
	isRunning   bool
	stopCh      chan struct{}
	msgHandlers map[MessageType]MessageHandler
	
	// 新增：优先级消息处理
	priorityQueue    *PriorityQueue
	congestionCtrl   *CongestionController
	bandwidthLimiter *BandwidthLimiter
	
	// 消息去重和路由优化
	messageCache map[string]time.Time // 消息缓存，用于去重
	cacheExpiry  time.Duration        // 缓存过期时间
	maxCacheSize int                  // 最大缓存数量

	// 健康检查
	healthCheckInterval time.Duration
	maxFailCount        int

	// 统计信息
	stats NetworkStats
}

// MessageHandler 消息处理器接口
type MessageHandler func(msg *Message, peer *Peer) error

// NewNetwork 创建新的网络管理器
func NewNetwork(nodeID, listenAddr string) *Network {
	// 创建优先级队列
	priorityQueue := &PriorityQueue{
		highPriority:   make(chan *Message, 1000),
		mediumPriority: make(chan *Message, 5000),
		lowPriority:    make(chan *Message, 10000),
		maxSize:        16000,
		stopCh:         make(chan struct{}),
	}

	// 创建拥塞控制器
	congestionCtrl := &CongestionController{
		windowSize:       100,
		throughputHistory: make([]float64, 100),
		latencyHistory:   make([]time.Duration, 100),
		adaptiveMode:     true,
		thresholds: CongestionThresholds{
			LatencyThreshold: 100 * time.Millisecond,
			ThroughputMin:    1024 * 1024, // 1MB/s
			QueueSizeMax:     15000,
			DropRateMax:      0.05, // 5%
		},
	}

	// 创建带宽限制器
	bandwidthLimiter := &BandwidthLimiter{
		limit:         10 * 1024 * 1024, // 10MB/s
		bucketSize:    10 * 1024 * 1024,
		tokens:        10 * 1024 * 1024,
		bytesPerToken: 1,
		lastReset:     time.Now(),
	}

	return &Network{
		nodeID:              nodeID,
		listenAddr:          listenAddr,
		peers:               make(map[string]*Peer),
		stopCh:              make(chan struct{}),
		msgHandlers:         make(map[MessageType]MessageHandler),
		priorityQueue:       priorityQueue,
		congestionCtrl:      congestionCtrl,
		bandwidthLimiter:    bandwidthLimiter,
		messageCache:        make(map[string]time.Time),
		cacheExpiry:         time.Minute * 5, // 5分钟过期
		maxCacheSize:        10000,
		healthCheckInterval: time.Second * 30, // 30秒健康检查
		maxFailCount:        3,                // 最大失败3次
		stats: NetworkStats{
			StartTime: time.Now(),
		},
	}
}

// generateMessageID 生成消息唯一ID
func (n *Network) generateMessageID(msg *Message) string {
	data := fmt.Sprintf("%s-%d-%s-%d", msg.NodeID, msg.Type, string(msg.Data), msg.Timestamp)
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

// isDuplicateMessage 检查消息是否重复
func (n *Network) isDuplicateMessage(messageID string) bool {
	n.mu.Lock()
	defer n.mu.Unlock()

	// 清理过期缓存
	n.cleanExpiredCache()

	if _, exists := n.messageCache[messageID]; exists {
		n.stats.DuplicateMessages++
		return true
	}

	// 检查缓存大小限制
	if len(n.messageCache) >= n.maxCacheSize {
		// 简单的缓存清理策略：清空一半
		for k := range n.messageCache {
			delete(n.messageCache, k)
			if len(n.messageCache) <= n.maxCacheSize/2 {
				break
			}
		}
	}

	n.messageCache[messageID] = time.Now()
	return false
}

// cleanExpiredCache 清理过期的缓存条目
func (n *Network) cleanExpiredCache() {
	now := time.Now()
	for messageID, timestamp := range n.messageCache {
		if now.Sub(timestamp) > n.cacheExpiry {
			delete(n.messageCache, messageID)
		}
	}
}

// Start 启动网络服务
func (n *Network) Start() error {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.isRunning {
		return fmt.Errorf("network already running")
	}

	// 启动监听
	listener, err := net.Listen("tcp", n.listenAddr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", n.listenAddr, err)
	}

	n.listener = listener
	n.isRunning = true
	n.stopCh = make(chan struct{})
	n.stats.StartTime = time.Now()

	// 启动优先级队列
	n.priorityQueue.isRunning = true
	n.priorityQueue.stopCh = make(chan struct{})

	// 启动各种协程
	go n.acceptLoop()
	go n.messageLoop()
	go n.healthCheckLoop()
	go n.cacheCleanupLoop()
	go n.congestionMonitorLoop() // 新增：拥塞监控

	return nil
}

// Stop 停止网络服务
func (n *Network) Stop() {
	n.mu.Lock()
	defer n.mu.Unlock()

	if !n.isRunning {
		return
	}

	n.isRunning = false
	close(n.stopCh)

	// 停止优先级队列
	n.priorityQueue.isRunning = false
	close(n.priorityQueue.stopCh)

	if n.listener != nil {
		n.listener.Close()
	}

	// 关闭所有连接
	for _, peer := range n.peers {
		if peer.Conn != nil {
			peer.Conn.Close()
		}
	}
}

// RegisterHandler 注册消息处理器
func (n *Network) RegisterHandler(msgType MessageType, handler MessageHandler) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.msgHandlers[msgType] = handler
}

// BroadcastBlock 广播新区块
func (n *Network) BroadcastBlock(block *types.Block) error {
	data, err := json.Marshal(block)
	if err != nil {
		return fmt.Errorf("failed to marshal block: %w", err)
	}

	msg := &Message{
		Type:      MsgNewBlock,
		Data:      data,
		Timestamp: time.Now().Unix(),
		NodeID:    n.nodeID,
		TTL:       10, // 最多传播10跳
	}

	msg.MessageID = n.generateMessageID(msg)
	return n.broadcast(msg)
}

// BroadcastTransaction 广播新交易
func (n *Network) BroadcastTransaction(tx *types.Transaction) error {
	data, err := json.Marshal(tx)
	if err != nil {
		return fmt.Errorf("failed to marshal transaction: %w", err)
	}

	msg := &Message{
		Type:      MsgNewTransaction,
		Data:      data,
		Timestamp: time.Now().Unix(),
		NodeID:    n.nodeID,
		TTL:       5, // 交易传播范围稍小
	}

	msg.MessageID = n.generateMessageID(msg)
	return n.broadcast(msg)
}

// ConnectToPeer 连接到其他节点
func (n *Network) ConnectToPeer(address string) error {
	n.mu.Lock()
	defer n.mu.Unlock()

	// 检查是否已连接
	for _, peer := range n.peers {
		if peer.Address == address {
			return fmt.Errorf("already connected to %s", address)
		}
	}

	// 建立连接
	conn, err := net.DialTimeout("tcp", address, 10*time.Second)
	if err != nil {
		return fmt.Errorf("failed to connect to %s: %w", address, err)
	}

	peer := &Peer{
		ID:        fmt.Sprintf("peer_%s", address),
		Address:   address,
		LastSeen:  time.Now(),
		Conn:      conn,
		IsHealthy: true,
		FailCount: 0,
		LastPing:  time.Now(),
	}

	n.peers[peer.ID] = peer
	n.stats.ActiveConnections++

	// 启动处理该连接的协程
	go n.handleConnection(peer)

	return nil
}

// acceptLoop 接受新连接的循环
func (n *Network) acceptLoop() {
	for {
		select {
		case <-n.stopCh:
			return
		default:
		}

		conn, err := n.listener.Accept()
		if err != nil {
			if n.isRunning {
				fmt.Printf("Accept error: %v\n", err)
			}
			continue
		}

		peer := &Peer{
			ID:        fmt.Sprintf("peer_%s", conn.RemoteAddr().String()),
			Address:   conn.RemoteAddr().String(),
			LastSeen:  time.Now(),
			Conn:      conn,
			IsHealthy: true,
			FailCount: 0,
			LastPing:  time.Now(),
		}

		n.mu.Lock()
		n.peers[peer.ID] = peer
		n.stats.ActiveConnections++
		n.mu.Unlock()

		go n.handleConnection(peer)
	}
}

// handleConnection 处理单个连接
func (n *Network) handleConnection(peer *Peer) {
	defer func() {
		peer.Conn.Close()
		n.mu.Lock()
		delete(n.peers, peer.ID)
		n.stats.ActiveConnections--
		n.mu.Unlock()
	}()

	decoder := json.NewDecoder(peer.Conn)

	for {
		select {
		case <-n.stopCh:
			return
		default:
		}

		var msg Message
		if err := decoder.Decode(&msg); err != nil {
			if n.isRunning {
				fmt.Printf("Failed to decode message from %s: %v\n", peer.ID, err)
				n.mu.Lock()
				peer.FailCount++
				if peer.FailCount >= n.maxFailCount {
					peer.IsHealthy = false
				}
				n.mu.Unlock()
			}
			return
		}

		peer.LastSeen = time.Now()
		peer.FailCount = 0 // 重置失败计数
		peer.IsHealthy = true
		peer.MessagesRecv++

		// 更新统计信息
		n.mu.Lock()
		n.stats.TotalMessagesReceived++
		n.stats.TotalBytesReceived += uint64(len(msg.Data))
		n.mu.Unlock()

		// 检查消息是否重复
		if n.isDuplicateMessage(msg.MessageID) {
			continue // 跳过重复消息
		}

		// 处理消息
		n.handleMessage(&msg, peer)

		// 转发消息（如果TTL > 0）
		if msg.TTL > 0 && msg.NodeID != n.nodeID {
			msg.TTL--
			n.forwardMessage(&msg, peer)
		}
	}
}

// handleMessage 处理接收到的消息
func (n *Network) handleMessage(msg *Message, peer *Peer) {
	n.mu.RLock()
	handler, exists := n.msgHandlers[msg.Type]
	n.mu.RUnlock()

	if exists {
		if err := handler(msg, peer); err != nil {
			fmt.Printf("Message handler error: %v\n", err)
		}
	} else {
		fmt.Printf("No handler for message type %d\n", msg.Type)
	}
}

// forwardMessage 转发消息到其他节点（除了发送方）
func (n *Network) forwardMessage(msg *Message, sender *Peer) {
	n.mu.RLock()
	defer n.mu.RUnlock()

	for _, peer := range n.peers {
		if peer.ID != sender.ID && peer.IsHealthy && peer.Conn != nil {
			go n.sendToPeer(msg, peer)
		}
	}
}

// messageLoop 消息发送循环
func (n *Network) messageLoop() {
	for {
		select {
		case <-n.stopCh:
			return
		case msg := <-n.priorityQueue.highPriority:
			n.sendToPeers(msg)
		case msg := <-n.priorityQueue.mediumPriority:
			// 检查拥塞情况，如果没有高优先级消息再处理
			select {
			case highMsg := <-n.priorityQueue.highPriority:
				// 先处理高优先级消息
				n.sendToPeers(highMsg)
				// 再放回中优先级消息
				select {
				case n.priorityQueue.mediumPriority <- msg:
				default:
					n.stats.DroppedMessages++
				}
			default:
				n.sendToPeers(msg)
			}
		case msg := <-n.priorityQueue.lowPriority:
			// 检查是否有更高优先级消息
			select {
			case highMsg := <-n.priorityQueue.highPriority:
				n.sendToPeers(highMsg)
				select {
				case n.priorityQueue.lowPriority <- msg:
				default:
					n.stats.DroppedMessages++
				}
			case mediumMsg := <-n.priorityQueue.mediumPriority:
				n.sendToPeers(mediumMsg)
				select {
				case n.priorityQueue.lowPriority <- msg:
				default:
					n.stats.DroppedMessages++
				}
			default:
				// 检查拥塞级别，决定是否处理低优先级消息
				if n.congestionCtrl.congestionLevel < 0.8 {
					n.sendToPeers(msg)
				} else {
					// 拥塞时丢弃低优先级消息
					n.stats.DroppedMessages++
				}
			}
		}
	}
}

// healthCheckLoop 健康检查循环
func (n *Network) healthCheckLoop() {
	ticker := time.NewTicker(n.healthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-n.stopCh:
			return
		case <-ticker.C:
			n.performHealthCheck()
		}
	}
}

// performHealthCheck 执行健康检查
func (n *Network) performHealthCheck() {
	n.mu.Lock()
	defer n.mu.Unlock()

	now := time.Now()
	for _, peer := range n.peers {
		// 检查最后通信时间
		if now.Sub(peer.LastSeen) > n.healthCheckInterval*2 {
			peer.IsHealthy = false
			peer.FailCount++

			// 发送ping消息
			n.sendPing(peer)
		}

		// 清理长时间不健康的节点
		if peer.FailCount >= n.maxFailCount*2 {
			if peer.Conn != nil {
				peer.Conn.Close()
			}
		}
	}
}

// sendPing 发送ping消息
func (n *Network) sendPing(peer *Peer) {
	pingMsg := &Message{
		Type:      MsgPing,
		Data:      []byte(fmt.Sprintf("{\"timestamp\":%d}", time.Now().Unix())),
		Timestamp: time.Now().Unix(),
		NodeID:    n.nodeID,
		TTL:       1,
	}
	pingMsg.MessageID = n.generateMessageID(pingMsg)

	go n.sendToPeer(pingMsg, peer)
}

// cacheCleanupLoop 缓存清理循环
func (n *Network) cacheCleanupLoop() {
	ticker := time.NewTicker(time.Minute * 2) // 每2分钟清理一次
	defer ticker.Stop()

	for {
		select {
		case <-n.stopCh:
			return
		case <-ticker.C:
			n.mu.Lock()
			n.cleanExpiredCache()
			n.mu.Unlock()
		}
	}
}

// broadcast 广播消息到优先级队列
func (n *Network) broadcast(msg *Message) error {
	// 设置消息优先级
	msg.Priority = GetMessagePriority(msg.Type)
	msg.Size = len(msg.Data)
	
	// 检查带宽限制
	if !n.checkBandwidth(msg.Size) {
		n.stats.DroppedMessages++
		return fmt.Errorf("bandwidth limit exceeded")
	}
	
	// 根据优先级将消息放入相应队列
	switch msg.Priority {
	case PriorityHigh:
		select {
		case n.priorityQueue.highPriority <- msg:
			n.stats.HighPriorityMessages++
			return nil
		default:
			n.stats.DroppedMessages++
			return fmt.Errorf("high priority queue full")
		}
	case PriorityMedium:
		select {
		case n.priorityQueue.mediumPriority <- msg:
			n.stats.MediumPriorityMessages++
			return nil
		default:
			n.stats.DroppedMessages++
			return fmt.Errorf("medium priority queue full")
		}
	case PriorityLow:
		select {
		case n.priorityQueue.lowPriority <- msg:
			n.stats.LowPriorityMessages++
			return nil
		default:
			n.stats.DroppedMessages++
			return fmt.Errorf("low priority queue full")
		}
	default:
		return fmt.Errorf("unknown message priority")
	}
}

// sendToPeers 发送消息给所有健康的连接节点
func (n *Network) sendToPeers(msg *Message) {
	n.mu.RLock()
	defer n.mu.RUnlock()

	for _, peer := range n.peers {
		if peer.IsHealthy && peer.Conn != nil {
			go n.sendToPeer(msg, peer)
		}
	}
}

// sendToPeer 发送消息给单个节点
func (n *Network) sendToPeer(msg *Message, peer *Peer) {
	start := time.Now()
	encoder := json.NewEncoder(peer.Conn)

	if err := encoder.Encode(msg); err != nil {
		fmt.Printf("Failed to send message to %s: %v\n", peer.ID, err)
		n.mu.Lock()
		peer.FailCount++
		if peer.FailCount >= n.maxFailCount {
			peer.IsHealthy = false
		}
		n.stats.FailedSends++
		n.mu.Unlock()
		return
	}

	// 更新统计信息
	n.mu.Lock()
	latency := time.Since(start)
	peer.Latency = latency
	peer.MessagesSent++
	peer.BytesSent += uint64(len(msg.Data))
	n.stats.TotalMessagesSent++
	n.stats.TotalBytesSent += uint64(len(msg.Data))
	n.mu.Unlock()
}

// GetPeerCount 获取连接的节点数量
func (n *Network) GetPeerCount() int {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return len(n.peers)
}

// GetHealthyPeerCount 获取健康节点数量
func (n *Network) GetHealthyPeerCount() int {
	n.mu.RLock()
	defer n.mu.RUnlock()

	count := 0
	for _, peer := range n.peers {
		if peer.IsHealthy {
			count++
		}
	}
	return count
}

// GetPeers 获取所有连接的节点
func (n *Network) GetPeers() []*Peer {
	n.mu.RLock()
	defer n.mu.RUnlock()

	peers := make([]*Peer, 0, len(n.peers))
	for _, peer := range n.peers {
		peerCopy := *peer
		peerCopy.Conn = nil // 不返回连接对象
		peers = append(peers, &peerCopy)
	}

	return peers
}

// GetHealthyPeers 获取所有健康的节点
func (n *Network) GetHealthyPeers() []*Peer {
	n.mu.RLock()
	defer n.mu.RUnlock()

	var healthyPeers []*Peer
	for _, peer := range n.peers {
		if peer.IsHealthy {
			peerCopy := *peer
			peerCopy.Conn = nil
			healthyPeers = append(healthyPeers, &peerCopy)
		}
	}

	return healthyPeers
}

// SendHeartbeat 发送心跳消息
func (n *Network) SendHeartbeat() error {
	msg := &Message{
		Type:      MsgHeartbeat,
		Data:      []byte(fmt.Sprintf(`{"node_id":"%s","timestamp":%d}`, n.nodeID, time.Now().Unix())),
		Timestamp: time.Now().Unix(),
		NodeID:    n.nodeID,
		TTL:       3,
	}
	msg.MessageID = n.generateMessageID(msg)

	return n.broadcast(msg)
}

// RequestBlock 请求特定区块
func (n *Network) RequestBlock(blockHash types.Hash) error {
	data, err := json.Marshal(map[string]string{
		"block_hash": blockHash.String(),
	})
	if err != nil {
		return fmt.Errorf("failed to marshal block request: %w", err)
	}

	msg := &Message{
		Type:      MsgBlockRequest,
		Data:      data,
		Timestamp: time.Now().Unix(),
		NodeID:    n.nodeID,
		TTL:       5,
	}
	msg.MessageID = n.generateMessageID(msg)

	return n.broadcast(msg)
}

// GetNetworkStats 获取网络统计信息
func (n *Network) GetNetworkStats() map[string]interface{} {
	n.mu.RLock()
	defer n.mu.RUnlock()

	uptime := time.Since(n.stats.StartTime)
	stats := map[string]interface{}{
		"node_id":            n.nodeID,
		"listen_addr":        n.listenAddr,
		"is_running":         n.isRunning,
		"peer_count":         len(n.peers),
		"healthy_peers":      n.GetHealthyPeerCount(),
		"uptime_seconds":     uptime.Seconds(),
		"messages_sent":      n.stats.TotalMessagesSent,
		"messages_received":  n.stats.TotalMessagesReceived,
		"bytes_sent":         n.stats.TotalBytesSent,
		"bytes_received":     n.stats.TotalBytesReceived,
		"duplicate_messages": n.stats.DuplicateMessages,
		"failed_sends":       n.stats.FailedSends,
		"cache_size":         len(n.messageCache),
		"active_connections": n.stats.ActiveConnections,
	}

	return stats
}

// =============== 新增：优先级和拥塞控制方法 ===============

// checkBandwidth 检查带宽限制
func (n *Network) checkBandwidth(size int) bool {
	n.bandwidthLimiter.mu.Lock()
	defer n.bandwidthLimiter.mu.Unlock()
	
	now := time.Now()
	// 每秒重置令牌桶
	if now.Sub(n.bandwidthLimiter.lastReset) >= time.Second {
		n.bandwidthLimiter.tokens = n.bandwidthLimiter.bucketSize
		n.bandwidthLimiter.used = 0
		n.bandwidthLimiter.lastReset = now
	}
	
	// 检查是否有足够的令牌
	requiredTokens := int64(size) / n.bandwidthLimiter.bytesPerToken
	if requiredTokens <= 0 {
		requiredTokens = 1
	}
	
	if n.bandwidthLimiter.tokens >= requiredTokens {
		n.bandwidthLimiter.tokens -= requiredTokens
		n.bandwidthLimiter.used += int64(size)
		return true
	}
	
	return false
}

// updateCongestionLevel 更新拥塞级别
func (n *Network) updateCongestionLevel() {
	n.congestionCtrl.mu.Lock()
	defer n.congestionCtrl.mu.Unlock()
	
	now := time.Now()
	if now.Sub(n.congestionCtrl.lastUpdate) < time.Second {
		return // 避免过于频繁的更新
	}
	
	// 计算当前的拥塞指标
	queueSize := len(n.priorityQueue.highPriority) + 
				 len(n.priorityQueue.mediumPriority) + 
				 len(n.priorityQueue.lowPriority)
	
	// 计算拥塞级别 (0-1)
	congestionLevel := float64(queueSize) / float64(n.priorityQueue.maxSize)
	
	// 滑动窗口平均
	n.congestionCtrl.congestionLevel = (n.congestionCtrl.congestionLevel*0.8) + (congestionLevel*0.2)
	n.congestionCtrl.lastUpdate = now
	
	// 如果拥塞级别超过阈值，记录拥塞事件
	if n.congestionCtrl.congestionLevel > 0.8 {
		n.stats.CongestionEvents++
	}
}

// GetCongestionLevel 获取当前拥塞级别
func (n *Network) GetCongestionLevel() float64 {
	n.congestionCtrl.mu.RLock()
	defer n.congestionCtrl.mu.RUnlock()
	return n.congestionCtrl.congestionLevel
}

// GetQueueStats 获取队列统计信息
func (n *Network) GetQueueStats() map[string]interface{} {
	return map[string]interface{}{
		"high_priority_queue":   len(n.priorityQueue.highPriority),
		"medium_priority_queue": len(n.priorityQueue.mediumPriority),
		"low_priority_queue":    len(n.priorityQueue.lowPriority),
		"total_queue_size":      len(n.priorityQueue.highPriority) + len(n.priorityQueue.mediumPriority) + len(n.priorityQueue.lowPriority),
		"max_queue_size":        n.priorityQueue.maxSize,
		"dropped_messages":      n.stats.DroppedMessages,
		"congestion_level":      n.GetCongestionLevel(),
		"bandwidth_used":        n.bandwidthLimiter.used,
		"bandwidth_limit":       n.bandwidthLimiter.limit,
		"bandwidth_tokens":      n.bandwidthLimiter.tokens,
	}
}

// SetBandwidthLimit 设置带宽限制
func (n *Network) SetBandwidthLimit(limit int64) {
	n.bandwidthLimiter.mu.Lock()
	defer n.bandwidthLimiter.mu.Unlock()
	
	n.bandwidthLimiter.limit = limit
	n.bandwidthLimiter.bucketSize = limit
	n.bandwidthLimiter.tokens = limit
}

// GetBandwidthUsage 获取带宽使用情况
func (n *Network) GetBandwidthUsage() map[string]interface{} {
	n.bandwidthLimiter.mu.RLock()
	defer n.bandwidthLimiter.mu.RUnlock()
	
	usagePercent := float64(n.bandwidthLimiter.used) / float64(n.bandwidthLimiter.limit) * 100
	
	return map[string]interface{}{
		"limit":         n.bandwidthLimiter.limit,
		"used":          n.bandwidthLimiter.used,
		"available":     n.bandwidthLimiter.limit - n.bandwidthLimiter.used,
		"usage_percent": usagePercent,
		"tokens":        n.bandwidthLimiter.tokens,
		"bucket_size":   n.bandwidthLimiter.bucketSize,
	}
}

// congestionMonitorLoop 拥塞监控循环
func (n *Network) congestionMonitorLoop() {
	ticker := time.NewTicker(5 * time.Second) // 每5秒检查一次
	defer ticker.Stop()

	for {
		select {
		case <-n.stopCh:
			return
		case <-ticker.C:
			n.updateCongestionLevel()
		}
	}
}

// IsRunning 检查网络是否正在运行
func (n *Network) IsRunning() bool {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.isRunning
}
