# Domain Layer Patterns

This document describes the domain directory structure convention used by all backend services (excluding `api-gateway`).

## Directory Structure

```
services/{name}/
├── cmd/
│   ├── main.go        # Entry point — calls Run()
│   ├── run.go         # Initializes infrastructure, wires dependencies, starts server
│   └── config.go      # Configuration loading from environment
└── internal/
    └── domain/
        ├── models.go           # Domain entity structs
        ├── services.go         # Service interface definitions
        ├── repositories.go     # Repository interface definitions
        ├── factories.go        # Factory interface (creates repos with tx support)
        ├── mediators.go        # Mediator interface definitions (reusable business logic steps)
        ├── constants.go        # Service-local enum values
        ├── recovery_points.go  # Recovery point constants for idempotent operations
        └── mock/               # Generated mock implementations
            ├── factorymock/
            ├── mediatormock/
            ├── repositorymock/
            ├── servicemock/
            ├── clientmock/     # When the service has gRPC clients
            └── publishermock/  # When the service publishes messages
```

## Standard Files

### `models.go`
Domain entity structs. These are the core data types that flow through the service.

When a domain type is compared with `shared/audit.ComputeChanges` for audit events, add an **`audit` struct tag** on each field that should appear in audit timelines with a stable, API-oriented key. Fields without the tag are omitted from diffs. See `docs/patterns/audit-event-patterns.md`.

### `services.go`
Interface definitions for the service layer (business logic). Each interface method typically corresponds to a use case.

### `repositories.go`
Interface definitions for the data access layer. Implementations live in `internal/infrastructure/repository/`.

### `factories.go`
Factory interfaces that create repository instances, supporting transaction scoping.

### `mediators.go`
Interfaces for mediators — reusable multi-step business logic that can be composed across service methods.

### `recovery_points.go`
String constants identifying idempotent checkpoints within multi-step operations.

## Optional Files

| File | When to add |
|------|-------------|
| `clients.go` | Service makes gRPC calls to other services |
| `publishers.go` | Service publishes messages to RabbitMQ |
| `constants.go` | Service has enum values not shared with other services |

## Mock Generation

Mocks are generated via:

```bash
make mocks [service]   # e.g., make mocks auth
```

Each interface gets its own mock subdirectory under `mock/` (e.g., `servicemock/`, `repositorymock/`).

## Entry Point Pattern

All services use an identical `main.go`:

```go
func main() {
    ctx := context.Background()
    if err := Run(ctx, os.Getenv, os.Stdin, os.Stdout, os.Stderr); err != nil {
        fmt.Fprintf(os.Stderr, "%s\n", err)
        os.Exit(1)
    }
}
```

`Run()` in `run.go` handles all initialization and wiring. See `docs/main-delegates-to-run-pattern.md` for details.

## Exceptions

- **api-gateway**: HTTP service with minimal domain layer (no `services.go`). Does not follow the full pattern.
