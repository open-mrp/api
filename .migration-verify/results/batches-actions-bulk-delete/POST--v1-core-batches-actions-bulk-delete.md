# POST /v1/core/batches/actions/bulk-delete

## Status: Issues found and fixed

## What was compared

- **Validation**: Request body requires `batch_ids` array (Go) / `batchIDs` array (Dashboard) — both enforce required field
- **Permission checks**: Both check `checkIsInternalActor` and `checkHasPermission(batches, delete)` — parity confirmed
- **DB queries and logic**: Both delete batches by IDs scoped to account. Go additionally explicitly deletes `_batch_flow` and `_batches_machines` records (Dashboard relies on Prisma cascade) — functionally equivalent
- **Error handling**: Compared below
- **Side effects**: Both collect production run IDs from deleted batches and call `closeIfAllBatchesScannedOrDeleted` for each — parity confirmed
- **Response shape**: Dashboard returns 200 with `{}`; Go returns 204 No Content — acceptable Go API convention difference
- **Idempotency**: gRPC handler has `WithIdempotencyTracking`. Service layer does not use idempotency keys. Dashboard doesn't either. Bulk delete is inherently idempotent at the DB level.

## Issues found and fixed

### 1. Missing 404 when no batches found

**Dashboard behavior**: If none of the provided batch IDs exist for the account, the repository throws `HttpError.notFound('Batches not found.')`.

**Go behavior (before fix)**: Silently succeeded with no error, even if zero batches were found.

**Fix**: Added a `foundCount` tracker in the service. After iterating through batch IDs, if `foundCount == 0`, return `apierror.NewNotFoundError("Batches not found.")`.

## Remaining concerns

- None. The Go implementation now matches Dashboard business logic.
