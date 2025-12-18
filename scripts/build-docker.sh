#!/bin/bash

# Build Docker Images Script
# This script builds Docker images for all services for production

set -e

# Colors for output
GREEN='\033[0;32m'
NC='\033[0m'

print_status() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

# Default values
AWS_REGION=${AWS_REGION:-us-east-2}
AWS_ACCOUNT_ID=$(aws sts get-caller-identity --query Account --output text 2>/dev/null || echo "unknown")

if [ "$AWS_ACCOUNT_ID" == "unknown" ]; then
    echo "Warning: Could not determine AWS Account ID. Ensure you are logged in to AWS."
fi

ECR_REGISTRY="${AWS_ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com"
SERVICES="api-gateway auth-service logging-service notification-service"

for service in $SERVICES; do
    print_status "Building $service..."
    docker build --platform linux/amd64 -t "$ECR_REGISTRY/augno/$service:latest" -f "infra/production/docker/$service.Dockerfile" .
done

print_status "All images built successfully!"
