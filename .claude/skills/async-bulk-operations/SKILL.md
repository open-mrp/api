---
name: async-bulk-operations
description: Convert a heavy bulk write endpoint (bulk create or bulk upsert) from a blocking request to an asynchronous job, using the shared async-bulk-operation engine. Use when making a bulk operation async, adding a new bulk operation, or reviewing one — in the Go API's core-service and api-gateway.
---

# Async bulk operations

Heavy multi-row write fan-outs must not block a request. Per Dane's passively-safe-API
doctrine, they validate synchronously, record a **job**, return `202 Accepted` with a
`Location` header, and a worker executes them; the client polls the job. The job
machinery (`domain.JobSvc`, `JobSvcFactory`, `GET /v1/core/jobs/{id}`,
`POST /v1/core/jobs/{id}/cancel`,
`db.NullableRawMessage`) already exists. Each entity's bulk logic lives in its own
`internal/service/<entity>_bulk_service.go`. The exemplars:
`production_run_bulk_service.go` (bulk create, with `AcceptResults`) and
`location_bulk_service.go` (bulk upsert with reference resolution + per-row links).

## The two phases

1. **Accept** (synchronous, in the request): authorize, validate structure, resolve
   fuzzy references to IDs (fail fast with a row-indexed 400), record the resolved rows
   on a job **inside the outbox transaction**, enqueue only `{job_id}`, return
   `202` with **the canonical `Job` resource** (`object: "job"`) +
   `Location: /v1/core/jobs/{id}`. There is **no** per-operation response object — every
   async operation returns the same `Job` the client then polls (per Dane's Stripe
   review: shared canonical objects, not operation-specific response types).
2. **Execute** (a consumer → service method): load the job, guard against a settled
   one, run the writes in one transaction, settle the job **in that same transaction**,
   run non-fatal post-commit side effects. No idempotency envelope — delivery is
   **at-least-once, made effectively-once** by the inbox de-dup plus the engine's
   `IsTerminal` guard (don't call it "exactly-once"; the guard is load-bearing).

**Partial success.** The execute phase is not all-or-nothing: each row is written inside
its own SAVEPOINT, so a bad row rolls back only itself and the good rows still commit. A
completed job carries its successes in `results` and its per-row failures in the Job's
`errors` field. Every `errors` entry is `{index, error}` where `error` is the SAME
canonical error object a synchronous error response carries (`apierror.ResponseError`,
via `APIError.ToResponseError()`) — never a bespoke shape. The job therefore
**completes** even when some (or all) rows fail — `completed` means "processed", not
"all succeeded", so a client must read `errors`. Only an infrastructure failure (the bulk read, a doomed transaction, recording
the completion) rolls the whole batch back and **fails** the job.

**What "fails" actually means.** The bulk consumer classifies the failure before
deciding, on `APIError.IsTransient` (`apierror.IsTransientError` — server-side errors,
`idempotency_in_progress`, `rate_limit_exceeded`):

- **Transient** → the error is returned, so the delivery runs the broker-level
  `retry.WithBackoff` (1 initial + 3 backoff attempts in-process, `rabbitmq.go
  processDelivery`), then `Reject(false)` → **DLQ, no requeue**. A blip self-heals; an
  outage outlasting the window leaves a message to replay.
- **Deterministic** → the delivery is **acknowledged**. Redelivery would fail
  identically, and the execute phase already recorded the failure on the job, so the
  job's `failed` state is the outcome rather than four wasted attempts and DLQ noise.

`failed` is deliberately **non-terminal** (`Job.IsTerminal()` = completed/cancelled
only) so a DLQ replay or crash-recovery retry can re-drive the job; a retry that
succeeds lands `completed_at` beside the older `failed_at`, and `Job.Status()` resolves
that pair in favour of the completion. **The `UpdateJob` guard must agree**: it guards
on `completed_at`/`cancelled_at` and deliberately *not* `failed_at`, because guarding on
failure would freeze the row against the very retry the design promises. Nothing in the
type system couples the SQL to `IsTerminal`, so
`TestJobRepo_Update_GuardsOnTheTerminalStatesOnly` pins them. Crash mid-write (no ack)
retries via the inbox's crash-recovery path.

What the client no longer gets inline is recorded on the job's `results` for polling.
`results` is **row-indexed**, the success-side mirror of `errors`: an array of
`{index, id, action}` (`action` = `created`|`updated`; a production run also carries
`sub_resource_ids`, its batches). Together with `errors` every submitted row is
accounted for — each lands in exactly one of the two once the job completes. This
replaced the old flat `{created_ids, updated_ids}`, which couldn't map a returned id
back to its request row.
An **upsert** computes the create/update split at execute time against live rows — never
at accept (which would add a DB read and a TOCTOU window). A **create** already knows
its pre-generated IDs at accept, so it records the full results then (via the spec's
`AcceptResults` hook) — the `202`'s `Job` already carries them for the client to use
immediately, and the execute-phase `Write` records the same IDs, dropping any row that
fails. **Provisional-until-completed:** accept-time results are the rows the job intends
to write; a row can still fail (partial success), moving to `errors`. Results are final
only once `status` is `completed`.

### Results and errors are typed end to end — no JSON in the service layer

`domain.JobResult{Index, ID, Action, SubResourceIDs}` and `domain.JobError{Index, Error}`
are **tag-free domain types**, and they stay typed through every layer:

| layer | what it does |
|---|---|
| service (`Write`, `AcceptResults`, `FailJob`) | builds `[]domain.JobResult` / `[]domain.JobError` — **never marshals** |
| repository (`job_repository.go`) | `encodeJSONList`/`decodeJSONList` — the JSON column is storage, so its encoding lives here and nowhere else |
| gRPC handler | `jobResultsToProto`/`jobErrorsToProto` → `JobResultList`/`JobErrorList` |
| gateway presenter | `jobResultsFromProto`/`jobErrorsFromProto` → `apiresource.JobResult`/`JobError`, which own the snake_case HTTP tags |

The two halves are carried differently on purpose, and the rule is **share a type where
one exists, define a proto message where none can**:

- **Results are structured proto** (`JobResultInfo`). `domain.JobResult` lives in core's
  `internal/`, which the gateway cannot import, so there is no shared Go type to lean on
  — proto is the only place the contract can live. `Action` is
  `constants.JobResultAction`, whose `EnumValues()` the schema generator picks up, so it
  is a first-class SDK model.
- **Errors are the canonical error object as JSON** in `JobErrorInfo.error`. Both
  services already import `apierror.ResponseError` from `shared/`, so mirroring it into
  proto would only create a second definition to hand-maintain. Marshaling and
  unmarshaling the one shared type is drift-proof by construction, and the bytes are
  identical to the `error` object a synchronous failure returns. (This was briefly a
  `JobErrorDetail` mirror guarded by a parity test; the mirror was the only thing the
  test existed to police, so both went.)

Either way the public schema is generated from `apiresource`, so clients see the full
typed error object regardless of how it crossed gRPC.

Two rules this places on new code:
- **Never invent a per-entity results type.** Every bulk op returns `domain.JobResult`;
  a genuinely new extra (as `SubResourceIDs` was) is an added field, not a new type. Do
  not name such a field for one entity — `SubResourceIDs` is deliberately generic
  because "batch" already means both the `Batch` model and the set of rows in a bulk
  request.
- **Nil vs empty is load-bearing.** A nil results list means the job has recorded no
  results; an empty-but-non-nil one means it ran and wrote nothing. That distinction
  survives the column (NULL vs `[]`), the proto (unset vs present-empty wrapper message
  — which is why `JobResultList` wraps the repeated field), and the resource (`null` vs
  `[]`). So a `Write` must build its results with `make([]domain.JobResult, 0, len(rows))`,
  never `var results []domain.JobResult`, or a batch where every row failed would read as
  "no results recorded".

`job_items` stays `json.RawMessage` and is still marshaled in the engine — it is
genuinely heterogeneous per operation, so only the engine knows its type.

## The engine

`services/core-service/internal/service/async_bulk_operation.go` owns both phases for
both **create** and **upsert**. An entity plugs in a
`bulkOperationSpec[TInput, TResolved]` — the invariant plumbing (permissions scaffold,
idempotency, transaction, job lifecycle, outbox, result recording) is the engine's; the
variance is a few functions:

- `Actions []types.Action` — `{Create}` for a create, `{Create, Update}` for an upsert.
- `Validate(rows []TInput)` — structural + in-request duplicate checks, row-indexed. No DB.
  **Every bulk operation reports these identically** — see the accept-phase error rule below.
- `Resolve(ctx, repos, accountID, rows) ([]TResolved, err)` — fuzzy refs → IDs, and for a **create** also pre-generate the new rows' IDs. Runs **after** the idempotency key is claimed, because it reads the database (see the ordering rule below); `Validate` runs before it, because it does not. `TResolved` is stored on the job, so it must carry only resolved IDs, never fuzzy identifiers or maps keyed by structs. Keep it a **plain domain struct with no JSON tags**: the engine marshals and unmarshals `job_items` against this same type and it is an internal column, so default field names round-trip — serialization tags do not belong on a domain model.
- `AcceptResults(resolved) []domain.JobResult` — **optional**, and total (nothing to fail: it only builds domain values). A create returns its pre-generated IDs here — the same rows `Write` records — so the job the 202 returns already carries them; an upsert omits it (nil), since its split is unknown until `Write`. There is no `Acknowledge`/`TAck`: the raised `Job` is the 202 body.
- `Write(txCtx, txRepos, sp, accountID, rows) (BulkWriteResult, err)` — in one tx, **partial-success**: after the one-time bulk read, wrap **each row's** writes in `sp.Run(...)` (a `db.SavepointRunner`). A row that fails rolls back only itself (SAVEPOINT/ROLLBACK) and is collected via `newRowError(i, apiErr)`; the rest still commit. Accumulate successes into `results := make([]domain.JobResult, 0, len(rows))` (non-nil — see the nil-vs-empty rule) via `newJobResult(i, id, isCreate)`, then return `BulkWriteResult{Results: results, Errors: rowErrors, WrittenIDs: writtenIDsFromResults(results)}`. No marshaling — these are domain values. Return an `err` **only** for an infrastructure failure (the bulk read, a data invariant) that should roll the whole batch back and fail the job. Converges by natural key (upsert) or pre-generated ID (create) on redelivery.
- `AfterCommit(ctx, repos, accountID, writtenIDs) *apierror.APIError` — optional, non-fatal derived side effects (e.g. flow relinking) that must not roll the writes back. Return the failure rather than handling it; the engine traces it and the job stands completed.

`enqueueBulkOperation` runs Accept and returns the raised `*domain.Job`;
`executeBulkOperation` runs Execute. Full instances: `production_step_bulk_service.go`
(`bulkUpsertSpec`, upsert, with `AfterCommit`) and `production_run_bulk_service.go`
(`bulkCreateSpec`, create). `async_bulk_operation_test.go` shows how to test the engine
with a fake spec.

**Create vs upsert.** A create pre-generates IDs in `Resolve` and records them at accept
via `AcceptResults`, because the caller wants each new row's ID immediately; an upsert
cannot know the created/updated split until the writes run, so it omits `AcceptResults`
and the split lands in the job's `results` at execute time. Both return the same `Job`;
everything else is identical.

## Per-entity conversion checklist

Reuse `references/checklist.md` for the file-by-file steps. In summary:

1. **Domain** (`<entity>_models.go`): add a `Resolved…Row` type mirroring the fuzzy
   input with IDs inline — a plain struct, **no JSON tags** (see the `Resolve` note
   above). The service method returns `*domain.Job` (no per-entity result type).
2. **Service** (new `<entity>_bulk_service.go`, beside `<entity>_service.go`): the
   entity's whole bulk story lives in this one file. Add `JobSvcFactory` to the impl
   struct, config, `validate()`, constructor (in the existing service file); add an
   `asyncBulkDeps()` helper. Split the old method: keep the structural checks as a
   `Validate` func, hoist the in-tx `resolveBulk…RefsInTx` to a `Resolve` func returning
   resolved rows, move the write loop to a `Write` func returning `BulkWriteResult`,
   move any post-commit step to `AfterCommit`; a create adds `AcceptResults`. Build a
   `bulkUpsertSpec`, and make the public method `enqueueBulkOperation(...)` and a new
   `ExecuteBulkUpsert…` be `executeBulkOperation(...)`.
3. **Domain interface** (`services.go`): the `BulkUpsert…` method returns `*Job`; add
   `ExecuteBulkUpsert…(ctx, BulkOperationJobEvent)`.
4. **Messaging** (`shared/messaging/bulk_operations.go`): register one
   `BulkOperation{Slug: "bulk_upsert_<entity>"}` var and append it to `BulkOperations`.
   The routing key, queue, inbox handler key, and rabbitmq binding ALL derive from that
   slug — there is nothing to add in `contracts/amqp.go`, `queues.go`, or `rabbitmq.go`
   (bindings range over `BulkOperations`). The slug is persisted in the inbox, so it
   must never be renamed.
5. **Consumer**: no new file — add one line to the `bulkConsumers` list in `cmd/run.go`:
   `event.NewBulkOperationConsumer(rabbitmq, inboxRepo, messaging.BulkUpsert…, svc.ExecuteBulkUpsert…)`.
6. **Proto** (`proto/core/core_<entity>.proto`): response → `{ JobInfo job = 1; }`
   (`import "core/core_jobs.proto"`). The gRPC handler maps `*domain.Job` with the
   shared `jobToProto` (same package as the entity handler) and returns `{ Job: … }`.
7. **Gateway**: the service returns `*apiresource.Job` via `jobep.JobFromProto(resp.GetJob())`
   (`import jobep ".../endpoints/jobs"`) — there is **no** per-entity response resource
   or `ObjectType`. The endpoint's response type is `*apiresource.Job`, with
   `SuccessStatusCode: http.StatusAccepted`, `ObjectType: constants.ObjectTypeJob`, and
   `LocationFunc → /v1/core/jobs/{resp.ID}`. **Delete** the entity's now-dead
   `bulk_upsert_<entity>_response` `ObjectType` (const + `IsValid` + `AllObjectTypes`)
   and any leftover response resource/`Sample`.
8. **Regenerate**: `make proto`, `make sqlc core-service` (only if queries changed),
   `make mocks core-service`, `make openapi-stainless`.

## Rules that bite (each cost a debugging cycle)

- **Accept-phase errors are IDENTICAL across every bulk operation.** One shape, no
  per-entity dialects — a client that learned to read a bad `machines` request must read a
  bad `production_steps` request the same way. `Validate` and every accept-phase resolver
  accumulate into an `apierror.RowErrors` (`shared/errors/row_errors.go`) and end with
  `return rowErrs.Summary(<the spec's EntityName>)`, which renders
  `Invalid <entity plural> — <param>: <clause>; <param>: <clause>.` and puts the **first**
  offending param on the error. Three parts are load-bearing:
  - **Every failure carries a row-indexed param** — `machines[3].name`,
    `production_steps[2].consumptions[0].item`. Never `apierror.NewValidationError` (no
    param) in a bulk path: the published contract tells clients to "fix the value named by
    `param`" (docs.openmrp.ai/api/errors), and a param-less error in a 500-row request makes
    them string-match the message to find the row. A bare param like `"ratio_denominator"`
    is the same bug — it does not say *which* row.
  - **Collect every bad row, don't return on the first.** No job exists yet at accept, so
    the job's `errors` array cannot carry them — the message is the only channel, and
    `message` is documented as human-readable-but-not-parseable, so listing them all costs
    nothing and saves the caller a fix-one-retry loop.
  - **`AddValidation(index, param, clause)` takes a bare clause** — lowercase, no trailing
    period, and **not** prefixed with the param; `Summary` does the `"param: "` prefixing so
    the format cannot drift per entity. The accumulator keeps each failure whole as an
    `apierror.RowError` (`{index, ResponseError}`), so `Entries()` yields the same array a
    job's `errors` carries — `Summary` is a rendering of intact data, not the only copy.

  This applies to shared resolvers too (`resolveObjectIdentifiers`,
  `resolveDepartmentIdentifiersInTx`, `validateBulkCreateCategoriesInTx`) — they already
  use `RowErrors`, and a new one must. Execute-phase per-row failures are a different
  channel and keep their own shape: `newRowError(i, apiErr)` → the job's `errors`.

- **Nullable `json` columns need `db.NullableRawMessage`, never `json.RawMessage`** —
  `database/sql` cannot scan NULL into a named `[]byte` type, and the failure is a
  runtime 500 on read-back, not a compile error. See `job.results`/`job.errors` in
  `services/core-service/sqlc.yaml`; the repository decodes that raw value into the
  typed list.
- **The job is created inside the accept transaction, with the outbox message** — so
  the job, the message, and the idempotency cache commit atomically.
- **`StartJob`/`FailJob` run outside the write tx; `CompleteJob` inside it** — start and
  fail marks must survive a rollback; completion must commit with the work.
- **The consumer is a thin translator** — one service call, no repo or DB access, plus
  the one decision only it can make: whether a failure is worth redelivering (see the
  transient/deterministic split above). The generic `BulkOperationConsumer` already is
  one; older non-bulk consumers violate this — do not copy them.
- **Resolve EVERY reference at accept, including plain-ID lists** — the fail-fast
  promise is "bad references 400 synchronously". A field that "is already an ID" still
  needs an accept-phase existence check; only intra-batch references (a name matching
  another row in the same request, resolvable only after phase-1 writes — see
  `location_bulk_service.go`) may defer to the write phase.
- **Never acquire a serialized resource per-row inside the batch tx** — e.g.
  `GetNextNumber`'s `SELECT MAX … FOR UPDATE` per run holds an account-wide lock for
  the entire 1000-row transaction, blocking every concurrent create and inviting
  lock-wait timeouts that (given DLQ-on-error) permanently fail the job. Allocate
  scarce/serialized values once up front (one locked read, then increment locally).
- **`AfterCommit` returns its failure; the engine records it** — non-fatal never means
  silent, so the hook does not decide what to do with an error (that is how one gets
  swallowed: `_ = apiErr` was a real bug here). Return it and the engine calls
  `tracing.Trace(span, apiErr)` — **the** way this codebase records an error (~5k call
  sites; `slog` is reserved for the one canonical log line per gRPC call, see
  `docs/patterns/canonical-log-patterns.md`, so do not hand-write log lines in a
  service). It cannot go on the job either: by then the job is completed and terminal,
  so the guard refuses further writes. If the side effect can run in-tx, fold it into
  `Write` instead and it becomes a row error like any other.
- **`errors` entries are `domain.JobError{Index, Error}` — the ONLY shape.** `Error` is
  `apierror.ResponseError` (`shared/errors`), the platform's client-facing error object,
  the same one every synchronous error envelope carries; build entries with
  `newRowError(i, apiErr)` (→ `apiErr.ToResponseError()`). A whole-job failure
  (`FailJob`) records one entry with no `Index`. Never mint a parallel error shape —
  and specifically, never mirror `ResponseError` into proto: both services import it
  from `shared/`, so it crosses gRPC as its own JSON and stays one definition. A *failed*
  RPC is different and unrelated: that error rides the gRPC status via
  `ConvertAPIErrorToGRPC`'s `__API_ERROR__:` marker. Job errors are data on a
  *successful* response, which is why they need a representation of their own at all.
- **`results` entries are `{index, id, action, sub_resource_ids?}` — the success mirror.**
  Build with `newJobResult(i, id, isCreate)`; `action` is `constants.JobResultAction`
  (`created`/`updated`). Same both-sides-of-the-wire discipline: core-service writes
  `domain.JobResult`, and the gateway maps it from structured proto. Every submitted row
  ends in exactly one of results/errors, both keyed by `index`. A new sub-resource extra
  is an added field (see `SubResourceIDs`), never a new type.
- **`Validate` before the key, `Resolve` after it** — and the split is deliberate, so
  keep it. `Validate` reads no database and depends only on the body, which is fixed for
  a given key, so it answers identically on every attempt: running it first means a
  malformed request never claims a key, matching the synchronous endpoints. `Resolve`
  reads the database, so its answer depends on *when* it runs — running it before the
  key would let a reference deleted after the first attempt succeeded turn a replay of
  an already-accepted request into a 400 instead of handing back its job. A `Resolve`
  failure deliberately leaves the key unfinished, so a retry resolves again once the
  client supplies what was missing.
- **The consumer reads the job via `GetJobForExecution` (permission-free), not
  `GetJob`** — the worker holds the requester's identity for tenancy and attribution,
  not authority; that was settled when the endpoint accepted the work. Gating on
  `jobs:read` would silently dead-letter any non-admin.
- **Authorization stays on the endpoint's own domain** (e.g. `production_steps:*`). The
  `jobs` domain only needs `jobs:read`/`jobs:delete` seeded in `shared/db/seed/0004_auth.sql`.
- **`LocationFunc` fires on 202 as well as 201** (`api_endpoint.go`). Set `SDKMethodKey`
  if the verb-derived SDK method name reads wrong (e.g. DELETE-that-cancels → `cancel`).
- **Validations that need existing rows move to the execute phase** — e.g. a
  create-only field rule that reads matched rows becomes an async **job failure**, not
  a synchronous 400. Adjust e2e assertions accordingly (poll the job to `failed`).
- **`AfterCommit` runs after the job is marked completed** — a client can observe
  `completed` before a post-commit side effect finishes. Poll for such side effects in
  e2e rather than asserting them immediately.
- **`tools/` is a separate Go module** — run `make test` (not just
  `go test ./services/...`); it validates spec parity and sample-ID coherence. New
  endpoint groups go in `tools/apidocs/endpoint_groups.go`; new sample IDs in
  `sample_doc_ids.go`; new `{id}` route segments in `path_parameter_examples.go`.
- **Commit `stainless/internal/stainless.yml`** (regenerated by `make openapi-stainless`)
  — the SDK build reads it. `specs/*.json` are gitignored.
- **After any generated-code change, re-run the e2e stack**:
  `make e2e-down && make e2e-up && make e2e`.

## Job type

Reuse the generic `constants.JobTypeBulkUpsert` for all upserts (bulk create uses the
generic `JobTypeBulkCreate`) — consistent with the existing async convention, and one
fewer step per conversion.

## Known gaps (2026-07-22 design review)

Open issues in the shipped pattern. Don't propagate them into new conversions, and
don't "fix" one silently inside an unrelated conversion — they're deliberate follow-ups:

1. ~~Retry doesn't classify errors~~ **Fixed 2026-07-23**: the bulk consumer branches on
   `APIError.IsTransient` — deterministic failures are acknowledged (the job already
   records them), only transient ones retry and dead-letter. The same change removed a
   contradiction that made the retry path unreachable anyway: `UpdateJob` guarded on
   `failed_at IS NULL` while `IsTerminal()` reported `failed` as retryable, so every
   redelivery passed the Go check and was then refused by the query at `StartJob`.
   Still open: a transient failure outlasting the backoff window (4 attempts) leaves the
   job failed until someone replays the DLQ — there is no automated re-drive.
2. ~~Accept-time `results` contradicts the public Job contract~~ **Fixed 2026-07-22**:
   the Job resource now documents `results` as provisional-until-`completed`, and the
   row-indexed shape means a failed row moves to `errors` rather than lingering as a
   phantom success.
3. ~~`errors` has two shapes~~ **Fixed 2026-07-22**: every entry is now
   `{index?, error: <canonical ResponseError>}` (see the rule above).
4. ~~Results wire structs live per-service, untyped in SDKs~~ **Fixed 2026-07-23**:
   results and errors are domain types mapped onto the resource — results via a proto
   message, errors as the shared `ResponseError`'s own JSON — so the service layer
   marshals nothing and both are first-class SDK models. (The remaining generality
   question — a non-row-oriented job type whose results aren't `[]JobResult` — is
   deferred until such a type exists; it would need a discriminated `results` on the
   Job resource.)
5. **No job watchdog** — a job whose message vanishes entirely (purged DLQ, deleted
   queue) sits non-terminal forever; a reaper/completer would give it a visible end.
6. ~~Locations `child_ids` asymmetry~~ **Fixed** (children are full identifiers resolved
   at accept, with intra-batch references). Location parent links still have **no cycle
   guard** anywhere (pre-existing in the sync path too).
7. **No canonical log line on the consumer path.** `CanonicalLogInterceptor` is gRPC-only
   (`docs/patterns/canonical-log-patterns.md`), so an AMQP-driven execution emits no
   canonical record — the span is the only structured account of what happened, and the
   consumer falls back to `log.Printf`. A consumer-side equivalent belongs in
   `shared/logging`, not in ad-hoc `slog` calls scattered through the services it drives.
