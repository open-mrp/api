# DELETE /v1/core/sales-orders/{id}/lines/{lineId}

## Result: Issues found and fixed

## What was compared

- **Permission checks**: Actor type, permission domain, action
- **Validation**: Line-in-order check, account ownership filtering
- **DB operations**: Cascade deletes (pick lines, shipment lines, invoice lines, order line)
- **Side effects**: No inventory changes, no status updates (matches)
- **Error handling**: 404 for missing line (matches)
- **Response shape**: Dashboard returns 200 with deleted line object; Go returns 204 No Content
- **Idempotency**: DELETE endpoints are idempotent by default (matches)

## Issues found and fixed

### 1. Permission action mismatch (FIXED)

- **Dashboard**: Uses `checkHasPermission(identity, PermissionDomains.salesOrders, 'update')`
- **Go (before)**: Used `identity.CheckHasPermission(types.PermissionDomainSalesOrders, types.ActionDelete)`
- **Fix**: Changed to `types.ActionUpdate` to match Dashboard behavior
- **File**: `services/core-service/internal/service/sales_order_line_service.go` line 323

## Acceptable differences

### Response shape (204 vs 200)

The Dashboard returns HTTP 200 with the deleted order line object. The Go API returns HTTP 204 No Content. This follows Go API conventions for DELETE endpoints and is an intentional design choice, not a business logic gap.

### Extra cascade step in Go

The Go implementation additionally deletes `quantity` records associated with pick lines (`DeleteQuantitiesByPickLinesForLine`) before deleting the pick lines themselves. The Dashboard relies on Prisma's cascade behavior for this. The Go implementation is more explicit and correct — orphaned quantity records would be problematic.

### Account filtering in IsInOrder

The Go `IsLineInOrder` SQL query joins to `sales_order` and filters by `owner_account_id`, which is actually more secure than the Dashboard's `checkIfLineIsInOrder` (which only checks line-to-order membership without account filtering). The Dashboard separately validates account ownership in the `find` call. The Go approach is stricter but equivalent in effect.
