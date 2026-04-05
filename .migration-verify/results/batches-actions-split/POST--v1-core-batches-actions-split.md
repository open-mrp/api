# POST /v1/core/batches/actions/split — Migration Verification

## Result: Issues found and fixed

## What was compared

- **Validation rules**: required fields, batch count constraints, duplicate checks, quantity validation
- **Permission checks**: actor type, permission domain (batches), action (create)
- **Business logic**: single-part vs multi-part routing, batch flow graph traversal, batch creation, flow edge connection, auto-close behavior, close-if-fully-used, close-if-last-step
- **Side effects**: outbox event enqueuing for production step execution (firsts with produceInventory=true, seconds+waste with produceInventory=false)
- **Response shape**: returns created BaseBatch with HTTP 201
- **Idempotency**: POST uses idempotency keys with recovery points (confirmed)
- **Error handling**: error types and messages

## Issues found and fixed

### 1. Missing single-part batch count validation
- **Dashboard**: `splitSinglePartBatch` checks `batches.length !== 1` and throws "Cannot split multiple batches."
- **Go (before)**: No validation — would silently use only the first batch ID and ignore extras.
- **Fix**: Added `len(params.BatchIDs) != 1` check at the top of `splitSinglePartBatch`.

### 2. Missing multi-part batch count validation
- **Dashboard**: `splitMultiPartBatch` checks `batches.length <= 1` and throws "Cannot split a single batch."
- **Go (before)**: No validation.
- **Fix**: Added `len(params.BatchIDs) <= 1` check at the top of `splitMultiPartBatch`.

### 3. Missing duplicate batch check
- **Dashboard**: After resolving available batches in flow, checks `uniqueBatches.size !== batches.length` and throws "Duplicate batches provided."
- **Go (before)**: No duplicate check — two different input batch IDs could resolve to the same available batch in the flow graph.
- **Fix**: Added duplicate detection on resolved source batches in `splitMultiPartBatch` (only relevant for multi-part since single-part requires exactly 1 batch).

### 4. Broken close logic in `splitSinglePartBatch`
- **Dashboard**: Calls `connectOneToOne({autoClose: closeBatch})` once.
- **Go (before)**: Called `ConnectOneToOne(..., false)` first, then if `closeBatch=true`, called `ConnectOneToOne` again with `true` — which would fail on duplicate batch_flow insert. The error was silently swallowed, meaning **source batches were never closed when `closeBatch=true`**.
- **Fix**: Changed to single `ConnectOneToOne(txCtx, accountID, sourceBatch.ID, newBatchID, params.CloseBatch)` call.

### 5. Broken close logic in `splitMultiPartBatch`
- **Dashboard**: Calls `connectManyToOne({autoClose: closeBatch})`.
- **Go (before)**: Always passed `false` to `ConnectManyToOne`, meaning source batches were never closed when `closeBatch=true`.
- **Fix**: Changed to pass `params.CloseBatch` to `ConnectManyToOne`.

## Remaining concerns

### `CloseIfFullyUsed` does not account for production step conversion ratios
- **Dashboard**: `closeIfFullyUsed` calls `calculateNextStepQuantities` which computes a production-to-consumption ratio (e.g., 10 input units → 5 output units) and compares the *expected output quantity* against actual output batch quantities.
- **Go**: `CloseIfFullyUsed` compares the source batch's raw `Quantity.Measure` directly against the sum of output batch quantities. If a conversion ratio exists (not 1:1), this will incorrectly compute remaining quantity and may fail to close batches that are fully used (or close them prematurely).
- **Impact**: Only affects production steps with non-1:1 conversion ratios. Fixing this requires adding a `CalculateNextStepQuantities` method to the Go production step query repo and updating `CloseIfFullyUsed` to use it. This is a cross-cutting concern that also affects merge and other batch operations, so it should be addressed separately.
