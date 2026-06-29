#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

PROTO_DIR="api/proto"
PROTO_FILE="${PROTO_DIR}/exchange/v1/exchange.proto"

go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

if command -v buf >/dev/null 2>&1; then
  buf generate
else
  protoc \
    --proto_path="${PROTO_DIR}" \
    --go_out="${PROTO_DIR}" --go_opt=paths=source_relative \
    --go-grpc_out="${PROTO_DIR}" --go-grpc_opt=paths=source_relative \
    "${PROTO_FILE}"
fi

echo "Proto generation complete."
