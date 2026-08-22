---
name: openmrp-layers
description: The OpenMRP layer contract — what each layer (edge/transport, controller/handler, service, mediator, repository, database) does, must not do, and where every cross-cutting concern lives (auth, permissions, validation, idempotency, transactions, mutation, messaging, pagination, errors). Use when adding or reviewing any endpoint, service, or repository in the Go API or the dashboard Express API.
---

# The OpenMRP layer contract

Both APIs implement the same doctrine (see the `dane-api-design` skill for
the philosophy) with different embodiments:

- **Go API** (`api/`): HTTP edge = api-gateway (middleware chain + declarative
  `APIEndpoint.Execute`), then per backend service:
  **gRPC handler → service → (mediator) → repository → sqlc → DB**.
- **Dashboard API** (`dashboard/apps/api`): Express —
  **middleware → controller → service → (mediator) → repository → Prisma → DB**.

Code-anchored maps with file:line evidence: `references/go-api.md` and
`references/dashboard-api.md`. Read the relevant one before touching a layer.

## The layers — does / does not

### 1. Edge / transport (gateway middleware + `Execute`; Express middleware)

DOES: routing; authentication (credential extraction + validation — delegated
to auth-service / `authHandler`), attaching an Identity to the request
context; rate limiting; CLIENT idempotency keys (`Idempotency-Key` header:
replay / in-progress / hash-mismatch); request-shape validation (validate
tags / Zod schemas); unknown-field and explicit-null rejection; logging,
redaction of `sensitive` fields, request IDs; error → HTTP status mapping.

DOES NOT: business rules, permission checks, touching the application
database (exception: its own infrastructure tables — request logs, outbox),
calling repositories.

### 2. Controller / handler shell (gateway `service.go` per resource; gRPC handler; Express `*.ctrl.ts`)

A thin, logic-free translator. DOES: map transport types ↔ domain/service
types; invoke exactly one service method; shape the response (status code,
Location, presenter mapping). DOES NOT: validate business rules, check
permissions, branch on domain state, open transactions, import repositories
or the DB client. If you're writing an `if` about domain state here, it
belongs in the service.

### 3. Service — the business layer and the only place that decides

DOES: **authorization/permissions** (Identity `Check*` helpers /
`checkHasPermission` — every service method enforces its own access, never
trusting the edge); business-rule validation (existence, state transitions,
conflicts); **owns the transaction boundary** (`withTx` /
`prisma.$transaction` — begins, commits, rolls back, hands a tx-scoped
factory/context downward); internal idempotency (upsert key, recovery
points, cache result in-tx); enqueues outbox messages inside the
transaction; orchestrates repositories and mediators; returns domain objects
and `APIError`s.

DOES NOT: parse HTTP/proto types beyond its params structs; write SQL or
call the ORM directly; format HTTP responses; assign roles.

### 4. Mediator — reusable business steps

DOES: multi-repository business logic shared by several services. DOES NOT:
open transactions (consumes the caller's), talk to transport, hold state.
Use one only when two or more services need the same step.

### 5. Repository — the only database-aware layer

DOES: all reads and writes (sqlc queries / Prisma via adapters); translate
domain params ↔ DB types; decode/encode pagination cursors and build page
info (only it knows the keyset columns); map driver errors to `APIError`
(duplicate keys, not-found). DOES NOT: business logic, permission checks,
transactions of its own (it accepts the service's tx via tx-scoped
factory / `context.tx`), network calls.

### 6. Database

DOES: constraints (unique keys back the inbox/idempotency de-dup),
generated columns, referential integrity — invariants the app cannot drift
from. Schema changes ride migrations; sqlc/Prisma regenerate from them.

## Concern placement — the lookup table

| Concern | Layer that owns it |
|---|---|
| Authentication (tokens, API keys, cookies) | Edge middleware; authoritative logic in auth-service (Go) / `authHandler` (TS) |
| Authorization / permissions | **Service** — per method, via Identity helpers; never the edge, never the repo |
| Tenant scoping | Service threads `identity.targetAccountID` into every repo call |
| Request-shape validation | Edge/controller (validate tags in `Execute`; Zod in ctrl) |
| Business-rule validation | Service (and mediators) |
| Client idempotency (`Idempotency-Key`) | Edge middleware (+ platform-service store in Go) |
| Internal idempotency / recovery points | Service, via idempotency mediator, in its own DB |
| Transactions | Service opens/commits; repos and mediators consume |
| Data mutation / SQL / ORM | Repository ONLY |
| Outbox enqueue | Service, inside the business transaction |
| Inbox consume / de-dup | Event consumers wrapping the inbox (unique constraint) |
| Pagination cursors | Repository decodes/encodes; service clamps limits; edge maps page info |
| Error typing | Every layer returns `APIError`; edge maps to HTTP (lossless across gRPC via the `__API_ERROR__` marker) |
| Logging, redaction, request IDs | Edge middleware; `sensitive` tags declared on request structs |
| Role assignment | NOWHERE in code — granted in SQL by an operator |

## Dependency rules

- Dependencies point strictly downward/inward; every layer depends on
  interfaces declared in `domain` (Go) — implementations live in
  infrastructure/service and are chosen ONLY in the composition root
  (`cmd/run.go` per service; `src/index.ts` in the dashboard).
- A layer never imports the layer above it, and never skips a layer
  downward (controller → repo is forbidden).
- Backend services never import each other's `internal`; cross-service =
  gRPC against `shared/proto` (one sanctioned exception: auth-service's
  `pkg/types` Identity). `shared/*` never imports any service.
- Edge duplicating a service rule, or a service trusting the edge to have
  checked permissions, are both bugs: the rule lives ONCE, in the service.

## Enforcement honesty

`services/structure_adherence_test.go` enforces the file skeleton
(cmd/main+run+config, domain files, main→Run) — NOT import direction. The
dashboard has no mechanical enforcement at all. Layer discipline is
convention + review: check the does/does-not lists above in every review.
