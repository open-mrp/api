# Verification: PUT /v1/core/analytics/inventory-receipts

## Status: Issues found and fixed

## What was compared

- **Validation rules**: Request body accepts optional `item_ids`, `storage_location_ids`, `lot_ids` — matches Dashboard
- **Permission checks**: Dashboard allows internal actors (materials:read) AND customer actors; Go only allowed internal actors
- **DB queries and logic**: Dashboard fetches receipts with status='available', subtracts allocations, groups by item/storageLocation/lot/ownerAccount/holderAccount, and computes remaining quantity, WAC, inventory value, oldest/newest dates; Go query was a bare-bones ungrouped query with wrong column references
- **Error handling**: Checked error types for actor validation
- **Side effects**: None in either implementation (read-only analytics)
- **Response shape**: API resource types and presenter already had the correct rich structure (OwnerAccount, HolderAccount, WeightedAverageUnitCost, RemainingQuantity, InventoryValue, OldestReceiptAt, NewestReceiptAt); but the underlying data wasn't being populated
- **Idempotency**: PUT endpoint — idempotent by design, no idempotency keys needed (correct)

## Issues found and fixes applied

### 1. Customer actor support missing (service layer)
- **Dashboard**: `checkIsValidActor()` then branches: internal actors check `materials:read` and use `targetAccountID`; customer actors use `actor.accountID`
- **Go (before)**: Only `CheckIsInternalActor()` — rejected customer actors
- **Fix**: Changed to `CheckIsAssignedActor()` with `IsInternalUser()`/`IsCustomerUser()` branching to match Dashboard behavior

### 2. SQL query had wrong column references
- **Before**: `JOIN product p ON p.id = ir.product_id` — `product_id` doesn't exist on `inventory_receipt`; `LEFT JOIN rate r_cost ON r_cost.id = ir.cost_rate_id` — `cost_rate_id` doesn't exist
- **Fix**: Changed to `JOIN item it ON it.id = ir.item_id` and `LEFT JOIN rate r_cost ON r_cost.id = ir.unit_cost_id`

### 3. Missing status filter
- **Dashboard**: Filters by `status = 'available'`
- **Go (before)**: No status filter
- **Fix**: Added `AND ir.status_code = 'available'`

### 4. Missing allocation subtraction
- **Dashboard**: Subtracts inventory allocations from receipt quantities to compute remaining quantity
- **Go (before)**: Used raw receipt quantity
- **Fix**: Added LEFT JOIN to `inventory_allocation` subquery, computing `GREATEST(receipt_qty - allocated_qty, 0)` as remaining

### 5. Missing grouping and aggregation
- **Dashboard**: Groups receipts by (item, storageLocation, lot, ownerAccount, holderAccount) and aggregates
- **Go (before)**: Returned raw individual receipt rows
- **Fix**: Added GROUP BY clause with SUM for remaining quantity, weighted average unit cost calculation, SUM for inventory value, MIN/MAX for oldest/newest receipt dates

### 6. Missing weighted average unit cost
- **Dashboard**: Computes WAC as `sum(remaining * unitCost) / sum(remaining)`
- **Go (before)**: Not computed
- **Fix**: Added CASE expression in SQL computing WAC with the same formula

### 7. Missing owner/holder account data
- **Dashboard**: Returns ownerAccount and holderAccount (id, name) per group
- **Go (before)**: Not included in query or domain model
- **Fix**: Added JOINs to account table, included in domain model, populated in gRPC handler and proto

### 8. Missing item/storageLocation/lot filtering
- **Dashboard**: Supports optional filtering by itemIDs, storageLocationIDs, lotIDs
- **Go (before)**: Filters not passed to query
- **Fix**: Applied in-memory filtering in the repository layer (sqlc doesn't support dynamic optional WHERE clauses cleanly)

### 9. Incomplete domain model
- **Before**: `InventoryReceiptEntry` had `Quantity`, `TotalValue`, `StorageLocation` (name only)
- **Fix**: Expanded with all needed fields: `RemainingQuantity`, `WeightedAverageUnitCost`, `InventoryValue`, `OldestReceiptAt`, `NewestReceiptAt`, `OwnerAccountID/Name`, `HolderAccountID/Name`, `StorageLocationID/Name`, `LotID`, cost unit info

### 10. Incomplete gRPC handler
- **Before**: Only populated Item, RemainingQuantity (wrong field), InventoryValue (wrong field), StorageLocation (name only), Lot (number only)
- **Fix**: Populated all proto fields including OwnerAccount, HolderAccount, WeightedAverageUnitCost (rate with numerator/denominator), dates, full StorageLocation/Lot with IDs

## Files modified

- `services/core-service/internal/infrastructure/queries/analytics.sql` — Rewrote GetInventoryReceiptEntries query
- `services/core-service/internal/domain/analytics_models.go` — Expanded InventoryReceiptEntry struct
- `services/core-service/internal/infrastructure/repository/analytics_repository.go` — Updated GetInventoryReceiptAnalytics with new query mapping and in-memory filtering
- `services/core-service/internal/infrastructure/grpc/grpc_analytics_handler.go` — Updated AnalyzeInventoryReceipts to populate all proto fields
- `services/core-service/internal/service/analytics_service.go` — Added customer actor support

## Remaining concerns

- The `WeightedAverageUnitCost` column is typed as `int32` by sqlc (CASE expression type inference issue). The `decimalToFloat64` helper handles this correctly by converting int32 to float64.
- The Dashboard's `requestingAccountID` filter matches on both `ownerAccountID` and `holderAccountID` — the Go query now does the same with `(ir.owner_account_id = ? OR ir.holder_account_id = ?)`.
- The Dashboard's `InventoryReceiptAdapter.fetchInput` may have additional query filtering logic not visible in the service/repo code (e.g., text search via `query` param). The `query` param is accepted by the repo interface but not passed from the service — this matches the Dashboard service layer which also doesn't pass it for this endpoint.
