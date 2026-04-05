# DELETE /v1/core/purchase-orders/{id}/lines/{lineId}

## Status: Issues found and fixed

## What was compared

- **Permission checks**: Actor type, permission domain, action
- **Validation**: Account ownership, line-to-order relationship
- **DB queries and cascade logic**: Receiving order line deletion, quantity cleanup, order line deletion
- **Error handling**: Error types and messages
- **Side effects**: Cascade deletes (receiving order lines, quantities)
- **Response shape**: Status code, response body
- **Idempotency**: DELETE is idempotent by design (no idempotency keys needed)

## Issues found and fixed

### 1. Permission action mismatch (FIXED)
- **Dashboard**: Uses `checkHasPermission(purchaseOrders, 'update')` — deleting a line is considered an update to the purchase order
- **Go (before)**: Used `types.ActionDelete`
- **Go (after)**: Changed to `types.ActionUpdate` to match dashboard behavior
- **Note**: The Create and Update methods in the same Go file already correctly use `ActionUpdate`

### 2. Missing account ownership validation (FIXED)
- **Dashboard**: Calls `purchaseOrderRepo.checkIsInAccount({ id: purchaseOrderID, accountID: ownerAccountID })` before checking line membership
- **Go (before)**: Only checked `IsInOrder` (line belongs to order) without verifying the order belongs to the caller's account — potential cross-account security issue
- **Go (after)**: Added `orderRepo.Get(ctx, params.AccountID, params.SalesOrderID)` before the `IsInOrder` check, matching the pattern used in `CreatePurchaseOrderLine`

## Items verified with no issues

- **Actor type check**: Both require internal actor (`checkIsInternalActor`)
- **Target account check**: Both require a target account ID
- **Line-to-order validation**: Both verify the line belongs to the specified purchase order
- **Cascade deletes**: Both delete associated receiving order lines before deleting the PO line itself
- **Transaction wrapping**: Both wrap the delete in a transaction
- **No idempotency keys**: Correct for DELETE endpoints

## Notes

- **Response shape difference**: Dashboard returns the deleted line object with HTTP 200; Go returns empty with HTTP 204 (NoContent). This is an intentional Go API convention (DELETE returns no content), not a parity issue.
- **Redundant receiving order line deletion**: The Go service calls `txReceivingRepo.DeleteLinesByOrderLineID` and then `txLineRepo.DeleteCascade` which also deletes receiving order lines. The second delete is a no-op since lines are already gone. Not a bug, just slightly wasteful — left as-is since it doesn't affect correctness.
- **Quantity cleanup**: Go's `DeleteCascade` also deletes quantity records associated with receiving order lines (`DeleteQuantitiesByReceivingOrderLinesForLine`). The dashboard relies on Prisma's cascade behavior for this. Both achieve the same result.
