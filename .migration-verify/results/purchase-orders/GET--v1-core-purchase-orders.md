# Verification: GET /v1/core/purchase-orders

**Status: Issues found and fixed, with remaining concerns noted**

## What was compared

| Aspect | Dashboard (Express.js) | Go API | Match? |
|--------|----------------------|--------|--------|
| Permission check: internal actor | `checkIsInternalActor` | `types.CheckIsInternalActor` | Yes |
| Permission check: domain/action | `purchaseOrders` / `read` | `PermissionDomainPurchaseOrders` / `ActionRead` | Yes |
| Account isolation | `ownerAccountID` + `buyerAccountID` | `owner_account_id` + `type_code = 'purchase_order'` | Yes |
| Filter: status codes | Single status or "all" | Array of status codes (superset) | Yes |
| Filter: item IDs | `lines.some { itemID in [...] }` | EXISTS subquery on `sales_order_line` | Yes |
| Filter: supplier IDs | `sellerAccountID in [...]` | `seller_account_id IN (...)` | Yes |
| Filter: date range | `createdAt gte/lte` | `created_at >= / <=` | Yes |
| Search: PO number | Yes (`number` field) | Yes (`so.number LIKE`) | Yes |
| Search: supplier name | Yes (via `SupplierAccountAdapter`) | **Was missing — FIXED** | Fixed |
| Search: customer PO number | Yes (`customerPO` field) | **Was missing — FIXED** | Fixed |
| Pagination | Offset-based (take/skip) | Cursor-based | Intentional upgrade |
| Sorting | Relevance on number + createdAt desc | createdAt desc, id desc | Acceptable |
| Response: id, number | Yes | Yes | Yes |
| Response: supplier (id, name, number) | Yes (sub-object) | Yes (sub-object) | Yes |
| Response: status (code, name) | Yes | Yes | Yes |
| Response: type (code, name) | N/A in Dashboard summary | Yes | OK |
| Response: priority (code, name) | Yes | Yes | Yes |
| Response: line_count | Yes | Yes | Yes |
| Response: is_acknowledgment_sent | `emailSent` | `is_acknowledgment_sent` | Yes (renamed) |
| Response: timestamps | createdAt, updatedAt, issuedAt, completedAt | Same set | Yes |
| Response: promisedAt | Yes | **Missing from summary** | See concerns |
| Response: deliveryProgress | Yes (calculated) | **Missing from summary** | See concerns |
| Response: shipTo (light address) | Yes | **Missing from summary** | See concerns |
| Response: receivingOrder | Yes (id, number) | **Missing from summary** | See concerns |
| Response: acceptsEmails | Yes | **Missing from summary** | See concerns |

## Issues found and fixed

### 1. Search query missing supplier name and customer PO number

**File:** `services/core-service/internal/infrastructure/queries/purchase_order.sql`

The Dashboard's `PurchaseOrderAdapter.fetchInput()` searches across three fields using OR logic:
- `number` (PO number)
- `sellerAccount` name (supplier account name)
- `customerPO` (customer PO number)

The Go SQL queries (both `ListPurchaseOrdersForward` and `ListPurchaseOrdersBackward`) only searched on `so.number`. This was inconsistent with the sales order list query in `sales_order.sql`, which already searches on `so.number`, `so.customer_po_number`, and `ba.name`.

**Fix:** Added `OR sa.name LIKE sqlc.narg('search_query')` and `OR so.customer_po_number LIKE sqlc.narg('search_query')` to both forward and backward query search conditions. Regenerated sqlc.

## Remaining concerns

These are Dashboard summary fields not present in the Go `PurchaseOrderSummary` response. They would require proto message changes, domain model updates, SQL query modifications, and full regeneration:

1. **`promisedAt`** — The Dashboard includes `promisedAt` in the summary. The Go detail endpoint has it, but the list summary does not. The column is already selected in the `GetPurchaseOrder` query but not in the list queries.

2. **`deliveryProgress`** — The Dashboard calculates `(totalQuantityDelivered / totalQuantityOrdered) * 100` from receiving order line quantities. This is a computed field requiring additional JOINs to `receiving_order_line` and `quantity` tables. This would add complexity to the list query.

3. **`shipTo`** (light address) — The Dashboard includes a light shipping address sub-object. Would require LEFT JOINs to `address` and `geolocation` tables in the list query.

4. **`receivingOrder`** — The Dashboard includes a reference to the associated receiving order (`{id, number}`). Would require a LEFT JOIN to `receiving_order` in the list query.

5. **`acceptsEmails`** — Whether the supplier has notification preferences for PO submission. Would require joining `account_relation_notification_preference`.

These fields enhance the list view's usefulness but are not critical for basic functionality. They should be added in a follow-up to achieve full parity.
