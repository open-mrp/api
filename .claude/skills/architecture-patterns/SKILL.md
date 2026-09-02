---
name: architecture-patterns
description: >-
  How to implement services, mediators, repositories, withTx, idempotency recovery
  points, tracing, APIError, and the inventory ledger lock order. Use when writing a
  service, mediator, repository, or transaction; when touching inventory_issue,
  inventory_receipt, inventory_allocation, ledgerlock, or any ledger-writing flow.
---

# Architecture (implementation)

Layer *does / does not* lives in the `openmrp-layers` skill. This skill is how to implement the Go backend. Human spec: `docs/patterns/architecture-patterns.md`. Doctrine: `dane-api-design`.

```
gRPC Handler
  └─ Service        (transaction boundary, idempotency, orchestration)
       ├─ Mediator   (reusable business logic; never opens a tx)
       │    └─ Repository
       └─ Repository (simple reads)
```

Business logic lives only in a service or mediator.

## Services

- Own the transaction: `withTx` builds a **new service instance** with a tx-scoped `RepoFactory` so every repo/mediator shares the `sql.Tx`.
- Own idempotency for mutating RPCs. POST/PATCH gRPC handlers also call `contracts.WithIdempotencyTracking`.
- Config-struct DI + `Default*Config` for production wiring.
- Call mediators for business logic / side effects / multi-step work; call repos directly for simple reads.

### Recovery points

1. Upsert the idempotency key (actor + service + handler + client key).
2. Switch:
   - `Finished` — return the cached response.
   - `Started` — do work; cache **success inside the tx** so cache and business state commit together.
3. Cache **only non-transient** errors. Transient errors must remain retryable.
4. Foreign (out-of-process) mutations get their own atomic phase with a recovery point so retries resume, never redo.
5. Unexpected recovery point → invariant violation.

Never make a network call inside a DB transaction.

## Mediators

- One discrete business step. Never open or manage a transaction; use the factory they were given.
- Rebuilt per tx via `svc.mediators()` so they see tx-scoped repos.
- May depend on other mediators; wired at factory build time.
- Domain interface and implementation share the same docstring (summary + numbered steps).

## Repositories

- The only database-aware layer. Wrap `*sqlc.Queries`, implement a domain interface.
- Map SQL errors with `db.MapSQLError()`. Nulls via `db.NullString` / `StringFromNullString` / `TimeFromNullTime`.
- Own tracer: `tracing.GetTracer("service.repo_name")`. Span names: `"<layer>.<entity>.<operation>"`.
- Never business logic, never permission checks, never their own transactions.

`RepoFactory` constructs every domain repo. When `queries` is bound to a `sql.Tx`, every repo from that factory shares it.

## Errors and tracing

- Business-layer functions return `*apierror.APIError`, never bare `error`.
- Record errors with `tracing.Trace(span, apiErr)`.
- Every method: `ctx, span := tracer.Start(ctx, "service.user.login"); defer span.End()`.

## Inventory ledger lock order

Applies to anything that writes `inventory_issue`, `inventory_receipt`, or `inventory_allocation`. Do this **before** any other ledger statement in the transaction.

1. Resolve the full set of item IDs the tx will touch.
2. `ledgerlock.Acquire(txCtx, locker, itemIDs)` as the **first** ledger statement. It sorts unique IDs and locks each via `LockItemForLedger`.
3. Pass the returned `*ledgerlock.Scope` into every ledger-writing repo method. Do not call `LockItemForLedger` yourself.
4. Mechanism: `INSERT INTO inventory_item_lock … ON DUPLICATE KEY UPDATE item_id = item_id`. **Never** `SELECT`, range-scan, or `DELETE` that table (gap locks on a cold insert deadlock the ordering root). See `inventory_item_lock.sql`.
5. Do not join `unit` (or any shared catalog table) under `FOR UPDATE`. Do not use `FOR UPDATE OF` (vtgate rejects it).

Late acquisition via `Scope.EnsureLocked` logs an error — it is a bug in the caller, not a supported path.
