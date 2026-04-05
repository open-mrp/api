# GET /v1/core/scanning-stations — Verification Result

## Status: Issues Found and Fixed

## What Was Compared

- **Permission checks**: Both check `isInternalActor` + `read` permission on `scanningStations` domain + target account ID ✅
- **Query parameters**: Dashboard supports `query`, `take`, `skip`; Go supports `q` (search), `cursor`, `limit` (cursor-based pagination) ✅ (expected difference — Go uses cursor-based pagination per project convention)
- **Search filtering**: Dashboard searches by scanning station name OR department name; Go now also searches by both ✅
- **DB queries**: Dashboard runs `findMany` + `count` with account scoping; Go runs cursor-paginated queries with account scoping ✅
- **Response shape**: Dashboard returns `{ items, count }`; Go returns `{ data, page_info }` with cursor pagination ✅ (expected difference)
- **Sub-resources**: Dashboard returns `departmentID` as a flat field; Go returns `department` as a sub-resource object with `id`, `object`, `name` ✅ (correct per Go API conventions)
- **Production steps**: Go includes production steps as a sub-resource (expandable list) — Dashboard does not include them in list response, but Go's `ScanningStationFull` resource includes them ✅ (enhancement, not a regression)
- **Sorting**: Dashboard returns results in DB default order; Go returns results ordered by `created_at DESC, id DESC` ✅
- **Error handling**: Both return appropriate errors for missing identity, insufficient permissions ✅
- **Idempotency**: Not applicable for GET endpoint ✅

## Issues Found and Fixed

### 1. Repository List method was a stub (CRITICAL)
The `List` method in `scanning_station_repository.go` returned an empty slice with no actual database query. This was marked as "pending sqlc generation for list queries."

**Fix:**
- Added `ListScanningStationsForward` and `ListScanningStationsBackward` SQL queries to `scanning_station_query.sql`
  - Joins `department` table to get `department_name`
  - Supports search by scanning station name OR department name (matching Dashboard behavior)
  - Uses cursor-based pagination with `created_at DESC, id DESC` ordering
- Ran `make sqlc core` to generate Go code
- Implemented full `List` method in `scanning_station_repository.go` following the department repository pattern:
  - Cursor decoding and bidirectional pagination
  - Row mapping for both forward and backward queries
  - Sub-resource attachment (production steps)
  - Page info generation via `pagination.BuildPageString`

## No Remaining Concerns

The endpoint now has full business-logic parity with the Dashboard implementation. The differences in response shape (cursor pagination vs offset pagination, sub-resource objects vs flat IDs) are expected and consistent with the Go API's conventions.
