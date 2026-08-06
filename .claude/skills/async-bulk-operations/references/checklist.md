# Per-entity async bulk conversion — file-by-file

Worked references: `production_step_bulk_service.go` (upsert),
`production_run_bulk_service.go` (create), and `location_bulk_service.go` (upsert with
reference resolution + per-row links) — each entity's bulk story lives in its own
`<entity>_bulk_service.go`. The engine lives in `async_bulk_operation.go`; a spec is
`bulkOperationSpec[TInput, TResolved]`. For a **create**, `Resolve` pre-generates IDs and
`AcceptResults` records them on the 202's job; for an **upsert**, there is no
`AcceptResults` and `Write` finds-existing + splits. `Write` is **partial-success**: it
takes a `db.SavepointRunner`, wraps each row in `sp.Run`, and returns
`BulkWriteResult{Results, Errors, WrittenIDs}` — successes in `Results`, per-row failures
in `Errors` (the job's `errors` field). It returns an `err` only for an infra failure that
should fail the whole job.

## Files to change / add

| # | Layer | File | Change |
|---|---|---|---|
| 1 | Domain models | `internal/domain/<entity>_models.go` | Add `Resolved…Row` (+ sub-types) mirroring the fuzzy input with IDs inline — a plain struct, **no renaming/reshaping JSON tags** (the `job_items` payload round-trips against this same type; tags are a serialization concern that stays out of domain). **One exception:** a `field.Clearable` field needs `json:",omitzero"` — its `MarshalJSON` errors on an unset value, so `json.Marshal(resolved)` fails without it; the empty tag name renames nothing, it only enables the marshal mode the type requires. Prefer flattening a `Clearable` to a plain nil-able type when the create/update split does not depend on unset-vs-clear (as unit-groups/locations do); keep `Clearable` + `omitzero` only when that distinction is load-bearing across the job boundary (as scanning stations' label codes are). |
| 2 | Domain (shared) | `internal/domain/async_bulk_models.go`, `job_models.go` | Already exist — `BulkOperationJobEvent`, plus `domain.RowResult` and `apierror.RowError` (`shared/errors`, the same `{index, ResponseError}` an accept-phase `RowErrors` accumulates), shared by every operation (no per-entity result type). Reuse; nothing to add. The repository encodes them into the job's JSON columns, the gRPC handler maps them to structured proto, and the gateway maps that onto `apiresource.Job`'s `Results`/`Errors`. |
| 3 | Service struct | `internal/service/<entity>_service.go` | Add `jobSvcFactory domain.JobSvcFactory` to the impl, config, `validate()`, constructor, `withTx`. |
| 4 | Service logic | `internal/service/<entity>_bulk_service.go` (new) | The whole bulk story in one file: `asyncBulkDeps()` helper, `validate…Rows` (structural), `resolve…Rows` (fuzzy→resolved — resolve **every** reference here, plain-ID lists included), `write…` (per-row `sp.Run` loop, `newRowResult(i, id, isCreate)` for successes and `apierror.NewRowError(i, apiErr)` for failures, no marshaling). Start results as `make([]domain.RowResult, 0, len(rows))` — **never `var results []domain.RowResult`**: a nil slice records "no results at all" instead of "ran and wrote nothing", so a batch where every row failed reads as though it never ran. Build `bulkUpsertSpec`. Public method = `enqueueBulkOperation(...)`; add `ExecuteBulkUpsert…` = `executeBulkOperation(...)`. |
| 5 | Domain interface | `internal/domain/services.go` | `BulkUpsert…` returns `*Job`; add `ExecuteBulkUpsert…(ctx, BulkOperationJobEvent) *apierror.APIError`. |
| 6 | Messaging | `shared/messaging/bulk_operations.go` | Add `BulkUpsert… = BulkOperation{Slug: "bulk_upsert_<entity>"}` and append to `BulkOperations`. Routing key, queue, inbox handler key, and rabbitmq binding all derive from the slug (bindings range over `BulkOperations`) — nothing to touch in `contracts/amqp.go`, `queues.go`, or `rabbitmq.go`. The slug is persisted in the inbox: never rename it. |
| 7 | Wiring | `cmd/run.go` | Pass `JobSvcFactory: jobSvcFactory` to the service config; add one `event.NewBulkOperationConsumer(rabbitmq, inboxRepo, messaging.BulkUpsert…, svc.ExecuteBulkUpsert…)` line to the `bulkConsumers` list. No per-entity consumer file. |
| 8 | Proto | whichever file declares `BulkUpsert…Request` — **grep for it** | There is no `core_<entity>.proto` rule: bulk messages sit in `core.proto`, `core_items.proto`, `core_consumption.proto`, `core_invoices.proto` and `core_users_territories.proto`, and one entity's may appear in two of them (departments is in `core.proto` *and* `core_users_territories.proto`). Then `import "core/core_jobs.proto";` and `BulkUpsert…Response` → `{ JobInfo job = 1; }`. |
| 9 | gRPC handler | `internal/infrastructure/grpc/grpc_<entity>_handler.go` | Return `&pb.BulkUpsert…Response{Job: jobToProto(job)}` (shared mapper, same package). |
| 10 | Dead response plumbing | `shared/constants/object_type.go`, `api-gateway/pkg/resource/<entity>_resource.go`, `api-gateway/endpoints/<entity>/presenter.go` | The 202 returns the canonical Job, so the entity's own bulk response type dies with the sync path — leave none of it behind. **Delete** the `bulk_upsert_<entity>_response` ObjectType (const + `IsValid` case + `AllObjectTypes` entry), the `BulkUpsert…Response` resource struct and its sample, and the `BulkUpsert…Presenter` function. No new response resource replaces them. |
| 11 | Gateway service | `api-gateway/endpoints/<entity>/service.go` | `import jobep ".../endpoints/jobs"`; return `jobep.JobFromProto(resp.GetJob())` (`*apiresource.Job`). |
| 12 | Endpoint | `api-gateway/endpoints/<entity>/endpoint_bulk_upsert_<entity>.go` | Response type `*apiresource.Job`; `SuccessStatusCode: http.StatusAccepted`; `ObjectType: constants.ObjectTypeJob`; `LocationFunc → /v1/core/jobs/{resp.ID}`. |

## Regenerate, in order

```
make proto
make sqlc core-service      # only if you changed queries
make mocks core-service
make openapi-stainless      # commit stainless/internal/stainless.yml
make test                   # includes the tools/ module — must pass
```

Then rebuild + run the e2e stack and adapt the entity's
`tests/e2e/api/bulk_upsert_<entity>_test.go`: happy paths → 202 then poll the job;
synchronous-resolution failures stay 400; validations needing existing rows → poll the
job to `failed`.

## Tests

- Reuse `async_bulk_operation_test.go` — it covers the engine generically, so a new
  entity needs only spec-level coverage (its `Validate`/`Resolve`/`Write` specifics)
  plus the accept-phase rejection rows (permissions, empty, too-many, duplicates),
  which the entity's existing `*_service_test.go` already has once `SetupTest` gets a
  `JobSvcFactory`.

## Conversions

Done: production runs (create), production steps, units, unit groups, locations,
departments, item categories (name-key match + fuzzy unit-group ref; property names are
**not** resolved at accept — they are found-or-created by `Write`, so there is nothing to
fail on; the system-category and same-unit-type rules both need the existing row and moved
to `Write` as per-row errors), parts (SKU-key match + fuzzy category ref; `category` is
create-only but `required` on every row, so **all** categories resolve at accept — including
the type/base-unit check — and no SKU lookup is needed to split create from update. The
rate-unit rule and the cross-type SKU conflict are found by the write, so both became per-row
job errors: the cross-type conflict was a synchronous 409), products (same as parts plus an
optional `product_line` ref, also resolved at accept), materials (same as parts, plus
`order_point`/`lead_time` quantities carried through untouched; the duplicate-attribute-value
rule lives in `resolvePropertyAttributesInTx`, which runs **before** the row loop, so a
conflict fails the whole job rather than one row), machines (dual-key name-or-serial matching + department-intent rules —
`matchMachineForUpsert` decides update/create/collision per row, so the intent
rejections that were synchronous 400s became per-row job `errors`), product lines (single
name-key match + fuzzy unit-group ref; the system/default-line "cannot be modified" rule
moved to `Write` as a per-row error, and `upsertProductLineInTx` became a free function
taking `txRepos`), scanning stations (name-key match + fuzzy department ref; the
immutable-department and immutable-type rules moved to `Write` as per-row errors. **First
conversion with `field.Clearable` fields on the resolved row: they need `json:",omitzero"`
tags — the ONE allowed exception to the no-tags rule — because `Clearable.MarshalJSON` errors
on an unset value, so the engine's `json.Marshal(resolved)` would fail otherwise**).

Only **properties** (`property_service.go`) is left — the optional 9th, never in the original
list. The other still-synchronous bulk endpoints predate this work and are out of its scope:
`items`/`production-steps` bulk-create (201 with inline per-row results) and the five
bulk-deletes (batches, purchase orders, suppliers, customers, sales orders).
