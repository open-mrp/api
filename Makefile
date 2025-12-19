.PHONY: help dev gen-sqlc gen-proto migrate-up migrate-down migrate-status migrate-reset db-dump test test-verbose install-tools docs mocks seed gosec static-check check-format docker-build-prod docker-push-prod deploy-prod jaeger-tracing connect-minikube connect-eks check-ci

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

# Default target
help: ## Show this help message
	@echo "Available targets:"
	@grep -E '^[a-zA-Z_-]+:.*## .*$$' Makefile | sort | awk 'BEGIN {FS = ":.*## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

check-ci:
	@./scripts/check-ci.sh

docker-build-prod: check-ci ## Build Docker images for production. Usage: make docker-build-prod [service]
	@./scripts/build-docker.sh $(ARGS)

docker-push-prod: check-ci ## Push Docker images to ECR. Usage: make docker-push-prod [service]
	@./scripts/push-docker.sh $(ARGS)

deploy-prod: check-ci ## Deploy application to EKS
	@./scripts/deploy.sh

tf-init: ## Initialize Terraform
	terraform -chdir=infra/production/terraform init

tf-fmt: ## Check Terraform formatting
	terraform -chdir=infra/production/terraform fmt -check

tf-validate: ## Validate Terraform configuration
	terraform -chdir=infra/production/terraform validate

tf-plan: ## Generate Terraform execution plan
	terraform -chdir=infra/production/terraform plan -out=tfplan

tf-apply: check-ci ## Apply Terraform changes
	terraform -chdir=infra/production/terraform apply -auto-approve tfplan

reset-prod: check-ci
	kubectl rollout restart deployment api-gateway auth-service logging-service notification-service

manual-deploy-prod: check-ci
	$(MAKE) docker-build-prod
	$(MAKE) docker-push-prod
	$(MAKE) deploy-prod
	$(MAKE) reset-prod
	
connect-minikube: ## Switch kubectl context to minikube
	@kubectl config use-context minikube

connect-eks: ## Switch kubectl context to EKS production cluster
	@aws eks update-kubeconfig --region us-east-2 --name augno-prod

dev: ## Run the API in development mode
	tilt up

docs: ## Generate OpenAPI documentation
	go run ./cmd/apidocs --name api
	@./scripts/validate-docs.sh

gen-sqlc: ## Generate code from SQL queries using sqlc. Usage: make gen-sqlc [services]
	@$(SQLC_SCRIPT) $(ARGS)

gen-proto: ## Generate Go protobuf bindings
	protoc \
		--proto_path=$(PROTO_DIR) \
		--go_out=$(GO_OUT) \
		--go-grpc_out=$(GO_OUT) \
		$(PROTO_SRC)

migrate-up: ## Run database migrations up
	@./scripts/migrate.sh up

migrate-down: ## Rollback database migrations
	@./scripts/migrate.sh down

migrate-status: ## Show migration status
	@./scripts/migrate.sh status

migrate-reset: ## Reset database
	@./scripts/migrate.sh reset

db-dump: ## Dump the database
	@./scripts/dump-dev-db.sh

seed: ## Seed the database with development data
	@./scripts/seed-db.sh

test: ## Run tests
	@echo "Running tests..."
	@time go test ./...

test-verbose: ## Run tests with verbose output
	@echo "Running tests with verbose output..."
	@time go test -v ./...

install-tools: ## Install required development tools
	@./scripts/install-tools.sh

install-ci-tools: ## Install minimum tools for CI
	@./scripts/install-ci-tools.sh

mocks: ## Generate mocks. Usage: make mocks [services]
	@$(MOCK_SCRIPT) $(ARGS)
	@if [ -z "$(ARGS)" ] || echo "$(ARGS)" | grep -q "integration"; then ./scripts/generate-integration-mocks.sh; fi

gosec: ## Run gosec
	@echo "Running gosec..."
	@gosec -exclude-generated -exclude-dir=sqlc,proto ./...

static-check: ## Run staticcheck
	@echo "Running static check..."
	@staticcheck ./...

check-format: ## Check formatting
	@echo "Checking formatting..."
	@if [ -n "$$(gofmt -l .)" ]; then \
		echo "The following files are not formatted. Please run 'go fmt ./...':"; \
		gofmt -l .; \
		exit 1; \
	fi

jaeger-tracing: ## Open Jaeger UI via port-forward
	@echo "Opening Jaeger UI at http://localhost:16686"
	kubectl port-forward svc/jaeger 16686:16686

# This catch-all target prevents make from complaining about unknown targets
%:
	@:
