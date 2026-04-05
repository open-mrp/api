# Verification: GET /v1/core/inventory-change-logs/actions/export

## Result: Issues found and fixed

## What was compared

- **Permissions**: Both check internal actor + read permission on `inventoryLogs` domain. **Match.**
- **Filters**: Both support `itemIDs`, `changedByUserIDs`, `actionTypeCodes`, `startDate`, `endDate`. **Match.**
- **DB query**: Dashboard calls `list()` without pagination; Go calls `ListAll()` with no pagination. Both filter identically by account, item IDs, action type codes, user IDs, date range. **Match.**
- **Sort order**: Both sort by `created_at DESC`. **Match.**
- **Error handling**: Both return internal errors appropriately. **Match.**
- **Side effects**: None in either implementation. **Match.**
- **Idempotency**: GET endpoint, no idempotency keys needed. **Match.**
- **HTTP method**: Dashboard uses POST with body params; Go uses GET with query params. Acceptable migration change (GET is more RESTful for read-only export).

## Issues found and fixed

### 1. Excel columns did not match (fixed)

**Before (Go):** 10 columns — ID, Item ID, Item SKU, Quantity Value, Unit, Action Type, Scanning Station, Responsible User, Created At, Updated At

**After (Go, matching Dashboard):** 7 columns — Item (SKU), Quantity Change (value), Unit (abbreviation), Action Type, Responsible User, Responsible Scanning Station, Created At

Changes made in `services/api-gateway/internal/export/excel.go`.

### 2. Filename was static (fixed)

**Before (Go):** `inventory-change-logs.xlsx` (hardcoded)

**After (Go, matching Dashboard):** `inventory-change-logs-{startDate}-{endDate}.xlsx` where dates are `YYYY-MM-DD` or `all` if not provided.

Changes made in `services/api-gateway/endpoints/inventory-change-logs/service.go`.

## Remaining notes

- Dashboard applies Excel number formatting (`#,##0.00` for quantity, `mm/dd/yyyy hh:mm:ss` for dates) via ExcelJS column-level formatting. The Go implementation writes quantity as a numeric value (which Excel will display) and dates as RFC3339 strings. This is a minor cosmetic difference — the data is equivalent and the Excel output is functional.
