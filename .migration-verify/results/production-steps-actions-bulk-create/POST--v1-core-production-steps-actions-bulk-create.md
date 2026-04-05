# Verification: POST /v1/core/production-steps/actions/bulk-create

## Status: Issues Found and Fixed

The Go implementation was a **stub** — the service method returned `nil, nil`, the gRPC handler was missing, and the request model was missing most fields. A full implementation was written to match the Dashboard's business logic.

## What Was Compared

| Aspect | Dashboard | Go (Before) | Go (After) |
|--------|-----------|-------------|------------|
| **Request: name** | Required string | Required string | Required string |
| **Request: consumptions** | Array of {sku, measure, instructions?} | Missing | Array of {sku, measure, instructions?} |
| **Request: productions** | Array of {sku, measure} (min 1) | Missing | Array of {sku, measure} (min 1) |
| **Request: laborRate** | Required number (positive) | Missing | Required number (positive) |
| **Request: laborTime** | Required number (positive) | Missing | Required number (positive) |
| **Request: laborTimeUnit** | Optional string (default: "hr") | Missing | Optional string (default: "hr") |
| **Request: overheadRate** | Required number (positive) | Missing | Required number (positive) |
| **Request: allowances** | Optional number (default: 0) | Optional float64 | Optional number (default: "0") |
| **Request: levelingFactor** | Optional number (default: 0) | Optional float64 | Optional number (default: "0") |
| **Request: station** | Optional string (resolved by name) | Missing (was scanningStationID) | Optional string (resolved by name) |
| **Permission check** | Internal actor + productionSteps:create | Missing (stub) | Internal actor + productionSteps:create |
| **Upsert behavior** | Creates new / updates existing by name | Missing | Creates new / updates existing by name |
| **Item resolution** | By SKU via ItemRepo.findIDBySku | Missing | By SKU via ItemRepo.FetchItemsBySKU |
| **Station resolution** | By name | Missing | By name via ScanningStationRepo.FindIDByName |
| **Rate creation** | Labor ($/hr), Overhead ($/hr), LaborTime (unit/prodUnit) | Missing | Same pattern |
| **Production flow linking** | Non-fatal LinkFlow after create/update | Missing | Non-fatal LinkFlow after create/update |
| **Error handling** | Row-level skip with reason | Missing | Row-level skip with reason |
| **Idempotency** | N/A (Dashboard doesn't use) | Missing | Uses idempotency keys (Go convention) |
| **Response shape** | {createdItems, updatedItems, skippedItems} | {object, data: [{name, success, error, id}]} | {object, data: [{name, success, error, id, action}]} |

## Issues Found and Fixed

### 1. Service was a stub (Critical)
**Before:** `return nil, nil`
**After:** Full implementation with permission checks, idempotency, batch processing, upsert logic, production flow linking.

### 2. gRPC handler was missing (Critical)
**Before:** No `BulkCreateProductionSteps` handler existed in the gRPC handler file.
**After:** Full handler that maps proto to domain types and delegates to service.

### 3. Request model was missing most fields (Critical)
**Before:** Only `name`, `departmentID`, `scanningStationID`, `levelingFactor`, `allowances`.
**After:** Full Dashboard-compatible input: `name`, `consumptions`, `productions`, `laborRate`, `laborTime`, `laborTimeUnit`, `overheadRate`, `allowances`, `levelingFactor`, `station`.

### 4. No item resolution by SKU (Critical)
**Before:** Not supported.
**After:** Uses `ItemRepo.FetchItemsBySKU()` to batch-resolve all SKUs upfront.

### 5. No scanning station resolution by name (Critical)
**Before:** Accepted station ID directly.
**After:** New SQL query `FindScanningStationIDByName` and repo method to resolve by name.

### 6. No upsert behavior (Critical)
**Before:** No create-or-update logic.
**After:** Checks `FindIDByName` for existing steps; updates existing (deletes old consumptions/productions, recreates) or creates new.

### 7. No rate creation (Critical)
**Before:** Not implemented.
**After:** Creates labor rate ($/hr), overhead rate ($/hr), and labor time (laborTimeUnit/productionUnit) rates.

### 8. No production flow linking (Medium)
**Before:** Not implemented.
**After:** Calls `ProductionFlowMed.LinkFlow()` after each successful create/update (non-fatal).

### 9. Response missing action field (Low)
**Before:** No `action` field.
**After:** Added `action` field ("created", "updated", "skipped") to distinguish outcomes (Dashboard uses separate arrays for this).

## Response Shape Differences

The Dashboard returns three separate arrays (`createdItems`, `updatedItems`, `skippedItems`), while the Go API returns a single `data` array with an `action` field per item. This follows the Go API's existing bulk create convention (see `BulkCreateItems`) and conveys equivalent information.

## New SQL Queries Added

- `FindProductionStepIDByName` — find step ID by name for upsert check
- `FindScanningStationIDByName` — find station ID by name
- `DeleteConsumptionQuantitiesByStepID` — bulk delete consumption quantities
- `DeleteConsumptionsByStepID` — bulk delete consumptions by step
- `DeleteProductionQuantitiesByStepID` — bulk delete production quantities
- `DeleteProductionsByStepID` — bulk delete productions by step
- `UpdateProductionStepFull` — update step fields during bulk update

## Validation Rules (Matching Dashboard)

| Rule | Behavior |
|------|----------|
| Name required | Skip: "Name is required" |
| Productions ≥ 1 | Skip: "No productions found in row" |
| Station name must resolve | Skip: "Invalid station name" |
| Labor time unit in [hr, minute, min, second, sec, day] | Skip: "Invalid labor time unit" |
| Item SKU must exist | Skip: "Missing item for [consumption/production] SKU: {sku}" |
| Create/Update failure | Skip: "Create/Update failed" |

## Remaining Work Required

1. **Run `make proto`** — proto definition was updated; generated `.pb.go` files need regeneration.
2. **Run `make sqlc core`** — new SQL queries were added; generated `.sql.go` files need regeneration.
3. **Run `make mocks core`** — new interface methods were added to `ProductionStepRepo` and `ScanningStationRepo`; mocks need regeneration.
4. **Run `make test`** — verify compilation and existing tests still pass.
