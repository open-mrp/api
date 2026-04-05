# DELETE /v1/core/sales-orders/{id} — Migration Verification

## Result: Issues found and fixed

## What was compared

- **Permission checks**: Internal actor + `salesOrders:delete` permission — matched
- **Validation**: Order must exist, must not be fulfilled (`completedAt != nil`) — matched (Dashboard checks both `status != fulfilled` AND `completedAt: null`, but these are always set together so checking `completedAt` alone is equivalent)
- **Type code check**: Dashboard checks `typeCode: salesOrder` to exclude purchase orders. Go uses a separate `sales_order` table, so this is implicitly handled — no issue
- **Seller account check**: Dashboard checks `sellerAccountID == ownerAccountID`. Go's `Get` filters by `owner_account_id` which is equivalent since the sales_order table only contains sales orders owned by the account — acceptable
- **Response shape**: Dashboard returns the deleted order (HTTP 200); Go returns HTTP 204 empty — intentional Go API convention
- **Idempotency**: DELETE is idempotent by nature, no idempotency key needed — matched
- **Side effects**: No notifications sent on delete — matched

## Issues found and fixed

### 1. Missing shipment line cascade deletion
Dashboard's order line deletion cascades to `shipment_line` records. Go was not deleting shipment lines or their associated quantities before deleting order lines.

**Fix**: Added `DeleteShipmentLineQuantitiesBySalesOrder` and `DeleteShipmentLinesBySalesOrder` SQL queries and calls in `DeleteCascade`.

### 2. Missing invoice line cascade deletion
Dashboard's order line deletion cascades to `invoice_line` records. Go was not deleting invoice lines or their associated quantities before deleting order lines.

**Fix**: Added `DeleteInvoiceLineQuantitiesBySalesOrder` and `DeleteInvoiceLinesBySalesOrder` SQL queries and calls in `DeleteCascade`.

### 3. Missing reserved inventory release
Dashboard calls `inventoryIssueRepo.releaseMaterialsForProductionRun()` which deletes `inventory_issue` records with status `reserved` for the order. Go was not performing this step.

**Fix**: Added `DeleteReservedInventoryIssuesBySalesOrder` SQL query and call in `DeleteCascade`.

## Files modified

- `services/core-service/internal/infrastructure/queries/sales_order.sql` — Added 5 new SQL queries for cascade deletion
- `services/core-service/internal/infrastructure/repository/sales_order_repo.go` — Added calls to new queries in `DeleteCascade`
- `services/core-service/internal/infrastructure/sqlc/sales_order.sql.go` — Regenerated via `make sqlc core`

## Updated deletion order in `DeleteCascade`

1. Delete quantities by pick lines
2. Delete pick lines
3. Delete pick
4. Delete email contacts
5. Delete payment intents
6. **Delete shipment line quantities** (NEW)
7. **Delete shipment lines** (NEW)
8. **Delete invoice line quantities** (NEW)
9. **Delete invoice lines** (NEW)
10. **Release reserved inventory issues** (NEW)
11. Delete sales order lines
12. Delete sales order

## Remaining concerns

- Pre-existing build errors on branch (`NoteFirstShipAt` missing, `MarkShipped` signature mismatch) are unrelated to this endpoint
