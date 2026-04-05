# GET /v1/core/production-runs — Migration Verification

**Status: Issues found and fixed**

## What was compared

- **Permission checks**: Internal actor + `productionRuns:read` + target account required. Matches Dashboard.
- **Query parameters**: cursor, limit, query, status, item_ids, machine_ids, start_date, end_date. All Dashboard filters are supported.
- **DB queries**: Filters on account_id, number search, status (open/closed via completed_at), item IDs (EXISTS subquery on batches), machine IDs (EXISTS subquery on batch-machine join), date range on created_at. Cursor-based pagination (intentional change from Dashboard's offset-based).
- **Response shape**: Returns `ProductionRunSummary` with id, object, number, responsible_user (sub-resource), batch_count, started_at, completed_at, created_at, updated_at. Dashboard returns full batches array; Go returns batch_count (intentional summary optimization).
- **Error handling**: Standard identity/permission/validation errors. Matches patterns.
- **Side effects**: None (read-only). Matches Dashboard.
- **Idempotency**: Not required for GET. Correct.

## Issues found and fixed

### 1. Missing default status filter (fixed)
**Dashboard**: Defaults `status` to `"open"` when not provided — only non-completed production runs are returned by default.
**Go (before)**: No default — when `status` was nil, the status filter was skipped entirely, returning both open and closed runs.
**Fix**: Added default status of `"open"` in the gRPC handler (`grpc_production_run_handler.go`) when `req.Status` is nil.

### 2. Missing batch ID search in query filter (fixed)
**Dashboard**: The `query` parameter searches both `number` (contains) and `batch.id` (startsWith prefix match).
**Go (before)**: Only searched `number` via `LIKE %query%`.
**Fix**: Added an `OR EXISTS` clause to both forward and backward SQL queries (`production_run.sql`) that matches batch IDs with `LIKE query%` (prefix match). Updated the repository to build and pass a separate `batch_id_query` parameter. Regenerated sqlc.

## Intentional differences (not issues)

- **Pagination**: Dashboard uses offset-based (take/skip with total count). Go uses cursor-based pagination. This is a deliberate architectural choice for the Go API.
- **Sorting**: Dashboard sorts by Prisma full-text relevance on `number` then `createdAt DESC`. Go sorts by `createdAt DESC` only. Full-text relevance sorting is a Prisma-specific feature; the Go API still supports number search via LIKE.
- **Response detail**: Dashboard returns full `batches` array and `responsibleUser` with full user object (firstName, lastName, email). Go returns `batchCount` and `responsibleUser` as a light sub-resource (id, object, name). This matches Go API resource conventions.
