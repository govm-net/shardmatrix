package network

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/libp2p/go-libp2p/p2p/discovery/mdns"
	"github.com/multiformats/go-multiaddr"

	"github.com/govm-net/shardmatrix/pkg/types"
)

// LibP2PConfig libp2p网络配置
type LibP2PConfig struct {
	NodeID         string        `json:"node_id"`
	ListenAddrs    []string      `json:"listen_addrs"`    // 监听地址列表
	BootstrapPeers []string      `json:"bootstrap_peers"` // 引导节点
	ConnTimeout    time.Duration `json:"conn_timeout"`    // 连接超时
	ChainID        uint64        `json:"chain_id"`        // 链ID
	Version        string        `json:"version"`         // 版本
	MaxPeers       int           `json:"max_peers"`       // 最大节点数
}

// DefaultLibP2PConfig 默认libp2p配置
func DefaultLibP2PConfig() *LibP2PConfig {
	return &LibP2PConfig{
		NodeID: generateRandomNodeID(),
		ListenAddrs: []string{
			"/ip4/0.0.0.0/tcp/0",
			"/ip6/::/tcp/0",
		},
		BootstrapPeers: []string{},
		ConnTimeout:    time.Second * 30,
		ChainID:        1,
		Version:        "1.0.0",
		MaxPeers:       50,
	}
}

// Protocol IDs
const (
	ShardMatrixProtocolID = protocol.ID("/shardmatrix/1.0.0")
	BlockTopic            = "shardmatrix-blocks"
	TransactionTopic      = "shardmatrix-transactions"
)

// generateRandomNodeID 生成随机节点ID
func generateRandomNodeID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return fmt.Sprintf("%x", bytes)
}

// LibP2PNetwork libp2p网络管理器
type LibP2PNetwork struct {
	config *LibP2PConfig
	host   host.Host
	pubsub *pubsub.PubSub
	mdns   mdns.Service

	// 订阅管理
	blockTopic *pubsub.Topic
	txTopic    *pubsub.Topic
	blockSub   *pubsub.Subscription
	txSub      *pubsub.Subscription

	// 状态管理
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	isRunning bool
	mutex     sync.RWMutex

	// 回调函数
	onPeerConnected    func(peerID peer.ID)
	onPeerDisconnected func(peerID peer.ID)
	onBlockReceived    func(block *types.Block, peerID peer.ID)
	onTxReceived       func(tx *types.Transaction, peerID peer.ID)
	onMessageReceived  func(msg *Message, peerID peer.ID)
}

// NewLibP2PNetwork 创建LibP2P网络
func NewLibP2PNetwork(config *LibP2PConfig) (*LibP2PNetwork, error) {
	if config == nil {
		config = DefaultLibP2PConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())

	// 创建 multiaddrs
	listenAddrs := make([]multiaddr.Multiaddr, len(config.ListenAddrs))
	for i, addr := range config.ListenAddrs {
		ma, err := multiaddr.NewMultiaddr(addr)
		if err != nil {
			cancel()
			return nil, fmt.Errorf("invalid listen address %s: %v", addr, err)
		}
		listenAddrs[i] = ma
	}

	// 创建host
	host, err := libp2p.New(
		libp2p.ListenAddrs(listenAddrs...),
	)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create libp2p host: %v", err)
	}

	return &LibP2PNetwork{
		config:    config,
		host:      host,
		ctx:       ctx,
		cancel:    cancel,
		isRunning: false,
	}, nil
}

// SetCallbacks 设置回调函数
func (n *LibP2PNetwork) SetCallbacks(
	onConnected func(peer.ID),
	onDisconnected func(peer.ID),
	onBlock func(*types.Block, peer.ID),
	onTx func(*types.Transaction, peer.ID),
	onMessage func(*Message, peer.ID),
) {
	n.onPeerConnected = onConnected
	n.onPeerDisconnected = onDisconnected
	n.onBlockReceived = onBlock
	n.onTxReceived = onTx
	n.onMessageReceived = onMessage
}

// Start 启动LibP2P网络
func (n *LibP2PNetwork) Start() error {
	n.mutex.Lock()
	defer n.mutex.Unlock()

	if n.isRunning {
		return fmt.Errorf("LibP2P network is already running")
	}

	// 初始化pubsub
	ps, err := pubsub.NewGossipSub(n.ctx, n.host)
	if err != nil {
		return fmt.Errorf("failed to create pubsub: %v", err)
	}
	n.pubsub = ps

	// 创建主题
	n.blockTopic, err = n.pubsub.Join(BlockTopic)
	if err != nil {
		return fmt.Errorf("failed to join block topic: %v", err)
	}

	n.txTopic, err = n.pubsub.Join(TransactionTopic)
	if err != nil {
		return fmt.Errorf("failed to join transaction topic: %v", err)
	}

	// 订阅主题
	n.blockSub, err = n.blockTopic.Subscribe()
	if err != nil {
		return fmt.Errorf("failed to subscribe to block topic: %v", err)
	}

	n.txSub, err = n.txTopic.Subscribe()
	if err != nil {
		return fmt.Errorf("failed to subscribe to transaction topic: %v", err)
	}

	// 设置连接事件监听
	n.host.Network().Notify(&network.NotifyBundle{
		ConnectedF: func(net network.Network, conn network.Conn) {
			if n.onPeerConnected != nil {
				n.onPeerConnected(conn.RemotePeer())
			}
		},
		DisconnectedF: func(net network.Network, conn network.Conn) {
			if n.onPeerDisconnected != nil {
				n.onPeerDisconnected(conn.RemotePeer())
			}
		},
	})

	// 设置流处理器
	n.host.SetStreamHandler(ShardMatrixProtocolID, n.handleStream)

	// 启动mDNS发现
	mdnsService := mdns.NewMdnsService(n.host, "shardmatrix", n)
	if err := mdnsService.Start(); err != nil {
		return fmt.Errorf("failed to start mDNS: %v", err)
	}
	n.mdns = mdnsService

	n.isRunning = true

	// 启动消息处理goroutines
	n.wg.Add(2)
	go n.handleBlockMessages()
	go n.handleTxMessages()

	// 连接引导节点
	n.wg.Add(1)
	go n.connectToBootstrapPeers()

	fmt.Printf("LibP2P network started. Host ID: %s\n", n.host.ID())
	fmt.Printf("Listen addresses: %v\n", n.host.Addrs())

	return nil
}

// Stop 停止LibP2P网络
func (n *LibP2PNetwork) Stop() error {
	n.mutex.Lock()
	defer n.mutex.Unlock()

	if !n.isRunning {
		return nil
	}

	n.isRunning = false

	// 停止mDNS
	if n.mdns != nil {
		n.mdns.Close()
	}

	// 关闭订阅
	if n.blockSub != nil {
		n.blockSub.Cancel()
	}
	if n.txSub != nil {
		n.txSub.Cancel()
	}

	// 关闭主题
	if n.blockTopic != nil {
		n.blockTopic.Close()
	}
	if n.txTopic != nil {
		n.txTopic.Close()
	}

	// 取消context
	n.cancel()

	// 关闭host
	if n.host != nil {
		n.host.Close()
	}

	// 等待所有goroutine结束
	n.wg.Wait()

	fmt.Println("LibP2P network stopped")
	return nil
}

// HandlePeerFound 处理发现的节点 (mDNS回调)
func (n *LibP2PNetwork) HandlePeerFound(pi peer.AddrInfo) {
	if pi.ID == n.host.ID() {
		return // 不连接自己
	}

	// 连接到发现的节点
	ctx, cancel := context.WithTimeout(n.ctx, n.config.ConnTimeout)
	defer cancel()

	if err := n.host.Connect(ctx, pi); err != nil {
		fmt.Printf("Failed to connect to discovered peer %s: %v\n", pi.ID, err)
	} else {
		fmt.Printf("Connected to discovered peer: %s\n", pi.ID)
	}
}

// connectToBootstrapPeers 连接引导节点
func (n *LibP2PNetwork) connectToBootstrapPeers() {
	defer n.wg.Done()

	for _, peerAddr := range n.config.BootstrapPeers {
		select {
		case <-n.ctx.Done():
			return
		default:
		}

		// 解析peer地址
		ma, err := multiaddr.NewMultiaddr(peerAddr)
		if err != nil {
			fmt.Printf("Invalid bootstrap peer address %s: %v\n", peerAddr, err)
			continue
		}

		pi, err := peer.AddrInfoFromP2pAddr(ma)
		if err != nil {
			fmt.Printf("Failed to parse bootstrap peer %s: %v\n", peerAddr, err)
			continue
		}

		// 连接
		ctx, cancel := context.WithTimeout(n.ctx, n.config.ConnTimeout)
		if err := n.host.Connect(ctx, *pi); err != nil {
			fmt.Printf("Failed to connect to bootstrap peer %s: %v\n", pi.ID, err)
		} else {
			fmt.Printf("Connected to bootstrap peer: %s\n", pi.ID)
		}
		cancel()
	}
}

// handleStream 处理流
func (net *LibP2PNetwork) handleStream(s network.Stream) {
	defer s.Close()

	// 读取消息
	buf := make([]byte, 1024*1024) // 1MB buffer
	n, err := s.Read(buf)
	if err != nil {
		fmt.Printf("Failed to read from stream: %v\n", err)
		return
	}

	// 解析消息
	var msg Message
	if err := json.Unmarshal(buf[:n], &msg); err != nil {
		fmt.Printf("Failed to unmarshal message: %v\n", err)
		return
	}

	// 处理消息
	if net.onMessageReceived != nil {
		net.onMessageReceived(&msg, s.Conn().RemotePeer())
	}
}

// handleBlockMessages 处理区块消息
func (n *LibP2PNetwork) handleBlockMessages() {
	defer n.wg.Done()

	for {
		select {
		case <-n.ctx.Done():
			return
		default:
		}

		msg, err := n.blockSub.Next(n.ctx)
		if err != nil {
			if err != context.Canceled {
				fmt.Printf("Failed to get block message: %v\n", err)
			}
			return
		}

		// 过滤自己的消息
		if msg.ReceivedFrom == n.host.ID() {
			continue
		}

		// 解析区块
		var block types.Block
		if err := json.Unmarshal(msg.Data, &block); err != nil {
			fmt.Printf("Failed to unmarshal block: %v\n", err)
			continue
		}

		// 处理区块
		if n.onBlockReceived != nil {
			n.onBlockReceived(&block, msg.ReceivedFrom)
		}
	}
}

// handleTxMessages 处理交易消息
func (n *LibP2PNetwork) handleTxMessages() {
	defer n.wg.Done()

	for {
		select {
		case <-n.ctx.Done():
			return
		default:
		}

		msg, err := n.txSub.Next(n.ctx)
		if err != nil {
			if err != context.Canceled {
				fmt.Printf("Failed to get tx message: %v\n", err)
			}
			return
		}

		// 过滤自己的消息
		if msg.ReceivedFrom == n.host.ID() {
			continue
		}

		// 解析交易
		var tx types.Transaction
		if err := json.Unmarshal(msg.Data, &tx); err != nil {
			fmt.Printf("Failed to unmarshal transaction: %v\n", err)
			continue
		}

		// 处理交易
		if n.onTxReceived != nil {
			n.onTxReceived(&tx, msg.ReceivedFrom)
		}
	}
}

// BroadcastBlock 广播区块
func (n *LibP2PNetwork) BroadcastBlock(block *types.Block) error {
	if !n.isRunning || n.blockTopic == nil {
		return fmt.Errorf("network not running or block topic not available")
	}

	// 序列化区块
	data, err := json.Marshal(block)
	if err != nil {
		return fmt.Errorf("failed to marshal block: %v", err)
	}

	// 发布到主题
	return n.blockTopic.Publish(n.ctx, data)
}

// BroadcastTransaction 广播交易
func (n *LibP2PNetwork) BroadcastTransaction(tx *types.Transaction) error {
	if !n.isRunning || n.txTopic == nil {
		return fmt.Errorf("network not running or tx topic not available")
	}

	// 序列化交易
	data, err := json.Marshal(tx)
	if err != nil {
		return fmt.Errorf("failed to marshal transaction: %v", err)
	}

	// 发布到主题
	return n.txTopic.Publish(n.ctx, data)
}

// SendMessage 发送消息给指定节点
func (n *LibP2PNetwork) SendMessage(peerID peer.ID, msg *Message) error {
	if !n.isRunning {
		return fmt.Errorf("network not running")
	}

	// 开启流
	s, err := n.host.NewStream(n.ctx, peerID, ShardMatrixProtocolID)
	if err != nil {
		return fmt.Errorf("failed to open stream: %v", err)
	}
	defer s.Close()

	// 序列化消息
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %v", err)
	}

	// 发送消息
	_, err = s.Write(data)
	return err
}

// GetConnectedPeers 获取已连接的节点
func (n *LibP2PNetwork) GetConnectedPeers() []peer.ID {
	return n.host.Network().Peers()
}

// GetHostID 获取本地节点ID
func (n *LibP2PNetwork) GetHostID() peer.ID {
	return n.host.ID()
}

// GetListenAddrs 获取监听地址
func (n *LibP2PNetwork) GetListenAddrs() []multiaddr.Multiaddr {
	return n.host.Addrs()
}

// IsConnectedToPeer 检查是否连接到指定节点
func (n *LibP2PNetwork) IsConnectedToPeer(peerID peer.ID) bool {
	return n.host.Network().Connectedness(peerID) == network.Connected
}

// DisconnectFromPeer 断开与指定节点的连接
func (n *LibP2PNetwork) DisconnectFromPeer(peerID peer.ID) error {
	return n.host.Network().ClosePeer(peerID)
}

// GetConfig 获取网络配置
func (n *LibP2PNetwork) GetConfig() *LibP2PConfig {
	return n.config
}

// IsRunning 检查网络是否运行中
func (n *LibP2PNetwork) IsRunning() bool {
	n.mutex.RLock()
	defer n.mutex.RUnlock()
	return n.isRunning
}

// GetPeerCount 获取连接节点数量
func (n *LibP2PNetwork) GetPeerCount() int {
	return len(n.GetConnectedPeers())
}
