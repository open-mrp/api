# PATCH /v1/core/purchase-orders/{id} — Verification Result

**Status: Issues found and fixed**

## What was compared

| Aspect | Dashboard (Express.js) | Go API | Match? |
|--------|----------------------|--------|--------|
| Permission: internal actor check | `checkIsInternalActor` | `identity.CheckIsInternalActor()` | Yes |
| Permission: domain/action | `purchaseOrders` / `update` | `PermissionDomainPurchaseOrders` / `ActionUpdate` | Yes |
| Permission: target account required | Yes | `identity.CheckTargetAccountSet()` | Yes |
| Duplicate number check | Checks with exclusion of current ID | `IsDuplicateOrderNumber` with exclude ID | Yes |
| Duplicate error message | "This purchase order number is already taken." | "A purchase order with this number already exists." | Minor wording diff (acceptable) |
| Updatable fields: note | Yes | Yes | Yes |
| Updatable fields: number | Yes | Yes | Yes |
| Updatable fields: priority | Via priority object (code) | Via `priority_code` string | Yes (equivalent) |
| Updatable fields: billing address | Via full address object (upsert) | Via `billing_address_id` (ID reference) | By design |
| Updatable fields: shipping address | Via full address object (upsert + link) | Via `shipping_address_id` (ID reference) | By design |
| Updatable fields: scheduled/promised date | `scheduledAt` | `promised_at` | Yes |
| Updatable fields: contacts | Full replace (delete+create) by notification type | Full replace (delete all + create) | Yes (minor filter diff, see below) |
| SQL type filter on update | `typeCode = purchaseOrder` | **Missing** — fixed | Fixed |
| SQL buyer_account_id filter on update | `buyerAccountID = ownerAccountID` | **Missing** — fixed | Fixed |
| SQL type filter on get | `typeCode = purchaseOrder` (implicit via Prisma) | Already present | Yes |
| Idempotency | Not present in Dashboard | Full idempotency key support | Go is better |
| Transaction wrapping | Dashboard update is NOT transactional | Go wraps in transaction | Go is better |
| Response shape | Full PurchaseOrder with nested objects | PurchaseOrderDetail with includes | Yes |

## Issues found and fixed

### 1. Missing `sales_order_type_code` filter on `UpdatePurchaseOrder` SQL query

**File:** `services/core-service/internal/infrastructure/queries/purchase_order.sql`

The Dashboard's update WHERE clause includes `typeCode: 'purchase_order'` to ensure only purchase orders can be updated via this endpoint. Since purchase orders and sales orders share the same `sales_order` table and use the same `OrderIDPrefix`, a missing type filter could allow accidentally updating a sales order via the purchase order endpoint.

**Fix:** Added `AND sales_order_type_code = 'purchase_order'` to the `UpdatePurchaseOrder` query.

### 2. Missing `buyer_account_id` filter on `UpdatePurchaseOrder` SQL query

The Dashboard's update WHERE clause includes `buyerAccountID: ownerAccountID` as a safety check. For purchase orders, the owner account is always the buyer. This prevents edge cases where data integrity issues could cause unexpected updates.

**Fix:** Added `AND buyer_account_id = sqlc.arg('account_id')` to the `UpdatePurchaseOrder` query (reuses existing `account_id` param).

## Minor differences (not fixed, acceptable)

1. **Address handling**: Dashboard accepts full address objects with upsert logic; Go accepts address IDs. This is by design — the Go API manages addresses through separate address endpoints.

2. **Contact deletion scope**: Dashboard deletes only contacts with `notificationTypeCode = 'purchaseOrderSubmission'`; Go deletes all contacts for the order. Functionally equivalent since purchase orders only have `purchaseOrderSubmission` contacts. Not changed because the same delete query is also used in cascade delete where deleting all contacts is correct.

3. **COALESCE pattern for nullable fields**: The Go `COALESCE` pattern means nullable fields (note, promised_at) cannot be cleared once set. Dashboard's Prisma can set them to null. This is a known codebase-wide pattern trade-off, not specific to this endpoint.

4. **Error message wording**: Dashboard says "This purchase order number is already taken." vs Go says "A purchase order with this number already exists." Functionally equivalent.

## Regenerated files

- Ran `make sqlc core` to regenerate `services/core-service/internal/infrastructure/sqlc/purchase_order.sql.go`
- The `UpdatePurchaseOrderParams` struct was unchanged (sqlc reuses the `AccountID` param for `buyer_account_id`)
- No repository code changes needed
