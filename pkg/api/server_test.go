package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/govm-net/shardmatrix/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAPIServer(t *testing.T) {
	// 创建测试配置
	cfg := &config.Config{}
	cfg.API.Host = "127.0.0.1"
	cfg.API.Port = 8081
	cfg.Blockchain.ChainID = 1
	cfg.Blockchain.BlockInterval = 3
	cfg.Blockchain.MaxBlockSize = 2097152

	// 创建测试服务器
	apiServer := NewAPIServer(cfg, nil, nil)

	// 测试健康检查端点
	t.Run("HealthCheck", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/api/v1/health", nil)
		require.NoError(t, err)

		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(apiServer.handleHealth)

		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		// 检查响应内容
		expected := `{"success":true,"data":{"status":"ok","timestamp":"`
		assert.Contains(t, rr.Body.String(), expected)
	})

	// 测试状态端点
	t.Run("Status", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/api/v1/status", nil)
		require.NoError(t, err)

		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(apiServer.handleStatus)

		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Contains(t, rr.Body.String(), `"success":true`)
	})
}

func TestAPIResponseHelpers(t *testing.T) {
	// 创建测试配置
	cfg := &config.Config{}
	cfg.API.Host = "127.0.0.1"
	cfg.API.Port = 8081

	// 创建测试服务器
	apiServer := NewAPIServer(cfg, nil, nil)

	t.Run("SuccessResponse", func(t *testing.T) {
		rr := httptest.NewRecorder()

		data := map[string]interface{}{
			"test": "value",
		}
		apiServer.writeSuccessResponse(rr, data)

		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Contains(t, rr.Body.String(), `"success":true`)
		assert.Contains(t, rr.Body.String(), `"test":"value"`)
	})

	t.Run("ErrorResponse", func(t *testing.T) {
		rr := httptest.NewRecorder()

		apiServer.writeErrorResponse(rr, http.StatusBadRequest, "test error", "details")

		assert.Equal(t, http.StatusBadRequest, rr.Code)
		assert.Contains(t, rr.Body.String(), `"success":false`)
		assert.Contains(t, rr.Body.String(), `"error":"test error"`)
	})
}

func TestMiddleware(t *testing.T) {
	// 创建测试配置
	cfg := &config.Config{}
	cfg.API.Host = "127.0.0.1"
	cfg.API.Port = 8081

	// 创建测试服务器
	apiServer := NewAPIServer(cfg, nil, nil)

	t.Run("CORSMiddleware", func(t *testing.T) {
		req, err := http.NewRequest("OPTIONS", "/", nil)
		require.NoError(t, err)

		rr := httptest.NewRecorder()
		handler := apiServer.corsMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Equal(t, "*", rr.Header().Get("Access-Control-Allow-Origin"))
	})

	t.Run("ContentTypeMiddleware", func(t *testing.T) {
		req, err := http.NewRequest("POST", "/", nil)
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		handler := apiServer.contentTypeMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
	})
}
