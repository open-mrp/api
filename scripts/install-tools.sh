#!/bin/bash

# Install Tools Script
# This script installs all required development tools

set -e

# Colors for output
GREEN='\033[0;32m'
NC='\033[0m'

print_status() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

install_tool() {
    local name=$1
    local package=$2
    print_status "Installing/updating $name..."
    go install "$package"
}

print_status "Installing required development tools..."

install_tool "sqlc" "github.com/sqlc-dev/sqlc/cmd/sqlc@latest"
install_tool "goose" "github.com/pressly/goose/v3/cmd/goose@latest"
install_tool "gotestsum" "gotest.tools/gotestsum@latest"
install_tool "mockgen" "go.uber.org/mock/mockgen@latest"
install_tool "vacuum" "github.com/daveshanley/vacuum@latest"
install_tool "protoc-gen-go" "google.golang.org/protobuf/cmd/protoc-gen-go@latest"
install_tool "protoc-gen-go-grpc" "google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest"
install_tool "goreleaser" "github.com/goreleaser/goreleaser/v2@latest"
install_tool "gosec" "github.com/securego/gosec/v2/cmd/gosec@latest"
install_tool "staticcheck" "honnef.co/go/tools/cmd/staticcheck@latest"

print_status "All tools installed successfully!"
