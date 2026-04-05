# POST /v1/core/sales-orders/actions/bulk-delete

## Status: Issues found and fixed

## What was compared

| Aspect | Dashboard | Go | Match |
|--------|-----------|-----|-------|
| **Permission checks** | `checkIsInternalActor` + `checkHasPermission(salesOrders, delete)` | `CheckIsInternalActor()` + `CheckHasPermission(SalesOrders, Delete)` + `CheckTargetAccountSet()` | Yes |
| **Validation: order exists** | Bulk query with `id: { in: ids }`, `ownerAccountID` | Per-order `Get()` with `owner_account_id` filter | Yes (different approach, same result) |
| **Validation: seller = owner** | `sellerAccountID: ownerAccountID` | SQL `GetSalesOrder` has `seller_account_id = owner_account_id` | Yes |
| **Validation: is sales order** | `typeCode: OrderTypeCodes.salesOrder` | Separate `sales_order` table (inherently only sales orders) | Yes |
| **Validation: not fulfilled** | `NOT status = fulfilled` AND `completedAt = null` | Only checked `CompletedAt != nil` | **Fixed** |
| **Error on invalid** | "Some of these orders cannot be found." (generic) | "Cannot delete a fulfilled sales order: {id}" (specific per order) | Acceptable — Go gives better error UX |
| **Cascade deletes** | picks, pickLines, orderLines, orders | quantities (pick lines), pickLines, picks, emailContacts, paymentIntents, orderLines, order | Go is more thorough (deletes related quantities, email contacts, payment intents) |
| **Transaction** | Prisma `$transaction` wrapping all deletes | `withTx` wrapping all deletes | Yes |
| **Response** | `{}` with 200 OK | Empty with 204 No Content | Acceptable (REST convention improvement) |
| **Idempotency** | N/A (Dashboard doesn't use idempotency keys) | Not used (this is a POST action endpoint without idempotency) | Acceptable — bulk delete is inherently idempotent via Get checks |

## Issues found and fixed

### 1. Missing fulfilled status check (fixed)

**File:** `services/core-service/internal/service/sales_order_service.go:618`

The Dashboard checks both `status !== fulfilled` AND `completedAt === null` before allowing deletion. The Go code only checked `CompletedAt != nil`. While these are set atomically in `UpdateSalesOrderStatus`, added the status code check for strict parity:

```go
// Before:
if order.CompletedAt != nil {

// After:
if order.CompletedAt != nil || order.SalesOrderStatusCode == string(constants.SalesOrderStatusCodeFulfilled) {
```

## Remaining concerns

- None. The Go implementation is functionally equivalent to the Dashboard with the fix applied. The Go version is actually more thorough in cascade deletion (cleaning up quantities, email contacts, and payment intents that the Dashboard doesn't explicitly delete).
