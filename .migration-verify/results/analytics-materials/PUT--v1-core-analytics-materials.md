# PUT /v1/core/analytics/materials — Migration Verification

## Status: Issues Found and Fixed

## What Was Compared

- **Validation rules**: Request body accepts optional `salesOrderIDs` and `supplierIDs` arrays — matches dashboard
- **Permission checks**: Internal actor + `materials:read` permission — matches dashboard (`PermissionDomains.materials`, `'read'`)
- **Target account**: Required via `Augno-Account-ID` header — matches dashboard (`this.identity.targetAccountID`)
- **DB queries and logic**: Inventory calculations (on-hand, reserved, open, available-to-promise)
- **Response shape**: MaterialAnalyticsEntry with BaseQuantity, UnitGroup, supplier info
- **Side effects**: None (read-only endpoint) — matches dashboard
- **Idempotency**: PUT method, idempotent by design — correct

## Issues Found and Fixed

### 1. SQL Query Returned Hardcoded Zeros (Critical)
**Before**: `GetMaterialAnalyticsEntries` returned `0 AS quantity_required, 0 AS quantity_available, 0 AS quantity_short` — no actual inventory data.
**After**: Replaced with 6 new SQL queries:
- `GetMaterialsWithDetails` — fetches materials with item details, order point, lead time, and unit group
- `GetMaterialOnHandByItem` — computes on-hand = receipt qty - allocated qty for available receipts
- `GetMaterialReservedByItem` — computes reserved issues remaining qty
- `GetMaterialOpenByItem` — computes open (demand) issues remaining qty
- `GetMaterialUnitGroupUnits` — fetches all units in each material's unit group
- `GetMaterialSupplierInfo` — fetches supplier names and part numbers when supplier IDs provided

### 2. Wrong Field Mappings in gRPC Handler (Critical)
**Before**: `ItemId: e.MaterialID` (should be item ID), `Sku: e.MaterialName` (should be item SKU)
**After**: Corrected to `ItemId: e.ItemID`, `Sku: e.Sku`

### 3. Missing Response Fields (Critical)
**Before**: Missing `Description`, `OrderPoint`, `LeadTime`, `UnitGroup`, `SupplierPartNumbers`
**After**: All fields now populated matching the dashboard's `MaterialAnalytics` response shape

### 4. Unused Request Parameters (Moderate)
**Before**: `supplierIDs` parameter was accepted but never used in the SQL query
**After**: `supplierIDs` is now used to query `supplier_material` table for supplier names and part numbers (matching dashboard's `findPartNumberAndNameFromItemIDAndSupplierIDs`)

### 5. Missing Inventory Calculations (Critical)
**Before**: No inventory ledger calculations at all
**After**: Repository now computes:
- On-hand = remaining receipt quantities (receipt qty - allocations) for available receipts
- Reserved = remaining reserved issue quantities
- Open (demand) = remaining open issue quantities
- Available to promise = on-hand - reserved - open
- Normalization to order point unit (matching dashboard's `BaseQuantityUtils.updateUnit`)

### 6. Domain Model Too Simple (Critical)
**Before**: `MaterialAnalyticsEntry` had flat fields (`MaterialName`, `SupplierName`, `QuantityRequired`, `QuantityAvailable`, `QuantityShort`, `Unit`)
**After**: Rich domain types with `MaterialBaseQuantity`, `MaterialUnitGroup`, `MaterialUnitGroupUnit` matching the dashboard's response structure

## Notes

- The dashboard's `salesOrderIDs` parameter is accepted but aliased as `_salesOrderIDs` (unused in dashboard implementation). The Go implementation also accepts but doesn't use it, maintaining parity.
- The presenter and proto types were already correctly structured for the rich response — only the backend (SQL, repository, gRPC handler) needed fixes.
- Pre-existing compilation errors in `sales_order_repo.go`, `shipment_repository.go`, and `shipment_service.go` are unrelated to this endpoint.
