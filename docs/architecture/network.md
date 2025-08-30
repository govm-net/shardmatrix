# ShardMatrix 网络层设计

## 概述

ShardMatrix 网络层基于 libp2p 构建，提供去中心化的 P2P 网络通信。

## 网络架构

- **传输协议**：TCP, WebSocket, QUIC
- **节点发现**：Kademlia DHT
- **消息传播**：Gossip 协议 + 直接请求/响应协议
- **连接管理**：连接池和复用

## 节点管理

### 节点类型
```go
type NodeType int
const (
    NodeTypeValidator NodeType = iota
    NodeTypeFull
    NodeTypeLight
    NodeTypeArchive
)

type Node struct {
    ID              []byte
    Type            NodeType
    Address         string
    Port            uint16
    PublicKey       []byte
    NodeAddress     Address
    Capabilities    []string
    LastSeen        int64
    Status          NodeStatus
}
```

### 节点发现
- 引导节点连接
- Kademlia DHT 查找
- 定期节点发现
- 节点状态监控

## 消息传播

### 广播消息类型
```go
type BroadcastMessageType int
const (
    BroadcastTypeBlock BroadcastMessageType = iota
    BroadcastTypeTransaction
    BroadcastTypeConsensus
    BroadcastTypeValidatorUpdate
    BroadcastTypeNetworkAlert
)

type BroadcastMessage struct {
    Type        BroadcastMessageType
    From        Address
    Data        []byte
    Signature   []byte
    Timestamp   int64
    TTL         uint32
    MessageID   string
}
```

### 点对点消息类型
```go
type P2PMessageType int
const (
    P2PTypeBlockRequest P2PMessageType = iota
    P2PTypeBlockResponse
    P2PTypeTransactionRequest
    P2PTypeTransactionResponse
    P2PTypeStateRequest
    P2PTypeStateResponse
    P2PTypeSyncRequest
    P2PTypeSyncResponse
    P2PTypePing
    P2PTypePong
)

type P2PMessage struct {
    Type        P2PMessageType
    From        Address
    To          Address
    RequestID   string
    Data        []byte
    Signature   []byte
    Timestamp   int64
}
```

### 广播传播协议
- Gossip 传播
- 消息去重
- TTL 控制
- 优先级队列
- 消息ID去重

### 点对点传播协议
- 直接连接传输
- 请求-响应模式
- 超时重试机制
- 连接状态监控

## 连接管理

### 连接池
```go
type ConnectionPool struct {
    connections     map[string]*Connection
    maxConnections  int
    connectionTTL   time.Duration
}

type Connection struct {
    ID              string
    PeerID          []byte
    PeerAddress     Address
    Address         string
    Established     time.Time
    LastActivity    time.Time
    Status          ConnectionStatus
    Bandwidth       uint64
    Latency         time.Duration
}
```

### 连接策略
- 最大连接数限制
- 连接复用
- 自动重连
- 负载均衡

## 安全机制

### 节点认证
- 公钥验证
- 证书验证
- 签名验证
- 黑名单机制

### 消息加密
- 对称加密
- 非对称加密
- 消息签名
- 完整性校验

## 网络监控

### 监控指标
- 节点数量
- 连接数量
- 带宽使用
- 延迟分布
- 丢包率

### 网络分区检测
- 心跳机制
- 超时检测
- 自动恢复
- 手动干预

## 性能优化

### 连接优化
- 连接复用
- 连接池管理
- 自动重连
- 负载均衡

### 消息优化
- 消息压缩
- 批量传输
- 优先级队列
- 流量控制

## 故障恢复

### 自动恢复
- 节点重连
- 连接修复
- 状态同步
- 网络重建

### 手动干预
- 节点黑名单
- 连接重置
- 网络隔离
- 紧急修复

## 点对点交互

### Gossip 协议在 P2P 交互中的应用
Gossip 协议是一种高效的分布式通信机制，适用于去中心化网络中的信息传播。在 ShardMatrix 中，Gossip 协议主要用于以下点对点交互场景：
- **新区块广播**：当一个节点生成或接收到新区块时，通过 Gossip 协议快速传播给其他节点。
- **交易传播**：新交易通过 Gossip 协议在网络中传播，确保所有节点都能看到待处理的交易。
- **节点状态共享**：节点可以通过 Gossip 协议共享自己的状态信息（如当前高度、可用性），帮助新节点发现同步目标。

Gossip 协议的优势在于其简单性和高效性，信息能够在网络中以指数级速度传播，即使部分节点不可用，信息仍能通过其他路径到达。但对于需要可靠传输的大量数据（如历史区块同步），Gossip 协议不是最优选择，应结合直接的点对点请求机制。

### 消息去重与优化
为了避免 Gossip 协议带来的重复消息问题，ShardMatrix 实现以下优化：
- **消息 ID**：每个消息都有唯一标识符，节点记录已处理的消息 ID，避免重复处理。
- **TTL（生存时间）**：设置消息的传播跳数限制，防止无限传播。
- **选择性转发**：节点根据邻居的状态选择性地转发消息，避免向已知已接收的节点重复发送。

## 新节点同步机制

新加入网络的节点需要同步历史区块和当前状态，以达到与其他节点的同步。ShardMatrix 采用主动请求-响应模式，确保同步过程高效可靠：

### 同步流程
1. **节点发现**：新节点通过种子节点或 Kademlia DHT 发现其他活跃节点。
2. **状态查询**：新节点向邻居节点查询当前区块链高度，确定自己需要同步的区块范围。
3. **区块请求**：新节点向拥有最新数据的节点发送请求，获取缺失的区块数据。
4. **区块验证**：接收到区块后，新节点验证区块的哈希、签名和数据完整性。
5. **状态更新**：验证通过后，更新本地区块链状态，并继续请求下一批区块，直到完全同步。
6. **状态同步**：在同步区块的同时，新节点也可以请求最新的状态树数据（如账户余额、合约状态），以加速后续验证。

### 同步优化
- **并行请求**：新节点可以同时向多个节点请求不同范围的区块，提高同步速度。
- **优先级队列**：优先同步较新的区块，以便尽快参与网络共识。
- **断点续传**：记录同步进度，遇到网络中断时可从断点继续同步。
- **状态快照**：对于较旧的节点，可以提供状态快照，减少从头同步的开销。

### 与 Gossip 协议的结合
虽然区块数据的实际传输依赖直接的点对点请求，但 Gossip 协议可以在同步过程中起到辅助作用：
- **同步需求广播**：新节点通过 Gossip 协议广播自己的同步需求，寻找拥有所需区块的节点。
- **可用节点通知**：拥有数据的节点可以通过 Gossip 协议通知新节点自己的可用性和数据范围。

这种结合方式充分利用了 Gossip 协议的信息传播优势，同时通过直接请求确保数据传输的可靠性。

## 点对点请求/响应协议

为了支持两个节点之间的可靠通信，ShardMatrix 实现了专门的请求/响应协议，不依赖 Gossip 协议，适用于需要可靠传输和确认的场景：

### 协议设计
```go
type P2PRequest struct {
    RequestID   string         `json:"request_id"`
    Type        P2PMessageType `json:"type"`
    From        Address        `json:"from"`
    To          Address        `json:"to"`
    Data        []byte         `json:"data"`
    Timestamp   int64          `json:"timestamp"`
    Signature   []byte         `json:"signature"`
}

type P2PResponse struct {
    RequestID   string         `json:"request_id"`
    Type        P2PMessageType `json:"type"`
    From        Address        `json:"from"`
    To          Address        `json:"to"`
    Data        []byte         `json:"data"`
    Error       string         `json:"error,omitempty"`
    Timestamp   int64          `json:"timestamp"`
    Signature   []byte         `json:"signature"`
}
```

### 支持的消息类型
1. **区块请求/响应** (`P2PTypeBlockRequest/P2PTypeBlockResponse`)：请求特定高度的区块数据
2. **交易请求/响应** (`P2PTypeTransactionRequest/P2PTypeTransactionResponse`)：请求特定交易详情
3. **状态请求/响应** (`P2PTypeStateRequest/P2PTypeStateResponse`)：请求账户或合约状态
4. **同步请求/响应** (`P2PTypeSyncRequest/P2PTypeSyncResponse`)：请求同步特定范围的区块
5. **心跳请求/响应** (`P2PTypePing/P2PTypePong`)：用于连接保活和延迟测试

### 协议特性
- **可靠性**：每个请求都有唯一ID，支持重传和确认机制
- **安全性**：所有消息都经过数字签名验证
- **超时控制**：设置请求超时时间，避免无限等待
- **并发支持**：支持多个并发请求，通过请求ID区分
- **错误处理**：响应中包含错误信息，便于调试和重试

### 实现示例
```go
type P2PProtocol struct {
    nodeID      Address
    connections map[Address]*Connection
    requests    map[string]*PendingRequest
    timeout     time.Duration
    mutex       sync.RWMutex
}

type PendingRequest struct {
    Request     *P2PRequest
    Response    chan *P2PResponse
    Timer       *time.Timer
    RetryCount  int
}

func (pp *P2PProtocol) SendRequest(req *P2PRequest) (*P2PResponse, error) {
    // 1. 生成请求ID
    req.RequestID = generateRequestID()
    req.From = pp.nodeID
    req.Timestamp = time.Now().Unix()
    
    // 2. 签名请求
    req.Signature = pp.signRequest(req)
    
    // 3. 创建响应通道
    responseChan := make(chan *P2PResponse, 1)
    timer := time.AfterFunc(pp.timeout, func() {
        pp.handleTimeout(req.RequestID)
    })
    
    // 4. 记录待处理请求
    pp.mutex.Lock()
    pp.requests[req.RequestID] = &PendingRequest{
        Request:    req,
        Response:   responseChan,
        Timer:      timer,
        RetryCount: 0,
    }
    pp.mutex.Unlock()
    
    // 5. 发送请求
    conn := pp.getConnection(req.To)
    if conn == nil {
        return nil, errors.New("no connection to target node")
    }
    
    err := conn.SendMessage(req)
    if err != nil {
        pp.removePendingRequest(req.RequestID)
        return nil, err
    }
    
    // 6. 等待响应
    select {
    case resp := <-responseChan:
        return resp, nil
    case <-timer.C:
        return nil, errors.New("request timeout")
    }
}

func (pp *P2PProtocol) HandleRequest(req *P2PRequest) (*P2PResponse, error) {
    // 1. 验证请求签名
    if !pp.verifyRequest(req) {
        return pp.createErrorResponse(req, "invalid signature"), nil
    }
    
    // 2. 根据请求类型处理
    var data []byte
    var err error
    
    switch req.Type {
    case P2PTypeBlockRequest:
        data, err = pp.handleBlockRequest(req)
    case P2PTypeTransactionRequest:
        data, err = pp.handleTransactionRequest(req)
    case P2PTypeStateRequest:
        data, err = pp.handleStateRequest(req)
    case P2PTypeSyncRequest:
        data, err = pp.handleSyncRequest(req)
    case P2PTypePing:
        data, err = pp.handlePingRequest(req)
    default:
        err = errors.New("unknown request type")
    }
    
    // 3. 创建响应
    if err != nil {
        return pp.createErrorResponse(req, err.Error()), nil
    }
    
    return pp.createSuccessResponse(req, data), nil
}

func (pp *P2PProtocol) HandleResponse(resp *P2PResponse) {
    pp.mutex.Lock()
    pending, exists := pp.requests[resp.RequestID]
    pp.mutex.Unlock()
    
    if !exists {
        return // 请求已超时或被移除
    }
    
    // 停止超时定时器
    pending.Timer.Stop()
    
    // 发送响应到等待的协程
    pending.Response <- resp
    
    // 移除待处理请求
    pp.removePendingRequest(resp.RequestID)
}

### 具体请求处理方法
```go
func (pp *P2PProtocol) handleBlockRequest(req *P2PRequest) ([]byte, error) {
    var blockReq struct {
        Height uint64 `json:"height"`
    }
    
    if err := json.Unmarshal(req.Data, &blockReq); err != nil {
        return nil, err
    }
    
    // 从本地存储获取区块
    block, err := pp.storage.GetBlock(blockReq.Height)
    if err != nil {
        return nil, err
    }
    
    return json.Marshal(block)
}

func (pp *P2PProtocol) handleSyncRequest(req *P2PRequest) ([]byte, error) {
    var syncReq struct {
        StartHeight uint64 `json:"start_height"`
        EndHeight   uint64 `json:"end_height"`
        BatchSize   int    `json:"batch_size"`
    }
    
    if err := json.Unmarshal(req.Data, &syncReq); err != nil {
        return nil, err
    }
    
    // 获取指定范围的区块
    blocks := make([]*Block, 0)
    for height := syncReq.StartHeight; height <= syncReq.EndHeight; height += uint64(syncReq.BatchSize) {
        end := height + uint64(syncReq.BatchSize) - 1
        if end > syncReq.EndHeight {
            end = syncReq.EndHeight
        }
        
        batch, err := pp.storage.GetBlockRange(height, end)
        if err != nil {
            return nil, err
        }
        blocks = append(blocks, batch...)
    }
    
    return json.Marshal(blocks)
}

func (pp *P2PProtocol) handlePingRequest(req *P2PRequest) ([]byte, error) {
    pong := struct {
        NodeID    Address `json:"node_id"`
        Timestamp int64   `json:"timestamp"`
        Latency   int64   `json:"latency"`
    }{
        NodeID:    pp.nodeID,
        Timestamp: time.Now().Unix(),
        Latency:   time.Now().Unix() - req.Timestamp,
    }
    
    return json.Marshal(pong)
}
```

### 连接管理
```go
type ConnectionManager struct {
    connections map[Address]*Connection
    mutex       sync.RWMutex
    maxConn     int
    timeout     time.Duration
}

func (cm *ConnectionManager) Connect(peerAddress Address) (*Connection, error) {
    cm.mutex.Lock()
    defer cm.mutex.Unlock()
    
    // 检查连接数量限制
    if len(cm.connections) >= cm.maxConn {
        return nil, errors.New("max connections reached")
    }
    
    // 检查是否已连接
    if conn, exists := cm.connections[peerAddress]; exists && conn.IsActive() {
        return conn, nil
    }
    
    // 建立新连接
    conn := NewConnection(peerAddress, cm.timeout)
    if err := conn.Connect(); err != nil {
        return nil, err
    }
    
    cm.connections[peerAddress] = conn
    return conn, nil
}

func (cm *ConnectionManager) Disconnect(peerAddress Address) error {
    cm.mutex.Lock()
    defer cm.mutex.Unlock()
    
    if conn, exists := cm.connections[peerAddress]; exists {
        conn.Close()
        delete(cm.connections, peerAddress)
    }
    
    return nil
}

func (cm *ConnectionManager) GetConnection(peerAddress Address) *Connection {
    cm.mutex.RLock()
    defer cm.mutex.RUnlock()
    
    if conn, exists := cm.connections[peerAddress]; exists && conn.IsActive() {
        return conn
    }
    
    return nil
}

func (cm *ConnectionManager) Broadcast(message interface{}) error {
    cm.mutex.RLock()
    defer cm.mutex.RUnlock()
    
    var errors []error
    for _, conn := range cm.connections {
        if conn.IsActive() {
            if err := conn.SendMessage(message); err != nil {
                errors = append(errors, err)
            }
        }
    }
    
    if len(errors) > 0 {
        return fmt.Errorf("broadcast errors: %v", errors)
    }
    
    return nil
}

## 协议选择策略

ShardMatrix 网络层根据不同的使用场景选择合适的通信协议：

### Gossip 协议适用场景
- **新区块广播**：快速传播新生成的区块到整个网络
- **交易传播**：广播新交易，确保所有节点都能看到待处理交易
- **状态更新**：传播重要的状态变更信息
- **节点发现**：帮助新节点发现网络中的其他节点

### 直接请求/响应协议适用场景
- **区块同步**：新节点请求历史区块数据
- **状态查询**：查询特定账户或合约的当前状态
- **验证者信息**：获取验证者列表和投票权重
- **连接测试**：测量节点间的网络延迟和连接质量
- **错误恢复**：当 Gossip 传播失败时的备用机制

### 协议选择逻辑
```go
type ProtocolSelector struct {
    gossipProtocol    *GossipProtocol
    p2pProtocol       *P2PProtocol
    config            *NetworkConfig
}

func (ps *ProtocolSelector) SelectProtocol(messageType interface{}, target Address) Protocol {
    switch msg := messageType.(type) {
    case BroadcastMessageType:
        // 广播消息使用 Gossip 协议
        return ps.gossipProtocol
        
    case P2PMessageType:
        // 点对点消息使用 P2P 协议
        return ps.p2pProtocol
        
    default:
        // 根据消息类型判断
        if ps.isBroadcastMessage(messageType) {
            return ps.gossipProtocol
        } else {
            return ps.p2pProtocol
        }
    }
}

func (ps *ProtocolSelector) isBroadcastMessage(messageType interface{}) bool {
    switch messageType {
    case BroadcastTypeBlock, BroadcastTypeTransaction, BroadcastTypeConsensus:
        return true
    default:
        return false
    }
}
```

### 广播协议实现
```go
type GossipProtocol struct {
    nodeID      Address
    neighbors   map[Address]*Connection
    messageLog  map[string]bool // 消息去重
    config      *GossipConfig
    mutex       sync.RWMutex
}

type GossipConfig struct {
    Fanout          int           // 每次传播的邻居数量
    TTL             uint32        // 消息生存时间
    GossipInterval  time.Duration // 传播间隔
    MessageTimeout  time.Duration // 消息超时时间
}

func (gp *GossipProtocol) Broadcast(message *BroadcastMessage) error {
    // 1. 生成消息ID
    message.MessageID = generateMessageID(message)
    message.From = gp.nodeID
    message.Timestamp = time.Now().Unix()
    
    // 2. 签名消息
    message.Signature = gp.signMessage(message)
    
    // 3. 记录消息（去重）
    gp.mutex.Lock()
    if gp.messageLog[message.MessageID] {
        gp.mutex.Unlock()
        return nil // 消息已处理
    }
    gp.messageLog[message.MessageID] = true
    gp.mutex.Unlock()
    
    // 4. 选择邻居节点传播
    neighbors := gp.selectNeighbors(gp.config.Fanout)
    
    // 5. 异步传播消息
    for _, neighbor := range neighbors {
        go gp.sendToNeighbor(neighbor, message)
    }
    
    return nil
}

func (gp *GossipProtocol) HandleMessage(message *BroadcastMessage) error {
    // 1. 验证消息签名
    if !gp.verifyMessage(message) {
        return errors.New("invalid message signature")
    }
    
    // 2. 检查消息是否已处理
    gp.mutex.Lock()
    if gp.messageLog[message.MessageID] {
        gp.mutex.Unlock()
        return nil // 消息已处理
    }
    gp.messageLog[message.MessageID] = true
    gp.mutex.Unlock()
    
    // 3. 处理消息内容
    err := gp.processMessage(message)
    if err != nil {
        return err
    }
    
    // 4. 继续传播（如果TTL > 0）
    if message.TTL > 0 {
        message.TTL--
        neighbors := gp.selectNeighbors(gp.config.Fanout)
        for _, neighbor := range neighbors {
            go gp.sendToNeighbor(neighbor, message)
        }
    }
    
    return nil
}

func (gp *GossipProtocol) selectNeighbors(count int) []*Connection {
    gp.mutex.RLock()
    defer gp.mutex.RUnlock()
    
    neighbors := make([]*Connection, 0, len(gp.neighbors))
    for _, conn := range gp.neighbors {
        if conn.IsActive() {
            neighbors = append(neighbors, conn)
        }
    }
    
    // 随机选择指定数量的邻居
    if len(neighbors) > count {
        rand.Shuffle(len(neighbors), func(i, j int) {
            neighbors[i], neighbors[j] = neighbors[j], neighbors[i]
        })
        neighbors = neighbors[:count]
    }
    
    return neighbors
}
```

### 协议切换机制
当一种协议失败时，系统会自动切换到另一种协议：
- **Gossip 失败回退**：如果 Gossip 传播失败，自动切换到 P2P 协议
- **P2P 请求重试**：P2P 请求失败时，可以重试或切换到其他节点
- **协议健康检查**：定期检查各协议的健康状态，动态调整使用策略
```
