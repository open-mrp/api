# PATCH /v1/core/scanning-stations/{id} — Migration Verification

## Result: Issues found and fixed

## What was compared

- **Validation rules**: Both implementations validate optional fields; Go uses COALESCE in SQL to only update provided fields
- **Permission checks**: Both require internal actor + `scanningStations` domain + `update` action — confirmed matching
- **DB queries and logic**: Dashboard uses Prisma `update()` with `where: { id, accountID }`; Go now uses sqlc `UpdateScanningStation` with same account-scoped WHERE clause
- **Name uniqueness check**: Both check for duplicate name excluding the current ID before updating
- **Error handling**: Both return 404 if scanning station not found, 409 if name conflicts
- **Side effects**: Neither implementation has side effects beyond the update itself
- **Response shape**: Both return the full scanning station object with department and production steps sub-resources
- **Idempotency**: Go implementation correctly uses idempotency keys for the PATCH endpoint (Dashboard does not have this pattern)

## Issues found and fixed

### 1. Repository `Update` method was a stub (FIXED)
**File**: `services/core-service/internal/infrastructure/repository/scanning_station_repository.go`
**Issue**: The `Update` method returned `"Not yet implemented"` error.
**Fix**: Implemented the method following the established pattern (department_repository.go):
- Calls sqlc `UpdateScanningStation` with nullable params
- Checks `RowsAffected()` for 404 handling
- Delegates to `Get()` to return full domain model with sub-resources

### 2. SQL `UpdateScanningStation` query was missing (FIXED)
**File**: `services/core-service/internal/infrastructure/queries/scanning_station_query.sql`
**Issue**: No UPDATE query existed for scanning stations.
**Fix**: Added `UpdateScanningStation :execresult` query using COALESCE pattern for all optional fields (`name`, `notes`, `label_size_code`, `label_type_code`, `material_check_required`), scoped by `id` and `account_id`.

### 3. `ExistsByName` was already implemented
The `CountScanningStationsByName` SQL query and the repository `ExistsByName` method were already in place (likely added in a prior batch).

## Field mapping (Dashboard → Go)

| Dashboard Field | Go Field | DB Column |
|---|---|---|
| `name` | `Name` | `name` |
| `notes` | `Notes` | `notes` |
| `batchLabelTagSize` | `LabelSizeCode` | `label_size_code` |
| `batchLabelType` | `LabelTypeCode` | `label_type_code` |
| `materialCheckRequired` | `MaterialCheckRequired` | `material_check_required` |

## Remaining concerns

- **Nullable field clearing**: The Dashboard (via Prisma) supports explicitly setting `notes`, `batchLabelTagSize`, and `batchLabelType` to `null` (clearing them). The Go implementation uses COALESCE, which means `nil` = "keep existing value". This is consistent with other Go endpoints in this codebase (e.g., department, machine) but means you cannot clear these fields to null via the Go API. This is a known pattern limitation across the codebase, not specific to this endpoint.
