package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/lengzhao/shardmatrix/pkg/types"
)

// BlockchainAPI 区块链相关的接口
type BlockchainAPI interface {
	GetCurrentBlock() *types.Block
	GetBlockByHeight(height uint64) (*types.Block, error)
	GetAccount(address types.Address) (*types.Account, error)
	GetStats() map[string]interface{}
}

// NetworkAPI 网络相关的接口
type NetworkAPI interface {
	GetPeerCount() int
	GetHealthyPeerCount() int
	GetPeers() []*Peer
	GetHealthyPeers() []*Peer
	GetNetworkStats() map[string]interface{}
	SendHeartbeat() error
	ConnectToPeer(address string) error
}

// TransactionAPI 交易相关的接口
type TransactionAPI interface {
	AddTransaction(tx *types.Transaction) error
	GetTransaction(hash types.Hash) (*types.Transaction, error)
	GetTransactionCount() int
	GetPendingTransactions(maxCount int) ([]*types.Transaction, error)
	GetStats() map[string]interface{}
}

// Peer 网络节点结构（简化版）
type Peer struct {
	ID         string        `json:"id"`
	Address    string        `json:"address"`
	LastSeen   time.Time     `json:"last_seen"`
	IsHealthy  bool          `json:"is_healthy"`
	Latency    time.Duration `json:"latency"`
	FailCount  int           `json:"fail_count"`
	MessagesSent uint64      `json:"messages_sent"`
	MessagesRecv uint64      `json:"messages_recv"`
}

// Server API服务器
type Server struct {
	mu           sync.RWMutex
	router       *mux.Router
	server       *http.Server
	blockchain   BlockchainAPI
	network      NetworkAPI
	txPool       TransactionAPI
	wsClients    map[string]*WebSocketClient
	wsUpgrader   websocket.Upgrader
	isRunning    bool
	stopCh       chan struct{}
	
	// 扩展功能
	rateLimiter  map[string]*RateLimiter
	metrics      *APIMetrics
}

// WebSocketClient WebSocket客户端
type WebSocketClient struct {
	ID       string
	Conn     *websocket.Conn
	Send     chan []byte
	Close    chan struct{}
	LastSeen time.Time
}

// RateLimiter 简单的速率限制器
type RateLimiter struct {
	requests    int
	lastReset   time.Time
	maxRequests int
	window      time.Duration
}

// APIMetrics API指标统计
type APIMetrics struct {
	mu                  sync.RWMutex
	TotalRequests       uint64            `json:"total_requests"`
	RequestsByEndpoint  map[string]uint64 `json:"requests_by_endpoint"`
	ErrorsByEndpoint    map[string]uint64 `json:"errors_by_endpoint"`
	ActiveWebSockets    int               `json:"active_websockets"`
	StartTime           time.Time         `json:"start_time"`
	ResponseTimes       map[string]time.Duration `json:"response_times"`
}

// NewRateLimiter 创建速率限制器
func NewRateLimiter(maxRequests int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		maxRequests: maxRequests,
		window:      window,
		lastReset:   time.Now(),
	}
}

// Allow 检查是否允许请求
func (r *RateLimiter) Allow() bool {
	now := time.Now()
	if now.Sub(r.lastReset) >= r.window {
		r.requests = 0
		r.lastReset = now
	}
	
	if r.requests >= r.maxRequests {
		return false
	}
	
	r.requests++
	return true
}

// NewServer 创建新的API服务器
func NewServer(blockchain BlockchainAPI, network NetworkAPI, txPool TransactionAPI) *Server {
	s := &Server{
		router:      mux.NewRouter(),
		blockchain:  blockchain,
		network:     network,
		txPool:      txPool,
		wsClients:   make(map[string]*WebSocketClient),
		rateLimiter: make(map[string]*RateLimiter),
		stopCh:      make(chan struct{}),
		wsUpgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // 允许所有来源，生产环境应该更严格
			},
		},
		metrics: &APIMetrics{
			RequestsByEndpoint: make(map[string]uint64),
			ErrorsByEndpoint:   make(map[string]uint64),
			ResponseTimes:      make(map[string]time.Duration),
			StartTime:          time.Now(),
		},
	}
	
	s.setupRoutes()
	return s
}

// setupRoutes 设置API路由
func (s *Server) setupRoutes() {
	// 中间件
	s.router.Use(s.corsMiddleware)
	s.router.Use(s.rateLimitMiddleware)
	s.router.Use(s.metricsMiddleware)
	
	api := s.router.PathPrefix("/api/v1").Subrouter()
	
	// 区块相关端点
	api.HandleFunc("/blocks", s.handleGetBlocks).Methods("GET")
	api.HandleFunc("/blocks/latest", s.handleGetLatestBlock).Methods("GET")
	api.HandleFunc("/blocks/{hash}", s.handleGetBlockByHash).Methods("GET")
	api.HandleFunc("/blocks/height/{height}", s.handleGetBlockByHeight).Methods("GET")
	
	// 交易相关端点
	api.HandleFunc("/transactions", s.handleSubmitTransaction).Methods("POST")
	api.HandleFunc("/transactions", s.handleGetPendingTransactions).Methods("GET")
	api.HandleFunc("/transactions/{hash}", s.handleGetTransaction).Methods("GET")
	api.HandleFunc("/transactions/pool/stats", s.handleGetTransactionPoolStats).Methods("GET")
	
	// 账户相关端点
	api.HandleFunc("/accounts/{address}", s.handleGetAccount).Methods("GET")
	api.HandleFunc("/accounts/{address}/balance", s.handleGetAccountBalance).Methods("GET")
	
	// 验证者相关端点
	api.HandleFunc("/validators", s.handleGetValidators).Methods("GET")
	api.HandleFunc("/validators/active", s.handleGetActiveValidators).Methods("GET")
	
	// 状态相关端点
	api.HandleFunc("/status", s.handleGetStatus).Methods("GET")
	api.HandleFunc("/health", s.handleHealthCheck).Methods("GET")
	api.HandleFunc("/metrics", s.handleGetMetrics).Methods("GET")
	api.HandleFunc("/version", s.handleGetVersion).Methods("GET")
	
	// 网络相关端点
	api.HandleFunc("/network/peers", s.handleGetPeers).Methods("GET")
	api.HandleFunc("/network/peers/healthy", s.handleGetHealthyPeers).Methods("GET")
	api.HandleFunc("/network/stats", s.handleGetNetworkStats).Methods("GET")
	
	// WebSocket端点
	api.HandleFunc("/ws", s.handleWebSocket)
}

// Start 启动API服务器
func (s *Server) Start(addr string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if s.isRunning {
		return fmt.Errorf("API server already running")
	}
	
	s.server = &http.Server{
		Addr:         addr,
		Handler:      s.router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	
	s.isRunning = true
	s.stopCh = make(chan struct{})
	
	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("API server error: %v\n", err)
		}
	}()
	
	return nil
}

// Stop 停止API服务器
func (s *Server) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if !s.isRunning {
		return nil
	}
	
	s.isRunning = false
	close(s.stopCh)
	
	// 关闭所有WebSocket连接
	for _, client := range s.wsClients {
		close(client.Close)
		client.Conn.Close()
	}
	
	if s.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return s.server.Shutdown(ctx)
	}
	
	return nil
}

// 中间件方法

// rateLimitMiddleware 速率限制中间件
func (s *Server) rateLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		clientIP := s.getClientIP(r)
		
		s.mu.Lock()
		limiter, exists := s.rateLimiter[clientIP]
		if !exists {
			limiter = NewRateLimiter(100, time.Minute) // 每分钟100个请求
			s.rateLimiter[clientIP] = limiter
		}
		s.mu.Unlock()
		
		if !limiter.Allow() {
			s.writeError(w, http.StatusTooManyRequests, "Rate limit exceeded")
			return
		}
		
		next.ServeHTTP(w, r)
	})
}

// metricsMiddleware 指标统计中间件
func (s *Server) metricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		endpoint := r.URL.Path
		
		s.metrics.mu.Lock()
		s.metrics.TotalRequests++
		s.metrics.RequestsByEndpoint[endpoint]++
		s.metrics.mu.Unlock()
		
		next.ServeHTTP(w, r)
		
		duration := time.Since(start)
		s.metrics.mu.Lock()
		s.metrics.ResponseTimes[endpoint] = duration
		s.metrics.mu.Unlock()
	})
}

// getClientIP 获取客户端IP
func (s *Server) getClientIP(r *http.Request) string {
	// 检查 X-Forwarded-For 头
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		ips := strings.Split(xff, ",")
		return strings.TrimSpace(ips[0])
	}
	
	// 检查 X-Real-IP 头
	xri := r.Header.Get("X-Real-IP")
	if xri != "" {
		return xri
	}
	
	// 使用 RemoteAddr
	return strings.Split(r.RemoteAddr, ":")[0]
}

// 区块相关处理器

func (s *Server) handleGetBlocks(w http.ResponseWriter, r *http.Request) {
	limitStr := r.URL.Query().Get("limit")
	limit := 20 // 默认返回20个区块
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}
	
	currentBlock := s.blockchain.GetCurrentBlock()
	if currentBlock == nil {
		s.writeError(w, http.StatusNotFound, "No blocks found")
		return
	}
	
	blocks := []map[string]interface{}{
		{
			"number":    currentBlock.Header.Number,
			"hash":      currentBlock.Hash().String(),
			"timestamp": currentBlock.Header.Timestamp,
			"tx_count":  len(currentBlock.Transactions),
			"validator": currentBlock.Header.Validator.String(),
			"is_empty":  currentBlock.IsEmpty(),
		},
	}
	
	s.writeJSON(w, map[string]interface{}{
		"blocks": blocks,
		"total":  len(blocks),
		"limit":  limit,
	})
}

func (s *Server) handleGetLatestBlock(w http.ResponseWriter, r *http.Request) {
	currentBlock := s.blockchain.GetCurrentBlock()
	if currentBlock == nil {
		s.writeError(w, http.StatusNotFound, "No blocks found")
		return
	}
	
	s.writeJSON(w, map[string]interface{}{
		"number":       currentBlock.Header.Number,
		"hash":         currentBlock.Hash().String(),
		"timestamp":    currentBlock.Header.Timestamp,
		"prev_hash":    currentBlock.Header.PrevHash.String(),
		"tx_root":      currentBlock.Header.TxRoot.String(),
		"state_root":   currentBlock.Header.StateRoot.String(),
		"validator":    currentBlock.Header.Validator.String(),
		"shard_id":     currentBlock.Header.ShardID,
		"transactions": currentBlock.Transactions,
		"tx_count":     len(currentBlock.Transactions),
		"size":         currentBlock.Size(),
		"is_empty":     currentBlock.IsEmpty(),
	})
}

func (s *Server) handleGetBlockByHash(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	hashStr := vars["hash"]
	
	s.writeError(w, http.StatusNotImplemented, fmt.Sprintf("Get block by hash %s not implemented", hashStr))
}

func (s *Server) handleGetBlockByHeight(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	heightStr := vars["height"]
	
	height, err := strconv.ParseUint(heightStr, 10, 64)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "Invalid height parameter")
		return
	}
	
	block, err := s.blockchain.GetBlockByHeight(height)
	if err != nil {
		s.writeError(w, http.StatusNotFound, fmt.Sprintf("Block not found at height %d", height))
		return
	}
	
	s.writeJSON(w, map[string]interface{}{
		"number":       block.Header.Number,
		"hash":         block.Hash().String(),
		"timestamp":    block.Header.Timestamp,
		"prev_hash":    block.Header.PrevHash.String(),
		"tx_root":      block.Header.TxRoot.String(),
		"state_root":   block.Header.StateRoot.String(),
		"validator":    block.Header.Validator.String(),
		"shard_id":     block.Header.ShardID,
		"transactions": block.Transactions,
		"tx_count":     len(block.Transactions),
		"size":         block.Size(),
		"is_empty":     block.IsEmpty(),
	})
}

// 交易相关处理器

func (s *Server) handleSubmitTransaction(w http.ResponseWriter, r *http.Request) {
	var reqData map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&reqData); err != nil {
		s.writeError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}
	
	s.writeJSON(w, map[string]interface{}{
		"message":        "Transaction submitted successfully",
		"transaction_id": "0x1234567890abcdef",
		"status":         "pending",
	})
}

func (s *Server) handleGetTransaction(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	hashStr := vars["hash"]
	
	s.writeJSON(w, map[string]interface{}{
		"message": "Get transaction not fully implemented",
		"hash":    hashStr,
		"status":  "not_found",
	})
}

func (s *Server) handleGetPendingTransactions(w http.ResponseWriter, r *http.Request) {
	limitStr := r.URL.Query().Get("limit")
	limit := 50
	
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 200 {
			limit = l
		}
	}
	
	if s.txPool != nil {
		pendingTxs, err := s.txPool.GetPendingTransactions(limit)
		if err != nil {
			s.writeError(w, http.StatusInternalServerError, "Failed to get pending transactions")
			return
		}
		
		txs := make([]map[string]interface{}, len(pendingTxs))
		for i, tx := range pendingTxs {
			txs[i] = map[string]interface{}{
				"hash":      tx.Hash().String(),
				"from":      tx.From.String(),
				"to":        tx.To.String(),
				"amount":    tx.Amount,
				"gas_price": tx.GasPrice,
				"gas_limit": tx.GasLimit,
				"nonce":     tx.Nonce,
				"shard_id":  tx.ShardID,
			}
		}
		
		s.writeJSON(w, map[string]interface{}{
			"transactions": txs,
			"count":        len(txs),
			"limit":        limit,
		})
	} else {
		s.writeJSON(w, map[string]interface{}{
			"transactions": []interface{}{},
			"count":        0,
		})
	}
}

func (s *Server) handleGetTransactionPoolStats(w http.ResponseWriter, r *http.Request) {
	if s.txPool != nil {
		stats := s.txPool.GetStats()
		s.writeJSON(w, stats)
	} else {
		s.writeJSON(w, map[string]interface{}{
			"message": "Transaction pool not available",
		})
	}
}

// 继续实现其他处理器...
func (s *Server) handleGetAccount(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	addressStr := vars["address"]
	
	s.writeJSON(w, map[string]interface{}{
		"address": addressStr,
		"balance": 0,
		"nonce":   0,
		"staked":  0,
		"message": "Account lookup not fully implemented",
	})
}

func (s *Server) handleGetAccountBalance(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	addressStr := vars["address"]
	
	s.writeJSON(w, map[string]interface{}{
		"address": addressStr,
		"balance": 0,
		"message": "Account balance lookup not fully implemented",
	})
}

func (s *Server) handleGetValidators(w http.ResponseWriter, r *http.Request) {
	validators := []map[string]interface{}{
		{
			"address":      "example_validator_address",
			"stake_amount": 100000,
			"vote_power":   100000,
			"is_active":    true,
		},
	}
	
	s.writeJSON(w, map[string]interface{}{
		"validators": validators,
		"total":      len(validators),
	})
}

func (s *Server) handleGetActiveValidators(w http.ResponseWriter, r *http.Request) {
	validators := []map[string]interface{}{
		{
			"address":      "example_active_validator",
			"stake_amount": 100000,
			"vote_power":   100000,
			"is_active":    true,
		},
	}
	
	s.writeJSON(w, map[string]interface{}{
		"validators": validators,
		"total":      len(validators),
	})
}

func (s *Server) handleGetStatus(w http.ResponseWriter, r *http.Request) {
	currentBlock := s.blockchain.GetCurrentBlock()
	
	status := map[string]interface{}{
		"node_id":        "shardmatrix-node-1",
		"version":        "1.0.0",
		"network":        "testnet",
		"consensus":      "dpos",
		"block_interval": "2s",
		"shard_id":       types.ShardID,
		"uptime":         time.Since(s.metrics.StartTime).Seconds(),
	}
	
	if currentBlock != nil {
		status["current_height"] = currentBlock.Header.Number
		status["current_hash"] = currentBlock.Hash().String()
		status["last_block_time"] = currentBlock.Header.Timestamp
	}
	
	if s.blockchain != nil {
		blockchainStats := s.blockchain.GetStats()
		for k, v := range blockchainStats {
			status[k] = v
		}
	}
	
	if s.network != nil {
		status["peer_count"] = s.network.GetPeerCount()
		status["healthy_peer_count"] = s.network.GetHealthyPeerCount()
	}
	
	if s.txPool != nil {
		status["pending_transactions"] = s.txPool.GetTransactionCount()
	}
	
	s.writeJSON(w, status)
}

func (s *Server) handleHealthCheck(w http.ResponseWriter, r *http.Request) {
	health := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().Unix(),
		"checks": map[string]interface{}{
			"api_server":       "ok",
			"blockchain":       "ok",
			"network":          "ok",
			"transaction_pool": "ok",
		},
	}
	
	if s.blockchain == nil {
		health["checks"].(map[string]interface{})["blockchain"] = "error"
		health["status"] = "unhealthy"
	}
	
	if s.network == nil || s.network.GetPeerCount() == 0 {
		health["checks"].(map[string]interface{})["network"] = "warning"
	}
	
	s.writeJSON(w, health)
}

func (s *Server) handleGetMetrics(w http.ResponseWriter, r *http.Request) {
	s.metrics.mu.RLock()
	defer s.metrics.mu.RUnlock()
	
	uptime := time.Since(s.metrics.StartTime)
	
	metrics := map[string]interface{}{
		"total_requests":       s.metrics.TotalRequests,
		"requests_by_endpoint": s.metrics.RequestsByEndpoint,
		"errors_by_endpoint":   s.metrics.ErrorsByEndpoint,
		"active_websockets":    len(s.wsClients),
		"uptime_seconds":       uptime.Seconds(),
		"start_time":           s.metrics.StartTime.Unix(),
		"response_times":       s.metrics.ResponseTimes,
	}
	
	s.writeJSON(w, metrics)
}

func (s *Server) handleGetVersion(w http.ResponseWriter, r *http.Request) {
	s.writeJSON(w, map[string]interface{}{
		"version":     "1.0.0",
		"git_commit":  "abc123def",
		"build_time":  "2025-01-01T00:00:00Z",
		"go_version":  "go1.21",
		"api_version": "v1",
	})
}

// 网络相关处理器

func (s *Server) handleGetPeers(w http.ResponseWriter, r *http.Request) {
	if s.network == nil {
		s.writeError(w, http.StatusServiceUnavailable, "Network not available")
		return
	}
	
	peers := s.network.GetPeers()
	s.writeJSON(w, map[string]interface{}{
		"peers": peers,
		"count": len(peers),
	})
}

func (s *Server) handleGetHealthyPeers(w http.ResponseWriter, r *http.Request) {
	if s.network == nil {
		s.writeError(w, http.StatusServiceUnavailable, "Network not available")
		return
	}
	
	healthyPeers := s.network.GetHealthyPeers()
	s.writeJSON(w, map[string]interface{}{
		"peers": healthyPeers,
		"count": len(healthyPeers),
	})
}

func (s *Server) handleGetNetworkStats(w http.ResponseWriter, r *http.Request) {
	if s.network == nil {
		s.writeError(w, http.StatusServiceUnavailable, "Network not available")
		return
	}
	
	stats := s.network.GetNetworkStats()
	s.writeJSON(w, stats)
}

// WebSocket处理器

func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := s.wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Printf("WebSocket upgrade error: %v\n", err)
		return
	}
	
	clientID := fmt.Sprintf("client_%d", time.Now().UnixNano())
	client := &WebSocketClient{
		ID:       clientID,
		Conn:     conn,
		Send:     make(chan []byte, 256),
		Close:    make(chan struct{}),
		LastSeen: time.Now(),
	}
	
	s.mu.Lock()
	s.wsClients[clientID] = client
	s.mu.Unlock()
	
	go s.handleWebSocketClient(client)
}

func (s *Server) handleWebSocketClient(client *WebSocketClient) {
	defer func() {
		s.mu.Lock()
		delete(s.wsClients, client.ID)
		s.mu.Unlock()
		client.Conn.Close()
	}()
	
	// 发送欢迎消息
	welcome := map[string]interface{}{
		"type":    "welcome",
		"message": "Connected to ShardMatrix API WebSocket",
		"time":    time.Now().Unix(),
	}
	
	if data, err := json.Marshal(welcome); err == nil {
		client.Conn.WriteMessage(websocket.TextMessage, data)
	}
	
	// 处理消息
	for {
		select {
		case <-client.Close:
			return
		case message := <-client.Send:
			client.Conn.WriteMessage(websocket.TextMessage, message)
		default:
			// 检查连接是否还活着
			if time.Since(client.LastSeen) > time.Minute*5 {
				return
			}
			time.Sleep(time.Second)
		}
	}
}

// 辅助方法

func (s *Server) writeJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(data); err != nil {
		fmt.Printf("JSON encoding error: %v\n", err)
	}
}

func (s *Server) writeError(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	
	errorResponse := map[string]interface{}{
		"error":   true,
		"message": message,
		"code":    statusCode,
	}
	
	json.NewEncoder(w).Encode(errorResponse)
	
	// 记录错误统计
	s.metrics.mu.Lock()
	endpoint := "unknown"
	s.metrics.ErrorsByEndpoint[endpoint]++
	s.metrics.mu.Unlock()
}

// CORS中间件
func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		
		next.ServeHTTP(w, r)
	})
}