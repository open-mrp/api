# GET /v1/core/scanning-stations/{id} — Verification Result

**Status: Issues found and fixed**

## What was compared

- **Validation rules**: Path parameter `id` (scanning station ID) — matches
- **Permission checks**: `CheckIsInternalActor()` + `CheckHasPermission(ScanningStations, Read)` + `CheckTargetAccountSet()` — matches Dashboard's `checkIsInternalActor` + `checkHasPermission(scanningStations, read)`
- **DB query and logic**: Both filter by `(id, account_id)` — matches
- **Error handling**: Go uses `db.MapSQLError` which returns 404 on `sql.ErrNoRows`, equivalent to Dashboard's `findUnique` returning null → `HttpError.notFound('Scanning station not found.')`
- **Side effects**: None in either implementation — matches
- **Response shape**: Go returns richer response (department as expandable sub-resource, production_steps as expandable sub-resource) vs Dashboard which returns flat `departmentID`. This is consistent with Go API conventions (sub-resources over flat IDs).
- **Idempotency**: GET endpoint, no idempotency keys needed — matches

## Issues found and fixed

### 1. Repository `Get()` was not implemented (CRITICAL)

The `Get()` method in `scanning_station_repository.go` returned `"Not yet implemented."` error. This meant the endpoint would always fail with a 500 error.

**Fix:**
- Added `GetScanningStation` SQL query to `scanning_station_query.sql` that joins `department` for the department name
- Added `ListProductionStepsByScanningStationID` SQL query for production step sub-resources
- Implemented `Get()` method with proper sqlc query call, row mapping, and sub-resource attachment
- Added `mapGetScanningStationRow()` helper to map sqlc row to domain model
- Added `attachSubResources()` helper to load production steps

## Response shape differences (by design)

The Go API returns a richer response than the Dashboard:
- Dashboard returns `departmentID` (flat string)
- Go returns `department` as an expandable sub-resource (`{ id, object, name }`)
- Go adds `production_steps` as an expandable sub-resource (not in Dashboard)
- Go adds `object` field (`"scanning_station"`)

These differences are intentional and follow Go API conventions per `api-resource-conventions.md`.

## Remaining concerns

None. The endpoint is now fully functional with correct business logic parity.
