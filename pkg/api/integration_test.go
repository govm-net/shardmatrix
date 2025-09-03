package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/govm-net/shardmatrix/pkg/blockchain"
	"github.com/govm-net/shardmatrix/pkg/config"
	"github.com/govm-net/shardmatrix/pkg/storage"
	"github.com/govm-net/shardmatrix/pkg/validator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAPIIntegration(t *testing.T) {
	// 创建测试配置
	cfg := &config.Config{}
	cfg.API.Host = "127.0.0.1"
	cfg.API.Port = 8081

	// 创建存储层
	blockStore := storage.NewMemoryBlockStore()
	txStore := storage.NewMemoryTransactionStore()

	// 创建验证器
	validatorConfig := validator.DefaultValidationConfig()
	blockValidator := validator.NewValidator(validatorConfig, blockStore, txStore, nil, nil)

	// 创建区块链管理器
	blockchainConfig := blockchain.DefaultBlockchainConfig()
	blockchainManager, err := blockchain.NewBlockchain(blockchainConfig, blockStore, blockValidator)
	require.NoError(t, err)

	// 创建API服务器
	apiServer := NewAPIServer(cfg, blockchainManager, nil)

	t.Run("GetLatestBlock", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/api/v1/blocks/latest", nil)
		require.NoError(t, err)

		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(apiServer.handleGetLatestBlock)

		handler.ServeHTTP(rr, req)

		// 应该返回创世区块
		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Contains(t, rr.Body.String(), `"success":true`)
		assert.Contains(t, rr.Body.String(), `"height":0`)
	})

	t.Run("GetBlocks", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/api/v1/blocks", nil)
		require.NoError(t, err)

		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(apiServer.handleGetBlocks)

		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Contains(t, rr.Body.String(), `"success":true`)
	})

	t.Run("GetStatus", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/api/v1/status", nil)
		require.NoError(t, err)

		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(apiServer.handleStatus)

		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Contains(t, rr.Body.String(), `"success":true`)
		assert.Contains(t, rr.Body.String(), `"chain_id":1`)
	})

	t.Run("SubmitTransaction", func(t *testing.T) {
		// 创建测试交易数据
		txData := map[string]interface{}{
			"from":   "0x1234567890123456789012345678901234567890",
			"to":     "0x0987654321098765432109876543210987654321",
			"amount": 1000,
			"fee":    10,
			"nonce":  1,
		}

		// 将数据编码为JSON
		jsonData, err := json.Marshal(txData)
		require.NoError(t, err)

		req, err := http.NewRequest("POST", "/api/v1/transactions", bytes.NewBuffer(jsonData))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(apiServer.handleSubmitTransaction)

		handler.ServeHTTP(rr, req)

		// 应该返回成功响应
		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Contains(t, rr.Body.String(), `"success":true`)
		assert.Contains(t, rr.Body.String(), `"status":"pending"`)
	})

	t.Run("GetTransactions", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/api/v1/transactions", nil)
		require.NoError(t, err)

		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(apiServer.handleGetTransactions)

		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Contains(t, rr.Body.String(), `"success":true`)
	})
}

func TestAPIErrorHandling(t *testing.T) {
	// 创建测试配置
	cfg := &config.Config{}
	cfg.API.Host = "127.0.0.1"
	cfg.API.Port = 8081

	// 创建API服务器
	apiServer := NewAPIServer(cfg, nil, nil)

	t.Run("InvalidBlockHash", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/api/v1/blocks/invalid_hash", nil)
		require.NoError(t, err)

		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(apiServer.handleGetBlockByHash)

		handler.ServeHTTP(rr, req)

		// 应该返回错误响应
		assert.Equal(t, http.StatusBadRequest, rr.Code)
		assert.Contains(t, rr.Body.String(), `"success":false`)
		assert.Contains(t, rr.Body.String(), "Invalid hash parameter")
	})

	t.Run("InvalidBlockHeight", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/api/v1/blocks/height/invalid", nil)
		require.NoError(t, err)

		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(apiServer.handleGetBlockByHeight)

		handler.ServeHTTP(rr, req)

		// 应该返回错误响应
		assert.Equal(t, http.StatusBadRequest, rr.Code)
		assert.Contains(t, rr.Body.String(), `"success":false`)
		assert.Contains(t, rr.Body.String(), "Invalid height parameter")
	})

	t.Run("MissingBlockchain", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/api/v1/blocks/latest", nil)
		require.NoError(t, err)

		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(apiServer.handleGetLatestBlock)

		handler.ServeHTTP(rr, req)

		// 应该返回服务不可用错误
		assert.Equal(t, http.StatusServiceUnavailable, rr.Code)
		assert.Contains(t, rr.Body.String(), `"success":false`)
		assert.Contains(t, rr.Body.String(), "Blockchain not available")
	})
}
