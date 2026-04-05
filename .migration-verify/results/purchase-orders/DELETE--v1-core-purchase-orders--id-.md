# DELETE /v1/core/purchase-orders/{id}

## Result: PARITY CONFIRMED

No code changes needed.

## What was compared

| Aspect | Dashboard | Go | Match |
|--------|-----------|-----|-------|
| Permission: internal actor | `checkIsInternalActor` | `CheckIsInternalActor` | Yes |
| Permission: domain/action | `purchaseOrders` / `delete` | `PurchaseOrders` / `Delete` | Yes |
| Existence check | `find({ id, ownerAccountID })` → 404 | `repo.Get(accountID, id)` → 404 | Yes |
| Cascade: receiving order lines | `receivingOrderLine.deleteMany` (via order) | `DeleteReceivingOrderLinesByOrderID` (JOIN) | Yes |
| Cascade: receiving orders | `receivingOrder.deleteMany` (via order) | `DeleteReceivingOrderByOrderID` | Yes |
| Cascade: order lines | `orderLine.deleteMany` (via order) | `DeletePurchaseOrderLinesBySalesOrder` | Yes |
| Cascade: email contacts | `orderEmailContact.deleteMany` | `DeleteOrderEmailContactsByOrder` | Yes |
| Cascade: order itself | `order.delete` | `DeletePurchaseOrder` (by id + account) | Yes |
| Transaction wrapping | Prisma `$transaction` | `withTx` | Yes |

## Minor differences (acceptable)

1. **Response code**: Dashboard returns HTTP 200 with the deleted order object. Go returns HTTP 204 with empty body. This follows the Go API convention for DELETE endpoints and is an intentional design choice.

2. **CompletedAt guard**: The Dashboard repository has a `findFirst` that checks `status != fulfilled` and `completedAt == null`, but the result is never used to prevent deletion — it's a no-op. Go explicitly checks `CompletedAt != nil` and returns a validation error. This is a **stricter guard** than the Dashboard effectively enforces, representing the likely intended behavior.

## No remaining concerns

The cascading delete SQL queries correctly mirror the Dashboard's Prisma operations. All permission checks, existence validation, and transaction safety are properly implemented.
