#!/bin/bash

# Generate Integration Mocks Script
# This script generates mocks for interfaces in internal/infra/integration

set -e

# Colors for output
GREEN='\033[0;32m'
NC='\033[0m'

print_status() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_status "Generating integration mocks..."

MOCK_DEST="./internal/infra/integration/mock"
mkdir -p "$MOCK_DEST"

for file in ./internal/infra/integration/*.go; do
    if [ -f "$file" ] && [[ ! "$file" =~ (mock|_test\.go) ]]; then
        interfaces=$(grep -h "^type.*interface" "$file" | sed 's/type \([A-Za-z0-9_]*\).*/\1/')
        for interface in $interfaces; do
            if [ -n "$interface" ]; then
                print_status "Generating mock for $interface..."
                # Convert CamelCase to snake_case for filename
                filename=$(echo "$interface" | sed 's/\([A-Z][a-z0-9]*\)/_\1/g' | tr '[:upper:]' '[:lower:]' | sed 's/^_//')
                mockgen -destination "$MOCK_DEST/${filename}_mock.go" -package mock github.com/augno/api/internal/infra/integration "$interface"
            fi
        done
    fi
done

print_status "Integration mocks generated successfully!"
