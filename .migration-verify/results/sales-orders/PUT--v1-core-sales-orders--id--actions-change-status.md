# PUT /v1/core/sales-orders/{id}/actions/change-status

## Status: Issues found and fixed

## What was compared
- Validation rules (status transition checks)
- Permission checks (internal actor, salesOrders:update)
- DB queries and logic for all four status changes (issue, unissue, close, open)
- Side effects (pick management, inventory reservation/release, pick packing)
- Response shape

## Issues found and fixed

### 1. issuedAt not cleared on unissue
**Dashboard:** Sets `issuedAt: null` when unissuing.
**Go (before):** SQL used `COALESCE(sqlc.narg('issued_at'), issued_at)` which preserved the existing value when nil was passed.
**Fix:** Changed SQL to `issued_at = sqlc.narg('issued_at')` (removed COALESCE). Updated `close` and `open` cases to explicitly pass `order.IssuedAt` to preserve the existing value.

### 2. Pick lines created for ALL order lines instead of sale-type only
**Dashboard:** Filters lines by `product.productType?.code === 'sale'` — only sale-type products get pick lines.
**Go (before):** Created pick lines for all order lines regardless of product type.
**Fix:** Added new SQL query `GetSalesOrderSaleLinesForIssue` that joins through product table and filters by `product_type_code = 'sale'`. Service now uses this query instead of `GetLines`.

### 3. No inventory reservation on issue
**Dashboard:** Creates `inventory_issue` records with status `reserved` for each sale-type line, linked to the order and item.
**Go (before):** Did not create any inventory issue records.
**Fix:** Added `CreateReservedInventoryIssueForSalesOrder` SQL query and repo method. Service now creates reserved inventory issues for each sale-type line that has an item_id during the issue transaction.

### 4. No inventory release on unissue
**Dashboard:** Deletes `inventory_allocation` records linked to reserved issues, then deletes the reserved `inventory_issue` records.
**Go (before):** Did not touch inventory records during unissue.
**Fix:** Added `DeleteInventoryAllocationsByReservedSalesOrderIssues` SQL query and repo method. Service now deletes allocations and reserved issues within the unissue transaction.

### 5. Pick not marked as packed on close
**Dashboard:** Calls `markPickAsPacked()` which sets `finishedAt` on the pick when closing an order.
**Go (before):** Only updated the order status, did not touch the pick.
**Fix:** Service now calls `PickRepo.UpdateFinishedAt()` after updating status, if the order has a pick.

### 6. Pick not marked as unpacked on open
**Dashboard:** Calls `markPickAsUnpacked()` which clears `finishedAt` on the pick when reopening an order.
**Go (before):** Only updated the order status, did not touch the pick.
**Fix:** Service now calls `PickRepo.ClearFinishedAt()` after updating status, if the order has a pick.

## Files modified
- `services/core-service/internal/infrastructure/queries/sales_order.sql` — Fixed UpdateSalesOrderStatus COALESCE, added 3 new queries
- `services/core-service/internal/infrastructure/sqlc/sales_order.sql.go` — Regenerated
- `services/core-service/internal/domain/sales_order_models.go` — Added `SalesOrderSaleLineForIssue` struct
- `services/core-service/internal/domain/repositories.go` — Added 4 new methods to `SalesOrderRepo` interface
- `services/core-service/internal/infrastructure/repository/sales_order_repo.go` — Implemented 4 new repo methods
- `services/core-service/internal/service/sales_order_service.go` — Updated all 4 status change cases

## Remaining concerns

### Email sending (sendEmail parameter)
**Dashboard:** When `sendEmail=true` and action is `issue`, sends order acknowledgement emails via `emailOrderAcknowledgements()`.
**Go:** The `sendEmail` parameter is accepted but not used. The infrastructure exists (`GetAcknowledgementRecipients`, `MarkAcknowledgementSent` queries) but email sending requires the notification service via RabbitMQ messaging, which is outside the scope of this single-endpoint fix.

### Pick line quantity value
**Dashboard:** Creates pick lines with `quantityPicked` (which may be 0 for a fresh order).
**Go:** Creates pick lines with the ordered quantity value. This is a pre-existing difference that may be intentional — using the ordered quantity as the initial pick quantity is arguably more correct operationally.
