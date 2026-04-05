# POST /v1/core/batches/actions/initialize

**Status: Issues found and fixed**

## What was compared

- Permission checks (internal actor, batches/create)
- Validation rules (scanning station in account, batch exists, batch not closed, batch not scanned, production run exists, production step exists)
- Validation order
- Error messages and error types
- Transaction logic (mark scanned, connect production step, connect scanning station, close if last step)
- Post-transaction side effects (start production run, close if all batches scanned/deleted)
- Async side effect (execute production step via outbox message)
- Response shape (BaseBatch resource)
- Idempotency key handling

## Issues found and fixed

### 1. Missing production run validation
- **Dashboard:** Checks `if (!productionRunID) throw HttpError.notFound('Production run not found.')` before proceeding.
- **Go (before fix):** Did not check if the batch had an associated production run. The post-transaction code handled nil gracefully by skipping, but the Dashboard treats this as a hard error that aborts the operation.
- **Fix:** Added `if batch.ProductionRun == nil` check returning `apierror.NewResourceNotFoundError("Production run not found.")`.

### 2. Validation order mismatch
- **Dashboard:** Checks closed → scanned.
- **Go (before fix):** Checked scanned → closed.
- **Fix:** Reordered to check closed before scanned, matching Dashboard behavior.

### 3. Error message wording
- **Dashboard:** "This batch is closed." / "This batch has been scanned already."
- **Go (before fix):** "Batch is closed." / "Batch has already been scanned."
- **Fix:** Updated messages to match Dashboard wording exactly.

## Remaining concerns

### Plan/limit check not implemented
- **Dashboard** calls `AccountPlanSvc.canScanBatch(accountID)` which checks billing period batch scan limits against the account's plan.
- **Go** has no equivalent. The billing service exists but has no batch scan limit enforcement.
- This is a **cross-service concern** that would require adding a gRPC call from core-service to billing-service. It should be tracked as a separate task.

## Parity confirmed for

- Permission checks: identical (internal actor + batches/create)
- Scanning station validation: identical (IsInAccount check)
- Batch retrieval: identical (Find by accountID + batchID)
- Production step lookup: identical (FindIDByScanningStationAndProducedBlock with same params)
- Transaction operations: identical (mark scanned, connect step, connect station, close if last)
- Execute production step side effect: Dashboard fires async after response; Go uses outbox message pattern (equivalent)
- Production run post-transaction ops: identical (start, close if all batches done)
- Idempotency: Go correctly uses idempotency keys (POST endpoint)
- Response: HTTP 201 with BaseBatch resource
