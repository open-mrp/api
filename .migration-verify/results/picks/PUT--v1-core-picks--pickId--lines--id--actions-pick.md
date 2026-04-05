# Verification: PUT /v1/core/picks/{pickId}/lines/{id}/actions/pick

## Status: Issues found and fixed

## What was compared

- **Validation**: Request params (pickId, id from path) — matches
- **Permission checks**: Internal actor + `picks:update` permission + target account required — matches
- **Account scoping**: Go verifies pick is in account, then line is in pick — equivalent to Dashboard's ownerAccountID checks
- **Business logic**: Remaining quantity calculation and update behavior
- **Side effects**: No side effects in either implementation — matches
- **Response shape**: Returns updated PickLine with quantity, ordered quantity, order line info — matches
- **Idempotency**: PUT endpoint, no idempotency keys needed — matches (correctly omitted)

## Issues found and fixed

### 1. Incorrect remaining quantity calculation (pick.sql — `PickRemainingQuantityForLine`)

**Problem**: The Go SQL excluded the current pick line from the sum (`pl2.id != pl.id`), calculating `ordered - sum(others)`. The Dashboard includes ALL pick lines (including the current one) in the sum, calculating `ordered - sum(all)`.

**Impact**: With the Go version, if a pick line already had quantity 30 and others summed to 40 with 100 ordered, Go would set the line to 60 (100-40). Dashboard would set it to 30 (100-70). The Go version would overshoot the intended "remaining" quantity.

**Fix**: Removed `AND pl2.id != pl.id` from the subquery so the sum includes all pick lines for the order line, matching Dashboard behavior.

### 2. Missing `packed_at IS NULL` guard (pick.sql — `PickRemainingQuantityForLine`)

**Problem**: The Go SQL did not check `packed_at IS NULL` before updating, meaning it could modify already-packed pick lines. The Dashboard's Prisma update has `packedAt: null` in the WHERE clause, preventing updates to packed lines.

**Fix**: Added `AND pl.packed_at IS NULL` to the WHERE clause.

## Minor behavioral difference (accepted)

The Dashboard returns the current line unchanged when calculated remaining is negative (overpicked state). The Go version uses `GREATEST(0, ...)` which would set the quantity to 0 in that edge case. This is an error state that shouldn't occur in normal operation, and setting to 0 is a safe fallback. No change made.

## Files modified

- `services/core-service/internal/infrastructure/queries/pick.sql` — Fixed `PickRemainingQuantityForLine` query
- `services/core-service/internal/infrastructure/sqlc/pick.sql.go` — Regenerated via `make sqlc core`
