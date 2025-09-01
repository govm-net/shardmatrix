package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/govm-net/shardmatrix/pkg/types"
)

// HealthResponse 健康检查响应
type HealthResponse struct {
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
	Version   string    `json:"version"`
	Uptime    string    `json:"uptime"`
}

// StatusResponse 状态响应
type StatusResponse struct {
	NodeInfo       NodeInfo       `json:"node_info"`
	BlockchainInfo BlockchainInfo `json:"blockchain_info"`
	NetworkInfo    NetworkInfo    `json:"network_info"`
	ConsensusInfo  ConsensusInfo  `json:"consensus_info"`
}

// NodeInfo 节点信息
type NodeInfo struct {
	Version   string    `json:"version"`
	NodeID    string    `json:"node_id"`
	StartTime time.Time `json:"start_time"`
	Uptime    string    `json:"uptime"`
}

// BlockchainInfo 区块链信息
type BlockchainInfo struct {
	ChainID      uint64    `json:"chain_id"`
	LatestHeight uint64    `json:"latest_height"`
	LatestHash   string    `json:"latest_hash"`
	LatestTime   time.Time `json:"latest_time"`
	TotalBlocks  uint64    `json:"total_blocks"`
	TotalTxs     uint64    `json:"total_txs"`
}

// NetworkInfo 网络信息
type NetworkInfo struct {
	PeerCount     int      `json:"peer_count"`
	InboundPeers  int      `json:"inbound_peers"`
	OutboundPeers int      `json:"outbound_peers"`
	ListenAddr    []string `json:"listen_addr"`
}

// ConsensusInfo 共识信息
type ConsensusInfo struct {
	Type             string `json:"type"`
	ValidatorCount   int    `json:"validator_count"`
	ActiveValidators int    `json:"active_validators"`
	BlockInterval    string `json:"block_interval"`
}

// BlockResponse 区块响应
type BlockResponse struct {
	Hash         string                `json:"hash"`
	Height       uint64                `json:"height"`
	PrevHash     string                `json:"prev_hash"`
	MerkleRoot   string                `json:"merkle_root"`
	Timestamp    time.Time             `json:"timestamp"`
	Validator    string                `json:"validator"`
	Signature    string                `json:"signature"`
	TxCount      int                   `json:"tx_count"`
	Transactions []TransactionResponse `json:"transactions"`
	Size         int                   `json:"size"`
}

// TransactionResponse 交易响应
type TransactionResponse struct {
	Hash        string    `json:"hash"`
	From        string    `json:"from"`
	To          string    `json:"to"`
	Amount      uint64    `json:"amount"`
	Fee         uint64    `json:"fee"`
	Nonce       uint64    `json:"nonce"`
	Timestamp   time.Time `json:"timestamp"`
	Status      string    `json:"status"`
	BlockHash   string    `json:"block_hash,omitempty"`
	BlockHeight uint64    `json:"block_height,omitempty"`
}

// AccountResponse 账户响应
type AccountResponse struct {
	Address string `json:"address"`
	Balance uint64 `json:"balance"`
	Nonce   uint64 `json:"nonce"`
	TxCount uint64 `json:"tx_count"`
}

// ValidatorResponse 验证者响应
type ValidatorResponse struct {
	Address        string    `json:"address"`
	Stake          uint64    `json:"stake"`
	Commission     float64   `json:"commission"`
	Status         string    `json:"status"`
	BlocksProduced uint64    `json:"blocks_produced"`
	LastActive     time.Time `json:"last_active"`
}

// handleHealth 健康检查处理器
func (api *APIServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	response := HealthResponse{
		Status:    "ok",
		Timestamp: time.Now(),
		Version:   "1.0.0", // TODO: 从构建信息获取
		Uptime:    "0s",    // TODO: 计算实际运行时间
	}

	api.writeSuccessResponse(w, response)
}

// handleStatus 状态处理器
func (api *APIServer) handleStatus(w http.ResponseWriter, r *http.Request) {
	// 构建状态响应
	response := StatusResponse{
		NodeInfo: NodeInfo{
			Version:   "1.0.0",
			NodeID:    "node-1", // TODO: 从节点获取实际ID
			StartTime: time.Now(),
			Uptime:    "0s",
		},
		BlockchainInfo: api.getBlockchainInfo(),
		NetworkInfo:    api.getNetworkInfo(),
		ConsensusInfo:  api.getConsensusInfo(),
	}

	api.writeSuccessResponse(w, response)
}

// handleGetBlocks 获取区块列表处理器
func (api *APIServer) handleGetBlocks(w http.ResponseWriter, r *http.Request) {
	if api.blockchain == nil {
		api.writeErrorResponse(w, http.StatusServiceUnavailable, "Blockchain not available")
		return
	}

	// 解析查询参数
	page := 1
	limit := 10
	var fromHeight, toHeight uint64

	if pageStr := r.URL.Query().Get("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	// 处理高度范围查询
	if fromStr := r.URL.Query().Get("from"); fromStr != "" {
		if f, err := strconv.ParseUint(fromStr, 10, 64); err == nil {
			fromHeight = f
		}
	}

	if toStr := r.URL.Query().Get("to"); toStr != "" {
		if t, err := strconv.ParseUint(toStr, 10, 64); err == nil {
			toHeight = t
		}
	}

	// 获取区块列表
	var blocks []*types.Block
	var err error
	var total uint64

	if fromHeight > 0 && toHeight > 0 {
		// 按高度范围查询
		blocks, err = api.blockchain.GetBlockRange(fromHeight, toHeight)
		if err != nil {
			api.writeErrorResponse(w, http.StatusInternalServerError, "Failed to get block range", err.Error())
			return
		}
		total = toHeight - fromHeight + 1
	} else {
		// 获取最近的区块
		count := limit
		blocks, err = api.blockchain.GetRecentBlocks(count)
		if err != nil {
			api.writeErrorResponse(w, http.StatusInternalServerError, "Failed to get recent blocks", err.Error())
			return
		}

		// 获取总数
		if stats, err := api.blockchain.GetChainStats(); err == nil {
			total = stats.TotalBlocks
		}
	}

	// 转换为响应格式
	blockResponses := make([]BlockResponse, 0, len(blocks))
	for _, block := range blocks {
		blockResponse := api.convertBlockToResponse(block)
		blockResponses = append(blockResponses, blockResponse)
	}

	response := map[string]interface{}{
		"blocks": blockResponses,
		"pagination": map[string]interface{}{
			"page":     page,
			"limit":    limit,
			"total":    total,
			"has_more": len(blockResponses) == limit,
		},
	}

	api.writeSuccessResponse(w, response)
}

// handleGetLatestBlock 获取最新区块处理器
func (api *APIServer) handleGetLatestBlock(w http.ResponseWriter, r *http.Request) {
	if api.blockchain == nil {
		api.writeErrorResponse(w, http.StatusServiceUnavailable, "Blockchain not available")
		return
	}

	// 获取最新区块
	latestBlock, err := api.blockchain.GetBestBlock()
	if err != nil {
		api.writeErrorResponse(w, http.StatusNotFound, "No blocks found", err.Error())
		return
	}

	// 转换为响应格式
	blockResponse := api.convertBlockToResponse(latestBlock)
	api.writeSuccessResponse(w, blockResponse)
}

// handleGetBlockByHash 根据哈希获取区块处理器
func (api *APIServer) handleGetBlockByHash(w http.ResponseWriter, r *http.Request) {
	hash, err := api.parseHashParam(r, "hash")
	if err != nil {
		api.writeErrorResponse(w, http.StatusBadRequest, "Invalid hash parameter", err.Error())
		return
	}

	if api.blockchain == nil {
		api.writeErrorResponse(w, http.StatusServiceUnavailable, "Blockchain not available")
		return
	}

	// 获取区块
	block, err := api.blockchain.GetBlock(hash)
	if err != nil {
		api.writeErrorResponse(w, http.StatusNotFound, "Block not found", err.Error())
		return
	}

	// 转换为响应格式
	blockResponse := api.convertBlockToResponse(block)
	api.writeSuccessResponse(w, blockResponse)
}

// handleGetBlockByHeight 根据高度获取区块处理器
func (api *APIServer) handleGetBlockByHeight(w http.ResponseWriter, r *http.Request) {
	height, err := api.parseUint64Param(r, "height")
	if err != nil {
		api.writeErrorResponse(w, http.StatusBadRequest, "Invalid height parameter", err.Error())
		return
	}

	if api.blockchain == nil {
		api.writeErrorResponse(w, http.StatusServiceUnavailable, "Blockchain not available")
		return
	}

	// 获取区块
	block, err := api.blockchain.GetBlockByHeight(height)
	if err != nil {
		api.writeErrorResponse(w, http.StatusNotFound, "Block not found", err.Error())
		return
	}

	// 转换为响应格式
	blockResponse := api.convertBlockToResponse(block)
	api.writeSuccessResponse(w, blockResponse)
}

// handleSubmitTransaction 提交交易处理器
func (api *APIServer) handleSubmitTransaction(w http.ResponseWriter, r *http.Request) {
	var txRequest struct {
		From   string `json:"from"`
		To     string `json:"to"`
		Amount uint64 `json:"amount"`
		Fee    uint64 `json:"fee"`
		Nonce  uint64 `json:"nonce"`
		Data   string `json:"data,omitempty"` // 可选的数据字段，hex编码
	}

	if err := json.NewDecoder(r.Body).Decode(&txRequest); err != nil {
		api.writeErrorResponse(w, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	// 验证必需字段
	if txRequest.From == "" {
		api.writeErrorResponse(w, http.StatusBadRequest, "From address is required")
		return
	}
	if txRequest.To == "" {
		api.writeErrorResponse(w, http.StatusBadRequest, "To address is required")
		return
	}
	if txRequest.Amount == 0 {
		api.writeErrorResponse(w, http.StatusBadRequest, "Amount must be greater than 0")
		return
	}
	if txRequest.Fee == 0 {
		api.writeErrorResponse(w, http.StatusBadRequest, "Fee must be greater than 0")
		return
	}

	// 解析地址
	fromAddr, err := types.AddressFromString(txRequest.From)
	if err != nil {
		api.writeErrorResponse(w, http.StatusBadRequest, "Invalid from address", err.Error())
		return
	}

	toAddr, err := types.AddressFromString(txRequest.To)
	if err != nil {
		api.writeErrorResponse(w, http.StatusBadRequest, "Invalid to address", err.Error())
		return
	}

	// 解析可选的数据字段
	var data []byte
	if txRequest.Data != "" {
		// 假设数据是hex编码的
		if len(txRequest.Data) > 2 && txRequest.Data[:2] == "0x" {
			txRequest.Data = txRequest.Data[2:] // 移除0x前缀
		}
		// TODO: 添加hex解码，目前简单处理
		data = []byte(txRequest.Data)
	}

	// 创建交易
	tx := types.NewTransaction(fromAddr, toAddr, txRequest.Amount, txRequest.Fee, txRequest.Nonce, data)

	// TODO: 这里应该进行交易签名，但现在我们跳过签名验证
	// 在生产环境中，客户端应该签名交易，然后API验证签名
	tx.Signature = []byte("placeholder_signature") // 临时占位符

	// 验证交易基本有效性
	if !tx.IsValid() {
		api.writeErrorResponse(w, http.StatusBadRequest, "Transaction is invalid")
		return
	}

	// TODO: 添加到交易池
	// 现在我们只是返回成功响应
	txHash := tx.Hash()

	// 构建响应
	response := map[string]interface{}{
		"tx_hash":   txHash.String(),
		"status":    "pending",
		"message":   "Transaction submitted successfully",
		"timestamp": time.Now(),
	}

	api.writeSuccessResponse(w, response)
}

// handleGetTransactions 获取交易列表处理器
func (api *APIServer) handleGetTransactions(w http.ResponseWriter, r *http.Request) {
	// 解析查询参数
	page := 1
	limit := 10
	var address string
	var status string

	if pageStr := r.URL.Query().Get("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	address = r.URL.Query().Get("address")
	status = r.URL.Query().Get("status")

	// TODO: 从存储层获取交易列表
	// 现在返回空列表
	transactions := make([]TransactionResponse, 0)

	response := map[string]interface{}{
		"transactions": transactions,
		"pagination": map[string]interface{}{
			"page":     page,
			"limit":    limit,
			"total":    0,
			"has_more": false,
		},
		"filters": map[string]interface{}{
			"address": address,
			"status":  status,
		},
	}

	api.writeSuccessResponse(w, response)
}

// handleGetTransactionByHash 根据哈希获取交易处理器
func (api *APIServer) handleGetTransactionByHash(w http.ResponseWriter, r *http.Request) {
	hash, err := api.parseHashParam(r, "hash")
	if err != nil {
		api.writeErrorResponse(w, http.StatusBadRequest, "Invalid hash parameter", err.Error())
		return
	}

	// TODO: 从存储层获取交易
	// 现在返回模拟数据
	if hash.IsZero() {
		api.writeErrorResponse(w, http.StatusNotFound, "Transaction not found")
		return
	}

	// 模拟交易数据
	txResponse := TransactionResponse{
		Hash:        hash.String(),
		From:        "0x1234567890123456789012345678901234567890", // 模拟地址
		To:          "0x0987654321098765432109876543210987654321", // 模拟地址
		Amount:      1000000,
		Fee:         1000,
		Nonce:       1,
		Timestamp:   time.Now().Add(-time.Hour), // 模拟1小时前
		Status:      "confirmed",
		BlockHash:   "0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
		BlockHeight: 100,
	}

	api.writeSuccessResponse(w, txResponse)
}

// handleGetAccount 获取账户处理器
func (api *APIServer) handleGetAccount(w http.ResponseWriter, r *http.Request) {
	address, err := api.parseAddressParam(r, "address")
	if err != nil {
		api.writeErrorResponse(w, http.StatusBadRequest, "Invalid address parameter", err.Error())
		return
	}

	// TODO: 从存储层获取账户信息
	// 现在返回模拟数据
	accountResponse := AccountResponse{
		Address: address.String(),
		Balance: 1000000, // 模拟余额
		Nonce:   5,       // 模拟 nonce
		TxCount: 10,      // 模拟交易数量
	}

	api.writeSuccessResponse(w, accountResponse)
}

// handleGetAccountBalance 获取账户余额处理器
func (api *APIServer) handleGetAccountBalance(w http.ResponseWriter, r *http.Request) {
	address, err := api.parseAddressParam(r, "address")
	if err != nil {
		api.writeErrorResponse(w, http.StatusBadRequest, "Invalid address parameter", err.Error())
		return
	}

	// TODO: 从存储层获取账户余额
	// 现在返回模拟数据
	balanceResponse := map[string]interface{}{
		"address":   address.String(),
		"balance":   uint64(1000000), // 模拟余额
		"timestamp": time.Now(),
	}

	api.writeSuccessResponse(w, balanceResponse)
}

// handleGetAccountTransactions 获取账户交易处理器
func (api *APIServer) handleGetAccountTransactions(w http.ResponseWriter, r *http.Request) {
	address, err := api.parseAddressParam(r, "address")
	if err != nil {
		api.writeErrorResponse(w, http.StatusBadRequest, "Invalid address parameter", err.Error())
		return
	}

	// 解析查询参数
	page := 1
	limit := 10

	if pageStr := r.URL.Query().Get("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	// TODO: 从存储层获取账户交易
	// 现在返回空列表
	transactions := make([]TransactionResponse, 0)

	response := map[string]interface{}{
		"address":      address.String(),
		"transactions": transactions,
		"pagination": map[string]interface{}{
			"page":     page,
			"limit":    limit,
			"total":    0,
			"has_more": false,
		},
	}

	api.writeSuccessResponse(w, response)
}

// handleGetValidators 获取验证者列表处理器
func (api *APIServer) handleGetValidators(w http.ResponseWriter, r *http.Request) {
	// 解析查询参数
	page := 1
	limit := 10
	status := r.URL.Query().Get("status") // active, inactive, all

	if pageStr := r.URL.Query().Get("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	// TODO: 从共识模块获取验证者列表
	// 现在返回模拟数据
	validators := []ValidatorResponse{
		{
			Address:        "0xf5a849e14271805ae7501a4ca4bcba6b8c6b5833",
			Stake:          10000,
			Commission:     0.1,
			Status:         "active",
			BlocksProduced: 100,
			LastActive:     time.Now().Add(-time.Minute * 5),
		},
	}

	// 按状态筛选
	if status != "" && status != "all" {
		filteredValidators := make([]ValidatorResponse, 0)
		for _, v := range validators {
			if v.Status == status {
				filteredValidators = append(filteredValidators, v)
			}
		}
		validators = filteredValidators
	}

	response := map[string]interface{}{
		"validators": validators,
		"pagination": map[string]interface{}{
			"page":     page,
			"limit":    limit,
			"total":    len(validators),
			"has_more": false,
		},
		"filters": map[string]interface{}{
			"status": status,
		},
	}

	api.writeSuccessResponse(w, response)
}

// handleGetValidator 获取验证者处理器
func (api *APIServer) handleGetValidator(w http.ResponseWriter, r *http.Request) {
	address, err := api.parseAddressParam(r, "address")
	if err != nil {
		api.writeErrorResponse(w, http.StatusBadRequest, "Invalid address parameter", err.Error())
		return
	}

	// TODO: 从共识模块获取验证者信息
	// 现在返回模拟数据
	validatorResponse := ValidatorResponse{
		Address:        address.String(),
		Stake:          10000,
		Commission:     0.1,
		Status:         "active",
		BlocksProduced: 100,
		LastActive:     time.Now().Add(-time.Minute * 5),
	}

	api.writeSuccessResponse(w, validatorResponse)
}

// handleGetValidatorDelegators 获取验证者委托者处理器
func (api *APIServer) handleGetValidatorDelegators(w http.ResponseWriter, r *http.Request) {
	address, err := api.parseAddressParam(r, "address")
	if err != nil {
		api.writeErrorResponse(w, http.StatusBadRequest, "Invalid address parameter", err.Error())
		return
	}

	// 解析查询参数
	page := 1
	limit := 10

	if pageStr := r.URL.Query().Get("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	// TODO: 从共识模块获取委托者列表
	// 现在返回空列表
	delegators := make([]map[string]interface{}, 0)

	response := map[string]interface{}{
		"validator_address": address.String(),
		"delegators":        delegators,
		"total_delegated":   uint64(0),
		"pagination": map[string]interface{}{
			"page":     page,
			"limit":    limit,
			"total":    0,
			"has_more": false,
		},
	}

	api.writeSuccessResponse(w, response)
}

// handleGetNetworkInfo 获取网络信息处理器
func (api *APIServer) handleGetNetworkInfo(w http.ResponseWriter, r *http.Request) {
	info := api.getNetworkInfo()
	api.writeSuccessResponse(w, info)
}

// handleGetPeers 获取节点列表处理器
func (api *APIServer) handleGetPeers(w http.ResponseWriter, r *http.Request) {
	// 解析查询参数
	page := 1
	limit := 10
	status := r.URL.Query().Get("status") // connected, disconnected, all

	if pageStr := r.URL.Query().Get("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	// TODO: 从网络模块获取实际的节点列表
	// 现在返回模拟数据
	peers := []map[string]interface{}{
		{
			"peer_id":      "12D3KooWExample1",
			"address":      "/ip4/127.0.0.1/tcp/8080",
			"status":       "connected",
			"connected_at": time.Now().Add(-time.Hour),
			"latency":      "50ms",
			"version":      "1.0.0",
		},
	}

	// 按状态筛选
	if status != "" && status != "all" {
		filteredPeers := make([]map[string]interface{}, 0)
		for _, peer := range peers {
			if peer["status"] == status {
				filteredPeers = append(filteredPeers, peer)
			}
		}
		peers = filteredPeers
	}

	response := map[string]interface{}{
		"peers": peers,
		"pagination": map[string]interface{}{
			"page":     page,
			"limit":    limit,
			"total":    len(peers),
			"has_more": false,
		},
		"filters": map[string]interface{}{
			"status": status,
		},
		"summary": map[string]interface{}{
			"total_peers":        len(peers),
			"connected_peers":    1,
			"disconnected_peers": 0,
		},
	}

	api.writeSuccessResponse(w, response)
}

// handleGetNetworkStats 获取网络统计处理器
func (api *APIServer) handleGetNetworkStats(w http.ResponseWriter, r *http.Request) {
	// TODO: 从网络模块获取实际的网络统计
	// 现在返回模拟数据
	stats := map[string]interface{}{
		"network_info": map[string]interface{}{
			"node_id": "12D3KooWDDsqVJ8ZekJaBhK8hZiU7steYxVFkeJxyJHBudY4ckNf",
			"listen_addresses": []string{
				"/ip4/127.0.0.1/tcp/8080",
				"/ip4/192.168.3.25/tcp/8080",
			},
			"protocol_version": "1.0.0",
			"network_id":       "shardmatrix",
		},
		"connection_stats": map[string]interface{}{
			"total_peers":        1,
			"connected_peers":    1,
			"inbound_peers":      0,
			"outbound_peers":     1,
			"max_peers":          50,
			"connection_rate":    "1.0/min",
			"disconnection_rate": "0.0/min",
		},
		"message_stats": map[string]interface{}{
			"messages_sent":     100,
			"messages_received": 95,
			"bytes_sent":        1024000,
			"bytes_received":    980000,
			"message_rate":      "10.5/sec",
			"bandwidth_usage":   "50KB/s",
		},
		"discovery_stats": map[string]interface{}{
			"dht_peers":          10,
			"mdns_discoveries":   5,
			"bootstrap_peers":    3,
			"discovery_rate":     "2.0/min",
			"routing_table_size": 8,
		},
		"health_metrics": map[string]interface{}{
			"uptime":          "2h30m45s",
			"last_block_time": time.Now().Add(-time.Second * 2),
			"network_latency": "25ms",
			"sync_status":     "synced",
			"error_rate":      "0.1%",
		},
		"timestamp": time.Now(),
	}

	api.writeSuccessResponse(w, stats)
}

// handleGetConsensusInfo 获取共识信息处理器
func (api *APIServer) handleGetConsensusInfo(w http.ResponseWriter, r *http.Request) {
	info := api.getConsensusInfo()
	api.writeSuccessResponse(w, info)
}

// handleGetConsensusValidators 获取共识验证者处理器
func (api *APIServer) handleGetConsensusValidators(w http.ResponseWriter, r *http.Request) {
	// TODO: 从共识模块获取当前活跃验证者
	// 现在返回模拟数据
	validators := map[string]interface{}{
		"current_epoch":    28334458,
		"epoch_start_time": time.Now().Add(-time.Hour),
		"epoch_duration":   "1h",
		"active_validators": []map[string]interface{}{
			{
				"address":         "0xd839ecfa599e5ed329c3905870c484695e17a509",
				"stake":           10000,
				"voting_power":    1.0,
				"status":          "active",
				"blocks_produced": 50,
				"blocks_missed":   0,
				"uptime":          "100%",
				"last_active":     time.Now().Add(-time.Minute * 2),
			},
		},
		"validator_stats": map[string]interface{}{
			"total_validators":    1,
			"active_validators":   1,
			"inactive_validators": 0,
			"jailed_validators":   0,
			"total_stake":         10000,
			"average_uptime":      "100%",
		},
		"next_epoch": map[string]interface{}{
			"epoch_number":  28334459,
			"start_time":    time.Now().Add(time.Hour),
			"validator_set": 1,
			"stake_changes": 0,
		},
	}

	api.writeSuccessResponse(w, validators)
}

// 辅助方法

// convertBlockToResponse 将区块转换为响应格式
func (api *APIServer) convertBlockToResponse(block *types.Block) BlockResponse {
	if block == nil {
		return BlockResponse{}
	}

	response := BlockResponse{
		Hash:       block.Hash().String(),
		Height:     block.Header.Number,
		PrevHash:   block.Header.PrevHash.String(),
		MerkleRoot: block.Header.TxRoot.String(),
		Timestamp:  time.Unix(block.Header.Timestamp, 0),
		Validator:  block.Header.Validator.String(),
		TxCount:    len(block.Transactions),
		Size:       0, // TODO: 计算区块大小
	}

	// 转换交易哈希列表
	for _, txHash := range block.Transactions {
		txResponse := TransactionResponse{
			Hash:        txHash.String(),
			From:        "", // 需要从存储层获取完整交易
			To:          "", // 需要从存储层获取完整交易
			Amount:      0,  // 需要从存储层获取完整交易
			Fee:         0,  // 需要从存储层获取完整交易
			Nonce:       0,  // 需要从存储层获取完整交易
			Timestamp:   time.Unix(block.Header.Timestamp, 0),
			Status:      "confirmed",
			BlockHash:   block.Hash().String(),
			BlockHeight: block.Header.Number,
		}
		response.Transactions = append(response.Transactions, txResponse)
	}

	return response
}

// getBlockchainInfo 获取区块链信息
func (api *APIServer) getBlockchainInfo() BlockchainInfo {
	info := BlockchainInfo{
		ChainID:     1, // TODO: 从配置获取
		TotalBlocks: 0,
		TotalTxs:    0,
	}

	if api.blockchain != nil {
		if latestBlock, err := api.blockchain.GetBestBlock(); err == nil {
			info.LatestHeight = latestBlock.Header.Number
			info.LatestHash = latestBlock.Hash().String()
			info.LatestTime = time.Unix(latestBlock.Header.Timestamp, 0)
		}

		// TODO: 获取统计信息
		stats, err := api.blockchain.GetChainStats()
		if err == nil && stats != nil {
			info.TotalBlocks = stats.TotalBlocks
			info.TotalTxs = stats.TotalTransactions
		}
	}

	return info
}

// getNetworkInfo 获取网络信息
func (api *APIServer) getNetworkInfo() NetworkInfo {
	info := NetworkInfo{
		PeerCount:     0,
		InboundPeers:  0,
		OutboundPeers: 0,
		ListenAddr:    []string{},
	}

	// TODO: 从网络模块获取实际信息
	if api.config != nil {
		addr := fmt.Sprintf("%s:%d", api.config.Network.Host, api.config.Network.Port)
		info.ListenAddr = append(info.ListenAddr, addr)
	}

	return info
}

// getConsensusInfo 获取共识信息
func (api *APIServer) getConsensusInfo() ConsensusInfo {
	info := ConsensusInfo{
		Type:             "dpos",
		ValidatorCount:   0,
		ActiveValidators: 0,
		BlockInterval:    "2s",
	}

	if api.config != nil {
		info.ValidatorCount = api.config.Consensus.ValidatorCount
		info.BlockInterval = fmt.Sprintf("%ds", api.config.Blockchain.BlockInterval)
	}

	// TODO: 获取活跃验证者数量

	return info
}

// handleGetNetworkHealth 获取网络健康状态处理器
func (api *APIServer) handleGetNetworkHealth(w http.ResponseWriter, r *http.Request) {
	// TODO: 从网络模块获取实际的健康状态
	// 现在返回模拟数据
	health := map[string]interface{}{
		"status": "healthy",
		"checks": map[string]interface{}{
			"connectivity": map[string]interface{}{
				"status":  "pass",
				"message": "All network connections are stable",
				"details": map[string]interface{}{
					"connected_peers": 1,
					"target_peers":    3,
					"success_rate":    "100%",
				},
			},
			"latency": map[string]interface{}{
				"status":  "pass",
				"message": "Network latency is within acceptable range",
				"details": map[string]interface{}{
					"average_latency": "25ms",
					"max_latency":     "50ms",
					"threshold":       "100ms",
				},
			},
			"discovery": map[string]interface{}{
				"status":  "pass",
				"message": "Node discovery is working properly",
				"details": map[string]interface{}{
					"dht_peers":          10,
					"mdns_discoveries":   5,
					"bootstrap_success":  true,
					"routing_table_size": 8,
				},
			},
			"sync": map[string]interface{}{
				"status":  "pass",
				"message": "Node is fully synchronized",
				"details": map[string]interface{}{
					"sync_status":     "synced",
					"last_block_time": time.Now().Add(-time.Second * 2),
					"block_lag":       0,
					"sync_progress":   "100%",
				},
			},
		},
		"overall_score": 100,
		"timestamp":     time.Now(),
		"uptime":        "2h30m45s",
	}

	api.writeSuccessResponse(w, health)
}
