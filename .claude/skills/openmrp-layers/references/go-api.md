# Go API layer map (`api/`)

Monorepo, module `github.com/open-mrp/api`. Services under `services/`,
cross-cutting code under `shared/`, protobufs under `shared/proto/*`.
Inter-service transport is gRPC; async is RabbitMQ via outbox/inbox.

## Two structural shapes

### api-gateway (HTTP edge) — `services/api-gateway/`

- `endpoints/<resource>/` (~80 dirs) — declarative endpoints: one
  `endpoint_*.go` per operation declaring `APIEndpoint[Req,Resp]` via
  `Materialize()`; plus `service.go` (thin gRPC translator) and
  `presenter.go` (proto → apiresource).
- `pkg/endpoint` — the generic execution engine; `Execute`
  (`pkg/endpoint/api_endpoint.go:159`) runs the whole HTTP lifecycle.
- `internal/` — `router/` (custom mux), `middleware/` (auth, idempotency,
  rate limit, logging, cors, version, sandbox, subscription, recover),
  `http/` (bind/validate/respond), `grpc/` (outbound helpers),
  `resourceloaders/`, `infrastructure/` (the gateway's OWN sqlc/repos for
  request logs + outbox).
- `grpc-client/` — typed clients per backend service.

### Backend services (core, billing, auth, notification, platform, agent)

Standard skeleton, enforced by `services/structure_adherence_test.go`:

| layer | directory |
|---|---|
| composition root | `cmd/` (`main.go` → `Run` in `run.go`, `config.go`) |
| domain (models + ALL interfaces + mocks) | `internal/domain/` (`models.go`, `services.go`, `repositories.go`, `factories.go`, `mediators.go`, `mock/`) |
| service | `internal/service/` |
| mediator | `internal/mediator/` |
| repository | `internal/infrastructure/repository/` |
| SQL | `internal/infrastructure/queries/*.sql` → sqlc → `internal/infrastructure/sqlc/` |
| gRPC transport | `internal/infrastructure/grpc/` |
| event consumers (inbox) | `internal/event/` |

There is NO controller layer in backend services — the gRPC handler is the
logic-free transport shell (AGENTS.md:226). Chain:
**gRPC handler → service → (mediator) → repository → sqlc → DB**.

The adherence test checks only the skeleton (cmd files exist, domain files
exist, repositories.go ⇒ factories.go, main calls Run) — not imports.

## Concern placement, with evidence

- **Authentication** — `AuthMiddleware`
  (`api-gateway/internal/middleware/auth_middleware.go:34`) extracts
  Authorization header / access cookie, delegates to auth-service
  `ValidateCredential` (`auth-service/internal/service/auth_service.go:103`,
  backed by `internal/token/`, `internal/apikey/`), and ALWAYS attaches an
  Identity (possibly `unauthenticated`) to context.
- **Authorization** — in each backend service method via
  `(*types.Identity).Check*` from `services/auth-service/pkg/types`. E.g.
  `core-service/internal/service/unit_service.go:167-177`
  (`CheckIsInternalActor` + `CheckHasPermission(PermissionDomainUnits,
  ActionCreate)`). Model is `domain:action`; admins bypass. NEVER checked at
  the gateway (docs/patterns/authentication-patterns.md:20-23).
- **Shape validation** — `validate` tags on request structs
  (`endpoints/units/endpoint_create_unit.go:15-36`), enforced centrally in
  `Execute`: unknown-field rejection (:292), explicit-null rejection (:305),
  empty-PATCH rejection (:311), enum validation (:318), `validate.Validate`
  (:323). Binding via `internal/http/handler.go:350`.
- **Business validation** — service layer only, e.g. zero-denominator and
  duplicate-name checks in `unit_service.go:179-227`. Never duplicated at
  the edge (AGENTS.md:102).
- **Idempotency, two tiers**:
  - Client-facing: `IdempotencyMiddleware`
    (`internal/middleware/idempotency_middleware.go:179`) — POST/PATCH only,
    scope+body hashes (`shared/idempotency/hash.go`), keys stored via
    platform-service; handles REPLAY / IN_PROGRESS / HASH_MISMATCH / NEW;
    stores the buffered response before the client sees it.
  - Internal: every mutating gRPC handler wraps
    `contracts.WithIdempotencyTracking`
    (`shared/contracts/idempotency_interceptor.go:41`); the service upserts
    a key via the idempotency mediator
    (`core-service/internal/mediator/idempotency_mediator.go:53`) in its OWN
    DB and switches on `RecoveryPoint` — success cached inside the tx
    (`unit_service.go:195-259`).
- **Transactions** — service-owned `withTx` (`unit_service.go:87-100`) →
  shared generic `db.TransactionManager` (`shared/db/transaction.go`), which
  builds a tx-scoped RepoFactory from `sqlc.Queries.WithTx(tx)`. Mediators
  and repositories never open transactions.
- **Mutation/SQL** — repositories only, via sqlc queries
  (`unit_repository.go:254-275`); errors mapped with `db.MapSQLError*`.
- **Messaging** — service enqueues to outbox INSIDE its tx (e.g. audit:
  `unit_service.go:238` via `repos.NewOutboxRepo()`); `messaging.Enqueuer`
  (started in `cmd/run.go`) drains to RabbitMQ; consumers in
  `internal/event/*_consumer.go` wrap `messaging.NewInboxConsumer` for
  de-dup; `InboxPurger` cleans up. Engine: `shared/messaging/`.
- **Pagination** — HMAC cursors (`shared/pagination`), key set at startup
  (`run.go:46`). Repository decodes/encodes cursors; gateway maps proto
  PageInfo via `grpcutil.MapProtoPageInfo` (`internal/grpc/page_info.go`).
- **Errors** — all layers return `*apierror.APIError` (`shared/errors`);
  HTTP mapping `GetHTTPStatusCode` (`api_error.go:398-684`); lossless gRPC
  crossing via the `__API_ERROR__:` marker
  (`shared/contracts/grpc.go:21-22,157,103-123`).
- **Logging/redaction/request IDs/rate limits** — gateway middleware;
  `sensitive:"true"` fields stripped by `shared/redact` (computed per
  endpoint, `api_endpoint.go:152-157`); request IDs `shared/appctx`; the
  request-log ID doubles as fallback idempotency key
  (`api_endpoint.go:187-189`); in-memory `RateLimitMiddleware` with backoff
  + jitter.

## Dependency direction

- `cmd` → `service` → `mediator` → `repository` → `sqlc`/`db`; transport
  (`infrastructure/grpc`) → `service`. Everything depends inward on
  `internal/domain` interfaces; implementations chosen only in `cmd/run.go`
  (core's builds the pool, RabbitMQ, sqlc, RepoFactory, TransactionManager,
  MediatorFactory, ~60 services, then registers gRPC).
- Services never import each other's `internal`; cross-service = gRPC
  against `shared/proto`. Sanctioned exception: auth-service `pkg/types`
  (Identity) imported by the gateway. `shared/*` never imports a service.
- Mocks generated into `domain/mock/` (GoMock).

## End-to-end: `POST /v1/catalog/units`

1. `router.InitEndpointGroups` (`internal/router/init_groups.go:12-76`)
   registers groups; middleware order (`:34-45`): Tracing → IPBlock →
   Platform → Logging → CORS → RateLimit → **Auth** → Subscription →
   SandboxBilling → Version → **Idempotency** → Recover.
2. Mux dispatch (`router.go:63-101`) → `APIEndpoint.Execute`
   (`api_endpoint.go:159`): bind → redact/log → decode → validate → call
   `ServiceHandler`.
3. Gateway translator `unitSvcImpl.CreateUnit`
   (`endpoints/units/service.go:101`) → `grpcutil.CallRPC` →
   core-service.
4. `gRPCHandler.CreateUnit` (`grpc_handler.go:813`) — idempotency tracking,
   proto→domain params, calls service, converts errors.
5. `unitSvcImpl.CreateUnit` (`unit_service.go:163`) — permissions →
   business validation → ID gen → idempotency upsert → `withTx`: duplicate
   checks → `txRepo.Create` → audit outbox publish → cache success in-tx.
6. `unitRepoImpl.Create` (`unit_repository.go:254`) → sqlc `InsertUnit` →
   error mapping → re-read.
7. Back out: domain → proto → gateway → resourceloaders → apiresource →
   `RespondWithJSON` 201 + `Location` (`api_endpoint.go:362-396`).

## Stated rules worth holding reviews to

- Gateways thin; business rules encoded once, in service/domain
  (AGENTS.md:102).
- Never skip a 5xx — root-cause in the correct layer; no bandaid skips,
  retries/sleeps, or broad nolint (AGENTS.md:87-104).
- POST/PATCH honor client idempotency keys; PUT/DELETE/GET idempotent by
  default; all inter-service calls idempotent via internal keys
  (AGENTS.md:318-322).
- Resources: every resource has `Object`; sub-resources not inline FK
  fields; expandables `null` unless `?include=`, never fabricated
  (AGENTS.md:170-192, 323-338).
- Nullability: value + `validate:"required"` vs `field.Optional[T]` vs
  `*field.Clearable[T]`; always `,omitzero`, never `omitempty`
  (AGENTS.md:172-182).
- Pattern catalog: `docs/patterns/` (architecture, authentication,
  domain-layer, audit-event, entity-id, canonical-log, api-versioning,
  main-delegates-to-run), indexed at AGENTS.md:343-362.
