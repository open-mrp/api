#!/bin/bash

# Push Docker Images Script
# This script pushes Docker images to ECR

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
    echo "Error: Could not determine AWS Account ID. Ensure you are logged in to AWS."
    exit 1
fi

ECR_REGISTRY="${AWS_ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com"
SERVICES="api-gateway auth-service logging-service notification-service"

print_status "Logging into Amazon ECR..."
aws ecr get-login-password --region "$AWS_REGION" | docker login --username AWS --password-stdin "$ECR_REGISTRY"

for service in $SERVICES; do
    print_status "Pushing $service..."
    docker push "$ECR_REGISTRY/augno/$service:latest"
done

print_status "All images pushed successfully!"
