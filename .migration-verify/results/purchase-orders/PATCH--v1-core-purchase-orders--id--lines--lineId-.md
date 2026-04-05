# Migration Verification: PATCH /v1/core/purchase-orders/{id}/lines/{lineId}

## Status: Issues found and fixed

## What was compared

- **Validation rules**: Required fields, formats, constraints
- **Permission checks**: Actor type, permission domain, action
- **DB queries and logic**: Filters, joins, ordering
- **Error handling**: Error types, messages
- **Side effects**: Receiving order line creation, supplier material linking
- **Response shape**: Field names, types, nested resources
- **Idempotency**: Idempotency key usage for PATCH

## Issues found and fixed

### 1. Missing account validation (security fix)

**Dashboard**: Calls `purchaseOrderRepo.checkIsInAccount({ id: purchaseOrderID, accountID })` before any update, ensuring the purchase order belongs to the user's account.

**Go (before fix)**: Only validated line-in-order via `IsInOrder()` SQL which checks `id = ? AND sales_order_id = ?` without account scoping. A user could theoretically update a line on another account's purchase order if they guessed the IDs.

**Fix**: Added `txOrderRepo.Get(txCtx, params.AccountID, params.SalesOrderID)` call before the line validation, matching the create endpoint's pattern and the Dashboard's behavior.

### 2. Missing receiving order line creation for remaining quantity

**Dashboard**: After updating a PO line, if a receiving order exists for the purchase order, calls `ReceivingOrderLineMed.createLineForRemainingQuantity()` which:
1. Calculates quantity yet to be received (ordered - already received/stocked)
2. Checks if there's already an unstocked receiving order line for this PO line
3. Only creates a new receiving order line if remaining > 0 AND no unstocked line exists

**Go (before fix)**: Did not create receiving order lines on update. The create endpoint had this logic, but the update endpoint was missing it entirely.

**Fix**:
- Added SQL query `HasUnstockedReceivingOrderLineForOrderLine` to check for existing unstocked receiving lines
- Added `CreateLineForRemainingQuantity` method on `ReceivingOrderRepo` that encapsulates the full logic (check unstocked, calculate remaining, create if needed)
- Added receiving order line creation logic to `UpdatePurchaseOrderLine` service method

## Files modified

- `services/core-service/internal/infrastructure/queries/receiving_order.sql` — Added `HasUnstockedReceivingOrderLineForOrderLine` query
- `services/core-service/internal/domain/repositories.go` — Added `HasUnstockedLineForOrderLine` and `CreateLineForRemainingQuantity` to `ReceivingOrderRepo` interface
- `services/core-service/internal/infrastructure/repository/receiving_order_repo.go` — Implemented both new methods
- `services/core-service/internal/service/purchase_order_line_service.go` — Added account validation and receiving order line creation to update flow
- `services/core-service/internal/infrastructure/sqlc/receiving_order.sql.go` — Regenerated via `make sqlc core`

## What matched correctly (no changes needed)

- **Permission checks**: Both use internal actor + purchaseOrders.update ✅
- **Supplier material link**: Both create link when item changes ✅
- **Idempotency**: Go properly uses idempotency keys with recovery points ✅
- **Partial updates**: Both use optional/nullable fields with COALESCE ✅
- **Response shape**: Returns full updated PurchaseOrderLine with quantities, rates, item ✅
- **Line-in-order validation**: Both validate line belongs to specified purchase order ✅

## Remaining concerns

- Mock files for `ReceivingOrderRepo` need regeneration (`make mocks core` was triggered but may need verification)
- Pre-existing build errors on this branch (unrelated to this endpoint: `SalesOrderRepo.NoteFirstShipAt` and `ShipmentRepo.MarkShipped` signature mismatches) prevent full compilation check, but the specific files modified compile cleanly
