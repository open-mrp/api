# POST /v1/core/batches/actions/move — Migration Verification

## Result: PARITY CONFIRMED

No code changes required. The Go implementation faithfully reproduces the Dashboard business logic.

## What Was Compared

### Validation Rules
- **Empty batch IDs**: Both reject with error (Dashboard: 400 "No batches to move", Go: validation error "At least one batch ID is required.")
- **Scanning station not in account**: Both return 404
- **Production step not in account**: Both return 404
- **Closed source batch**: Both reject (Dashboard: 400 "Batch is closed", Go: validation error "Source batch is closed.")
- **Duplicate batches (multi-part)**: Both reject (Dashboard: 400 "Duplicate batches provided", Go: validation error "Duplicate batch found in flow.")
- **Mismatched output quantities (multi-part)**: Both reject (Dashboard: 400 "All batches must have the same measure", Go: validation error "Calculated output quantities do not match across input batches.")
- **Batch not found**: Both return 404 — Dashboard checks `batches.length !== batchIDs.length`, Go errors on first missing batch in `FindNextAvailableBatchInFlow`. Same outcome.

### Permission Checks
- Both require internal actor (`checkIsInternalActor`)
- Both require `create` permission on `batches` domain
- Both require target account ID to be set

### Business Logic
- **Single-part flow**: Both find next available batch via BFS, calculate next step quantities, create new batch, connect source→target with auto-close, close if last step
- **Multi-part flow**: Both find all available batches, validate no duplicates, validate no closed batches, calculate quantities for each and verify consistency, create new batch, connect all sources→target with auto-close, close if last step
- **Auto-close behavior**: Dashboard `connectOneToOne` and `connectManyToOne` both default `autoClose = true`, matching Go's explicit `true` parameter

### Side Effects
- Dashboard: Calls `executeProductionStep()` as fire-and-forget after responding to client (error swallowed)
- Go: Enqueues `ExecuteProductionStepEvent` to outbox inside the transaction
- This is an intentional architectural improvement — the outbox pattern guarantees at-least-once delivery vs fire-and-forget

### Response Shape
- Both return HTTP 201 (Created)
- Both return a single `BaseBatch` resource

### Idempotency
- Go correctly implements idempotency keys with recovery points (Started, Finished) for this POST endpoint
- Dashboard had no idempotency support — this is an expected improvement

## Notes
- Minor error message wording differences are acceptable and don't affect behavior
- The outbox pattern for production step execution is a deliberate improvement over fire-and-forget
