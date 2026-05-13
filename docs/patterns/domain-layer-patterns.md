# Domain Layer Patterns

This document describes the domain directory structure convention used by all backend microservices (excluding **`api-gateway`**). HTTP routes—including public ones—live in api-gateway and call backends over gRPC; this document governs **`services/{name}/internal/domain/`** in each microservice, not endpoint files.

## Directory Structure

```
services/{name}/
├── cmd/
│   ├── main.go        # Entry point — calls Run()
│   ├── run.go         # Initializes infrastructure, wires dependencies, starts server
│   └── config.go      # Configuration loading from environment
└── internal/
    └── domain/
        ├── models.go           # Domain entity structs (may be split; see below)
        ├── services.go         # Service interface definitions
        ├── repositories.go     # Repository interface definitions
        ├── factories.go        # Factory interface (creates repos; supports transaction scoping)
        ├── constants.go        # Service-local enum values (when needed)
        ├── mediators.go        # When the service uses mediators (see Conditional files)
        ├── recovery_points.go  # When the service uses multi-step idempotency checkpoints
        └── mock/               # Generated mock implementations (see Mock generation)
            ├── factory/
            ├── mediator/
            ├── repository/
            ├── service/
            ├── client/       # When the service has gRPC clients
            └── publisher/    # When the service publishes messages
```

## Standard files

### `models.go`

Domain entity structs—core data types that flow through the service.

Large services may also use additional **`*_models.go`** files (for example **`core-service`**) alongside or instead of a single monolithic **`models.go`**. Same rules apply everywhere: keep domain types cohesive and grouped by subsystem when splitting.

When a domain type is compared with `shared/audit.ComputeChanges` for audit events, add an **`audit` struct tag** on each field that should appear in audit timelines with a stable, API-oriented key. Fields without the tag are omitted from diffs. See `docs/patterns/audit-event-patterns.md`.

### `services.go`

Interface definitions for the service layer (business logic). Each interface method typically corresponds to a use case.

### `repositories.go`

Interface definitions for the data access layer. Implementations live in `internal/infrastructure/repository/`.

### `factories.go`

Factory interfaces that create repository instances (`NewXRepo()`), supporting consistent wiring and optional transaction-scoped construction the same way other services do (`core-service`, **`auth-service`**, **`platform-service`**, etc.).

Thin services **must** still define **`factories.go`** when they own domain **`repositories`** interfaces—even if repos are backed by one shared **`sqlc.Queries`** today—so composition matches the shared pattern and mock generation picks up **`RepoFactory`**.

### `constants.go`

Add when the service has enum-like values **not shared** via `shared/constants`.

## Conditional files

| File | When to add |
|------|-------------|
| `clients.go` | Service makes gRPC calls to other services |
| `publishers.go` | Service publishes messages to RabbitMQ |
| `mediators.go` | Service composes mediator steps (**`auth-service`**, **`billing-service`**, **`core-service`**, **`agent-service`**). Omit when unused (**`platform-service`**, **`notification-service`** until needed). |
| `recovery_points.go` | Service uses **`RecoveryPoint`** idempotency checkpoints in multi-phase flows (**`auth-service`**, **`billing-service`**, **`core-service`**, **`notification-service`** when wired). Related constants may live in a domain-specific **`*_recovery_points.go`** (for example **`supplier_recovery_points.go`** in **`core-service`**) when clearer than one global file. |

## Mock generation

Mocks are generated via:

```bash
make mocks [service]   # e.g., make mocks auth-service
```

**`scripts/generate-mocks.sh`** reads **`factories.go`**, **`mediators.go`**, **`publishers.go`**, **`repositories.go`**, **`services.go`**, **`clients.go`** (each only if present and declaring interfaces).

Output directories under **`internal/domain/mock/`** are named **`factory/`**, **`mediator/`**, **`repository/`**, **`service/`**, **`client/`**, **`publisher/`**. Generated Go **package** names remain **`factorymock`**, **`mediatormock`**, **`repositorymock`**, **`servicemock`**, **`clientmock`**, **`publishermock`** (see the script).

## Entry point pattern

All services use an identical **`main.go`**:

```go
func main() {
    ctx := context.Background()
    if err := Run(ctx, os.Getenv, os.Stdin, os.Stdout, os.Stderr); err != nil {
        fmt.Fprintf(os.Stderr, "%s\n", err)
        os.Exit(1)
    }
}
```

**`Run()`** in **`run.go`** handles all initialization and wiring. See **`docs/main-delegates-to-run-pattern.md`** for details.

## Exceptions

- **api-gateway**: HTTP entry point with a **minimal** domain package (no **`services.go`**, no **`repositories.go`**, no transactional **`factories`** pattern). Do not retrofit the full backend domain layout here.
- **Repos outside `factories.go`**: Infra helpers that never participate in the same transactional scoping pattern as **`RepoFactory`** (for example **`LeaseRepo`**, **`InboxRepo`** in **`notification-service`** `run.go`) may stay constructed directly alongside the factory-built domain repos unless you unify them deliberately.
