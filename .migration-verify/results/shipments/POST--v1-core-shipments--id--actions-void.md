# POST /v1/core/shipments/{id}/actions/void — Verification Results

## Status: Issues Found and Fixed (with remaining TODOs)

## What Was Compared

- **Validation rules**: Shipment status check (must be "shipped")
- **Permission checks**: Internal actor + `shipments:update` permission
- **DB queries and logic**: Shipment void, shipping case void, invoice deletion, order unfulfillment
- **Error handling**: Error types and messages
- **Side effects**: Invoice deletion, inventory allocation reversal, order status reset, Shippo refunds, S3 label deletion
- **Response shape**: Returns updated shipment
- **Idempotency**: Uses idempotency keys with recovery points

## Issues Found and Fixed

### 1. SQL: `MarkShipmentVoided` missing `master_tracking_number = NULL`
**Dashboard**: Clears `masterTrackingNumber` when voiding a shipment.
**Go (before)**: Did not clear `master_tracking_number`.
**Fix**: Added `master_tracking_number = NULL` to the `MarkShipmentVoided` SQL query.

### 2. SQL: Missing `VoidShippingCasesByShipment` query
**Dashboard**: Clears `shippedAt`, `trackingNumber`, `shippoTransactionId`, `shippingLabelUrl`, and sets `freightAmount` to 0 on all shipping cases.
**Go (before)**: No SQL query existed for this; the repo method was a no-op stub.
**Fix**: Added `VoidShippingCasesByShipment` SQL query that clears all these fields via a JOIN UPDATE on `shipping_case` and `quantity`.

### 3. Service: `MarkVoided` call missing `accountID` parameter
**Interface**: `MarkVoided(ctx, accountID, shipmentID)`
**Service call (before)**: `MarkVoided(txCtx, params.ShipmentID)` — missing accountID.
**Fix**: Updated to `MarkVoided(txCtx, params.AccountID, params.ShipmentID)`.

### 4. Service: Missing invoice deletion
**Dashboard**: Finds invoice by shipment, reverses allocations, deletes invoice lines, then deletes invoice.
**Go (before)**: Only set `invoice_id = NULL` on shipment — did not delete the invoice or its lines.
**Fix**: Added `FindInvoiceIDByShipment` SQL query + repo method. Added `DeleteInvoiceLinesByInvoice` and `DeleteInvoice` SQL queries + repo methods. Updated service to find and delete the invoice within the transaction.

### 5. Service: Missing order unfulfillment
**Dashboard**: Resets the sales order to `issued` status, clears `completedAt` and `firstShipAt`.
**Go (before)**: Did not touch the sales order at all.
**Fix**: Added `MarkSalesOrderUnfulfilled` SQL query + `MarkUnfulfilled` repo method. Updated service to call it within the transaction.

### 6. Repository stubs: `MarkVoided` and `VoidByShipment` not implemented
**Before**: Both returned `nil` without calling sqlc.
**Fix**: Implemented `MarkVoided` to call `queries.MarkShipmentVoided` and `VoidByShipment` to call `queries.VoidShippingCasesByShipment`.

## Remaining Concerns

### 1. Inventory allocation reversal (TODO)
The Dashboard's `reverseAllocationsByInvoice` performs a complex LIFO reversal of inventory allocations per invoice line. This involves:
- Finding allocations for each order+item (newest first)
- Deleting or reducing allocations
- Updating receipt/issue statuses
- Creating reserved issues for unallocated quantities
- Creating change log entries
- Reallocating remaining open issues per item using FIFO

This is ~200 lines of business logic that needs a dedicated mediator or repository method. A TODO comment has been added to the service code.

### 2. Shippo refund and S3 label deletion (pre-existing TODO)
The Dashboard refunds Shippo transactions (in non-sandbox mode) and deletes labels from S3. The Go code has a pre-existing TODO for this as Phase 2 (foreign mutation). This is not a regression from this review — the Ship endpoint also has a corresponding TODO for creating labels.

### 3. sqlc regeneration required
New SQL queries were added. Run `make sqlc core` to regenerate the Go bindings, then `make mocks core` to regenerate mock implementations for the updated interfaces.

## Files Modified

- `services/core-service/internal/infrastructure/queries/shipment.sql` — Added `master_tracking_number = NULL` to void query, added `FindInvoiceIDByShipment` query
- `services/core-service/internal/infrastructure/queries/shipping_case.sql` — Added `VoidShippingCasesByShipment`, `MarkShippingCasesShippedByShipment`, `ListShippingCasesByShipment` queries
- `services/core-service/internal/infrastructure/queries/invoice.sql` — Added `DeleteInvoiceLinesByInvoice` and `DeleteInvoice` queries
- `services/core-service/internal/infrastructure/queries/sales_order.sql` — Added `MarkSalesOrderUnfulfilled` query
- `services/core-service/internal/domain/repositories.go` — Added `FindInvoiceIDByShipment` to ShipmentRepo, `DeleteLinesByInvoice`/`Delete` to InvoiceRepo, `MarkUnfulfilled` to SalesOrderRepo
- `services/core-service/internal/infrastructure/repository/shipment_repository.go` — Implemented `MarkVoided` and `FindInvoiceIDByShipment`
- `services/core-service/internal/infrastructure/repository/shipping_case_repository.go` — Implemented `VoidByShipment`
- `services/core-service/internal/infrastructure/repository/invoice_repo.go` — Added `DeleteLinesByInvoice` and `Delete` methods
- `services/core-service/internal/infrastructure/repository/sales_order_repo.go` — Added `MarkUnfulfilled` method
- `services/core-service/internal/service/shipment_service.go` — Updated `VoidShipment` with invoice deletion, order unfulfillment, and correct parameter passing
