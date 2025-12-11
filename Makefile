.PHONY: all build build-web build-server build-agent clean dev dev-web dev-server

# 版本信息
VERSION ?= 1.0.0
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
LDFLAGS := -ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME) -X main.GitCommit=$(GIT_COMMIT) -s -w"

# 输出目录
DIST_DIR := dist

all: build

# 构建所有
build: build-web build-server build-agent

# 构建前端
build-web:
	@echo "Building frontend..."
	cd web && npm install && npm run build
	@echo "Frontend build complete"

# 构建服务端
build-server: build-web
	@echo "Building server..."
	@mkdir -p $(DIST_DIR)
	@rm -rf cmd/letsyncd/dist
	@cp -r web/dist cmd/letsyncd/dist
	CGO_ENABLED=0 go build $(LDFLAGS) -o $(DIST_DIR)/letsyncd ./cmd/letsyncd
	@echo "Server build complete: $(DIST_DIR)/letsyncd"

# 构建 Agent
build-agent:
	@echo "Building agent..."
	@mkdir -p $(DIST_DIR)
	CGO_ENABLED=0 go build $(LDFLAGS) -o $(DIST_DIR)/letsync ./cmd/letsync
	@echo "Agent build complete: $(DIST_DIR)/letsync"

# 仅构建服务端(不重新构建前端)
build-server-only:
	@echo "Building server (without frontend rebuild)..."
	@mkdir -p $(DIST_DIR)
	CGO_ENABLED=0 go build $(LDFLAGS) -o $(DIST_DIR)/letsyncd ./cmd/letsyncd
	@echo "Server build complete"

# 跨平台构建
build-all-platforms: build-web
	@echo "Building for all platforms..."
	@mkdir -p $(DIST_DIR)
	@rm -rf cmd/letsyncd/dist
	@cp -r web/dist cmd/letsyncd/dist
	# Linux AMD64
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build $(LDFLAGS) -o $(DIST_DIR)/letsyncd-linux-amd64 ./cmd/letsyncd
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build $(LDFLAGS) -o $(DIST_DIR)/letsync-linux-amd64 ./cmd/letsync
	# Linux ARM64
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build $(LDFLAGS) -o $(DIST_DIR)/letsyncd-linux-arm64 ./cmd/letsyncd
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build $(LDFLAGS) -o $(DIST_DIR)/letsync-linux-arm64 ./cmd/letsync
	# macOS AMD64
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build $(LDFLAGS) -o $(DIST_DIR)/letsyncd-darwin-amd64 ./cmd/letsyncd
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build $(LDFLAGS) -o $(DIST_DIR)/letsync-darwin-amd64 ./cmd/letsync
	# macOS ARM64
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build $(LDFLAGS) -o $(DIST_DIR)/letsyncd-darwin-arm64 ./cmd/letsyncd
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build $(LDFLAGS) -o $(DIST_DIR)/letsync-darwin-arm64 ./cmd/letsync
	# Windows AMD64
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build $(LDFLAGS) -o $(DIST_DIR)/letsyncd-windows-amd64.exe ./cmd/letsyncd
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build $(LDFLAGS) -o $(DIST_DIR)/letsync-windows-amd64.exe ./cmd/letsync
	@echo "All platform builds complete"

# 清理
clean:
	@echo "Cleaning..."
	rm -rf $(DIST_DIR)
	rm -rf web/dist
	rm -rf web/node_modules
	rm -rf cmd/letsyncd/dist
	@echo "Clean complete"

# 开发模式 - 启动前端开发服务器
dev-web:
	cd web && npm run dev

# 开发模式 - 启动后端
dev-server:
	go run ./cmd/letsyncd -dev

# 开发模式 - 同时启动前后端 (需要 tmux 或两个终端)
dev:
	@echo "Please run in separate terminals:"
	@echo "  make dev-web    # Frontend dev server"
	@echo "  make dev-server # Backend server"

# 测试
test:
	go test ./...

# 格式化
fmt:
	go fmt ./...
	cd web && npm run lint || true
