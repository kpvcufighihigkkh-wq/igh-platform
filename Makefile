.PHONY: build-server build-edge build-edge-full build run-server run-edge dev clean tidy lint test
.PHONY: build-linux build-windows

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -s -w -X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME)

# --- Build targets ---

build-server:
	go build -ldflags "$(LDFLAGS)" -o bin/igh-server.exe ./cmd/server

build-edge:
	go build -ldflags "$(LDFLAGS)" -o bin/igh-edge.exe ./cmd/edge

build-siteimport:
	go build -ldflags "$(LDFLAGS)" -o bin/siteimport.exe ./cmd/siteimport

build: build-server build-edge build-siteimport

build-edge-full: web-build build-edge

# Cross-compilation
build-linux:
	GOOS=linux GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o bin/linux-amd64/igh-server ./cmd/server
	GOOS=linux GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o bin/linux-amd64/igh-edge ./cmd/edge

build-windows:
	GOOS=windows GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o bin/windows-amd64/igh-server.exe ./cmd/server
	GOOS=windows GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o bin/windows-amd64/igh-edge.exe ./cmd/edge

build-all: build-linux build-windows

# --- Frontend ---

# 前端产物有两个去处，作用不同：
#   internal/edge/app/dist —— 编进边端二进制，边端现场无网也能开页面
#   configs*/ui           —— 由中心服务按目录托管，改页面不用重编
# 两边都拷，避免"中心改了、边端还是旧的"这种版本漂移。
web-build:
	cd web && npm run build
	@if [ ! -d internal/edge/app/dist ]; then mkdir -p internal/edge/app/dist; fi
	rm -rf internal/edge/app/dist/*
	cp -r web/dist/* internal/edge/app/dist/
	@for d in configs/ui configs-dev3/ui; do \
		mkdir -p $$d; \
		rm -rf $$d/*; \
		cp -r web/dist/* $$d/; \
		echo "前端已拷贝到 $$d"; \
	done

web-dev:
	cd web && npm run dev

# --- Run targets ---

run-server:
	IGH_CONFIG_DIR=./configs DB_PASSWORD=igh_dev_2026 go run ./cmd/server

run-edge:
	IGH_CONFIG_DIR=./configs go run ./cmd/edge --config configs

# 把仓库里的初始配置导入中心"现场配置"
# 用法: make import-config EDGE=edge-winding-01 SERVER=http://127.0.0.1:8099
EDGE ?= edge-winding-01
SERVER ?= http://127.0.0.1:8099
PLC ?= configs/devices/instances/spinning_sides.yaml

import-config: build-siteimport
	./bin/siteimport.exe -server $(SERVER) -edge $(EDGE) \
	  -plc $(PLC) \
	  -loading configs/geometry/loading_sequences.yaml \
	  -heads configs/geometry/head_maps.yaml \
	  -remark "make import-config 导入初始配置"

# 只校验不写入
import-config-dry: build-siteimport
	./bin/siteimport.exe -server $(SERVER) -edge $(EDGE) -dry-run \
	  -plc $(PLC) \
	  -loading configs/geometry/loading_sequences.yaml \
	  -heads configs/geometry/head_maps.yaml

# --- Dev environment ---

dev:
	docker compose up -d postgres redis
	@echo "PostgreSQL: localhost:5432, Redis: localhost:6379"

# --- Code quality ---

tidy:
	go mod tidy

lint:
	golangci-lint run ./...

test:
	go test ./... -v -count=1

# --- Packaging ---

dist-edge-windows: build-edge-full
	@mkdir -p dist/igh-edge-windows
	cp bin/igh-edge.exe dist/igh-edge-windows/
	cp configs/edge.yaml dist/igh-edge-windows/
	cp deploy/windows/install.bat dist/igh-edge-windows/
	cp deploy/windows/uninstall.bat dist/igh-edge-windows/
	@echo "Edge Windows package: dist/igh-edge-windows/"

dist-edge-linux: web-build build-linux
	@mkdir -p dist/igh-edge-linux
	cp bin/linux-amd64/igh-edge dist/igh-edge-linux/
	cp configs/edge.yaml dist/igh-edge-linux/
	cp deploy/linux/igh-edge.service dist/igh-edge-linux/
	cp deploy/linux/install.sh dist/igh-edge-linux/
	@echo "Edge Linux package: dist/igh-edge-linux/"

# --- Cleanup ---

clean:
	rm -rf bin/ tmp/ dist/
