.PHONY: help dev sqlc proto buf-lint no-binaries test test-e2e tx-audit test-ledger test-verbose test-sql-prepare-smoke install-tools install-ci-tools docs mocks lint gosec gosec-fast govet static-check check-format jaeger-tracing connect-minikube version validate-openapi-specs httpie local-db local-db-cli local-db-down local-db-nuke setup teardown migrate-create migrate-create-data migrate-up migrate-down migrate-status migrate-baseline migrate-data-up migrate-data-status migrate-agent-db migrate-agent-create migrate-agent-create-data migrate-agent-data migrate-agent-status seed-agent-db seed-core seed-user-photos seed-stripe teardown-stripe teardown-all-stripe fmt stripe-webhook stripe-webhook-account view-otel e2e-up e2e-up-ci e2e e2e-down fix-minikube-dns openapi openapi-quiet gen-agent-tools stainless openapi-stainless openapi-stainless-quiet generate generate-quiet install-stlc stlc-internal-sdk stlc-public-typescript-sdk stlc-public-python-sdk stlc-public-go-sdk stlc-public-sdks stlc-sdks sdk-yalc

# Include .env file if it exists (optional for CI)
-include .env
# Include .env.test file if it exists (for local testing)
-include .env.test
export $(shell [ -f .env ] && sed 's/=.*//' .env || echo "")
export $(shell [ -f .env.test ] && sed 's/=.*//' .env.test || echo "")

# Default database for the migration targets. Override with TARGET=branch or TARGET=prod.
TARGET ?= local

PROTO_DIR := proto
PROTO_SRC := $(shell find $(PROTO_DIR) -name '*.proto' -print | sort)
GO_OUT := .
# Destination of the generated bindings (from each .proto's go_package). Contains
# nothing but generated files, so `make proto` sweeps it clean before regenerating.
PROTO_GEN_DIR := shared/proto
MOCK_SCRIPT := ./scripts/generate-mocks.sh
SQLC_SCRIPT := ./scripts/generate-sqlc.sh

# Extract arguments after the target (e.g., make mocks auth)
ARGS := $(wordlist 2,$(words $(MAKECMDGOALS)),$(MAKECMDGOALS))

# Reads a pinned tool version from tools/tool-versions
tool-version = $(shell grep '$(1) ' tools/tool-versions | awk '{print $$2}')

# protoc is a C++ binary (not `go install`-able), so it is pinned here and the
# `proto` target checks the local binary matches before regenerating. A drifting
# protoc rewrites the version stamp in every generated file.
PROTOC_VERSION := $(shell grep '^protoc ' tools/tool-versions | awk '{print $$2}')

# Default target
help: ## Show this help message
	@echo "Available targets:"
	@grep -E '^[a-zA-Z0-9_-]+:.*## .*$$' Makefile | sort | awk 'BEGIN {FS = ":.*## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'
	
connect-minikube: ## Switch kubectl context to minikube
	@kubectl config use-context minikube

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

openapi: ## Generate OpenAPI specifications (specs only, no Stainless configs)
	@mkdir -p specs
	@cd tools && GOTOOLCHAIN=go1.27.0 go run ./apidocs --name api --skip-stainless

openapi-quiet: ## Generate OpenAPI specifications without informational output (specs only)
	@mkdir -p specs
	@cd tools && GOTOOLCHAIN=go1.27.0 go run ./apidocs --name api --skip-stainless --quiet

gen-agent-tools: ## Generate the agent-service endpoint-tool catalog + DB seed from endpoints flagged AgentTool=true
	@cd tools && GOTOOLCHAIN=go1.27.0 go run ./apidocs --agent-tools --root ..

stainless: ## Generate Stainless SDK configs only (no OpenAPI specs)
	@cd tools && GOTOOLCHAIN=go1.27.0 go run ./apidocs --name api --only-stainless

openapi-stainless: ## Generate both OpenAPI specs and Stainless SDK configs
	@mkdir -p specs
	@cd tools && GOTOOLCHAIN=go1.27.0 go run ./apidocs --name api

openapi-stainless-quiet: ## Generate both OpenAPI specs and Stainless SDK configs (no informational output)
	@mkdir -p specs
	@cd tools && GOTOOLCHAIN=go1.27.0 go run ./apidocs --name api --quiet

generate: sqlc proto ## Generate sqlc code, protobuf bindings, OpenAPI specs, Stainless configs, and agent tools
	@mkdir -p specs
	@cd tools && GOTOOLCHAIN=go1.27.0 go run ./apidocs --name api --with-agent-tools

generate-quiet: sqlc proto ## Generate sqlc code, protobuf bindings, OpenAPI specs, Stainless configs, and agent tools (no informational output)
	@mkdir -p specs
	@cd tools && GOTOOLCHAIN=go1.27.0 go run ./apidocs --name api --with-agent-tools --quiet

httpie: ## Generate HTTPie workspace file
	@mkdir -p httpie
	@cd tools && go run ./apidocs --httpie --root ..

validate-openapi-specs: ## Validate OpenAPI specifications with vacuum
	@./scripts/validate-openapi-specs.sh

# --- STLC (Stainless CLI) ----------------------------------------------------
# Forks: github.com/sdk-gen/stlc, sdk-gen/stlc-{typescript,python,go} — see docs/stlc-sdk-codegen.md.
# Optional: STLC_BUILD_EXTRA='--commit "chore: regenerate SDK"'
install-stlc: ## Install stlc + the typescript/python/go workers from sdk-gen (uses gh auth token if STLC_READ_TOKEN unset)
	@./scripts/install-stlc.sh

stlc-internal-sdk: ## SDK: open-mrp/internal-sdk from stainless/internal
	@command -v stlc >/dev/null 2>&1 || { printf '%s\n' 'stlc not on PATH — install per docs/stlc-sdk-codegen.md' >&2; exit 127; }
	stlc build --workspace stainless/internal --targets typescript $(STLC_BUILD_EXTRA)

# The public TS target also generates the MCP server (packages/mcp-server). It needs
# the stlc-mcp worker installed and NODE_OPTIONS=--preserve-symlinks so the
# stlc-typescript + stlc-mcp plugins share one codegen.lib.mjs (else stlc reports the
# stlc-mcp plugin as missing). See docs/stlc-sdk-codegen.md.
stlc-public-typescript-sdk: ## SDK: open-mrp/typescript-sdk (@openmrp/sdk) + MCP server from stainless/public
	@command -v stlc >/dev/null 2>&1 || { printf '%s\n' 'stlc not on PATH — install per docs/stlc-sdk-codegen.md' >&2; exit 127; }
	NODE_OPTIONS=--preserve-symlinks stlc build --workspace stainless/public --targets typescript $(STLC_BUILD_EXTRA)

stlc-public-python-sdk: ## SDK: open-mrp/python-sdk (openmrp on PyPI) from stainless/public
	@command -v stlc >/dev/null 2>&1 || { printf '%s\n' 'stlc not on PATH — install per docs/stlc-sdk-codegen.md' >&2; exit 127; }
	stlc build --workspace stainless/public --targets python $(STLC_BUILD_EXTRA)

stlc-public-go-sdk: ## SDK: open-mrp/openmrp-go (github.com/open-mrp/openmrp-go) from stainless/public
	@command -v stlc >/dev/null 2>&1 || { printf '%s\n' 'stlc not on PATH — install per docs/stlc-sdk-codegen.md' >&2; exit 127; }
	stlc build --workspace stainless/public --targets go $(STLC_BUILD_EXTRA)

stlc-public-sdks: stlc-public-typescript-sdk stlc-public-python-sdk stlc-public-go-sdk ## Regenerate all public SDK targets

stlc-sdks: stlc-internal-sdk stlc-public-sdks ## Regenerate every SDK workspace/target

sdk-yalc: ## Rebuild @openmrp/internal-sdk from current api, publish to yalc, link into dashboard (local testing). Flags via the script: --skip-regen, --no-link.
	@./scripts/regen-sdk-yalc.sh

sqlc: ## Generate code from SQL queries using sqlc. Usage: make sqlc [services]
	@$(SQLC_SCRIPT) $(ARGS)
	@$(MAKE) fmt

proto: ## Generate Go protobuf bindings
	@have=$$(protoc --version | awk '{print $$2}'); \
		if [ "$$have" != "$(PROTOC_VERSION)" ]; then \
			echo "protoc $$have found, but $(PROTOC_VERSION) is pinned (tools/tool-versions)."; \
			echo "Install protoc $(PROTOC_VERSION) — see README — to avoid version-stamp drift."; \
			exit 1; \
		fi
	find $(PROTO_GEN_DIR) -name '*.pb.go' -delete
	protoc \
		--proto_path=$(PROTO_DIR) \
		--go_out=$(GO_OUT) \
		--go-grpc_out=$(GO_OUT) \
		$(PROTO_SRC)
	@$(MAKE) fmt

buf-lint: ## Run buf lint (requires: make install-tools, buf.work.yaml at repo root, GOPATH/bin on PATH)
	@command -v buf >/dev/null || (echo "buf not found. Run: make install-tools  (ensure go env GOPATH bin is on PATH)" && exit 1)
	@buf lint

local-db: ## Start local databases, apply migrations, and seed data
	@./scripts/setup-local-db.sh
	@if [ -n "$(STRIPE_SECRET_KEY)" ]; then \
		./scripts/seed-stripe-subscription.sh; \
	else \
		echo "\033[0;33m[WARN]\033[0m STRIPE_SECRET_KEY not set — skipping Stripe subscription seed. Run 'make seed-stripe' later."; \
	fi

local-db-cli: ## Open MySQL CLI to the local core database (uses DB_URL from .env)
	@./scripts/local-db-cli.sh $(ARGS)

local-db-down: ## Tear down local database containers (data preserved)
	@docker compose down

local-db-nuke: ## Tear down local databases, clean up Stripe resources, and delete all data
	@if [ -n "$(STRIPE_SECRET_KEY)" ]; then \
		./scripts/teardown-stripe-subscription.sh || true; \
	fi
	@docker compose down -v --remove-orphans

setup: ## Start minikube and local databases
	@minikube start --cpus=4 --memory=8192 --driver=docker && $(MAKE) local-db

teardown: ## Delete minikube, nuke local databases, and tear down the E2E stack
	@minikube delete && $(MAKE) local-db-nuke && $(MAKE) e2e-down

migrate-create: ## Create a core-service schema migration. Usage: make migrate-create name=add_foo
	@./scripts/migrate.sh create $(name)

migrate-create-data: ## Create a core-service data (backfill) migration. Usage: make migrate-create-data name=backfill_foo
	@./scripts/migrate.sh create-data $(name)

migrate-up: ## Apply pending core-service schema migrations. Usage: make migrate-up [TARGET=local|branch]
	@./scripts/migrate.sh up --target $(TARGET)

migrate-down: ## Roll back the most recent core-service schema migration. Usage: make migrate-down [TARGET=local|branch]
	@./scripts/migrate.sh down --target $(TARGET)

migrate-status: ## Show core-service schema migration status. Usage: make migrate-status [TARGET=local|branch|prod]
	@./scripts/migrate.sh status --target $(TARGET)

migrate-baseline: ## Record the baseline as applied on a database that already has the schema. Usage: make migrate-baseline TARGET=branch
	@./scripts/migrate.sh baseline --target $(TARGET)

migrate-data-up: ## Apply pending core-service data migrations. Usage: make migrate-data-up [TARGET=local|branch|prod]
	@./scripts/migrate.sh data-up --target $(TARGET)

migrate-data-status: ## Show core-service data migration status. Usage: make migrate-data-status [TARGET=local|branch|prod]
	@./scripts/migrate.sh data-status --target $(TARGET)

migrate-agent-db: ## Apply migrations to the agent-service PostgreSQL database
	@GOOSE_DRIVER=postgres GOOSE_DBSTRING="$$AGENT_DB_URL" \
		goose -dir services/agent-service/db/migrations up

migrate-agent-create: ## Create an agent-service schema migration. Usage: make migrate-agent-create name=add_foo
	@goose -s -dir services/agent-service/db/migrations create $(name) sql

migrate-agent-create-data: ## Create an agent-service data (backfill) migration. Usage: make migrate-agent-create-data name=backfill_foo
	@goose -s -table goose_db_version_data -dir services/agent-service/db/data-migrations create $(name) sql

migrate-agent-data: ## Apply agent-service data migrations to the PostgreSQL database
	@GOOSE_DRIVER=postgres GOOSE_DBSTRING="$$AGENT_DB_URL" \
		goose -table goose_db_version_data -dir services/agent-service/db/data-migrations up

migrate-agent-status: ## Show agent-service schema + data migration status
	@GOOSE_DRIVER=postgres GOOSE_DBSTRING="$$AGENT_DB_URL" \
		goose -dir services/agent-service/db/migrations status
	@GOOSE_DRIVER=postgres GOOSE_DBSTRING="$$AGENT_DB_URL" \
		goose -table goose_db_version_data -dir services/agent-service/db/data-migrations status

seed-agent-db: ## Seed agent-service DB with e2e test data
	@GOOSE_DRIVER=postgres GOOSE_DBSTRING="$$AGENT_DB_URL" \
		goose -table goose_db_version_seeds -dir services/agent-service/db/seeds up

seed-core: ## Seed core-service DB with sample data
	@./scripts/seed-core-db.sh $(ARGS)

seed-user-photos: ## Upload seeded account-users' avatar images to the user-photos S3 bucket
	@./scripts/seed-user-photos.sh

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

test-ledger: ## Run inventory ledger concurrency tests against the local MySQL (requires local-db)
	@echo "Running ledger concurrency tests..."
	@time go test -tags ledger -v -count=1 -timeout 300s -p 1 -parallel 1 \
		./services/core-service/internal/infrastructure/repository

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

lint: gosec static-check tx-audit vtparse no-binaries ## Run gosec + staticcheck + transaction-callback audit + Vitess parse check + committed-binary check

no-binaries: ## Check that no compiled binary or oversized file is tracked in git
	@./scripts/check-no-binaries.sh

tx-audit: ## Check that database transaction callbacks are safe to re-run after a deadlock
	@echo "Auditing transaction callbacks..."
	@cd tools && GOTOOLCHAIN=go1.27.0 go run ./txaudit --root ..

vtparse: ## Check that every generated MySQL query parses on Vitess (PlanetScale rejects valid MySQL)
	@echo "Parsing generated queries on Vitess..."
	@cd tools && GOTOOLCHAIN=go1.27.0 go run ./vtparse --root ..

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

fmt: ## Format Go source code
	@echo "Formatting Go source code..."
	@go fmt ./...
	@if command -v goimports >/dev/null; then \
		echo "Organizing imports with goimports..."; \
		goimports -w .; \
	fi

stripe-webhook: ## Run the Stripe webhook listener
	@stripe listen --forward-to localhost:8081/v1/webhooks/stripe

# Defaults to the seeded Acme Inc. vendor account (shared/db/seed/0003_accounts.sql).
stripe-webhook-account: ACCOUNT ?= ac_01k0a5smf9ekb8rqg12555zjqa
stripe-webhook-account: ## Forward a vendor account's Stripe webhooks to the per-account endpoint. Usage: make stripe-webhook-account [ACCOUNT=<account_id>] [API_KEY=sk_test_...]
	@echo "Forwarding Stripe webhooks for account $(ACCOUNT)"
	@stripe listen $(if $(API_KEY),--api-key $(API_KEY)) --events payment_intent.succeeded,payment_intent.payment_failed,payment_intent.canceled,payout.paid --forward-to localhost:8081/v1/webhooks/stripe/accounts/$(ACCOUNT)

check-format: ## Check formatting
	@echo "Checking formatting..."
	@if [ -n "$$(gofmt -l .)" ]; then \
		echo "The following files are not formatted. Please run 'go fmt ./...':"; \
		gofmt -l .; \
		exit 1; \
	fi

view-otel: ## Open Jaeger UI via port-forward
	@echo "Opening Jaeger UI at http://localhost:16686"
	kubectl port-forward svc/jaeger 16686:16686

e2e-up: openapi-quiet ## Start the E2E stack (isolated services + seeded DBs)
	@./scripts/run-quiet.sh "Building E2E service images" docker compose -f docker-compose.e2e.yml build --parallel
	@./scripts/run-quiet.sh "Clearing leftover E2E containers" ./scripts/e2e-rm-named-containers.sh
	@./scripts/run-quiet.sh "Starting E2E databases" docker compose -f docker-compose.e2e.yml up -d --wait mysql-e2e postgres-e2e rabbitmq minio-e2e
	@./scripts/setup-e2e-db.sh
	@./scripts/run-quiet.sh "Starting E2E services" ./scripts/start-e2e-services.sh

e2e-up-ci: openapi-quiet ## Start the E2E stack using pre-built images (for CI)
	@./scripts/run-quiet.sh "Clearing leftover E2E containers" ./scripts/e2e-rm-named-containers.sh
	@./scripts/run-quiet.sh "Starting E2E databases" docker compose -f docker-compose.e2e.yml up -d --wait mysql-e2e postgres-e2e rabbitmq minio-e2e
	@./scripts/setup-e2e-db.sh
	@./scripts/run-quiet.sh "Starting E2E services" ./scripts/start-e2e-services.sh

e2e: e2e-up ## Run API E2E tests against the full stack (brings the stack up first)
	@echo "Running API E2E tests..."
	@time ./scripts/run-e2e-tests.sh 600s

e2e-down: ## Tear down the E2E stack
	@./scripts/run-quiet.sh "Clearing leftover E2E containers" ./scripts/e2e-rm-named-containers.sh
	@./scripts/run-quiet.sh "Tearing down E2E stack" docker compose -f docker-compose.e2e.yml down -v --remove-orphans

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
