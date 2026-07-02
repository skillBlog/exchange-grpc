#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

echo "Installing protoc plugins..."
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

if command -v buf >/dev/null 2>&1; then
  echo "Generating with buf..."
  buf generate
else
  echo "Generating with protoc..."
  PROTO_DIR="proto"
  find "$PROTO_DIR" -name '*.proto' | while read -r f; do
    protoc \
      --proto_path="$PROTO_DIR" \
      --go_out=pb --go_opt=paths=source_relative \
      --go-grpc_out=pb --go-grpc_opt=paths=source_relative \
      "${f#$PROTO_DIR/}"
  done
fi

echo "Done."
