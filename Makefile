.PHONY: build clean test lint start-network

# 项目配置
BINARY_NAME=shardnode
OUTPUT_DIR=./cmd/shardnode
MAIN_PATH=./cmd/shardnode

# Go配置
GO=go
GOFLAGS=-v
LDFLAGS=-w -s

# 构建目标
build:
	@echo "Building ShardMatrix node..."
	$(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(OUTPUT_DIR)/$(BINARY_NAME) $(MAIN_PATH)
	@echo "Build completed: $(OUTPUT_DIR)/$(BINARY_NAME)"

# 清理构建文件
clean:
	@echo "Cleaning build files..."
	rm -f $(OUTPUT_DIR)/$(BINARY_NAME)
	rm -rf ./data ./dev-data ./test-data
	@echo "Clean completed"

# 运行测试
test:
	@echo "Running tests..."
	$(GO) test $(GOFLAGS) ./...

# 运行代码检查
lint:
	@echo "Running linter..."
	golangci-lint run

# 安装依赖
deps:
	@echo "Installing dependencies..."
	$(GO) mod download
	$(GO) mod tidy

# 初始化节点
init:
	@echo "Initializing node..."
	./$(OUTPUT_DIR)/$(BINARY_NAME) init

# 启动单节点
start:
	@echo "Starting single node..."
	./$(OUTPUT_DIR)/$(BINARY_NAME) start

# 启动三节点网络（如果脚本存在）
start-network:
	@if [ -f "./examples/three-nodes/scripts/start-network.sh" ]; then \
		echo "Starting three-node network..."; \
		bash ./examples/three-nodes/scripts/start-network.sh; \
	else \
		echo "Three-node network script not found"; \
	fi

# 查看节点状态
status:
	@echo "Checking node status..."
	./$(OUTPUT_DIR)/$(BINARY_NAME) status

# 显示版本信息
version:
	./$(OUTPUT_DIR)/$(BINARY_NAME) version --verbose

# 生成密钥
keys-generate:
	./$(OUTPUT_DIR)/$(BINARY_NAME) keys generate

# 显示配置
config-show:
	./$(OUTPUT_DIR)/$(BINARY_NAME) config show

# 验证配置
config-validate:
	./$(OUTPUT_DIR)/$(BINARY_NAME) config validate

# 完整的构建和测试流程
all: clean deps build test

# 开发模式：构建并运行
dev: build init start

# 帮助信息
help:
	@echo "ShardMatrix Node Makefile"
	@echo ""
	@echo "Available targets:"
	@echo "  build         - Build the shardnode binary"
	@echo "  clean         - Clean build files and data directories"
	@echo "  test          - Run all tests"
	@echo "  lint          - Run code linter"
	@echo "  deps          - Install/update dependencies"
	@echo "  init          - Initialize a new node"
	@echo "  start         - Start the node"
	@echo "  start-network - Start three-node network"
	@echo "  status        - Check node status"
	@echo "  version       - Show version information"
	@echo "  keys-generate - Generate new validator keys"
	@echo "  config-show   - Show current configuration"
	@echo "  config-validate - Validate configuration"
	@echo "  all           - Clean, deps, build, and test"
	@echo "  dev           - Build, init, and start (development mode)"
	@echo "  help          - Show this help message"