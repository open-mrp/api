# Verification: PUT /v1/core/picks/{id}/actions/void

## Result: Issues found and fixed

## What was compared

- **Permission checks**: Internal actor + `picks` domain `update` permission — matches
- **Validation**: Both check for shipped items before allowing void — matches
- **Error types/messages**: Both return appropriate errors for shipped items — matches (Go uses `NewValidationError`, Dashboard uses `HttpError.badRequest`)
- **HTTP method & status**: PUT returning 200 — matches
- **Idempotency**: PUT endpoint, no idempotency keys needed — correct
- **Response shape**: Returns the updated pick — matches
- **Transaction usage**: Both wrap mutations in a transaction — matches

## Issues found and fixed

### 1. VoidAllPickLines did not filter by `packed_at IS NULL`

**Dashboard behavior**: The `voidLine` method only updates pick lines where `packedAt` is null (unpacked lines). Packed lines are left unchanged.

**Go behavior (before fix)**: `VoidAllPickLines` SQL updated ALL pick line quantities to 0 regardless of packed status.

**Fix**: Added `AND packed_at IS NULL` filter to the `VoidAllPickLines` SQL query.

### 2. Missing `DeleteDuplicatePickLines` step

**Dashboard behavior**: After voiding lines, `deleteDuplicatePickLines` finds order lines with multiple pick lines, keeps only the first (by ID), and deletes the rest. This is a data cleanup step within the void transaction.

**Go behavior (before fix)**: This step was completely missing.

**Fix**: Added `DeleteDuplicatePickLines` SQL query, repository method, and service call (between VoidAllLines and ClearFinishedAt in the transaction).

## Files modified

- `services/core-service/internal/infrastructure/queries/pick.sql` — Added `packed_at IS NULL` filter; added `DeleteDuplicatePickLines` query
- `services/core-service/internal/infrastructure/sqlc/pick.sql.go` — Manual sqlc update for both changes
- `services/core-service/internal/infrastructure/sqlc/db.go` — Added prepared statement for `DeleteDuplicatePickLines`
- `services/core-service/internal/infrastructure/repository/pick_repository.go` — Added `DeleteDuplicatePickLines` method
- `services/core-service/internal/domain/repositories.go` — Added `DeleteDuplicatePickLines` to interface
- `services/core-service/internal/service/pick_svc.go` — Added `DeleteDuplicatePickLines` call in transaction

## Remaining concerns

- sqlc could not be regenerated due to a pre-existing syntax error in `GetPickShipmentNumbers` query (from other branch work). The sqlc generated file was manually updated to match. Should be regenerated when the pre-existing issue is resolved.
