# Agents.md

This file provides guidance when working with code in this repository.

## Read the Patterns Docs First (non-negotiable)

`docs/patterns/` is the normative spec for this codebase, not background reading. Code that contradicts a pattern doc is wrong even if it compiles, passes tests, and looks like the file next to it — neighboring code may predate the doc.

**Before writing a line of code, open the docs for the layers you are about to touch.** Do not infer conventions by copying a nearby file; a plausible-looking imitation of the wrong pattern is the single most common failure mode in this repo, and it gets caught in review rather than by the compiler. If a doc and existing code disagree, the doc wins — follow the doc and flag the drift.

| If you are... | Read first (in `docs/patterns/`) |
|---|---|
| Changing **any** existing public request/response shape or behavior | `api-versioning-patterns.md` — and see the breaking-change rule below |
| Adding or changing an endpoint's request/response shape | `api-resource-conventions.md`, `nullable-field-patterns.md` |
| Writing a service, mediator, repository, or transaction | `architecture-patterns.md` |
| Touching identity, permissions, or actor checks | `authentication-patterns.md`, `authorization-check-patterns.md` |
| Adding or changing a list endpoint, filter, or sort | `performant-list-endpoint-patterns.md` |
| Adding a domain model, mock, or entry point | `domain-layer-patterns.md` |
| Emitting or changing audit events | `audit-event-patterns.md` |
| Adding a new entity or ID prefix | `entity-id-patterns.md` |
| Adding or changing an enum | `constants-enum-patterns.md` |
| Writing or fixing e2e tests | `e2e-test-patterns.md` |
| Writing or editing any comment | `comment-conventions.md` |
| Changing config structs | `config-patterns.md` |
| Touching logging or tracing | `canonical-log-patterns.md` |
| Touching production step graphs | `production-step-graph-patterns.md` |

Read the doc **before** you write, not after review. Reading the section you need is enough; you do not have to read a doc end to end.

**Self-check before reporting work as done:** name the pattern docs that govern the files you changed, re-read the relevant sections, and confirm each rule actually holds in your diff. "I didn't know that doc existed" is not an acceptable outcome — the table above is the index, and `important-patterns.md` is the short list of rules that apply everywhere.

## Breaking Changes Require a New API Version (non-negotiable)

**Any breaking change to the public API requires a new API version plus a version transformer that preserves the old shape.** A client pinned to a supported `OpenMRP-Version` must keep receiving byte-compatible request and response shapes for as long as that version is supported. Shipping a shape change without a transformer silently breaks every pinned consumer — that is never an acceptable trade for a smaller diff, and it is not something to "follow up on later."

Breaking — needs a new version **and** a transformer:

- Removing or renaming a request or response field.
- Changing a field's type, format, nullability, or enum value set.
- Moving previously-unconditional data behind an `?include=` key.
- Changing a default, a status code, an error `type`/`param`, or pagination semantics.
- Tightening validation so a previously-accepted body is now rejected.
- Changing the meaning of an existing field or parameter, even at an identical shape.

Not breaking — ships straight on `Latest`: new endpoints, new **response** fields, new **optional** request fields or query parameters, new include keys, new values on a field documented as an open enum, and bug fixes where the documented contract was always the fixed behavior.

**When in doubt, treat it as breaking.** An unnecessary transformer costs little; a silently broken consumer costs a lot.

The backend only ever speaks `Latest` — no version conditionals anywhere below the gateway edge. Older versions exist purely as edge transformations in `services/api-gateway/internal/versiontransforms/`. A shape-changing change is not complete until all of these ship with it:

1. The new version declared in `shared/version/version.go` (constant added, prepended to `Supported`, `Latest` repointed) with `version_test.go` updated.
2. The transformer(s), `FromVersion` = new and `ToVersion` = previous, with unit tests covering single, list, nested, and data-missing payloads.
3. `ObjectType` declared on **every** endpoint that returns or accepts the changed resource — an endpoint without it is invisible to the transformer chain and leaks the new shape.
4. `ForcedIncludes` wherever the old shape needs data the new shape gates behind an include.
5. Version-compat e2e tests (`tests/e2e/api/version_compat_<resource>_test.go`) pinned to the previous version.
6. `make openapi` re-run and the regenerated spec committed.

Transformers reshape real data and **never** fabricate values, and a transformer is immutable once its `ToVersion` has shipped. `docs/patterns/api-versioning-patterns.md` is the full checklist — follow it, don't approximate it.

## Build & Development Commands

```bash
make dev                      # Start local dev environment (Tilt + K8s)
make test                     # Run all tests
go test ./services/auth-service/...  # Run tests for specific service
go test -run TestFunctionName ./path/to/package  # Run single test
make sqlc [service]       # Generate DB code from SQL (e.g., make sqlc auth)
make proto                # Generate protobuf bindings
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
- **agent-service**: Agent management and execution
- **billing-service**: Billing and subscription management

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

(this is not exhaustive)

- **contracts/**: gRPC interceptors, identity propagation, API error encoding
- **errors/**: Centralized API error types with HTTP/gRPC mapping
- **id/**: Custom ID generation with entity prefixes (usr*, acct*, org\_, etc.)
- **messaging/**: RabbitMQ integration with outbox/inbox patterns
- **db/**: Database pool management and migrations
- **constants/**: Domain enums (AccountMode, RoleType, PlanCode, etc.)

### Key Patterns

1. **Identity propagation**: Identity is serialized to gRPC metadata via `contracts.SetIdentityInMetadata()` and extracted with interceptors
2. **API error handling**: Errors are encoded in gRPC messages with `apiErrorMarker` prefix, decoded by `ConvertGRPCError()`
3. **Repository factory pattern**: Each service has a factory that creates repositories with transaction support
4. **sqlc for type-safe SQL**: Queries defined in `queries/*.sql`, generated code in `sqlc/`

## Database

PlanetScale (MySQL) with safe migrations. **Schema source of truth is the goose migrations in `shared/db/migrations`.** See `docs/patterns/database-migrations.md` for the full workflow — it explains the constraints the rules below come from.

`00001_initial.sql` is a frozen baseline: a dump of the schema at the goose cutover. Never edit or regenerate it. It begins by dropping every table, so applying it to a populated database destroys that database.

To add/modify schema:

1. `make migrate-create name=add_something` — writes a new numbered migration in `shared/db/migrations`.
2. Fill in the Up and Down halves. Keep the `-- +goose NO TRANSACTION` line the template starts with: Vitess rejects DDL inside an explicit transaction.
3. `make migrate-up` to apply it to the local Docker MySQL, then `make sqlc [service]` for affected services. sqlc reads the whole migrations directory, so no dump needs regenerating.
4. Update the Prisma schema in `dashboard/packages/db` to match, in the same change — it is the dashboard's model definition and cannot be derived from the database (no foreign keys, `relationMode = "prisma"`). Nothing enforces this automatically; it is caught in review.
5. Shipping is automated — the release PR opens a PlanetScale deploy request for review, and merging it deploys the schema before any service image rolls. Never run DDL against prod yourself.

The agent-service Postgres schema works the same way but lives in `services/agent-service/db/migrations` (`make migrate-agent-create`). Postgres has no deploy requests — PlanetScale applies Postgres DDL directly — so those migrations are applied on merge.

Data backfills are not schema: they go in `shared/db/data-migrations` (`make migrate-create-data`) or `services/agent-service/db/data-migrations` (`make migrate-agent-create-data`). A PlanetScale deploy request diffs schema only, so DML written into a schema migration silently never reaches production. Backfills run after both schemas are live and before any image rolls.

## Code Style

- **Never write agent-generated comments.** Do not add comments to code you write or edit unless the user explicitly asks for them. This is absolute: no explanatory comments, no narration of what the code does, no rationale blocks, no "helpful" annotations — even when they seem useful. When a comment is genuinely warranted the user will ask for it; absent that instruction, leave the code uncommented and do not remove or rewrite existing human-authored comments.
- Do not create README files or examples unless explicitly requested
- **Comments: never hard-wrap a prose paragraph across multiple `//` lines** — one paragraph is one physical line (let the editor soft-wrap). Distinct paragraphs get a blank `//` separator; numbered/TODO lists may use one line per item. Internal doc comments explain business intent, side effects, ordering/transactionality, idempotency, and failure modes — not the mechanics. See `docs/patterns/comment-conventions.md`.
- **Commits must follow the Conventional Commits conventions in `README.md` (the "Committing" section).** release-please parses these prefixes (feat:, fix:, feat!:, etc.) to calculate the next version — a non-conforming prefix is ignored, so the change won't make it into a Release PR and no release gets cut.
- Review `docs/patterns/` for the conventions governing whatever you are touching — see the routing table at the top of this file. These are requirements, not suggestions.

## End-to-end (e2e) tests

When working on e2e tests, do not let production bugs slip through by relaxing assertions, skipping failures, or only noting broken behavior in comments on production code. End-to-end tests exist to exercise the real stack and expose defects in production code; when a test reveals a problem, fix the underlying issue until the e2e suite passes (or correct the test if its expectations were wrong).

**Never skip 5xx errors.** If an endpoint returns a 500 (or any 5xx) in an e2e run, that is a production bug — fix the root cause in the service, repository, or gRPC layer. Do not add `t.Skip("backend 500 ...")`, `if status >= 500 { ... }` guards, `skipIfBackend500` helpers, retries that hide the error, or TODO comments that leave the failure latent. The test must either pass against a healthy backend or fail loudly until the backend is fixed.

**Avoid bandaid fixes.** Prefer root-cause fixes in the correct layer (repository SQL and sqlc queries, domain/service logic, gRPC contracts, gateway mapping) even when that means touching many files or packages. Do not paper over microservice or data-layer bugs with shortcuts at the edge.

Examples of what to avoid:

- **Skipping 5xx / internal errors** — always root-cause them (see above).
- **In-memory filtering or sorting** when the database should do the work (add or fix `queries/*.sql`, repository methods, and indexes as needed instead of loading large pages and filtering in Go).
- **Fixing only the API gateway** when the bug belongs in core-service (or another service)—drill down and fix ownership, permissions, validation, and persistence where they live.
- **Retries, sleeps, or inflated timeouts** to mask flakes, races, or ordering issues—fix idempotency, transactions, or the actual race.
- **Empty results or silent defaults** that hide errors or missing data—return proper errors and correct contract fields.
- **Duplicating business rules** in the gateway to “make the response look right”—keep gateways thin; encode rules once in the service/domain layer.
- **Broad `//nolint` or disabled checks** to silence static analysis—address the finding or refactor so the code is clean.

## Typical API Gateway Implementations

Here is a typical endpoint:

```go
// GetAPIKeyRequest is the request to retrieve a single API key by ID.
type GetAPIKeyRequest struct {
	// The ID of the API key to retrieve.
	APIKeyID string `path:"id"`
}

const getAPIKeyEndpointDescription string = `This endpoint returns a single API key's metadata by its ID.`

type GetAPIKeyEndpoint struct{}

func (e *GetAPIKeyEndpoint) Materialize() *apiendpoint.APIEndpoint[*GetAPIKeyRequest, *apiresource.APIKey] {
	return &apiendpoint.APIEndpoint[*GetAPIKeyRequest, *apiresource.APIKey]{
		Title:             "Get API Key",
		Description:       getAPIKeyEndpointDescription,
		Method:            http.MethodGet,
		Route:             "/v1/auth/api-keys/{id}",
		Request:           &GetAPIKeyRequest{},
		Response:          &apiresource.APIKey{},
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *GetAPIKeyRequest) (*apiresource.APIKey, *apierror.APIError) {
			return svc.(APIKeySvc).GetAPIKey
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeAPIKey,
			Fields:     []string{"role", "role.permissions"},
		}),
	}
}
```

Here is the resource:

```go
// APIKey represents an API key for authenticating API requests.
type APIKey struct {
	// The unique identifier for the API key.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=api_key"`
	// The human-readable name for the API key.
	Name string `json:"name" validate:"required"`
	// The redacted value of the API key for display purposes.
	RedactedValue string `json:"redacted_value" validate:"required"`
	// The role associated with this API key. Expandable.
	Role *Role `json:"role" expandable:"true"`
	// The timestamp when the API key was created.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// The timestamp when the API key was last updated.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
	// The timestamp when the API key was last used.
	LastUsedAt *time.Time `json:"last_used_at"`
	// The timestamp when the API key expires.
	ExpiresAt *time.Time `json:"expires_at"`
	// The timestamp when the API key was revoked.
	RevokedAt *time.Time `json:"revoked_at"`
}
```

Notice that we define some include parameters so that the user can fetch extra data (the role) if they desire. Make sure we always structure apiresources to use subresources when they can and never inline subresource values in the resource directly (e.g. we would not create fields like RoleID or RoleName in the api key resource but instead make it a sub resource).

### Request field tags (create / update bodies)

Request structs model field *presence* through the field's type, not a bare pointer. A bare Go pointer can't distinguish an absent key from an explicit `null`, so we use two wrapper types. **The type picks itself based on which states the endpoint must distinguish** — see `docs/patterns/nullable-field-patterns.md` for the full rationale, decision table, runtime pipeline, and proto/OpenAPI mapping.

| Context | Required / always present | Optional, not clearable | Clearable (accepts `null`) |
|---------|---------------------------|-------------------------|----------------------------|
| Create / action | `T` + `validate:"required"`, no omit tag | `field.Optional[T]` + `,omitzero` | — |
| Update / PATCH | `T` (path params) | `field.Optional[T]` + `,omitzero` | `*field.Clearable[T]` + `,omitzero` |
| Response | `T` + `validate:"required"` | `*T` (nullable, **no** omit tag) | `*T` (nullable, **no** omit tag) |

Rules: every **request** field uses `,omitzero` (never `,omitempty`) on its json tag; `validate:"omitempty,..."` is a separate validator keyword and stays. `field.Optional[T]` rejects an explicit `null` and a blank string (`400`); `*field.Clearable[T]` is the only request shape that accepts `null` (to clear). Never use a bare `*T` for an optional *request* field, and never use `omitempty` on a *response* field. After changing any request struct, run `make openapi` and commit the regenerated spec.

### Expandable subresources: real data on include, `null` otherwise (NON-NEGOTIABLE)

Expandable subresources (`expandable:"true"`) follow exactly one contract — the same one our other services use:

- **`null` unless explicitly included.** If the client did not pass `?include=<key>`, the field is `null`. The presenter leaves the field `nil`; it does **not** assign anything.
- **Real data only when included.** When `?include=<key>` is requested, the field is populated by a registered loader that fetches the **real** record from the source service (the `BatchGet<X>ByIDs` loader pattern in `internal/resourceloaders/`). What we serialize is exactly what the source returned — nothing invented.
- **Never fabricate.** Do not build "stub" / placeholder subresources, do not default enum or status fields to plausible-looking values (`status: "issued"`, `priority: "normal"`, unit `ratio_numerator: "1"`, etc.), and do not hand-assemble a partial subresource from whatever foreign keys happen to be on the parent proto. If the real value is not available, the answer is `null`, not a guess.

Mechanically: the presenter builds the resource with expandable fields left `nil` and stashes only the foreign-key **id** into the request-scoped load meta (`resourcekit.GetLoadMeta(ctx).Set(parentObjectType, parent.ID, "<fk>_id", id)`). The field is registered as a `SubField` in `internal/resourceregistry/registered_*.go` with a `Target`, an `ExtractIDs` that reads the stashed id, and a `Populate` that writes `loaded[id]` onto the parent. The include resolver only runs `Populate` for includes the client actually requested — so an un-requested field is never touched and serializes as `null`. There is no post-hoc "collapse" step that hides fabricated data; whatever the presenter puts on an expandable field ships to the client, which is exactly why presenters must never put fabricated data there.

## Typical gRPC Handler Shape

```go
func (h *gRPCHandler) CreateSandbox(ctx context.Context, req *pb.CreateSandboxRequest) (*pb.CreateSandboxResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	var mode constants.SandboxMode
	switch req.Mode {
	case pb.SandboxMode_SANDBOX_MODE_SEEDED:
		mode = constants.SandboxModeSeeded
	default:
		mode = constants.SandboxModeBlank
	}

	sandbox, apiErr := h.sandboxSvc.CreateSandbox(ctx, req.Name, mode)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.CreateSandboxResponse{
		Sandbox: sandboxToProto(sandbox),
	}, nil
}
```

Notice that `contracts.WithIdempotencyTracking(ctx)` is present in this case - it is used for POST and PATCH endpoints since these are made idempotent with idempotency keys.

The gRPC handlers are just responsible for transport concerns. They do not handle business logic.

## Typical Microservice Service Shape

```go
ctx, span := unitSvcTracer.Start(ctx, "service.unit.create")
defer span.End()

identity, ok := appctx.GetIdentityFromContext(ctx)
if !ok || identity == nil {
    return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
}

if apiErr := types.CheckIsInternalActor(identity); apiErr != nil {
    return nil, tracing.Trace(span, apiErr)
}
if apiErr := types.CheckHasPermission(identity, types.PermissionDomainUnits, types.ActionCreate); apiErr != nil {
    return nil, tracing.Trace(span, apiErr)
}
if identity.TargetAccountID == nil {
    return nil, tracing.Trace(span, apierror.NewAuthenticationError("The OpenMRP-Account header is required."))
}

unitID, apiErr := id.GenID(id.UnitIDPrefix, nil)
if apiErr != nil {
    return nil, tracing.Trace(span, apiErr)
}

params.AccountID = *identity.TargetAccountID

meds := s.mediators()

idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, identity)
if apiErr != nil {
    return nil, apiErr
}

switch domain.RecoveryPoint(idempotencyKey.RecoveryPoint) {
case domain.RecoveryPointFinished:
    cached, err := idempotency.UnmarshalCachedResponse[domain.Unit](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
    if err != nil {
        return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
    }
    return cached.Data, cached.Error

case domain.RecoveryPointStarted:
    var result *domain.Unit
    apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *unitSvcImpl) *apierror.APIError {
        txRepo := txSvc.repos.NewUnitRepo()

        exists, apiErr := txRepo.ExistsByName(txCtx, params.AccountID, params.Name, nil)
        if apiErr != nil {
            return apiErr
        }
        if exists {
            return apierror.NewConflictErrorWithParam("A unit with this name already exists.", "name")
        }

        exists, apiErr = txRepo.ExistsByAbbreviation(txCtx, params.AccountID, params.Abbreviation, nil)
        if apiErr != nil {
            return apiErr
        }
        if exists {
            return apierror.NewConflictErrorWithParam("A unit with this abbreviation already exists.", "abbreviation")
        }

        created, apiErr := txRepo.Create(txCtx, unitID, params)
        if apiErr != nil {
            return apiErr
        }
        result = created

        return txSvc.mediators().Idempotency.CacheSuccessResponse(txCtx, idempotencyKey.TypeID, result)
    })

    if apiErr != nil {
        return nil, meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, apiErr)
    }

    return result, nil

default:
    return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Unexpected recovery point: "+idempotencyKey.RecoveryPoint))
}
```

Notice that this first checks the identity to make sure we have the requisite permissions and that a valid target account ID is supplied. For any endpoint that mutates data, we want to do this idempotently using idempotency keys. These will have atomic phases, which are set by recovery points. The idea is that the request could be retried at any stage and continued where it left off. If we have to do any foreign mutation (mutate state on an API we do not control or across a microservice boundary), this should happen in the beginning of its own atomic phase. Any other atomic changes we can commit in a transaction should then happen immediately after that mutation. Since we have idempotency keys across our own stack and some of our third party providers (e.g. Stripe) support them, we can likely retry the stage if it were to fail after the foreign state mutation happened but before we were able to commit our atomic changes that followed. Notice that we cache responses and update the recovery point inside a transaction so that these are always in sync.

When we send messages we use the message inbox/outbox pattern to ensure delivery always happens at least once.

## Important Patterns

1. All PATCH and POST endpoints must respect idempotency keys sent by the user.
2. All PUT, DELETE, and GET endpoints must be designed to be idempotent by default without idempotency keys.
3. All microservice calls should be idempotent via internal idempotency keys even if the user does not supply them.
4. All services inside microservices should change the database atomically wherever possible.
5. All business logic should be inside a service or mediator. See `docs/patterns/architecture-patterns.md` for more information.
6. All apiresources should have an "Object" field.
7. Nested resources that are returned by the API should prefer json structures like so:

```json
{
  "user": { "id": "us_...", "object": "user" }
}
```

rather than:

```json
{
  "user_id": "us_..."
}
```

8. New endpoints should be added to the openapi spec generator.
9. Sensitive HTTP request or response fields that must not appear verbatim in persisted request logs should be tagged `sensitive:"true"` on the corresponding struct field (`shared/redact` redacts logged JSON at the gateway). Omit entire request-log rows via `Extras.SkipRequestLogging` when logging the request is unacceptable.
10. Any breaking change to an existing public request or response shape requires a new API version plus a transformer in `services/api-gateway/internal/versiontransforms/`. See the non-negotiable section at the top of this file and `docs/patterns/api-versioning-patterns.md`.
11. **`_parent_child_production_steps` column order is fixed**: **`A` = downstream step, `B` = upstream (parent)** — matches Prisma/dashboard. Never revert queries or seeds to "`A` = parent." See `docs/patterns/production-step-graph-patterns.md`.

## Reference Doc Index

The complete index of `docs/patterns/`. The routing table at the top of this file maps tasks to the docs you must read before writing code; this is the full set, including docs no routing row calls out.

- `docs/patterns/important-patterns.md` — the short list of rules that apply to every change, regardless of layer
- `docs/patterns/api-resource-conventions.md` — API resource field conventions (object field, no omitempty, sub-objects, expandable relations, include system, list responses, sample data)
- `docs/patterns/nullable-field-patterns.md` — **Request** field tags and nullability: `field.Optional[T]` vs. `*field.Clearable[T]` vs. value types, `omitzero` rule, the absent/null/value three-state model, the gateway null/blank-rejection pipeline, and proto/OpenAPI mapping
- `docs/patterns/api-versioning-patterns.md` — API version scheme, the no-breaking-changes contract, gateway-edge transformers (`versiontransforms/`), forced includes, and the version deprecation/removal process
- `docs/patterns/architecture-patterns.md` — Layered architecture (services, mediators, repositories, transaction management, idempotency, error handling, tracing)
- `docs/patterns/authentication-patterns.md` — Identity model, authorization checks, permission model, actor types
- `docs/patterns/authorization-check-patterns.md` — Cross-account authorization: actor vs. target account, `TargetRelationType`, and which permission domain applies when a merchant targets a customer or supplier
- `docs/patterns/domain-layer-patterns.md` — Domain directory structure, standard files, mock generation, entry point pattern
- `docs/patterns/audit-event-patterns.md` — Publishing audit events from services, outbox usage, `audit` struct tags on domain models for field-level diffs
- `docs/patterns/entity-id-patterns.md` — ID format, vocabulary codes, composable prefixes, adding new entities
- `docs/patterns/constants-enum-patterns.md` — Enum type conventions (IsValid, EnumValues), adherence tests
- `docs/patterns/config-patterns.md` — Config struct conventions (WithDefaults, validate, constructor pattern)
- `docs/patterns/canonical-log-patterns.md` — Canonical log lines, interceptor chain, tracing fields
- `docs/patterns/e2e-test-patterns.md` — E2E test conventions (CRUD lifecycle, field assertions, omitted fields, expandable fields, helpers, checklist)
- `docs/patterns/performant-list-endpoint-patterns.md` — Keyset list queries, selective-filter composite indexes, and how to add a user-facing list filter without a table scan
- `docs/patterns/production-step-graph-patterns.md` — `_parent_child_production_steps`: **`A` = downstream, `B` = upstream** (Prisma-aligned); flow SQL and seeds must stay consistent
- `docs/patterns/main-delegates-to-run-pattern.md` — main() → Run() delegation pattern
- `docs/patterns/comment-conventions.md` — internal-code comment style: one paragraph per physical line (no hard-wrapping), what doc comments must convey (intent/side effects/idempotency/failure modes), config-field optionality docs, actionable TODOs
- `docs/api-migration-instructions.md` — Dashboard API → Go API migration context
- `docs/migration-checklist.md` — Per-endpoint checklist for the dashboard → Go migration
- `docs/stlc-sdk-codegen.md` — Stainless SDK codegen: regeneration flow, config, and release
