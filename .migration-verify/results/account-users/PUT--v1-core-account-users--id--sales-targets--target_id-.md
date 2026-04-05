# PUT /v1/core/account-users/{id}/sales-targets/{target_id}

## Result: Issues found and fixed

## What was compared

- **Validation rules**: Request validation (date formats, required fields)
- **Permission checks**: Actor type, permission domain, action
- **Upsert logic**: Exists check, create vs update branching
- **Update behavior**: Which fields are modified on update
- **Create behavior**: Which fields are set on create
- **Error handling**: Error types and messages
- **Response shape**: Resource fields, sub-resources
- **Idempotency**: PUT is idempotent by design (no idempotency key needed) — matches both sides
- **Side effects**: None in either implementation

## Issues found and fixed

### 1. Update path modified too many fields (FIXED)

**Dashboard behavior**: When a sales target already exists, the update only modifies the quantity `measure` (value). It does NOT update `start_date`, `end_date`, or the quantity `unit_id`.

**Go behavior (before fix)**: The update path was updating `start_date`, `end_date` on the target record AND both `value` and `unit_id` on the quantity record.

**Fix**: Modified `services/core-service/internal/service/sales_target_service.go` to:
- Remove the `txRepo.Update(txCtx, params)` call (no longer updates dates)
- Pass `existing.AmountUnitID` instead of `params.AmountUnitID` to `UpdateQuantity` (preserves existing unit)

This now matches Dashboard behavior where only the quantity measure is updated on existing targets.

## Remaining concerns

### Account user validation (minor)

- **Dashboard**: Validates that the `userID` exists as an account_user in the target account before proceeding (returns 404 "Account user not found" if not).
- **Go**: Does not validate that the `SalesRepID` (account-user ID from path) exists in the target account. For creates, the FK constraint on `sales_rep_id` will catch invalid IDs (but with a less friendly error). For updates, the sales_rep_id isn't used since the target is looked up by its own ID.
- **Mitigation**: The route structures differ (Dashboard uses user ID, Go uses account-user ID directly), so the validation paths are inherently different. The Go approach will still fail on invalid IDs via FK constraints on create, and the update path doesn't use the sales_rep_id at all.

### Route structure difference (by design)

- Dashboard: `PUT /v1/identity/:accountID/users/:userID/targets/:targetID`
- Go: `PUT /v1/core/account-users/{id}/sales-targets/{target_id}`

The Go route uses account-user ID directly rather than user ID + account ID. This is consistent with the Go API's convention of using the `v1/core/` prefix and account-user IDs in paths.

## Parity summary

| Aspect | Status |
|--------|--------|
| Permission checks | ✅ Match (internal actor + salesTargets.update) |
| Create behavior | ✅ Match (inserts target + quantity with all fields) |
| Update behavior | ✅ Fixed (now only updates quantity measure) |
| Error handling | ✅ Match |
| Response shape | ✅ Match (sales_target with amount sub-resource) |
| Idempotency | ✅ Match (PUT, no idempotency key) |
| Account user validation | ⚠️ Minor gap (Go relies on FK constraints) |
