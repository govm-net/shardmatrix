# ShardMatrix Makefile

# 变量定义
BINARY_NAME=shardmatrix-node
MAIN_PATH=cmd/node/main.go
BUILD_DIR=build
VERSION=$(shell git describe --tags --always --dirty)
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS=-ldflags "-X main.Version=${VERSION} -X main.BuildTime=${BUILD_TIME}"

# 默认目标
.PHONY: all
all: clean build

# 构建项目
.PHONY: build
build:
	@echo "Building ShardMatrix node..."
	@mkdir -p ${BUILD_DIR}
	go build ${LDFLAGS} -o ${BUILD_DIR}/${BINARY_NAME} ${MAIN_PATH}
	@echo "Build completed: ${BUILD_DIR}/${BINARY_NAME}"

# 运行项目
.PHONY: run
run:
	@echo "Running ShardMatrix node..."
	go run ${MAIN_PATH}

# 运行示例
.PHONY: example
example:
	@echo "Running basic usage example..."
	go run examples/basic_usage.go

# 测试
.PHONY: test
test:
	@echo "Running tests..."
	go test -v ./pkg/...

# 测试覆盖率
.PHONY: test-coverage
test-coverage:
	@echo "Running tests with coverage..."
	go test -v -coverprofile=coverage.out ./pkg/...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# 代码格式化
.PHONY: fmt
fmt:
	@echo "Formatting code..."
	go fmt ./...

# 代码检查
.PHONY: lint
lint:
	@echo "Running linter..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not installed. Installing..."; \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; \
		golangci-lint run; \
	fi

# 竞态条件检查
.PHONY: test-race
test-race:
	@echo "Running tests with race detection..."
	go test -race -v ./pkg/...

# 基准测试
.PHONY: bench
bench:
	@echo "Running benchmarks..."
	go test -bench=. -benchmem ./pkg/...

# 完整质量检查
.PHONY: quality
quality: fmt lint test test-race
	@echo "All quality checks completed successfully!"

# 清理构建文件
.PHONY: clean
clean:
	@echo "Cleaning build files..."
	rm -rf ${BUILD_DIR}
	rm -f coverage.out coverage.html

# 安装依赖
.PHONY: deps
deps:
	@echo "Installing dependencies..."
	go mod download
	go mod tidy

# 生成文档
.PHONY: docs
docs:
	@echo "Generating documentation..."
	godoc -http=:6060

# 创建发布版本
.PHONY: release
release: clean test build
	@echo "Creating release version ${VERSION}..."
	tar -czf ${BUILD_DIR}/shardmatrix-${VERSION}.tar.gz -C ${BUILD_DIR} ${BINARY_NAME}

# 开发模式（自动重新构建）
.PHONY: dev
dev:
	@echo "Starting development mode..."
	air

# 帮助信息
.PHONY: help
help:
	@echo "ShardMatrix Makefile Commands:"
	@echo "  build        - Build the project"
	@echo "  run          - Run the node"
	@echo "  example      - Run the basic usage example"
	@echo "  test         - Run tests"
	@echo "  test-coverage- Run tests with coverage report"
	@echo "  fmt          - Format code"
	@echo "  lint         - Run linter"
	@echo "  clean        - Clean build files"
	@echo "  deps         - Install dependencies"
	@echo "  docs         - Generate documentation"
	@echo "  release      - Create release version"
	@echo "  dev          - Start development mode"
	@echo "  help         - Show this help"
