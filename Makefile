.PHONY: build-server build-edge run-server run-edge dev clean

build-server:
	go build -o bin/igh-server.exe ./cmd/server

build-edge:
	go build -o bin/igh-edge.exe ./cmd/edge

build: build-server build-edge

run-server:
	IGH_CONFIG_DIR=./configs DB_PASSWORD=igh_dev_2026 go run ./cmd/server

run-edge:
	IGH_CONFIG_DIR=./configs IGH_EDGE_ID=edge-winding-01 go run ./cmd/edge

dev:
	docker compose up -d postgres redis
	@echo "PostgreSQL: localhost:5432, Redis: localhost:6379"

clean:
	rm -rf bin/ tmp/

tidy:
	go mod tidy

lint:
	golangci-lint run ./...

test:
	go test ./... -v -count=1
