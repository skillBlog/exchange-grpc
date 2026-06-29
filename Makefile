.PHONY: proto deps build test lint clean

MODULE := github.com/exchange-grpc
PROTO_DIR := api/proto
PROTO_FILE := $(PROTO_DIR)/exchange/v1/exchange.proto

deps:
	go mod download
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

proto: deps
	@if command -v buf >/dev/null 2>&1; then \
		buf generate; \
	else \
		protoc \
			--proto_path=$(PROTO_DIR) \
			--go_out=$(PROTO_DIR) --go_opt=paths=source_relative \
			--go-grpc_out=$(PROTO_DIR) --go-grpc_opt=paths=source_relative \
			$(PROTO_FILE); \
	fi

build:
	go build ./...

run-spot:
	go run ./cmd/spot-instrument-service

run-order:
	go run ./cmd/order-service

run-client:
	go run ./cmd/client --user-id=user-1 --market-id=BTC-USDT --order-type=limit --price=42000 --quantity=0.01

test:
	go test ./...

test-integration:
	go test ./test/integration/... -v

test-race:
	go test -race ./...

lint:
	go vet ./...

compose-build:
	cd deployments && docker compose build

compose-up:
	cd deployments && docker compose up -d --build

compose-down:
	cd deployments && docker compose down

compose-logs:
	cd deployments && docker compose logs -f

clean:
	go clean ./...
