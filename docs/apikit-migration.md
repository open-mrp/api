# Migrating to apikit at forge.1

`github.com/open-mrp/apikit` holds the generic API framework that used to live here, rebuilt to the forge.1 contract. This repo keeps its own copies until forge.1 ships, then switches to the kit in one migration. This document is that migration: what moves, what OpenMRP has to provide in its place, what clients will see change, and the order to do it in.

The decision and the list of what the kit holds are in `docs/forge1-api-review.md` (Shared API kit). The kit's own package docs and tests are the reference for each API named here.

## Scale

How many files in this repo import each package that moves, when this plan was written:

| Package here | Kit package | Files |
|---|---|---|
| `shared/errors` | `apierror` | ~1,670 |
| `services/api-gateway/pkg/endpoint` | `endpoint` | ~810 |
| `shared/tracing` | `tracing` | ~640 |
| `services/api-gateway/pkg/example` | `example` | ~410 |
| `shared/field` | `field` | ~320 |
| `shared/appctx` | `appctx` | ~300 |
| `shared/db` | `db` | ~290 |
| `shared/pagination` | `pagination` | ~220 |
| `services/api-gateway/pkg/resourcekit` | `include` | ~210 |
| `shared/id` | `id` | ~170 |
| `shared/timeutil` | `timeutil` | ~115 |
| `shared/idempotency` | `idempotency` | ~95 |
| everything else | | < 50 each |

Almost all of this is a mechanical import and name swap. The hard parts are in a few places, called out below.

## Ground rules

- **One type, one owner.** `APIError`, `field.Optional`, the identity and the object type cross service boundaries and package boundaries, so each switches in a single change. Never run with both this repo's type and the kit's in flight; that is why the error switch is one PR across every service.
- **Script the rewrites; never hand-edit them.** Several `apierror` constructors changed from `(message, param)` to `(param, message)`. Both are strings, so a missed swap compiles and silently puts the message in `param`. Use an AST rewrite that swaps the two arguments of exactly those constructors, and review its output rather than the call sites. Moving `internal/http` into the kit alone turned up 18 such calls.
- **Pin the kit.** Develop against a local checkout with `go.work`, then require a tagged apikit release. Keep apikit's dependency versions in step with this repo's.
- **What stays here moves to OpenMRP-named packages.** Where a package here keeps an OpenMRP-only remainder, give that remainder its own name (for example `shared/omrpctx` for the platform and external-host context keys), so no file imports two packages called `appctx` or `tracing`.

## Step 0: Close the kit's gaps first

Before starting, add to apikit (each a normal kit PR):

1. **gRPC transport for `apierror`.** Services return `APIError`s to the gateway over gRPC, using `ToJSON` and `APIErrorFromJSON` here. The kit dropped them when gRPC was deferred. Add them back, carrying `Param`, `Errors`, `Details` (as raw JSON, re-decoded by the code's registered details type), `InternalMessage`, the wrapped error's text, and `Stack`.
2. **Anything the steps below show missing.** Run `go build ./...` here against the kit after Step 1 and add what the compiler asks for that is generic. OpenMRP-specific gaps get an OpenMRP package instead.

Packages deferred from the kit stay here but are rewired to the kit's types in the step that switches the type they use: `shared/contracts`, `shared/rpc`, `shared/messaging`, `shared/audit`, the gRPC canonical-log interceptor in `shared/logging`, the gRPC and RabbitMQ helpers in `shared/tracing`, and `idempotency.UnmarshalCachedResponse`.

## Step 1: Leaf packages

Swap imports only; the APIs did not change. `field` (except below), `pagination`, `crypto`, `ptrutil`, `safeconv`, `fuzzy`, `querytag`, `lease`.

- `shared/field/proto.go` and `convert.go` (the protobuf patch converters and `QuantityInput`) stay here, in a package such as `shared/fieldpb`.
- `pagination`'s documentation cursor key changed name, so every signed example cursor in the generated spec changes. That is expected noise in the spec diff.

## Step 2: Errors, and everything that returns them (one PR)

Switch `shared/errors` to `apierror` everywhere, together with the packages whose signatures carry `*APIError`: `validate`, `retry`, `cache`, `id`, `db`, `tracing`, `cloud/s3`, `cloud/sqs`, `blobstore`, `idempotency`, `metadata`, and the deferred packages listed in Step 0.

**Register OpenMRP's error codes** at startup, in an OpenMRP package (for example `shared/apperr`) that also holds their constructors:

| Code | Type | Status | Notes |
|---|---|---|---|
| `limit_exceeded` | `invalid_request_error` | 403 | `DetailsField: "quota"`, `DetailsExample: QuotaInfo{...}`; `WithQuota` becomes `WithDetails(QuotaInfo{...})` |
| `registration_closed` | `invalid_request_error` | 403 | |
| `payment_required` | `invalid_request_error` | 402 | |
| `agent_spending_cap_reached` | `invalid_request_error` | 402 | |
| `api_key_expired` | `invalid_request_error` | 401 | |
| `api_key_revoked` | `invalid_request_error` | 401 | |
| `verification_required` | `invalid_request_error` | 403 | New in forge.1 (G1) |

Set the doc URL builder: `apierror.SetDocURL(func(c apierror.Code) string { return "https://docs.openmrp.ai/api/errors#" + string(c) })`, or the per-code pages once G6 publishes them.

**Rename map.** Arguments in the second column are in the new order.

| Here | apikit |
|---|---|
| `ErrorCodeX`, `ErrorTypeX` | `CodeX`, `TypeX` |
| `ErrorCodeRateLimitExceeded` | `CodeRateLimited` |
| `GetHTTPStatusCode(code)` / `Is5XXErrorCode(code)` | `code.Status()` / `code.Status() >= 500` |
| `IsNotFound(err)` | `HasCode(err, CodeResourceNotFound)` |
| `NewAPIError(code, type, public, internal, opts...)` | `New(code, public, WithInternalMessage(internal), opts...)`; the type now comes from the code |
| `NewValidationError(msg)` | `NewValidationError(msg, fields...)` |
| `NewValidationErrorWithParam(msg, param)` | `NewFieldError(param, CodeInvalidFormat, msg)` |
| `NewMissingFieldError(msg, param)` / `NewInvalidFormatError(msg, param)` | `NewMissingFieldError(param, msg)` / `NewInvalidFormatError(param, msg)` |
| `NewParameter{Missing,Invalid,Unknown}Error(msg, param)` | same name, `(param, msg)` |
| `NewConflictErrorWithParam(msg, param)` | `NewConflictErrorWithParam(param, msg)` |
| `NewResourceNotFoundError` / `NewResourceConflictError` / `NewAlreadyDeletedError` | `NewNotFoundError` / `NewConflictError` / `NewGoneError` |
| `NewResourceExistsError(msg)` | `NewExistsError(param, msg)` (pass the field when known, else `""`) |
| `NewIdempotencyHashMismatchError(key)` | `NewIdempotencyKeyReusedError(key)` |
| `NewRateLimitExceededError` | `NewRateLimitedError` |
| `NewClientClosedRequestError(msg)` | `NewClientClosedRequestError()` |
| `NewAPIVersion{Required,Invalid}Error(...)` | gain the header name: `version.Header()` |
| `NewExpiredAPIKeyError`, `NewRevokedAPIKeyError`, `NewPaymentRequiredError`, `NewLimitExceededError`, `NewAgentSpendingCapReachedError`, `NewRegistrationClosedError` | OpenMRP constructors over the registered codes above |
| `ResponseError` / `APIErrorResponse` | `ErrorObject` / `Response` |
| `ToResponseError()` / `ToResponseMap()` | `Object()` / `Response()` |
| `WithDocURL` / the `DocURL` field | gone: built from the code |
| `RowErrors.AddValidation(i, param, msg)` | `AddField(i, param, CodeInvalidFormat, msg)` |
| `RowErrors.Summary(entityPlural)` | `Summary(listParam)`, listing each row's fields as `rows[3].sku` |

**Other renames in this step:**
- `validate`: callers that read `err.Param` after `Validate` now read `err.Errors` (every failing field) or `err.Param` (the first).
- `redact`: register OpenMRP's log classes with `redact.RegisterSensitiveTag("cost")` and `redact.RegisterKeyedMapTag("cost_keys", isCostName)`; `IsCostName` and its name list move to the OpenMRP side of `sensitive` (Step 4).
- `tracing`: `Config.Environment` is a string (`string(platformMode)`); `constants.Protocol` becomes `tracing.Protocol`; `WrapGatewayHandler` becomes `WrapHandler`.
- `id`: the prefix vocabulary and values stay in `shared/id` here, built with `id.ComposePrefix`.
- `metadata`: `PatchToProto`/`PatchFromProto` stay in a protobuf package here; `CheckLimit(m, param)` keeps its arguments, and its error is now a 422.

## Step 3: Identity, context, versions and object types

- **Identity.** `appctx.GetIdentityFromContext(ctx)` becomes `appctx.Identity[*types.Identity](ctx)`; `WithIdentity` is unchanged at call sites. Give `types.Identity`:
  - `LogAttrs() []slog.Attr` (`logging.IdentityAttrer`), returning what `extractIdentityAttrs` logs today;
  - `ForIncludeReads() any` (`include.IncludeReader`), returning the flagged copy. Callers that used its typed result assert it back.
- **The context keys that stay here.** `platform.go` and `external_host.go` move to an OpenMRP package (for example `shared/omrpctx`).
- **The request log.** The kit's `appctx.RequestLog` has no `AccountID`, `TargetAccountID` or `IdentityType`. The auth middleware sets `rl.Identity`, and the request-log saver derives those columns from it when it publishes.
- **Versions.** At startup: `version.SetHeader("OpenMRP-Version")`, `version.RegisterVersions(...)` with every version now in `shared/version`, and `version.RegisterTransformer(t)` for each transformer in `versiontransforms` (today's `version.Register`). `version.Latest` and `version.Supported` become calls. Give retiring versions `DeprecatedAt` and `SunsetAt` to send the `Deprecation` and `Sunset` headers (X10).
- **Object types.** Make `constants.ObjectType` an alias, `type ObjectType = object.Type`, so its constants keep working and the kit accepts them without a rewrite. Retire the alias later at leisure.

## Step 4: The HTTP layer

Switch together: `internal/http` and `internal/header` → `transport`, `pkg/endpoint` → `endpoint`, `pkg/resourcekit` → `include`, `internal/router` → `router`, `pkg/costguard` → `sensitive`, `pkg/example` → `example`, the generic middleware, and the list and delete shapes → `object`.

**Endpoint access policies.** The endpoint's `RequiredPermissions`, `RequiresAllPermissions`, `CounterpartyPermissions`, `SelfPathParam` and `RequiredRoleType` fold into one OpenMRP policy value, for example:

```go
type Policy struct {
	Permissions  types.AnyOfPermissions
	RequireAll   bool
	Counterparty apiendpoint.CounterpartyPermissions
	SelfPathParam string
	RoleType     constants.RoleType
}
```

Each endpoint sets `Auth: authz.Policy{...}` (scripted: move the five fields into the literal). Register `endpoint.SetAuthorizer` with today's `authorize` and `actsOnSelf` logic, reading `appctx.Identity[*types.Identity]` and the policy. The `DescribeAuth` hook in Step 5 and the agent-tool writer read the same policy.

**Restricted fields.** Register OpenMRP's two classes with `sensitive.Register`:
- `cost`, with `KeyedTag: "cost_keys"`, `MatchKey: isCostName` and `Visible: identity.CanReadCosts()`;
- `internal`, with `Visible: identity.IsInternalActor()`.

Types implementing `costguard.Redactor` rename `RedactCosts()` to `RedactSensitive(hidden sensitive.Set)` and check `hidden.Hides("cost")`.

**Lists.** `apiresource.PaginationRequest` becomes an embedded `object.ListRequest` (limit 25 by default, 100 at most) plus a per-endpoint `q` where the endpoint keeps search (the cross-cutting `q` item in the review). `apiresource.List` and `PageInfo` become `object.List` and `object.PageInfo` (a generic alias works for the transition). `NewList` keeps its signature; `HasPrevPage` becomes `HasPreviousPage`.

**Deletes and background work.** `EmptyResource` on a delete becomes `object.NewDeleted(id, objectType)`. `apiresource.Job` and `JobResult` become OpenMRP types that embed `object.AsyncJob[apiresource.Entity]` and add `created_by` and `export`. Job statuses and routes change as X11 says, with a data migration for stored statuses (`created` → `queued`, `started` → `running`, `cancelled` → `canceled`).

**Middleware wiring** (in `internal/router/configs.go`; the OpenMRP-side names in the right column, such as `requestLogSaver`, are illustrative):

| Today | apikit |
|---|---|
| `CORSMiddleware()` | `middleware.CORS(middleware.CORSConfig{AllowHeaders: []string{"OpenMRP-Account", "OpenMRP-Actor-Account"}})` |
| `AuthSecurityMiddleware()` | `middleware.SecurityHeaders()` |
| `RecoverMiddleware()`, `TracingMiddleware()` | `middleware.Recover()`, `middleware.Tracing()` |
| `IPBlockMiddleware(hops)` | `middleware.IPBlock([]string{"49.43.184.36"}, hops)` (the list becomes config) |
| `VersionMiddleware()` | `middleware.Version("/healthz")` |
| `RateLimitMiddleware(hops)` | `middleware.RateLimit(middleware.RateLimitConfig{Limiter: ratelimit.NewMemory(20, time.Second), TrustedProxyHops: hops, SkipPaths: []string{"/healthz"}, Skip: isTestOrDevPlatform})`, or `ratelimit.NewRedis` to share one count across pods |
| `IdempotencyMiddleware(cfg)` | `middleware.Idempotency(middleware.IdempotencyConfig{Store: platformIdempotencyStore, Scope: actorAndTargetAccount})`; the store adapts platform-service's `ProcessIdempotencyKey`, `SetIdempotencyKeyResponse` and `ReleaseIdempotencyKey` to `idempotency.Store` |
| `LoggingMiddleware(logger, next, saver, router, hops)` | `middleware.RequestLog(middleware.RequestLogConfig{Saver: requestLogSaver, Routes: router, NewID: newRequestID, TrustedProxyHops: hops, SkipPaths: []string{"/healthz"}})(router)` |

The auth, internal-auth, platform, sandbox, subscription and external-host middleware stay here, rewired to the kit's types.

## Step 5: Tooling

- **`tools/apidocs`** becomes a thin `main` around `openapi`. It keeps:
  - `endpoint_groups.go`;
  - the sample-data tables, behind `Examples.PathParam` (use `openapi.RouteSegmentBefore` for the route-segment table), `Examples.QueryParam` and `Examples.ListCursor` (the int64-cursor types);
  - `NamedEnums: []reflect.Type{reflect.TypeFor[constants.LocationTypeCode]()}`;
  - `DescribeAuth` over `Policy`, which is today's `authRequirementParagraph`;
  - `ParamDescriptions` for the refresh-token cookie;
  - the spec title, servers and `BearerAuth` scheme;
  - the Go writer for the agent-service tool catalog, over `openapi.AgentTools`, deriving `RequiredPermissions` and `RequiredRoleType` from each tool's `Auth`.

  It writes both specs with `openapi.WriteFile` and both Stainless workspaces with `openapi.WriteStainlessConfig`. The OpenMRP-specific apidocs tests stay: permission drift, cost fields, counterparty permissions, sample data, and doc sources.
- **Conformance.** Add `conformance.Assert(t, openAPIEndpointGroups())`. It fails until the forge.1 sweep is done, so while the sweep is in progress, run it with only the rules already met and add rules as each lands. It replaces `internal/http`'s source-scanning convention tests and the route-uniqueness test.
- **`make tx-audit`** becomes `go run github.com/open-mrp/apikit/cmd/txaudit -root .. -external stripeClient,shippoClient,hubspotClient,vercelClient,mapsClient,notificationClient,billingClient,coreClient,authClient,platformClient,agentClient,addressValidator,portalDomainProvider,checkoutClient`, which lists what the kit's defaults do not already cover. Delete `tools/txaudit`.
- `tools/vtparse`, `tools/schemasplit` and the HTTPie generator stay as they are.

## Step 6: Delete what moved

Remove every package and file named in Steps 1–5 that now lives in the kit, so nothing can drift back. A file left with only OpenMRP content gets its own package name, per the ground rules.

## What clients see change

These ship as part of forge.1. The dashboard, the SDKs and the e2e suite must expect them; the dashboard checklist in the review lists most of them, and these are the ones the kit itself causes:

- **Error object.** `errors[]` lists each failing field and is present on every error. `request_log_url` is gone. `quota` appears only on `limit_exceeded`. `doc_url` is built from the code.
- **Status codes.**
  - Validation failures are **422**, not 400.
  - A malformed body (bad JSON, unreadable) is a 400 `parameter_invalid`, not `validation_failed`.
  - Reusing an idempotency key with a different request is a 422 `idempotency_key_reused`.
  - `rate_limit_exceeded` becomes `rate_limited`.
- **Lists.** `page_info.has_prev_page` becomes `has_previous_page`; `limit` is 25 by default and at most 100; `q` exists only where an endpoint documents it.
- **Background work.** `/v1/core/jobs` becomes `/v1/core/async-jobs`; `object: "job"` becomes `"async_job"`; statuses and `job_result` change as X11 says.
- **Headers.** Responses to a deprecated version carry `Deprecation` and `Sunset`.

The dashboard reads several of these today (`has_prev_page`, `rate_limit_exceeded`, `validation_failed`, `/v1/core/jobs`), so they need its changes in the same release.

## Verifying it

1. `go build ./...`, `make test`, `make lint` (with the kit's `txaudit`).
2. **Spec diff.** Generate both specs and both Stainless configs before the migration and after. The only differences allowed are the ones under "What clients see change" plus the example-cursor churn from Step 1. Anything else is a migration bug.
3. Regenerate the internal SDK and type-check the dashboard against it.
4. The full e2e suite. This is the one time it is worth running everything rather than a filtered set, so ask before starting it.
5. Deploy to a sandbox environment and exercise sign-in, an order's lifecycle, a bulk upsert (async job) and an idempotent retry before production.
