#Requires -Version 5.1
$ErrorActionPreference = "Stop"

$Root = Split-Path -Parent $PSScriptRoot
Set-Location $Root

$ProtoDir = "api/proto"
$ProtoFile = "$ProtoDir/exchange/v1/exchange.proto"

Write-Host "Installing protoc plugins..."
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

$buf = Get-Command buf -ErrorAction SilentlyContinue
if ($buf) {
    Write-Host "Generating with buf..."
    buf generate
} else {
    $protoc = Get-Command protoc -ErrorAction SilentlyContinue
    if (-not $protoc) {
        throw "protoc not found. Install via: winget install Google.Protobuf"
    }
    Write-Host "Generating with protoc..."
    protoc `
        --proto_path=$ProtoDir `
        --go_out=$ProtoDir --go_opt=paths=source_relative `
        --go-grpc_out=$ProtoDir --go-grpc_opt=paths=source_relative `
        $ProtoFile
}

Write-Host "Done."
