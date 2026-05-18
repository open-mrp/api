.PHONY: help dev sqlc proto buf-lint db-dump test test-verbose test-sql-prepare-smoke install-tools docs mocks lint gosec gosec-fast govet static-check check-format jaeger-tracing connect-minikube connect-eks version validate-openapi-specs httpie local-db local-db-down local-db-nuke seed-core seed-stripe teardown-stripe teardown-all-stripe fmt stripe-webhook open-tracing e2e-up e2e-up-ci e2e e2e-down fix-minikube-dns openapi openapi-quiet

# Include .env file if it exists (optional for CI)
-include .env
# Include .env.test file if it exists (for local testing)
-include .env.test
export $(shell [ -f .env ] && sed 's/=.*//' .env || echo "")
export $(shell [ -f .env.test ] && sed 's/=.*//' .env.test || echo "")

PROTO_DIR := proto
PROTO_SRC := $(shell find $(PROTO_DIR) -name '*.proto' -print | sort)
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

fix-minikube-dns: ## Ensure host.minikube.internal resolves inside minikube pods
	@HOST_IP=$$(minikube ssh "ip route | grep default" 2>/dev/null | awk '{print $$3}'); \
	if [ -z "$$HOST_IP" ]; then echo "⚠ minikube not running, skipping DNS fix"; exit 0; fi; \
	echo "Fixing host.minikube.internal → $$HOST_IP"; \
	minikube ssh "sudo sed -i '/host.minikube.internal/d' /etc/hosts && echo '$$HOST_IP	host.minikube.internal' | sudo tee -a /etc/hosts > /dev/null"; \
	CURRENT=$$(kubectl get configmap coredns -n kube-system -o jsonpath='{.data.Corefile}' 2>/dev/null); \
	if echo "$$CURRENT" | grep -q "host.minikube.internal"; then \
		echo "CoreDNS already patched"; \
	else \
		echo "Patching CoreDNS..."; \
		kubectl get configmap coredns -n kube-system -o yaml | \
			sed "s|kubernetes cluster.local|hosts {\n           $$HOST_IP host.minikube.internal\n           fallthrough\n        }\n        kubernetes cluster.local|" | \
			kubectl apply -f -; \
		kubectl rollout restart deployment coredns -n kube-system; \
	fi

dev: ## Run the API in development mode
	@$(MAKE) fix-minikube-dns
	tilt up

openapi: ## Generate OpenAPI specifications
	@mkdir -p specs
	@cd tools && GOTOOLCHAIN=go1.26.2 go run ./apidocs --name api

openapi-quiet: ## Generate OpenAPI specifications without informational output
	@mkdir -p specs
	@cd tools && GOTOOLCHAIN=go1.26.2 go run ./apidocs --name api --quiet

httpie: ## Generate HTTPie workspace file
	@mkdir -p httpie
	@cd tools && go run ./apidocs --httpie --root ..

validate-openapi-specs: ## Validate OpenAPI specifications with vacuum
	@./scripts/validate-openapi-specs.sh

sqlc: ## Generate code from SQL queries using sqlc. Usage: make sqlc [services]
	@$(SQLC_SCRIPT) $(ARGS)
	@$(MAKE) fmt

proto: ## Generate Go protobuf bindings
	protoc \
		--proto_path=$(PROTO_DIR) \
		--go_out=$(GO_OUT) \
		--go-grpc_out=$(GO_OUT) \
		$(PROTO_SRC)
	@$(MAKE) fmt

buf-lint: ## Run buf lint (requires: make install-tools, buf.work.yaml at repo root, GOPATH/bin on PATH)
	@command -v buf >/dev/null || (echo "buf not found. Run: make install-tools  (ensure go env GOPATH bin is on PATH)" && exit 1)
	@buf lint

db-dump: ## Dump the database
	@./scripts/dump-dev-db.sh

local-db: ## Start local databases, apply migrations, and seed data
	@./scripts/setup-local-db.sh
	@if [ -n "$(STRIPE_SECRET_KEY)" ]; then \
		./scripts/seed-stripe-subscription.sh; \
	else \
		echo "\033[0;33m[WARN]\033[0m STRIPE_SECRET_KEY not set — skipping Stripe subscription seed. Run 'make seed-stripe' later."; \
	fi

local-db-down: ## Tear down local database containers (data preserved)
	@docker compose down

local-db-nuke: ## Tear down local databases, clean up Stripe resources, and delete all data
	@if [ -n "$(STRIPE_SECRET_KEY)" ]; then \
		./scripts/teardown-stripe-subscription.sh || true; \
	fi
	@docker compose down -v --remove-orphans

migrate-agent-db: ## Apply migrations to the agent-service PostgreSQL database
	@GOOSE_DRIVER=postgres GOOSE_DBSTRING="$$AGENT_DB_URL" \
		goose -dir services/agent-service/db/migrations up

seed-agent-db: ## Seed agent-service DB with e2e test data
	@GOOSE_DRIVER=postgres GOOSE_DBSTRING="$$AGENT_DB_URL" \
		goose -table goose_db_version_seeds -dir services/agent-service/db/seeds up

seed-core: ## Seed core-service DB with sample data
	@./scripts/seed-core-db.sh $(ARGS)

seed-stripe: ## Create Stripe test subscription for seeded account
	@./scripts/seed-stripe-subscription.sh

teardown-stripe: ## Delete Stripe test resources and clear local DB fields
	@./scripts/teardown-stripe-subscription.sh

teardown-all-stripe: ## Cancel all subscriptions and delete all customers in Stripe
	@./scripts/teardown-all-stripe.sh

test: ## Run tests
	@echo "Running tests..."
	@time go test ./...
	@echo "Running tools tests..."
	@cd tools && go test ./...

test-e2e: ## Run E2E tests against a running stack (requires e2e-up)
	@echo "Running E2E tests..."
	@time ./scripts/run-e2e-tests.sh 300s

test-sql-prepare-smoke: ## Run sqlc Prepare smoke tests for MySQL services (requires local-db)
	@echo "Running SQL prepare smoke tests..."
	@time go test -tags integration -v -count=1 -timeout 120s \
		./services/api-gateway/internal/infrastructure/sqlc \
		./services/auth-service/internal/infrastructure/sqlc \
		./services/billing-service/internal/infrastructure/sqlc \
		./services/core-service/internal/infrastructure/sqlc \
		./services/notification-service/internal/infrastructure/sqlc \
		./services/platform-service/internal/infrastructure/sqlc

test-verbose: ## Run tests with verbose output
	@echo "Running tests with verbose output..."
	@time go test -v ./...
	@echo "Running tools tests (verbose)..."
	@cd tools && go test -v ./...

install-tools: ## Install required development tools
	@go install github.com/bufbuild/buf/cmd/buf@$(call tool-version,github.com/bufbuild/buf/cmd/buf)
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
	@go install github.com/sqlc-dev/sqlc/cmd/sqlc@$(call tool-version,github.com/sqlc-dev/sqlc)
	@go install gotest.tools/gotestsum@$(call tool-version,gotest.tools/gotestsum)
	@go install github.com/pressly/goose/v3/cmd/goose@$(call tool-version,github.com/pressly/goose/v3)

mocks: ## Generate mocks. Usage: make mocks [services]
	@$(MOCK_SCRIPT) $(ARGS)

lint: gosec static-check ## Run gosec + staticcheck

gosec: ## Run gosec (all rules)
	@echo "Running gosec..."
	@gosec -exclude-generated -exclude-dir=sqlc -exclude-dir=proto -exclude-dir=tools --concurrency=24 ./...

# Fast rules: pattern-matching only, excludes taint-tracking rules (G107,G201,G202,G203,G204).
# Update this list when upgrading gosec — new rules land in the full scan only.
GOSEC_FAST_RULES := G101,G102,G103,G104,G401,G402,G404

gosec-fast: ## Run gosec (fast pattern-matching rules only)
	@echo "Running gosec (fast rules)..."
	@gosec -include=$(GOSEC_FAST_RULES) -exclude-generated -exclude-dir=sqlc -exclude-dir=proto -exclude-dir=tools --concurrency=24 ./...

govet: ## Run go vet
	@echo "Running go vet..."
	@go vet ./...

static-check: ## Run staticcheck
	@echo "Running static check..."
	@go run honnef.co/go/tools/cmd/staticcheck@$(call tool-version,honnef.co/go/tools) ./...

fmt: ## Format Go source code and Terraform
	@echo "Formatting Go source code..."
	@go fmt ./...
	@if command -v goimports >/dev/null; then \
		echo "Organizing imports with goimports..."; \
		goimports -w .; \
	fi
	@echo "Formatting Terraform..."
	@if command -v terraform >/dev/null 2>&1; then \
		terraform -chdir=infra/production/terraform fmt; \
	else \
		echo "Terraform not found, skipping Terraform formatting"; \
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

e2e-up: openapi-quiet ## Start the E2E stack (isolated services + seeded DBs)
	@./scripts/run-quiet.sh "Building E2E service images" docker compose -f docker-compose.e2e.yml build --parallel
	@./scripts/run-quiet.sh "Starting E2E databases" docker compose -f docker-compose.e2e.yml up -d --wait mysql-e2e postgres-e2e rabbitmq
	@./scripts/setup-e2e-db.sh
	@./scripts/run-quiet.sh "Starting E2E services" docker compose -f docker-compose.e2e.yml up -d --wait

e2e-up-ci: openapi-quiet ## Start the E2E stack using pre-built images (for CI)
	@./scripts/run-quiet.sh "Starting E2E databases" docker compose -f docker-compose.e2e.yml up -d --wait mysql-e2e postgres-e2e rabbitmq
	@./scripts/setup-e2e-db.sh
	@./scripts/run-quiet.sh "Starting E2E services" docker compose -f docker-compose.e2e.yml up -d --wait

e2e: e2e-up ## Run API E2E tests against the full stack
	@echo "Running API E2E tests..."
	@time ./scripts/run-e2e-tests.sh 600s

e2e-down: ## Tear down the E2E stack
	@./scripts/run-quiet.sh "Tearing down E2E stack" docker compose -f docker-compose.e2e.yml down -v

# Version management
version: ## Show current version
	@go run ./cmd/print-version

# Targets that accept trailing args (e.g., `make mocks auth-service`)
ARG_FORWARDING_TARGETS := sqlc mocks seed-core

# Optional include files should not be treated as unknown targets.
.env .env.test:
	@:

# Keep arg forwarding behavior for known targets, but fail on unknown commands.
ifneq ($(filter $(ARG_FORWARDING_TARGETS),$(firstword $(MAKECMDGOALS))),)
%:
	@:
else
%:
	@echo "Error: unknown make target '$@'" >&2
	@echo "Run 'make help' to see available targets." >&2
	@exit 1
endif
