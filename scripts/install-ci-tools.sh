#!/bin/bash

# Install CI Tools Script
# This script installs only the minimum necessary tools for CI to run fast.

set -e

# Colors for output
GREEN='\033[0;32m'
NC='\033[0m'

print_status() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

# Ensure go bin is in PATH
export PATH=$PATH:$(go env GOPATH)/bin

install_tool() {
    local name=$1
    local package=$2
    if ! command -v "$name" >/dev/null 2>&1; then
        print_status "Installing $name..."
        go install "$package"
    else
        print_status "$name is already installed."
    fi
}

print_status "Installing minimum CI tools..."

# Only install what's actually used in ci.yml
install_tool "gosec" "github.com/securego/gosec/v2/cmd/gosec@latest"
install_tool "staticcheck" "honnef.co/go/tools/cmd/staticcheck@latest"

print_status "CI tools installed successfully!"
