---
name: domain-layer
description: >-
  Domain directory layout for backend microservices: models, services, repositories,
  factories, mediators, recovery points, mock generation. Use when adding a domain
  model, interface, mock, factory, or entry point under services/*/internal/domain.
---

# Domain layer

Governs `services/{name}/internal/domain/` in each microservice — not api-gateway endpoints. Human spec: `docs/patterns/domain-layer-patterns.md`. Entry-point `main()` → `Run()`: `main-delegates-to-run` skill.

```
internal/domain/
  models.go           # or *_models.go in large services (core-service)
  services.go
  repositories.go
  factories.go        # required whenever the service owns repository interfaces
  constants.go        # service-local enums only; shared enums go in shared/constants
  mediators.go        # when the service uses mediators
  recovery_points.go  # when it uses RecoveryPoint checkpoints
  mock/{factory,mediator,repository,service,client,publisher}/
```

## Conditional files

| File | When |
|---|---|
| `clients.go` | gRPC calls to other services |
| `publishers.go` | publishes RabbitMQ messages |
| `mediators.go` | composes mediator steps (auth, billing, core, agent). Omit until needed (platform, notification). |
| `recovery_points.go` | multi-phase idempotency. May split (`supplier_recovery_points.go`) when clearer. |

## Rules

- Domain types that `shared/audit.ComputeChanges` diffs need an `audit` struct tag per timeline field. See `audit-events`.
- Thin services still define `factories.go` when they own `repositories.go`, so mock gen picks up `RepoFactory`.
- `make mocks [service]` — `scripts/generate-mocks.sh` reads the interface files that exist. Output dirs: `factory/`, `mediator/`, `repository/`, `service/`, `client/`, `publisher/` (packages `*mock`).
- Infra helpers that never join `RepoFactory` tx scoping (`LeaseRepo`, `InboxRepo`) may be constructed directly in `run.go`.

## Exceptions

**api-gateway** has a minimal domain package — no `services.go` / `repositories.go` / transactional factories. Do not retrofit the backend layout there.
