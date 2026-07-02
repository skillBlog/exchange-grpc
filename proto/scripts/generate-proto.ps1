#Requires -Version 5.1
$ErrorActionPreference = "Stop"

$Root = Split-Path -Parent $PSScriptRoot
Set-Location $Root

Write-Host "Installing protoc plugins..."
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

$buf = Get-Command buf -ErrorAction SilentlyContinue
if ($buf) {
    Write-Host "Generating with buf..."
    Set-Location proto
    buf generate
} else {
    $protoc = Get-Command protoc -ErrorAction SilentlyContinue
    if (-not $protoc) {
        throw "protoc not found. Install via: winget install Google.Protobuf"
    }
    Write-Host "Generating with protoc..."
    $ProtoDir = "proto/proto"
    Get-ChildItem -Path $ProtoDir -Recurse -Filter "*.proto" | ForEach-Object {
        protoc `
            --proto_path=$ProtoDir `
            --go_out=proto/pb --go_opt=paths=source_relative `
            --go-grpc_out=proto/pb --go-grpc_opt=paths=source_relative `
            $_.FullName.Substring((Resolve-Path $ProtoDir).Path.Length + 1)
    }
}

Write-Host "Done."
