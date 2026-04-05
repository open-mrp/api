# PATCH /v1/core/production-runs/{id} — Migration Verification

## Result: Issues found and fixed

## What was compared

- **Permission checks**: Internal actor + `productionRuns:update` permission — matches
- **Updatable fields**: `number` and `responsible_user_id` — matches
- **Duplicate number validation**: Both check uniqueness excluding current ID — matches
- **Responsible user validation**: Both look up account user by user ID and account ID, error if not found — matches
- **Account isolation**: Both scope queries by account ID — matches
- **Response shape**: `ProductionRunDetail` with sub-resource `ResponsibleUser`, batch count, timestamps — matches
- **Idempotency**: Go uses idempotency keys (PATCH pattern) — correct
- **Error handling**: Conflict for duplicate number, not found for invalid user — matches

## Issues found and fixed

### 1. Missing empty update validation (FIXED)

**Dashboard**: Throws `HttpError.badRequest('No valid fields to update.')` when neither `number` nor `responsibleUser` is provided in the request body.

**Go (before fix)**: Silently proceeded through the idempotency flow and returned the unchanged record without error.

**Fix**: Added a validation check in `production_run_service.go` before the `IsCompleted` check:
```go
if params.Number == nil && params.ResponsibleUserID == nil {
    return nil, tracing.Trace(span, apierror.NewValidationError("No valid fields to update."))
}
```

## Noted behavioral differences (intentional improvements)

### Completed run guard

**Dashboard**: Uses `completedAt: data.completedAt` in the Prisma WHERE clause. Since `completedAt` is typically `undefined` (not sent in the body), Prisma ignores it, effectively allowing updates to completed production runs.

**Go**: Explicitly checks `IsCompleted()` and rejects all updates to completed runs with "Cannot update a completed production run."

This is a stricter, safer behavior in Go. The Dashboard's implicit WHERE behavior appears unintentional rather than a deliberate feature — there's no good reason to allow updating completed production runs. No change made.
