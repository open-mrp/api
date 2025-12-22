#!/bin/bash

# Install CD Tools Script
# This script installs only the minimum necessary tools for CD to run fast.

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
    print_status "Installing/updating $name..."
    go install "$package"
}

print_status "Installing minimum CD tools..."

# Only install what's actually used in prepare-release.yml
install_tool "vacuum" "github.com/daveshanley/vacuum@latest"

print_status "CD tools installed successfully!"
