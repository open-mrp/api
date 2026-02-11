# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Development Commands

```bash
make dev                      # Start local dev environment (Tilt + K8s)
make test                     # Run all tests
go test ./services/auth-service/...  # Run tests for specific service
go test -run TestFunctionName ./path/to/package  # Run single test
make gen-sqlc [service]       # Generate DB code from SQL (e.g., make gen-sqlc auth)
make gen-proto                # Generate protobuf bindings
make mocks [service]          # Generate mock implementations
make fmt                      # Format Go code
make gosec                    # Security scanning
make static-check             # Static analysis
```

## Architecture Overview

Go microservices platform with gRPC inter-service communication and RabbitMQ for async messaging.

### Services

- **api-gateway**: HTTP entry point, routes to backend services via gRPC
- **auth-service**: Authentication, JWT tokens, password management
- **core-service**: Account and business entity management
- **notification-service**: Email notifications
- **platform-service**: Platform utilities and idempotency

### Service Internal Structure

Each service follows this layered pattern:
```
services/[name]/
├── cmd/                    # Entry point (main.go, run.go, config.go)
├── internal/
│   ├── service/            # Business logic layer
│   ├── domain/             # Domain models, interfaces, mocks
│   ├── infrastructure/
│   │   ├── repository/     # Data access implementations
│   │   ├── grpc/           # gRPC handlers and clients
│   │   ├── queries/        # SQL query definitions
│   │   └── sqlc/           # Generated DB code
│   └── mediator/           # Reusable business logic steps
└── pkg/                    # Public types for other services
```

### Shared Code (`shared/`)

- **contracts/**: gRPC interceptors, identity propagation, API error encoding
- **errors/**: Centralized API error types with HTTP/gRPC mapping
- **id/**: Custom ID generation with entity prefixes (usr_, acct_, org_, etc.)
- **messaging/**: RabbitMQ integration with outbox/inbox patterns
- **db/**: Database pool management and migrations
- **constants/**: Domain enums (AccountMode, RoleType, PlanCode, etc.)

### Key Patterns

1. **Identity propagation**: Identity is serialized to gRPC metadata via `contracts.SetIdentityInMetadata()` and extracted with interceptors
2. **API error handling**: Errors are encoded in gRPC messages with `apiErrorMarker` prefix, decoded by `ConvertGRPCError()`
3. **Repository factory pattern**: Each service has a factory that creates repositories with transaction support
4. **sqlc for type-safe SQL**: Queries defined in `queries/*.sql`, generated code in `sqlc/`

## Database

PlanetScale (MySQL) with safe migrations. Schema in `shared/db/migrations/0001_initial.sql`.

To add/modify schema:
1. Edit migration file
2. Run `make gen-sqlc [service]` for affected services
3. Create PlanetScale deploy request for production

## Code Style

- Do not create README files, examples, or comments unless explicitly requested
- Use Conventional Commits (feat:, fix:, feat!:)
