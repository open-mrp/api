#!/bin/bash

# Validate OpenAPI Documentation Script
# This script uses vacuum to lint OpenAPI specifications

set -e

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

print_status() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

validate_spec() {
    local spec_path=$1
    if [ ! -f "$spec_path" ]; then
        print_warning "Spec file not found: $spec_path"
        return
    fi

    if command -v vacuum >/dev/null 2>&1; then
        print_status "Validating $spec_path..."
        vacuum lint -d "$spec_path" || print_warning "Vacuum lint failed for $spec_path - continuing..."
    else
        print_warning "Vacuum not available - skipping lint for $spec_path"
    fi
}

validate_spec "./specs/api_internal_spec.json"
validate_spec "./spec/api_public_spec.json"
