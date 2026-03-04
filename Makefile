.PHONY: help dev gen-sqlc gen-proto db-dump test test-verbose install-tools docs mocks gosec static-check check-format jaeger-tracing connect-minikube connect-eks version validate-openapi-specs

# Include .env file if it exists (optional for CI)
-include .env
# Include .env.test file if it exists (for local testing)
-include .env.test
export $(shell [ -f .env ] && sed 's/=.*//' .env || echo "")
export $(shell [ -f .env.test ] && sed 's/=.*//' .env.test || echo "")

PROTO_DIR := proto
PROTO_SRC := $(wildcard $(PROTO_DIR)/*.proto)
GO_OUT := .
MOCK_SCRIPT := ./scripts/generate-mocks.sh
SQLC_SCRIPT := ./scripts/generate-sqlc.sh

# Extract arguments after the target (e.g., make mocks auth)
ARGS := $(wordlist 2,$(words $(MAKECMDGOALS)),$(MAKECMDGOALS))

# Reads a pinned tool version from tools/tool-versions
tool-version = $(shell grep '$(1) ' tools/tool-versions | awk '{print $$2}')

# Default target
help: ## Show this help message
	@echo "Available targets:"
	@grep -E '^[a-zA-Z_-]+:.*## .*$$' Makefile | sort | awk 'BEGIN {FS = ":.*## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'
	
connect-minikube: ## Switch kubectl context to minikube
	@kubectl config use-context minikube

connect-eks: ## Switch kubectl context to EKS production cluster
	@aws eks update-kubeconfig --region us-east-2 --name augno-prod

dev: ## Run the API in development mode
	tilt up

gen-openapi-specs: ## Generate OpenAPI specifications
	@cd tools && go run ./apidocs --name api

validate-openapi-specs: ## Validate OpenAPI specifications with vacuum
	@./scripts/validate-openapi-specs.sh

gen-sqlc: ## Generate code from SQL queries using sqlc. Usage: make gen-sqlc [services]
	@$(SQLC_SCRIPT) $(ARGS)

gen-proto: ## Generate Go protobuf bindings
	protoc \
		--proto_path=$(PROTO_DIR) \
		--go_out=$(GO_OUT) \
		--go-grpc_out=$(GO_OUT) \
		$(PROTO_SRC)

db-dump: ## Dump the database
	@./scripts/dump-dev-db.sh

test: ## Run tests
	@echo "Running tests..."
	@time go test ./...

test-verbose: ## Run tests with verbose output
	@echo "Running tests with verbose output..."
	@time go test -v ./...

install-tools: ## Install required development tools
	@go install github.com/sqlc-dev/sqlc/cmd/sqlc@$(call tool-version,github.com/sqlc-dev/sqlc)
	@go install github.com/pressly/goose/v3/cmd/goose@$(call tool-version,github.com/pressly/goose/v3)
	@go install gotest.tools/gotestsum@$(call tool-version,gotest.tools/gotestsum)
	@go install go.uber.org/mock/mockgen@$(call tool-version,go.uber.org/mock)
	@go install github.com/daveshanley/vacuum@$(call tool-version,github.com/daveshanley/vacuum)
	@go install google.golang.org/protobuf/cmd/protoc-gen-go@$(call tool-version,google.golang.org/protobuf)
	@go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@$(call tool-version,google.golang.org/grpc/cmd/protoc-gen-go-grpc)
	@go install github.com/goreleaser/goreleaser/v2@$(call tool-version,github.com/goreleaser/goreleaser/v2)
	@go install github.com/securego/gosec/v2/cmd/gosec@$(call tool-version,github.com/securego/gosec/v2)
	@go install honnef.co/go/tools/cmd/staticcheck@$(call tool-version,honnef.co/go/tools)
	@go install golang.org/x/tools/cmd/goimports@$(call tool-version,golang.org/x/tools)

install-ci-tools: ## Install minimum tools for CI
	@go install github.com/securego/gosec/v2/cmd/gosec@$(call tool-version,github.com/securego/gosec/v2)
	@go install honnef.co/go/tools/cmd/staticcheck@$(call tool-version,honnef.co/go/tools)
	@go install github.com/daveshanley/vacuum@$(call tool-version,github.com/daveshanley/vacuum)

mocks: ## Generate mocks. Usage: make mocks [services]
	@$(MOCK_SCRIPT) $(ARGS)

gosec: ## Run gosec
	@echo "Running gosec..."
	@gosec -exclude-generated -exclude-dir=sqlc -exclude-dir=proto -exclude-dir=tools ./...

static-check: ## Run staticcheck
	@echo "Running static check..."
	@staticcheck ./...

fmt: ## Format Go source code
	@echo "Formatting Go source code..."
	@go fmt ./...
	@if command -v goimports >/dev/null; then \
		echo "Organizing imports with goimports..."; \
		goimports -w .; \
	fi

stripe-webhook: ## Run the Stripe webhook listener
	@stripe listen --forward-to localhost:8081/v1/webhooks/stripe

check-format: ## Check formatting
	@echo "Checking formatting..."
	@if [ -n "$$(gofmt -l .)" ]; then \
		echo "The following files are not formatted. Please run 'go fmt ./...':"; \
		gofmt -l .; \
		exit 1; \
	fi

open-tracing: ## Open Jaeger UI via port-forward
	@echo "Opening Jaeger UI at http://localhost:16686"
	kubectl port-forward svc/jaeger 16686:16686

# Version management
version: ## Show current version
	@go run ./cmd/print-version

# This catch-all target prevents make from complaining about unknown targets
%:
	@:
