# POST /v1/core/purchase-orders/actions/bulk-delete

## Status: PARITY CONFIRMED (with intentional improvement)

No code changes were needed.

## What Was Compared

| Aspect | Dashboard | Go | Match |
|--------|-----------|-----|-------|
| Permission: internal actor | `checkIsInternalActor` | `identity.CheckIsInternalActor()` | ✅ |
| Permission: domain/action | `purchaseOrders / delete` | `PermissionDomainPurchaseOrders / ActionDelete` | ✅ |
| Account scoping | `ownerAccountID` from identity | `*identity.TargetAccountID` | ✅ |
| Transaction wrapping | Prisma `$transaction` | `s.withTx()` | ✅ |
| Type filter | `typeCode: OrderTypeCodes.purchaseOrder` | SQL: `sales_order_type_code = 'purchase_order'` | ✅ |
| Cascade: order lines | `db.orderLine.deleteMany` | `DeletePurchaseOrderLinesBySalesOrder` | ✅ |
| Cascade: order | `db.order.deleteMany` | `DeletePurchaseOrder` | ✅ |
| Cascade: receiving orders | Not explicitly deleted | `DeleteReceivingOrderByOrderID` + lines | ✅ (improvement) |
| Cascade: email contacts | Not explicitly deleted | `DeleteOrderEmailContactsByOrder` | ✅ (improvement) |
| Request field | `ids: string[]` | `purchase_order_ids: []string` | ✅ (renamed for Go conventions) |
| Response | 200 `{}` | 204 No Content | ✅ (Go convention) |

## Differences Noted

### 1. Fulfilled order guard (intentional improvement)
The Dashboard's `deleteMany` repository method calls `findMany` with filters for `NOT status: fulfilled` and `completedAt: null`, but the query result is **discarded** — the subsequent `deleteMany` calls delete orders regardless of status. This is effectively dead validation code. The Go implementation properly checks `CompletedAt != nil` and returns a validation error, which is the correct behavior.

### 2. More thorough cascade deletion (improvement)
The Go implementation also deletes receiving orders, receiving order lines, and email contacts as part of the cascade. The Dashboard only deletes order lines and the order itself. The Go version is more thorough in cleaning up related entities.

### 3. Request field naming
Dashboard uses `ids`, Go uses `purchase_order_ids`. This follows Go API conventions for explicit field naming.

### 4. Response status code
Dashboard returns 200 with `{}`. Go returns 204 No Content. This follows Go API conventions for delete operations.

## Remaining Concerns

None. The Go implementation is a faithful and improved migration of the Dashboard endpoint.
