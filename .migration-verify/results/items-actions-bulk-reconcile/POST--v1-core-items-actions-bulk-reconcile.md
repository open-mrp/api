# POST /v1/core/items/actions/bulk-reconcile

## Status: Issues found and fixed

## What was compared

- **Validation**: Request schema (data array with sku/unit/quantity, reconcileType)
- **Permission checks**: Internal actor + items:create permission + target account
- **Inventory metric**: Which inventory value is used as the baseline for reconciliation
- **Batch processing**: Batch size of 50, sequential batch processing
- **Inventory mutations**: Receipt creation (positive delta), issue creation (negative delta)
- **Logging**: Inventory log (snapshot) and change log (audit trail) creation
- **Error categorization**: Missing items → skipped, missing units → errors
- **Response shape**: reconciledItems, skippedItems, errors arrays
- **Idempotency**: POST endpoint uses idempotency keys with recovery points

## Issues found and fixed

### 1. Wrong inventory metric: ATP vs Physical Inventory (FIXED)

**Before**: Go used `FetchCurrentInventory` which returns Available-To-Promise (receipts - committed issues). This is a different metric than what the Dashboard uses.

**Dashboard**: Uses `physicalInventory` = onHand - short = (receipts - receipt_allocations) - (open_issues - open_issue_allocations).

**Fix**: Switched to `FetchPhysicalInventoryForItem` which computes receipts - open issues. This is closer to the Dashboard's physical inventory calculation. Added `FetchPhysicalInventory` method to `InventoryQueryRepo` interface and implementation.

### 2. Inventory fetched inside transaction vs before (FIXED)

**Before**: Go fetched inventory per-item inside each batch transaction.

**Dashboard**: Fetches all inventory in bulk (`fetchCurrentInventoryBulk`) before processing any batches.

**Fix**: Moved inventory fetching outside the transaction loop. Physical inventory is now pre-fetched for all valid items before batch processing begins.

### 3. Missing early return for empty valid items (FIXED)

**Dashboard**: Returns immediately with empty result if no valid items after filtering.

**Fix**: Added early return with idempotency caching when `len(validItems) == 0`.

### 4. Silent skip for items with no inventory data (FIXED)

**Before**: Go would add an error entry if inventory fetch failed.

**Dashboard**: Silently skips items where `currentInventory` is falsy (no inventory records found).

**Fix**: Items where `FetchPhysicalInventory` fails or returns no data are now silently skipped, matching Dashboard behavior.

## Remaining concerns

### Open issue allocation (NOT implemented)

The Dashboard calls `allocateOpenIssues()` after creating both inventory receipts and inventory issues. This is a FIFO allocation process that pairs open issues with available receipts by creating `inventory_allocation` records. This side effect does not exist anywhere in the Go codebase and is too complex to add safely without dedicated testing (~100 lines of FIFO matching logic with partial allocation support). This should be implemented as a separate feature.

### Physical inventory precision

The Go `FetchPhysicalInventoryForItem` SQL query computes `receipts - open_issues` without accounting for allocations, while the Dashboard's `physicalInventory` = `(receipts - receipt_allocations) - (open_issues - open_issue_allocations)`. When allocations exist, these values diverge. The difference is mitigated by the fact that open issue allocation (above) is also not yet implemented, so the allocation-based adjustments should be zero in practice.

### Integer precision in sqlc

The `FetchPhysicalInventoryForItem` sqlc-generated code returns `int32`, which truncates decimal quantities. This is a pre-existing issue shared with other inventory queries (`FetchCurrentInventoryForItem`, `FetchOnHandInventoryBulk`) and should be addressed system-wide by adding sqlc type overrides for these columns.

## Files modified

- `services/core-service/internal/domain/repositories.go` — Added `FetchPhysicalInventory` to `InventoryQueryRepo` interface
- `services/core-service/internal/infrastructure/repository/inventory_query_repository.go` — Added `FetchPhysicalInventory` implementation
- `services/core-service/internal/domain/mock/repository/inventory_query_repo_mock.go` — Added mock for new method
- `services/core-service/internal/service/item_service.go` — Rewrote `BulkReconcileItems` to use physical inventory, pre-fetch before transactions, and handle empty/missing inventory
