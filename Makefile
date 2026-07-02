.PHONY: proto deps build build-services test test-integration test-race lint clean

deps:
	go mod download
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

proto: deps
	@if command -v buf >/dev/null 2>&1; then \
		cd proto && buf generate; \
	else \
		cd proto && bash scripts/generate-proto.sh; \
	fi

build: build-services

build-services:
	go build ./userservice/...
	go build ./spotservice/...
	go build ./orderservice/...
	go build ./orderserviceclient/...

run-user:
	go run ./userservice

run-spot-service:
	go run ./spotservice

run-order-service:
	go run ./orderservice

run-client:
	go run ./orderserviceclient --register --email=user@example.com --password=password123 --market-id=BTC-USDT --order-side=buy --quantity=0.01

run-client-service: run-client

compose-up:
	cd infrastructure/compose && docker compose up -d --build

compose-down:
	cd infrastructure/compose && docker compose down

test:
	go test ./userservice/... ./spotservice/... ./orderservice/... ./orderserviceclient/... ./shared/... ./proto/...
	go test ./test/integration/...

test-integration:
	go test ./test/integration/... -v

test-race:
	go test -race ./userservice/... ./spotservice/... ./orderservice/... ./orderserviceclient/... ./shared/...
	go test -race ./test/integration/...

lint:
	go vet ./userservice/... ./spotservice/... ./orderservice/... ./orderserviceclient/... ./shared/... ./test/integration/...

clean:
	go clean -cache
