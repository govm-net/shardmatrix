package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"

	"github.com/govm-net/shardmatrix/pkg/blockchain"
	"github.com/govm-net/shardmatrix/pkg/config"
	"github.com/govm-net/shardmatrix/pkg/network"
	"github.com/govm-net/shardmatrix/pkg/types"
)

// APIServer REST API服务器
type APIServer struct {
	config     *config.Config
	blockchain *blockchain.Blockchain
	network    *network.Network
	router     *mux.Router
	server     *http.Server
	logger     *logrus.Logger
}

// APIResponse 统一的API响应格式
type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
	Message string      `json:"message,omitempty"`
}

// ErrorResponse 错误响应
type ErrorResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// NewAPIServer 创建新的API服务器
func NewAPIServer(cfg *config.Config, blockchain *blockchain.Blockchain, network *network.Network) *APIServer {
	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)

	api := &APIServer{
		config:     cfg,
		blockchain: blockchain,
		network:    network,
		logger:     logger,
		router:     mux.NewRouter(),
	}

	// 设置路由
	api.setupRoutes()

	// 设置中间件
	api.setupMiddleware()

	return api
}

// setupRoutes 设置API路由
func (api *APIServer) setupRoutes() {
	// API版本前缀
	v1 := api.router.PathPrefix("/api/v1").Subrouter()

	// 健康检查
	v1.HandleFunc("/health", api.handleHealth).Methods("GET")
	v1.HandleFunc("/status", api.handleStatus).Methods("GET")

	// 区块相关API
	blocks := v1.PathPrefix("/blocks").Subrouter()
	blocks.HandleFunc("", api.handleGetBlocks).Methods("GET")
	blocks.HandleFunc("/latest", api.handleGetLatestBlock).Methods("GET")
	blocks.HandleFunc("/{hash}", api.handleGetBlockByHash).Methods("GET")
	blocks.HandleFunc("/height/{height}", api.handleGetBlockByHeight).Methods("GET")

	// 交易相关API
	txs := v1.PathPrefix("/transactions").Subrouter()
	txs.HandleFunc("", api.handleGetTransactions).Methods("GET")
	txs.HandleFunc("", api.handleSubmitTransaction).Methods("POST")
	txs.HandleFunc("/{hash}", api.handleGetTransactionByHash).Methods("GET")

	// 账户相关API
	accounts := v1.PathPrefix("/accounts").Subrouter()
	accounts.HandleFunc("/{address}", api.handleGetAccount).Methods("GET")
	accounts.HandleFunc("/{address}/balance", api.handleGetAccountBalance).Methods("GET")
	accounts.HandleFunc("/{address}/transactions", api.handleGetAccountTransactions).Methods("GET")

	// 验证者相关API
	validators := v1.PathPrefix("/validators").Subrouter()
	validators.HandleFunc("", api.handleGetValidators).Methods("GET")
	validators.HandleFunc("/{address}", api.handleGetValidator).Methods("GET")
	validators.HandleFunc("/{address}/delegators", api.handleGetValidatorDelegators).Methods("GET")

	// 网络相关API
	network := v1.PathPrefix("/network").Subrouter()
	network.HandleFunc("/info", api.handleGetNetworkInfo).Methods("GET")
	network.HandleFunc("/peers", api.handleGetPeers).Methods("GET")
	network.HandleFunc("/stats", api.handleGetNetworkStats).Methods("GET")
	network.HandleFunc("/health", api.handleGetNetworkHealth).Methods("GET")

	// 共识相关API
	consensus := v1.PathPrefix("/consensus").Subrouter()
	consensus.HandleFunc("/info", api.handleGetConsensusInfo).Methods("GET")
	consensus.HandleFunc("/validators", api.handleGetConsensusValidators).Methods("GET")
}

// setupMiddleware 设置中间件
func (api *APIServer) setupMiddleware() {
	// CORS中间件
	api.router.Use(api.corsMiddleware)

	// 日志中间件
	api.router.Use(api.loggingMiddleware)

	// 错误恢复中间件
	api.router.Use(api.recoveryMiddleware)

	// 内容类型中间件
	api.router.Use(api.contentTypeMiddleware)
}

// Start 启动API服务器
func (api *APIServer) Start() error {
	addr := fmt.Sprintf("%s:%d", api.config.API.Host, api.config.API.Port)

	api.server = &http.Server{
		Addr:         addr,
		Handler:      api.router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	api.logger.Infof("Starting API server on %s", addr)

	go func() {
		if err := api.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			api.logger.Fatalf("Failed to start API server: %v", err)
		}
	}()

	return nil
}

// Stop 停止API服务器
func (api *APIServer) Stop() error {
	if api.server == nil {
		return nil
	}

	api.logger.Info("Stopping API server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	return api.server.Shutdown(ctx)
}

// 辅助方法

// writeJSONResponse 写入JSON响应
func (api *APIServer) writeJSONResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		api.logger.Errorf("Failed to encode JSON response: %v", err)
	}
}

// writeSuccessResponse 写入成功响应
func (api *APIServer) writeSuccessResponse(w http.ResponseWriter, data interface{}) {
	response := APIResponse{
		Success: true,
		Data:    data,
	}
	api.writeJSONResponse(w, http.StatusOK, response)
}

// writeErrorResponse 写入错误响应
func (api *APIServer) writeErrorResponse(w http.ResponseWriter, statusCode int, message string, details ...string) {
	var detail string
	if len(details) > 0 {
		detail = details[0]
	}

	response := APIResponse{
		Success: false,
		Error:   message,
	}

	if detail != "" {
		response.Data = ErrorResponse{
			Code:    statusCode,
			Message: message,
			Details: detail,
		}
	}

	api.writeJSONResponse(w, statusCode, response)
}

// parseUint64Param 解析uint64参数
func (api *APIServer) parseUint64Param(r *http.Request, param string) (uint64, error) {
	vars := mux.Vars(r)
	str, exists := vars[param]
	if !exists {
		return 0, fmt.Errorf("parameter %s not found", param)
	}

	value, err := strconv.ParseUint(str, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid %s: %v", param, err)
	}

	return value, nil
}

// parseAddressParam 解析地址参数
func (api *APIServer) parseAddressParam(r *http.Request, param string) (types.Address, error) {
	vars := mux.Vars(r)
	str, exists := vars[param]
	if !exists {
		return types.Address{}, fmt.Errorf("parameter %s not found", param)
	}

	address, err := types.AddressFromString(str)
	if err != nil {
		return types.Address{}, fmt.Errorf("invalid %s: %v", param, err)
	}

	return address, nil
}

// parseHashParam 解析哈希参数
func (api *APIServer) parseHashParam(r *http.Request, param string) (types.Hash, error) {
	vars := mux.Vars(r)
	str, exists := vars[param]
	if !exists {
		return types.Hash{}, fmt.Errorf("parameter %s not found", param)
	}

	hash, err := types.HashFromString(str)
	if err != nil {
		return types.Hash{}, fmt.Errorf("invalid %s: %v", param, err)
	}

	return hash, nil
}
