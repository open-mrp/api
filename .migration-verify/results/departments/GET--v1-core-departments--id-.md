# Verification: GET /v1/core/departments/{id}

## Result: PARITY CONFIRMED

No issues found. The Go implementation is a faithful drop-in replacement for the Dashboard endpoint.

## What was compared

- **Permission checks:** Both require internal actor + `departments:read` permission. Both require target account ID.
- **DB query:** Dashboard uses Prisma `findFirst({id, accountID})` with relation selects. Go uses a SQL query filtering on `id` and `account_id` with a LEFT JOIN on `storage_location` for the location name, plus two separate sub-resource queries for scanning stations and machines — functionally equivalent.
- **Not-found handling:** Dashboard returns `HttpError.notFound('Department not found.')` when result is null. Go returns a not-found error via `db.MapSQLError` when `sql.ErrNoRows` is encountered — equivalent behavior.
- **Response shape:** Both return id, name, notes, location (sub-resource with id/name), scanning_stations (list of id/name), machines (list of id/name), created_at, updated_at. Go resource includes `object` field per API conventions.
- **Expandable sub-resources:** Go endpoint correctly defines `IncludeConfig` for location, scanning_stations, and machines.
- **Side effects:** None in either implementation (correct for GET).
- **Idempotency:** Not applicable for GET (correct in both).
- **Error types/messages:** Consistent between implementations.

## Issues found

None.
