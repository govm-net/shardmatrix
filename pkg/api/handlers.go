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
	}

	if err := json.NewDecoder(r.Body).Decode(&txRequest); err != nil {
		api.writeErrorResponse(w, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	// TODO: 验证和处理交易
	api.writeErrorResponse(w, http.StatusNotImplemented, "Transaction submission not implemented")
}

// handleGetTransactions 获取交易列表处理器
func (api *APIServer) handleGetTransactions(w http.ResponseWriter, r *http.Request) {
	// TODO: 实现交易列表查询
	api.writeErrorResponse(w, http.StatusNotImplemented, "Transaction list not implemented")
}

// handleGetTransactionByHash 根据哈希获取交易处理器
func (api *APIServer) handleGetTransactionByHash(w http.ResponseWriter, r *http.Request) {
	// TODO: 实现交易查询
	api.writeErrorResponse(w, http.StatusNotImplemented, "Transaction query not implemented")
}

// handleGetAccount 获取账户处理器
func (api *APIServer) handleGetAccount(w http.ResponseWriter, r *http.Request) {
	// TODO: 实现账户查询
	api.writeErrorResponse(w, http.StatusNotImplemented, "Account query not implemented")
}

// handleGetAccountBalance 获取账户余额处理器
func (api *APIServer) handleGetAccountBalance(w http.ResponseWriter, r *http.Request) {
	// TODO: 实现余额查询
	api.writeErrorResponse(w, http.StatusNotImplemented, "Balance query not implemented")
}

// handleGetAccountTransactions 获取账户交易处理器
func (api *APIServer) handleGetAccountTransactions(w http.ResponseWriter, r *http.Request) {
	// TODO: 实现账户交易查询
	api.writeErrorResponse(w, http.StatusNotImplemented, "Account transactions not implemented")
}

// handleGetValidators 获取验证者列表处理器
func (api *APIServer) handleGetValidators(w http.ResponseWriter, r *http.Request) {
	// TODO: 实现验证者列表查询
	api.writeErrorResponse(w, http.StatusNotImplemented, "Validators list not implemented")
}

// handleGetValidator 获取验证者处理器
func (api *APIServer) handleGetValidator(w http.ResponseWriter, r *http.Request) {
	// TODO: 实现验证者查询
	api.writeErrorResponse(w, http.StatusNotImplemented, "Validator query not implemented")
}

// handleGetValidatorDelegators 获取验证者委托者处理器
func (api *APIServer) handleGetValidatorDelegators(w http.ResponseWriter, r *http.Request) {
	// TODO: 实现委托者查询
	api.writeErrorResponse(w, http.StatusNotImplemented, "Validator delegators not implemented")
}

// handleGetNetworkInfo 获取网络信息处理器
func (api *APIServer) handleGetNetworkInfo(w http.ResponseWriter, r *http.Request) {
	info := api.getNetworkInfo()
	api.writeSuccessResponse(w, info)
}

// handleGetPeers 获取节点列表处理器
func (api *APIServer) handleGetPeers(w http.ResponseWriter, r *http.Request) {
	// TODO: 实现节点列表查询
	api.writeErrorResponse(w, http.StatusNotImplemented, "Peers list not implemented")
}

// handleGetNetworkStats 获取网络统计处理器
func (api *APIServer) handleGetNetworkStats(w http.ResponseWriter, r *http.Request) {
	// TODO: 实现网络统计查询
	api.writeErrorResponse(w, http.StatusNotImplemented, "Network stats not implemented")
}

// handleGetConsensusInfo 获取共识信息处理器
func (api *APIServer) handleGetConsensusInfo(w http.ResponseWriter, r *http.Request) {
	info := api.getConsensusInfo()
	api.writeSuccessResponse(w, info)
}

// handleGetConsensusValidators 获取共识验证者处理器
func (api *APIServer) handleGetConsensusValidators(w http.ResponseWriter, r *http.Request) {
	// TODO: 实现共识验证者查询
	api.writeErrorResponse(w, http.StatusNotImplemented, "Consensus validators not implemented")
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
