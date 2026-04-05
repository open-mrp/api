# Verification: GET /v1/core/inventory-change-logs

## Result: Issues found and fixed

## What was compared

- **Permission checks**: Actor type (internal only), permission domain, action
- **Query parameters / filters**: itemIDs, actionTypeCodes, changedByUserIDs, startDate, endDate, search query
- **DB queries**: Joins (item, quantity, unit, scanning_station, user), filter logic, ordering
- **Search behavior**: item SKU, responsible user name, scanning station name (LIKE matching)
- **Date filtering**: createdAt >= startDate, createdAt <= endDate
- **Response shape**: Fields, nested sub-resources, expandable fields
- **Error handling**: Not-found, validation errors
- **Side effects**: None expected, none present

## Issues found

### 1. Permission domain mismatch (FIXED)

**Dashboard**: Uses `PermissionDomains.inventoryLogs` which maps to `'inventory_logs'`
**Go (before fix)**: Used `types.PermissionDomainInventoryChangeLogs` which maps to `"inventory_change_logs"`

Both constants exist in Go (`PermissionDomainInventoryLogs` and `PermissionDomainInventoryChangeLogs`), but the Dashboard uses `inventory_logs`. Fixed all three methods in `inventory_change_log_service.go` to use `types.PermissionDomainInventoryLogs`.

## Confirmed parity

- **Actor check**: Both restrict to internal actors only
- **Filters**: All five filter parameters match (itemIDs, actionTypeCodes, changedByUserIDs, startDate, endDate)
- **Search**: Both search across item SKU, user name, scanning station name using LIKE/contains
- **Date range**: Both use `>=` for startDate and `<=` for endDate on `createdAt`
- **Ordering**: Both order by `createdAt DESC` (Go adds `id DESC` as tiebreaker for cursor pagination)
- **Joins**: Both join item, quantity+unit, and LEFT JOIN scanning_station and user
- **Response fields**: id, actionTypeCode, quantity (with value/unit), item (with SKU), responsibleUser, responsibleScanningStation, createdAt, updatedAt — all present
- **Nullable fields**: responsibleUser and responsibleScanningStation are nullable in both implementations
- **Pagination**: Dashboard uses offset-based (take/skip), Go uses cursor-based — expected migration change
- **No idempotency needed**: GET endpoint, correctly has no idempotency key handling
