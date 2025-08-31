package network

import (
	"context"
	"crypto/rand"
	"encoding/pem"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/routing"
	"github.com/libp2p/go-libp2p/p2p/discovery/mdns"
	"github.com/libp2p/go-libp2p/p2p/net/connmgr"
	"github.com/libp2p/go-libp2p/p2p/security/noise"
	"github.com/multiformats/go-multiaddr"
)

// Network 网络层 - 基于libp2p的现代化P2P网络
type Network struct {
	config *NetworkConfig
	host   host.Host
	ctx    context.Context
	cancel context.CancelFunc

	// DHT自动管理（由libp2p.Routing自动处理）

	// Gossipsub消息传播
	pubsub   *pubsub.PubSub
	topics   map[string]*pubsub.Topic
	topicsMu sync.RWMutex

	// 消息处理
	messageHandlers map[string]MessageHandler
	handlersMu      sync.RWMutex

	// 点对点请求处理
	requestHandlers map[string]RequestHandler
	requestMu       sync.RWMutex

	// 节点发现
	mdnsService mdns.Service
}

var _ NetworkInterface = (*Network)(nil)
var log = slog.With("module", "network")

// New 创建新的网络实例
func New(cfg *NetworkConfig) (*Network, error) {
	ctx, cancel := context.WithCancel(context.Background())

	// 获取私钥
	priv, err := getOrGeneratePrivateKey(cfg.PrivateKeyPath)
	if err != nil {
		cancel()
		log.Error("failed to get private key", "error", err)
		return nil, fmt.Errorf("failed to get private key: %w", err)
	}

	// 创建连接管理器
	connManager, err := connmgr.NewConnManager(
		20,                                     // 最小连接数（默认值）
		cfg.MaxPeers,                           // 最大连接数
		connmgr.WithGracePeriod(time.Minute*5), // 优雅期
	)
	if err != nil {
		cancel()
		log.Error("failed to create connection manager", "error", err)
		return nil, fmt.Errorf("failed to create connection manager: %w", err)
	}

	// 解析 bootstrap 节点
	var bootstrapPeers []peer.AddrInfo
	for _, peerAddr := range cfg.BootstrapPeers {
		maddr, err := multiaddr.NewMultiaddr(peerAddr)
		if err != nil {
			log.Error("failed to parse bootstrap peer address", "peer_addr", peerAddr, "error", err)
			continue
		}

		addrInfo, err := peer.AddrInfoFromP2pAddr(maddr)
		if err != nil {
			log.Error("failed to parse bootstrap peer info", "peer_addr", peerAddr, "error", err)
			continue
		}

		bootstrapPeers = append(bootstrapPeers, *addrInfo)
	}
	if len(bootstrapPeers) == 0 {
		bootstrapPeers = dht.GetDefaultBootstrapPeerAddrInfos()
	}

	// 创建libp2p主机
	host, err := libp2p.New(
		// 使用生成的密钥对
		libp2p.Identity(priv),
		// 监听地址
		libp2p.ListenAddrStrings(
			fmt.Sprintf("/ip4/%s/tcp/%d", cfg.Host, cfg.Port),         // TCP连接
			fmt.Sprintf("/ip4/%s/udp/%d/quic-v1", cfg.Host, cfg.Port), // QUIC传输
		),
		// 支持TLS连接
		// libp2p.Security(libp2ptls.ID, libp2ptls.New),
		// 支持noise连接
		libp2p.Security(noise.ID, noise.New),
		// 支持默认传输协议
		libp2p.DefaultTransports,
		// 尝试使用uPNP为NAT主机开放端口
		libp2p.NATPortMap(),
		// 让此主机使用DHT查找其他主机，并设置bootstrap节点
		libp2p.Routing(func(h host.Host) (routing.PeerRouting, error) {
			dhtInstance, err := dht.New(context.Background(), h, dht.BootstrapPeers(bootstrapPeers...))
			return dhtInstance, err
		}),
		// 启用NAT服务以帮助其他节点检测NAT
		libp2p.EnableNATService(),
		// 启用自动中继
		libp2p.EnableAutoRelayWithStaticRelays([]peer.AddrInfo{}),
		// 启用洞穿
		libp2p.EnableHolePunching(),
		// 使用连接管理器
		libp2p.ConnectionManager(connManager),
	)
	if err != nil {
		cancel()
		log.Error("failed to create libp2p host", "error", err)
		return nil, fmt.Errorf("failed to create libp2p host: %w", err)
	}

	network := &Network{
		config:          cfg,
		host:            host,
		ctx:             ctx,
		cancel:          cancel,
		messageHandlers: make(map[string]MessageHandler),
		requestHandlers: make(map[string]RequestHandler),
		topics:          make(map[string]*pubsub.Topic),
	}

	// DHT由libp2p.Routing自动初始化

	// 初始化Gossipsub
	if err := network.initializeGossipsub(); err != nil {
		cancel()
		log.Error("failed to initialize Gossipsub", "error", err)
		return nil, fmt.Errorf("failed to initialize Gossipsub: %w", err)
	}

	network.host.SetStreamHandler("/chain/1.0.0/request", network.handleRequest)

	// 设置网络通知
	host.Network().Notify(network)
	log.Info("network initialized", "address", network.GetLocalAddresses())

	return network, nil
}

// Run 运行网络
func (n *Network) Run(ctx context.Context) error {
	// 启动mDNS发现
	n.startMDNSDiscovery()

	// 可信节点现在作为 DHT bootstrap 节点，DHT 会自动处理连接
	log.Info("network started, bootstrap peers configured as DHT bootstrap nodes")

	// 等待context取消
	<-ctx.Done()

	// 停止mDNS服务
	if n.mdnsService != nil {
		n.mdnsService.Close()
	}

	// 关闭主机
	if n.host != nil {
		n.host.Close()
	}

	return ctx.Err()
}

// initializeGossipsub 初始化Gossipsub
func (n *Network) initializeGossipsub() error {
	// 创建Gossipsub实例
	pubsubInstance, err := pubsub.NewGossipSub(n.ctx, n.host)
	if err != nil {
		return fmt.Errorf("failed to create Gossipsub: %w", err)
	}

	n.pubsub = pubsubInstance
	return nil
}

// handleRequest 处理点对点请求
func (n *Network) handleRequest(stream network.Stream) {
	// 不要在这里立即关闭流，让客户端先读取响应

	log.Info("handling request", "remote_peer", stream.Conn().RemotePeer().String())

	defer func() {
		time.Sleep(100 * time.Millisecond)
		stream.Close()
	}()

	// 读取请求数据 - 支持大数据量
	var data []byte
	buffer := make([]byte, 1024) // 1KB缓冲区

	// 读取所有可用数据
	bytesRead, err := stream.Read(buffer)
	if err != nil {
		log.Error("failed to read request data", "error", err)
		return
	}

	if bytesRead == 0 {
		log.Error("received empty request data")
		return
	}

	data = buffer[:bytesRead]

	if len(data) == 0 {
		log.Error("received empty request data")
		return
	}
	log.Info("received request data", "bytes_read", len(data), "data", string(data))

	// 解析请求
	var req Request
	if err := req.Deserialize(data); err != nil {
		log.Error("failed to deserialize request", "error", err)
		return
	}

	log.Info("parsed request", "type", req.Type, "data_len", len(req.Data))

	// 调用处理器
	handler, exists := n.requestHandlers[req.Type]
	if !exists {
		return
	}

	log.Info("calling handler", "type", req.Type, "peer", stream.Conn().RemotePeer().String())
	respData, err := handler(stream.Conn().RemotePeer().String(), req)
	if err != nil {
		log.Error("failed to process request", "error", err)
		// 发送错误响应
		resp := Response{
			Type: "error",
			Data: []byte(err.Error()),
		}
		respBytes, _ := resp.Serialize()
		if _, err := stream.Write(respBytes); err != nil {
			log.Error("failed to send error response", "error", err)
		}
		return
	}

	log.Info("handler completed", "type", req.Type, "response_len", len(respData))

	// 发送响应
	resp := Response{
		Type: req.Type,
		Data: respData,
	}
	respBytes, _ := resp.Serialize()
	log.Info("sending response", "type", req.Type, "response_bytes", len(respBytes))
	if _, err := stream.Write(respBytes); err != nil {
		log.Error("failed to send response", "error", err)
	} else {
		log.Info("response sent successfully")
	}
}

// startMDNSDiscovery 启动mDNS发现
func (n *Network) startMDNSDiscovery() {
	n.mdnsService = mdns.NewMdnsService(n.host, "chain-network", n)

	// 启动mDNS服务
	if err := n.mdnsService.Start(); err != nil {
		log.Error("failed to start mDNS service", "error", err)
		return
	}

	log.Info("mDNS service started", "service_name", "chain-network")
}

// BroadcastMessage 广播消息
func (n *Network) BroadcastMessage(topic string, data []byte) error {
	if n.pubsub == nil {
		return fmt.Errorf("gossipsub not initialized")
	}

	topicObj, err := n.getOrCreateTopic(topic)
	if err != nil {
		return fmt.Errorf("failed to get topic: %w", err)
	}

	if err := topicObj.Publish(n.ctx, data); err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}

	return nil
}

// getOrCreateTopic 获取或创建主题
func (n *Network) getOrCreateTopic(topicName string) (*pubsub.Topic, error) {
	n.topicsMu.Lock()
	defer n.topicsMu.Unlock()

	if topic, exists := n.topics[topicName]; exists {
		return topic, nil
	}

	topic, err := n.pubsub.Join(topicName)
	if err != nil {
		return nil, fmt.Errorf("failed to join topic: %w", err)
	}

	n.topics[topicName] = topic
	return topic, nil
}

// subscribeToTopicInternal 内部订阅主题方法
func (n *Network) subscribeToTopicInternal(topicName string) error {
	if n.pubsub == nil {
		return fmt.Errorf("gossipsub not initialized")
	}

	topic, err := n.getOrCreateTopic(topicName)
	if err != nil {
		return fmt.Errorf("获取主题失败: %w", err)
	}

	subscription, err := topic.Subscribe()
	if err != nil {
		return fmt.Errorf("failed to subscribe to topic: %w", err)
	}

	go n.handleSubscription(topicName, subscription)
	return nil
}

// handleSubscription 处理订阅
func (n *Network) handleSubscription(topicName string, subscription *pubsub.Subscription) {
	defer subscription.Cancel()

	for {
		msg, err := subscription.Next(n.ctx)
		if err != nil {
			if err == context.Canceled {
				return
			}
			log.Error("failed to receive message", "error", err)
			continue
		}

		if msg.ReceivedFrom == n.host.ID() {
			continue // 忽略自己发送的消息
		}

		go n.processPubsubMessage(topicName, msg)
	}
}

// processPubsubMessage 处理pubsub消息
func (n *Network) processPubsubMessage(topicName string, msg *pubsub.Message) {
	n.handlersMu.RLock()
	handler, exists := n.messageHandlers[topicName]
	n.handlersMu.RUnlock()

	if !exists {
		return
	}
	netMsg := NetMessage{
		From:  msg.ReceivedFrom.String(),
		Topic: topicName,
		Data:  msg.Data,
	}

	if err := handler(msg.ReceivedFrom.String(), netMsg); err != nil {
		log.Error("failed to process message", "error", err)
	}
}

// RegisterMessageHandler 注册消息处理器
// 自动订阅相应的主题以接收消息
func (n *Network) RegisterMessageHandler(topic string, handler MessageHandler) {
	n.handlersMu.Lock()
	defer n.handlersMu.Unlock()

	// 检查是否已经注册过该主题的处理器
	_, exists := n.messageHandlers[topic]
	n.messageHandlers[topic] = handler

	// 如果是新注册的主题，自动订阅
	if !exists {
		go func() {
			if err := n.subscribeToTopicInternal(topic); err != nil {
				log.Error("failed to auto-subscribe to topic", "topic", topic, "error", err)
			} else {
				log.Info("auto-subscribed to topic", "topic", topic)
			}
		}()
	}
}

// RegisterRequestHandler 注册点对点请求处理器
func (n *Network) RegisterRequestHandler(requestType string, handler RequestHandler) {
	n.requestMu.Lock()
	defer n.requestMu.Unlock()
	n.requestHandlers[requestType] = handler
}

// SendRequest 发送点对点请求
func (n *Network) SendRequest(peerID string, requestType string, data []byte) ([]byte, error) {
	pID, err := peer.Decode(peerID)
	if err != nil {
		return nil, fmt.Errorf("invalid peer ID: %w", err)
	}
	// 创建流
	stream, err := n.host.NewStream(n.ctx, pID, "/chain/1.0.0/request")
	if err != nil {
		return nil, fmt.Errorf("failed to create stream: %w", err)
	}
	defer stream.Close()

	// 构造请求
	req := Request{
		Type: requestType,
		Data: data,
	}

	// 序列化请求
	reqData, err := req.Serialize()
	if err != nil {
		return nil, fmt.Errorf("failed to serialize request: %w", err)
	}

	// 发送请求
	if _, err := stream.Write(reqData); err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	// 读取响应
	var responseData []byte
	buffer := make([]byte, 4096)
	for {
		bytesRead, err := stream.Read(buffer)
		if bytesRead > 0 {
			responseData = append(responseData, buffer[:bytesRead]...)
		}
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			log.Error("failed to read response", "error", err, "peer", peerID)
			return nil, fmt.Errorf("failed to read response: %w", err)
		}
		if bytesRead == 0 {
			break
		}
	}

	log.Info("read response", "bytes_read", len(responseData), "peer", peerID)

	// 解析响应
	var resp Response
	if err := resp.Deserialize(responseData); err != nil {
		return nil, fmt.Errorf("failed to deserialize response: %w", err)
	}

	// 检查是否是错误响应
	if resp.Type == "error" {
		return nil, fmt.Errorf("remote error: %s", string(resp.Data))
	}

	return resp.Data, nil
}

// GetPeers 获取连接的节点
func (n *Network) GetPeers() []string {
	peers := n.host.Network().Peers()
	result := make([]string, len(peers))
	for i, p := range peers {
		result[i] = p.String()
	}

	return result
}

// ConnectToPeer 连接到指定节点（外部API）
func (n *Network) ConnectToPeer(addr string) error {
	maddr, err := multiaddr.NewMultiaddr(addr)
	if err != nil {
		return fmt.Errorf("failed to parse address: %w", err)
	}

	info, err := peer.AddrInfoFromP2pAddr(maddr)
	if err != nil {
		return fmt.Errorf("failed to parse peer info: %w", err)
	}

	if err := n.host.Connect(n.ctx, *info); err != nil {
		return fmt.Errorf("failed to connect to peer: %w", err)
	}

	return nil
}

// IsPeerConnected 检查节点是否已连接
func (n *Network) IsPeerConnected(peerID peer.ID) bool {
	conns := n.host.Network().ConnsToPeer(peerID)
	return len(conns) > 0
}

// GetPubsub 获取Gossipsub实例
func (n *Network) GetPubsub() *pubsub.PubSub {
	return n.pubsub
}

// ListTopics 列出所有主题
func (n *Network) ListTopics() []string {
	if n.pubsub == nil {
		return []string{}
	}
	return n.pubsub.GetTopics()
}

// GetTopicPeers 获取主题中的节点
func (n *Network) GetTopicPeers(topicName string) []peer.ID {
	if n.pubsub == nil {
		return []peer.ID{}
	}
	topic, err := n.pubsub.Join(topicName)
	if err != nil {
		return []peer.ID{}
	}
	return topic.ListPeers()
}

// GetHost 获取libp2p主机
func (n *Network) GetHost() host.Host {
	return n.host
}

// IsMDNSEnabled 检查mDNS是否启用
func (n *Network) IsMDNSEnabled() bool {
	return n.mdnsService != nil
}

// GetMDNSStatus 获取mDNS状态
func (n *Network) GetMDNSStatus() map[string]interface{} {
	return map[string]interface{}{
		"enabled":      n.mdnsService != nil,
		"service":      n.mdnsService != nil,
		"service_name": "chain-network",
	}
}

// GetConnectionStats 获取连接统计信息
func (n *Network) GetConnectionStats() map[string]interface{} {
	stats := make(map[string]interface{})

	// 获取当前连接的节点数量
	peers := n.GetPeers()
	stats["current_peers"] = len(peers)

	// 获取配置的最大节点数
	stats["max_peers"] = n.config.MaxPeers

	// 计算连接使用率
	if n.config.MaxPeers > 0 {
		usage := float64(len(peers)) / float64(n.config.MaxPeers) * 100
		stats["usage_percentage"] = usage
	} else {
		stats["usage_percentage"] = 0.0
	}

	// 获取配置的 bootstrap 节点数量
	stats["bootstrap_peers"] = len(n.config.BootstrapPeers)

	return stats
}

// GetLocalAddresses 获取本地节点的地址列表
func (n *Network) GetLocalAddresses() []string {
	if n.host == nil {
		return []string{}
	}

	addresses := n.host.Addrs()
	result := make([]string, len(addresses))

	for i, addr := range addresses {
		// 将地址和节点ID组合成完整的p2p地址
		fullAddr := fmt.Sprintf("%s/p2p/%s", addr.String(), n.host.ID().String())
		result[i] = fullAddr
	}

	return result
}

// GetLocalPeerID 获取本地节点的Peer ID
func (n *Network) GetLocalPeerID() string {
	if n.host == nil {
		return ""
	}
	return n.host.ID().String()
}

// 实现network.Notifiee接口
func (n *Network) Listen(network.Network, multiaddr.Multiaddr)      {}
func (n *Network) ListenClose(network.Network, multiaddr.Multiaddr) {}

func (n *Network) Connected(net network.Network, conn network.Conn) {
	// peerID := conn.RemotePeer()
	// log.Info("peer connected", "peer_id", peerID.String())
}

func (n *Network) Disconnected(net network.Network, conn network.Conn) {
	// peerID := conn.RemotePeer()
	// log.Info("peer disconnected", "peer_id", peerID.String())
}

func (n *Network) OpenedStream(network.Network, network.Stream) {}
func (n *Network) ClosedStream(network.Network, network.Stream) {}

// 实现mdns.Notifee接口
func (n *Network) HandlePeerFound(pi peer.AddrInfo) {
	// 避免连接自己
	if pi.ID == n.host.ID() {
		return
	}

	log.Info("mDNS peer discovered", "peer_id", pi.ID.String(), "addresses", pi.Addrs)

	if err := n.host.Connect(n.ctx, pi); err != nil {
		log.Error("failed to connect to discovered peer", "peer_id", pi.ID.String(), "error", err)
	} else {
		log.Info("successfully connected to discovered peer", "peer_id", pi.ID.String())
	}
}

// getOrGeneratePrivateKey 获取或生成私钥
func getOrGeneratePrivateKey(keyPath string) (crypto.PrivKey, error) {
	// 如果指定了私钥文件路径
	if keyPath != "" {
		// 检查文件是否存在
		if _, err := os.Stat(keyPath); os.IsNotExist(err) {
			// 文件不存在，生成新私钥并保存
			log.Info("private key file not found, generating new key", "key_path", keyPath)
			priv, _, err := crypto.GenerateKeyPairWithReader(crypto.Ed25519, 2048, rand.Reader)
			if err != nil {
				return nil, fmt.Errorf("failed to generate private key: %w", err)
			}

			// 保存私钥到文件
			if err := SavePrivateKeyToFile(priv, keyPath); err != nil {
				return nil, fmt.Errorf("failed to save private key: %w", err)
			}

			log.Info("new private key saved", "key_path", keyPath)
			return priv, nil
		}

		// 文件存在，尝试读取
		priv, err := loadPrivateKeyFromFile(keyPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read private key file: %w", err)
		}

		log.Info("private key loaded from file", "key_path", keyPath)
		return priv, nil
	}

	// 没有指定文件路径，生成临时私钥
	log.Info("no private key path specified, generating temporary key")
	priv, _, err := crypto.GenerateKeyPairWithReader(crypto.Ed25519, 2048, rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate private key: %w", err)
	}

	return priv, nil
}

// loadPrivateKeyFromFile 从文件加载私钥
func loadPrivateKeyFromFile(path string) (crypto.PrivKey, error) {
	// 读取私钥文件
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read private key file: %w", err)
	}

	// 解析PEM格式
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("invalid PEM format private key file")
	}

	// 解析私钥
	priv, err := crypto.UnmarshalPrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	return priv, nil
}

// SavePrivateKeyToFile 保存私钥到文件
func SavePrivateKeyToFile(priv crypto.PrivKey, path string) error {
	// 序列化私钥
	privBytes, err := crypto.MarshalPrivateKey(priv)
	if err != nil {
		return fmt.Errorf("failed to serialize private key: %w", err)
	}

	// 创建PEM块
	block := &pem.Block{
		Type:  "LIBP2P PRIVATE KEY",
		Bytes: privBytes,
	}

	// 编码为PEM格式
	pemData := pem.EncodeToMemory(block)

	// 写入文件
	err = os.WriteFile(path, pemData, 0600) // 只有所有者可读写
	if err != nil {
		return fmt.Errorf("failed to write private key file: %w", err)
	}

	return nil
}
