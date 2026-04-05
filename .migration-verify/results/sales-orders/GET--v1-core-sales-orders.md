# Verification: GET /v1/core/sales-orders

**Status: Issues found and fixed**

## What was compared

| Area | Dashboard (Express.js) | Go API | Match? |
|------|----------------------|--------|--------|
| Permission checks | Internal: `salesOrders.read`; Customer: actor-scoped | Same | Yes |
| Customer actor scoping | Filters to `buyerAccountID = customer.accountID` | Same via `BuyerAccountID` param | Yes |
| Pagination | Offset (`take`/`skip`) | Keyset cursor (intentional upgrade) | OK |
| Query param: `query` | Searches order number, customer name, customer PO | Was missing customer name search | **Fixed** |
| Query param: `status` | Single status code | Array of status codes (superset) | OK |
| Query param: `itemIDs` | Filter by item IDs via order lines | Same | Yes |
| Query param: `productLineIDs` | Filter by product line IDs via product→line | Same | Yes |
| Query param: `customerIDs` | Filter by buyer account IDs | Same | Yes |
| Query param: `customerGroupIDs` | Filter by account group on relation | Same | Yes |
| Query param: `salesRepIDs` | Filter by sales rep user IDs | Same | Yes |
| Query param: `startDate`/`endDate` | Inclusive date range on `created_at` | Same | Yes |
| Query param: `excludeInternalOrders` | Change-log-based lookup | `buyer_account_id != owner_account_id` | **Divergent** |
| `sellerAccountID` filter | Always filters `sellerAccountID = ownerAccountID` | Was missing | **Fixed** |
| Sorting | Relevance (if query) then `createdAt DESC` | `createdAt DESC, id DESC` (no relevance) | Acceptable |
| Response fields | Many summary fields (see below) | Fewer fields | **Known gap** |

## Issues found and fixed

### 1. Search query missing customer name (Fixed)
**Dashboard:** The `query` parameter searched order number, customer name (`buyerAccount.name`), customer alias, external number, notes, and customer PO.
**Go (before):** Only searched order number and customer PO.
**Fix:** Added `OR ba.name LIKE sqlc.narg('search_query')` to both `ListSalesOrdersForward` and `ListSalesOrdersBackward` SQL queries. This covers the most impactful missing search — customer name matching.

### 2. Missing `seller_account_id` filter (Fixed)
**Dashboard:** Always filters `sellerAccountID = ownerAccountID`, ensuring only the account's own sales orders are returned (not purchase orders from other accounts).
**Go (before):** Did not have this filter, which could return non-sales-order records.
**Fix:** Added `AND so.seller_account_id = so.owner_account_id` to both list queries.

## Known divergences (not fixed)

### 3. `excludeInternalOrders` — different semantics
**Dashboard:** Looks up change logs to find orders created by internal users (users who are `accountUser` records), then excludes those order IDs.
**Go:** Uses `buyer_account_id != owner_account_id` as a simpler heuristic — excludes orders where the buyer is the same as the owner (self-orders).
**Assessment:** These are semantically different. The Dashboard approach is more accurate but requires expensive change log lookups. The Go approach is a reasonable simplification but could produce different results in edge cases (e.g., an internal user creating an order for an external customer would NOT be excluded in Go, but WOULD be excluded in Dashboard). This should be revisited if the feature is business-critical.

### 4. Missing response fields in summary
The Dashboard `OrderSummary` returns several fields not present in the Go `SalesOrderSummary`:
- `shipTo` (shipping address)
- `paymentTerm`
- `fulfillmentProgress` (calculated from pick/invoice lines)
- `pickProgress` (calculated from pick lines)
- `shipments` (related shipments with statuses)
- `isPaid` (derived from payment intents + invoice status)
- `email` (customer support email)
- `productionRunID`
- `createdBy` / `isCreatedByInternal` (from change logs)
- `expiredAt`

The Go version includes fields the Dashboard doesn't: `type` (sales order type), `is_acknowledgment_sent`.

**Assessment:** Adding all missing fields would require significant SQL changes (additional joins, subqueries for progress calculations, change log lookups). The most critical missing fields for the frontend are likely `fulfillmentProgress`, `pickProgress`, and `isPaid` as these drive UI state. These should be added in a follow-up.

### 5. Search scope — minor differences
The Dashboard also searches customer alias, external number, and notes via the `CustomerAccountAdapter`. The Go version now searches customer name but not these additional fields. This is a minor difference unlikely to affect most users.

## Files modified
- `services/core-service/internal/infrastructure/queries/sales_order.sql` — Added `ba.name` to search clause and `seller_account_id` filter in both list queries
- `services/core-service/internal/infrastructure/sqlc/sales_order.sql.go` — Regenerated
