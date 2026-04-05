# DELETE /v1/core/scanning-stations/{id} — Migration Verification

## Result: Issues found and fixed

## What was compared

- **Permission checks**: Internal actor, `scanningStations` domain, `delete` action — matches Dashboard
- **Account scoping**: Both scope deletion by `accountID` from identity context — matches
- **Existence check**: Go service verifies the station exists before deleting (via `Get`) — Dashboard relies on Prisma throwing if record not found; functionally equivalent
- **DB operation**: Dashboard does a hard delete (Prisma `delete()`); Go now does the same (`DELETE FROM scanning_station WHERE id = ? AND account_id = ?`)
- **Response shape**: Dashboard returns 200 with the deleted object; Go returns 204 No Content (EmptyResource) — acceptable divergence per Go API conventions
- **Side effects**: Neither implementation triggers notifications, webhooks, or cascading logic in application code
- **Idempotency**: DELETE is idempotent by design — no idempotency keys needed. Both implementations handle this correctly

## Issues found and fixed

1. **Repository `Delete` method was stubbed** — `scanning_station_repository.go:Delete()` returned `"Not yet implemented."` error. Implemented it with:
   - Added `DeleteScanningStation` SQL query (`:execresult`) to `scanning_station_query.sql`
   - Ran `make sqlc core` to generate the sqlc binding
   - Implemented the repository method following the standard pattern (execute delete, check `RowsAffected`, return not-found if 0)

## No remaining concerns

The endpoint is now fully functional with business-logic parity to the Dashboard implementation.
