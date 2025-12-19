#!/bin/bash

# check-ci.sh
# Helper script to ensure commands are run in CI unless overridden

RED='\033[0;31m'
NC='\033[0m'

if [ "$GITHUB_ACTIONS" != "true" ] && [ "$ALLOW_MANUAL_DEPLOY" != "true" ]; then
    echo -e "${RED}[ERROR]${NC} This command is restricted to CI (GitHub Actions) to prevent accidental manual production deployments."
    echo -e "To override this (USE WITH CAUTION!), set ALLOW_MANUAL_DEPLOY=true"
    exit 1
fi
