// api_usage.go - API使用示例
//
// 这个示例演示了如何使用ShardMatrix的RESTful API接口
// 包括获取区块链状态、查询区块、提交交易等操作
//
// 使用方法:
//   1. 启动ShardMatrix节点: go run cmd/node/main.go
//   2. 运行示例: go run examples/api_usage.go

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// APIResponse 统一的API响应格式
type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
	Message string      `json:"message,omitempty"`
}

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

func main() {
	// API基础URL
	baseURL := "http://localhost:8081/api/v1"

	fmt.Println("🚀 ShardMatrix API 使用示例")
	fmt.Println("========================")

	// 1. 健康检查
	fmt.Println("\n1. 健康检查:")
	healthResp, err := getHealth(baseURL)
	if err != nil {
		fmt.Printf("❌ 健康检查失败: %v\n", err)
		return
	}
	fmt.Printf("✅ 状态: %s\n", healthResp.Status)
	fmt.Printf("🕒 时间戳: %s\n", healthResp.Timestamp.Format("2006-01-02 15:04:05"))

	// 2. 获取节点状态
	fmt.Println("\n2. 节点状态:")
	statusResp, err := getStatus(baseURL)
	if err != nil {
		fmt.Printf("❌ 获取状态失败: %v\n", err)
		return
	}
	fmt.Printf("🔗 链ID: %d\n", statusResp.BlockchainInfo.ChainID)
	fmt.Printf("📈 最新区块高度: %d\n", statusResp.BlockchainInfo.LatestHeight)
	fmt.Printf("🔢 总区块数: %d\n", statusResp.BlockchainInfo.TotalBlocks)

	// 3. 获取最新区块
	fmt.Println("\n3. 最新区块:")
	latestBlock, err := getLatestBlock(baseURL)
	if err != nil {
		fmt.Printf("❌ 获取最新区块失败: %v\n", err)
		return
	}
	fmt.Printf("🔢 区块高度: %d\n", latestBlock.Height)
	fmt.Printf("🔐 区块哈希: %s\n", latestBlock.Hash)
	fmt.Printf("🕒 时间戳: %s\n", latestBlock.Timestamp.Format("2006-01-02 15:04:05"))
	fmt.Printf("👤 验证者: %s\n", latestBlock.Validator)

	// 4. 提交交易示例
	fmt.Println("\n4. 提交交易:")
	txResp, err := submitTransaction(baseURL)
	if err != nil {
		fmt.Printf("❌ 提交交易失败: %v\n", err)
	} else {
		fmt.Printf("✅ 交易提交成功\n")
		fmt.Printf("🆔 交易哈希: %s\n", txResp["tx_hash"])
		fmt.Printf("💬 状态: %s\n", txResp["status"])
	}

	// 5. 获取交易列表
	fmt.Println("\n5. 交易列表:")
	txsResp, err := getTransactions(baseURL)
	if err != nil {
		fmt.Printf("❌ 获取交易列表失败: %v\n", err)
	} else {
		fmt.Printf("✅ 成功获取交易列表\n")
		if data, ok := txsResp["pagination"]; ok {
			if pagination, ok := data.(map[string]interface{}); ok {
				fmt.Printf("📄 总数: %.0f\n", pagination["total"])
			}
		}
	}

	fmt.Println("\n🎉 API使用示例完成!")
}

// getHealth 执行健康检查
func getHealth(baseURL string) (*HealthResponse, error) {
	resp, err := http.Get(baseURL + "/health")
	if err != nil {
		return nil, fmt.Errorf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %v", err)
	}

	var apiResp APIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("解析JSON失败: %v", err)
	}

	if !apiResp.Success {
		return nil, fmt.Errorf("API错误: %s", apiResp.Error)
	}

	healthData, err := json.Marshal(apiResp.Data)
	if err != nil {
		return nil, fmt.Errorf("序列化数据失败: %v", err)
	}

	var healthResp HealthResponse
	if err := json.Unmarshal(healthData, &healthResp); err != nil {
		return nil, fmt.Errorf("解析健康数据失败: %v", err)
	}

	return &healthResp, nil
}

// getStatus 获取节点状态
func getStatus(baseURL string) (*StatusResponse, error) {
	resp, err := http.Get(baseURL + "/status")
	if err != nil {
		return nil, fmt.Errorf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %v", err)
	}

	var apiResp APIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("解析JSON失败: %v", err)
	}

	if !apiResp.Success {
		return nil, fmt.Errorf("API错误: %s", apiResp.Error)
	}

	statusData, err := json.Marshal(apiResp.Data)
	if err != nil {
		return nil, fmt.Errorf("序列化数据失败: %v", err)
	}

	var statusResp StatusResponse
	if err := json.Unmarshal(statusData, &statusResp); err != nil {
		return nil, fmt.Errorf("解析状态数据失败: %v", err)
	}

	return &statusResp, nil
}

// getLatestBlock 获取最新区块
func getLatestBlock(baseURL string) (*BlockResponse, error) {
	resp, err := http.Get(baseURL + "/blocks/latest")
	if err != nil {
		return nil, fmt.Errorf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %v", err)
	}

	var apiResp APIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("解析JSON失败: %v", err)
	}

	if !apiResp.Success {
		return nil, fmt.Errorf("API错误: %s", apiResp.Error)
	}

	blockData, err := json.Marshal(apiResp.Data)
	if err != nil {
		return nil, fmt.Errorf("序列化数据失败: %v", err)
	}

	var blockResp BlockResponse
	if err := json.Unmarshal(blockData, &blockResp); err != nil {
		return nil, fmt.Errorf("解析区块数据失败: %v", err)
	}

	return &blockResp, nil
}

// submitTransaction 提交交易
func submitTransaction(baseURL string) (map[string]interface{}, error) {
	// 创建交易数据
	txData := map[string]interface{}{
		"from":   "0x1234567890123456789012345678901234567890",
		"to":     "0x0987654321098765432109876543210987654321",
		"amount": 1000,
		"fee":    10,
		"nonce":  1,
	}

	// 将数据编码为JSON
	jsonData, err := json.Marshal(txData)
	if err != nil {
		return nil, fmt.Errorf("编码JSON失败: %v", err)
	}

	resp, err := http.Post(baseURL+"/transactions", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %v", err)
	}

	var apiResp APIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("解析JSON失败: %v", err)
	}

	if !apiResp.Success {
		return nil, fmt.Errorf("API错误: %s", apiResp.Error)
	}

	if data, ok := apiResp.Data.(map[string]interface{}); ok {
		return data, nil
	}

	return nil, fmt.Errorf("无效的响应数据格式")
}

// getTransactions 获取交易列表
func getTransactions(baseURL string) (map[string]interface{}, error) {
	resp, err := http.Get(baseURL + "/transactions")
	if err != nil {
		return nil, fmt.Errorf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %v", err)
	}

	var apiResp APIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("解析JSON失败: %v", err)
	}

	if !apiResp.Success {
		return nil, fmt.Errorf("API错误: %s", apiResp.Error)
	}

	if data, ok := apiResp.Data.(map[string]interface{}); ok {
		return data, nil
	}

	return nil, fmt.Errorf("无效的响应数据格式")
}
