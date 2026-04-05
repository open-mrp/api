# POST /v1/core/purchase-orders/{id}/lines

## Result: Issues found and fixed

## What was compared

- **Validation rules**: Required fields, formats, constraints
- **Permission checks**: Actor type, permission domain, action
- **DB queries and logic**: Line creation, quantity/rate records, line item numbering
- **Side effects**: Receiving order line creation, supplier material linking
- **Error handling**: Error types, messages, silent error swallowing
- **Response shape**: Field names, types, nested resources
- **Idempotency**: POST uses idempotency keys with recovery points

## Issues found and fixed

### 1. Permission action mismatch (FIXED)

- **Dashboard**: Uses `checkHasPermission(PermissionDomains.purchaseOrders, 'update')` for creating PO lines
- **Go (before fix)**: Used `types.ActionCreate`
- **Go (after fix)**: Changed to `types.ActionUpdate` to match Dashboard behavior

The Dashboard treats adding a line to an existing purchase order as an "update" operation on the purchase order, not a "create". This is consistent across all PO line operations in the Dashboard (create, update, delete all use `'update'` permission). Fixed in `purchase_order_line_service.go` line 82.

## Parity confirmed (no issues)

- **Actor check**: Both require internal actor (customer/supplier actors rejected)
- **Account validation**: Both validate the PO belongs to the requesting account
- **Line creation**: Go creates quantity, rate (unit price), optional rate (unit cost), and the line record with auto-incrementing line item number — matches Dashboard behavior
- **Receiving order line creation**: Go creates a receiving order line with the full ordered quantity when a receiving order exists for the PO. For a brand new line this is equivalent to the Dashboard's `calculateQuantityYetToBeReceived` (which would return the full quantity since nothing has been received yet)
- **Supplier material linking**: Both implementations check for existing link, look up material by item ID, and silently ignore creation errors (race condition protection). Go only runs this when `item_id` is provided, matching Dashboard behavior
- **Idempotency**: Go correctly uses idempotency keys with recovery points for this POST endpoint
- **Response shape**: Returns `PurchaseOrderLineDetail` with nested `Item`, `Quantity`, and `Rate` sub-resources
- **HTTP status**: Both return 201 Created

## Remaining concerns

- The Go implementation has additional fields not in Dashboard: `unit_cost` (optional rate), `product_id`, and `edi_line_item_id`. These are additive and don't break parity.
- The Go `CreatePurchaseOrderLineParams` field is named `SalesOrderID` internally (since PO lines share the `sales_order_line` table), but this is correctly mapped from `PurchaseOrderId` at the gRPC handler layer.
