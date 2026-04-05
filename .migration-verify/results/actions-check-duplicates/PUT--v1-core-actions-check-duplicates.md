# PUT /v1/core/actions/check-duplicates

**Status: Issues found and fixed**

## What was compared

| Aspect | Dashboard | Go | Match? |
|--------|-----------|-----|--------|
| HTTP method | PUT | PUT | Yes |
| Auth: actor check | `checkIsAssignedActor` (allows internal + customer) | `CheckIsAssignedActor` | Yes |
| Auth: permission check (internal only) | invoices→read, salesOrders→read | Same domains/actions | Yes |
| Target account required | Yes | Yes | Yes |
| Duplicate types | invoice_number, order_number, customer_po_number | Same | Yes |
| customer_id required for customer_po_number | Yes (400 if missing) | Yes (validation error) | Yes |
| Response shape | `{ isDuplicate, message }` | `{ is_duplicate, message }` | Yes (casing matches JSON convention) |
| Duplicate messages | "This invoice/sales order/customer PO number X already exists" | Same | Yes |
| Idempotency keys | Not needed (PUT) | Not used | Yes |
| Record number trimming | `.trim()` before all queries | **Missing** | Fixed |
| Invoice query filters | `account_id`, `number` | Same | Yes |
| Order query: seller filter | `sellerAccountID = ownerAccountID` | **Missing** `seller_account_id` filter | Fixed |
| Customer PO query: seller filter | `sellerAccountID = ownerAccountID` | **Missing** `seller_account_id` filter | Fixed |
| Order query: type filter | `typeCode = salesOrder` | N/A (dedicated `sales_order` table) | Yes |

## Issues found and fixed

### 1. Missing `strings.TrimSpace` on record number

**File:** `services/core-service/internal/service/utils_service.go`

The Dashboard trims whitespace from record numbers via `.trim()` before all duplicate queries. The Go service passed the raw value. Added `strings.TrimSpace(params.RecordNumber)` before querying.

### 2. Missing `seller_account_id` filter on sales order duplicate queries

**File:** `services/core-service/internal/infrastructure/queries/sales_order.sql`

The Dashboard's order repo filters by `sellerAccountID = ownerAccountID` to ensure only orders where the account is the seller are checked. The Go queries only filtered by `owner_account_id`. Added `AND seller_account_id = sqlc.arg('account_id')` to both `IsDuplicateOrderNumber` and `IsDuplicateCustomerPO` queries, and regenerated sqlc.

## Remaining concerns

- None. Pre-existing build errors in `sales_order_repo.go` (missing `NoteFirstShipAt`) and `shipment_repository.go` (wrong `MarkShipped` signature) are unrelated to this endpoint.
