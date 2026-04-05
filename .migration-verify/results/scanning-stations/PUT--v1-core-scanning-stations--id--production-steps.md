# PUT /v1/core/scanning-stations/{id}/production-steps

## Status: Issues found and fixed

## What was compared

- **Validation rules**: Request requires `name` (string, required) and scanning station ID (path param) — matches Dashboard
- **Permission checks**: Internal actor + `scanningStations.update` — matches Dashboard (`checkIsInternalActor` + `checkHasPermission(PermissionDomains.scanningStations, 'update')`)
- **Account scoping**: Both filter by target account ID
- **Scanning station existence check**: Go adds an explicit `IsInAccount` check before connecting — Dashboard relies on Prisma's `update` with `where: { id, accountID }` which would throw if not found. Equivalent behavior (404 if not found).
- **DB logic**: Dashboard finds production steps where `name CONTAINS query` (case-insensitive substring match via Prisma), then sets their `scanning_station_id` FK. Go now does the same via a single `UPDATE production_step SET scanning_station_id = ? WHERE account_id = ? AND name LIKE CONCAT('%', ?, '%')`.
- **Response shape**: Both return empty response body
- **Side effects**: None in either implementation
- **Idempotency**: PUT endpoint — idempotent by design (same request produces same result). No idempotency key needed.

## Issues found and fixed

1. **Repository not implemented**: `ConnectProductionStepsByName` in `scanning_station_repository.go` returned "Not yet implemented" error. Implemented it with the proper SQL UPDATE query that matches production steps by name substring and sets their `scanning_station_id`.

2. **SQL query missing**: Added `ConnectProductionStepsByName` query to `scanning_station_query.sql` that performs `UPDATE production_step SET scanning_station_id = ?, updated_at = NOW(3) WHERE account_id = ? AND name LIKE CONCAT('%', ?, '%')`.

3. **Status code mismatch**: Go endpoint returned 204 No Content, Dashboard returns 200 OK with `{}`. Changed Go to return 200 OK for parity.

## Notes

- The Go implementation uses a single UPDATE query which is more efficient than the Dashboard's two-step approach (find IDs, then connect). The behavior is equivalent: all production steps in the account whose name contains the query string get their `scanning_station_id` set to the given scanning station.
- The Dashboard's Prisma `connect` on a one-to-many relation sets the FK on child records without disconnecting existing ones. The Go UPDATE similarly only touches rows matching the name filter, leaving other already-connected steps unchanged. This is additive behavior in both implementations.
- The Go implementation also sets `updated_at = NOW(3)` on updated rows, which is good practice and consistent with how the Dashboard's Prisma `@updatedAt` works.
