APP_NAME = sysmonitord
VERSION ?= V0.1.0
BUILD_TIME = $(shell date +%Y-%m-%d_%H:%M:%S)
GIT_COMMIT = $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

LDFLAGS = -ldflags "-X 'sysmonitord/internal/version.Version=$(VERSION)' \
-X 'sysmonitord/internal/version.BuildTime=$(BUILD_TIME)' \
-X 'sysmonitord/internal/version.GitCommit=$(GIT_COMMIT)'"

all: build

build:
	@echo "开始编译 $(APP_NAME) (amd64) 版本: $(VERSION)"
	@mkdir -p dist
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o dist/$(APP_NAME) main.go
	@echo "正在打包 amd64 发布包..."
	@mkdir -p dist/pkg-tmp/$(APP_NAME)
	@cp dist/$(APP_NAME) dist/pkg-tmp/$(APP_NAME)/
	@cp scripts/sysmonitord.service dist/pkg-tmp/$(APP_NAME)/
	@cp config.yaml.example dist/pkg-tmp/$(APP_NAME)/
	@mkdir -p release
	@cd dist/pkg-tmp && tar czf ../../release/$(APP_NAME)-$(VERSION)-linux-amd64.tar.gz $(APP_NAME)
	@rm -rf dist/pkg-tmp
	@echo "编译完成: dist/$(APP_NAME)"
	@echo "发布包: release/$(APP_NAME)-$(VERSION)-linux-amd64.tar.gz"

build-arm64:
	@echo "开始编译 $(APP_NAME) (arm64) 版本: $(VERSION)"
	@mkdir -p dist
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o dist/$(APP_NAME)-arm64 main.go
	@echo "正在打包 arm64 发布包..."
	@mkdir -p dist/pkg-tmp/$(APP_NAME)
	@cp dist/$(APP_NAME)-arm64 dist/pkg-tmp/$(APP_NAME)/$(APP_NAME)
	@cp scripts/sysmonitord.service dist/pkg-tmp/$(APP_NAME)/
	@cp config.yaml.example dist/pkg-tmp/$(APP_NAME)/
	@mkdir -p release
	@cd dist/pkg-tmp && tar czf ../../release/$(APP_NAME)-$(VERSION)-linux-arm64.tar.gz $(APP_NAME)
	@rm -rf dist/pkg-tmp
	@echo "编译完成: dist/$(APP_NAME)-arm64"
	@echo "发布包: release/$(APP_NAME)-$(VERSION)-linux-arm64.tar.gz"

install:
	@echo "安装 $(APP_NAME) 到/usr/local/bin..."
	cp dist/$(APP_NAME) /usr/local/bin/
	@echo "安装完成"

clean:
	@echo "清理编译产物..."
	rm -rf dist/ release/
	@echo "清理完成"
