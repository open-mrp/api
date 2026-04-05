# PATCH /v1/core/sales-orders/{id}/lines/{lineId}

## Result: Issues found and fixed

## What was compared

- **Validation rules**: Required fields, line-in-order check, account ownership
- **Permission checks**: Internal actor, salesOrders:update permission
- **DB queries and logic**: Update queries, COALESCE pattern, quantity/rate updates
- **Error handling**: 404 for missing line, proper error propagation
- **Side effects**: Pick line creation for remaining quantity after update
- **Response shape**: SalesOrderLineDetail with nested sub-resources
- **Idempotency**: PATCH uses idempotency keys with recovery points
- **Rounding**: Monetary value rounding to nearest cent

## Issues found and fixed

### 1. Missing rounding in UpdateSalesOrderLine (fixed)
**Dashboard**: Rounds `unitPrice` and `unitCost` to nearest cent via `RateUtils.roundToNearestCent()` before updating.
**Go (before)**: No rounding in the Update method (Create had it, Update didn't).
**Fix**: Added `roundToNearestCent()` calls for `UnitPriceValue` and `UnitCostValue` in the Update service method.

### 2. Missing account ownership check in IsInOrder (fixed)
**Dashboard**: The Prisma update WHERE clause includes `order: { ownerAccountID }`, ensuring the line belongs to an order owned by the calling account.
**Go (before)**: `IsLineInOrder` SQL only checked `sales_order_line.id` + `sales_order_id` without verifying account ownership on the parent order.
**Fix**: Updated `IsLineInOrder` SQL query to JOIN `sales_order` and check `so.owner_account_id = ?`. Updated domain interface, repository, mock, and both service callers (Update + Delete) to pass `accountID`.

### 3. Missing unit cost creation when none exists (fixed)
**Dashboard**: When updating with `unitCost` data but the rate record doesn't exist yet (line has null `unit_cost_id`), creates a new rate record and links it.
**Go (before)**: Silently skipped the unit cost update when `unit_cost_id` was null.
**Fix**: Added logic in the repository `Update` method to create a new rate record via `CreateOrderLineRate` and link it via new `SetSalesOrderLineUnitCost` SQL query when all three cost fields are provided but no existing cost exists.

### 4. Missing pick line creation side effect in Update (fixed)
**Dashboard**: After updating a line, if the order has an associated pick and the product is a sale type, calls `createPickLineForRemainingQuantity()` to create pick lines for remaining quantity.
**Go (before)**: The Create method had this side effect but Update did not.
**Fix**: Added the same `createPickLineForRemainingQuantity` call to the Update transaction, using `txOrderRepo.GetPickID()` and the existing helper function.

## Files modified

- `services/core-service/internal/infrastructure/queries/sales_order_line.sql` — Added account_id to IsLineInOrder, added SetSalesOrderLineUnitCost query
- `services/core-service/internal/domain/repositories.go` — Updated IsInOrder signature to include accountID
- `services/core-service/internal/infrastructure/repository/sales_order_line_repo.go` — Updated IsInOrder to pass accountID, added unit cost creation logic
- `services/core-service/internal/service/sales_order_line_service.go` — Added rounding, accountID to IsInOrder calls, pick line side effect
- `services/core-service/internal/domain/mock/repository/sales_order_line_repo_mock.go` — Updated mock to match new IsInOrder signature

## Remaining concerns

None — all identified discrepancies have been addressed. The Go implementation now matches the Dashboard behavior for this endpoint.
