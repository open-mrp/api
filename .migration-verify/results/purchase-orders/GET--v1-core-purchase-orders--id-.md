# GET /v1/core/purchase-orders/{id} — Verification Result

**Status:** Issue found and fixed

## What Was Compared

- **Validation:** Request parameter validation (path ID)
- **Permission checks:** Internal actor + read permission on purchaseOrders domain
- **DB query:** Filters (id, owner_account_id, type_code), joins (supplier, addresses, status, priority, carrier, payment/shipping terms, receiving order)
- **Error handling:** 404 on not found via db.MapSQLError
- **Response shape:** PurchaseOrderDetail with expandable sub-resources (supplier, addresses, carrier, carrier_option, payment_term, shipping_term, receiving_order, lines, contacts)
- **Side effects:** None (GET endpoint)
- **Idempotency:** N/A (GET endpoint, idempotent by design)

## Issues Found and Fixed

### 1. Missing `sales_order_type_code` filter in SQL query (FIXED)

**File:** `services/core-service/internal/infrastructure/queries/purchase_order.sql` (GetPurchaseOrder query)

The `GetPurchaseOrder` SQL query was missing `AND so.sales_order_type_code = 'purchase_order'` in its WHERE clause. The Dashboard enforces `typeCode: OrderTypeCodes.purchaseOrder` in its Prisma query, which prevents fetching a sales order or other order type through the purchase order endpoint. The Go list queries already had this filter, but the get-by-ID query did not.

**Fix:** Added `AND so.sales_order_type_code = 'purchase_order'` to the WHERE clause and regenerated sqlc.

## Noted Differences (Acceptable)

### Dashboard `buyerAccountID` check
The Dashboard also checks `buyerAccountID: ownerAccountID`. For purchase orders, `buyer_account_id` always equals `owner_account_id` (set identically in the Create method), so the existing `owner_account_id` filter is sufficient. This is not a functional parity gap.

### Dashboard `createdBy` field
The Dashboard populates a `createdBy` field by querying the `changeLog` table for the user who created the record. No Go API resources implement this pattern — it is a deliberate architectural difference in the new API, not a parity gap.

### Go includes system vs Dashboard eager loading
The Dashboard eagerly loads all relations (lines, contacts, supplier, addresses, etc.). The Go API uses an `include` query parameter system where lines and contacts are conditionally loaded. This is consistent with Go API conventions (expandable sub-resources) and is an intentional improvement.

## Parity Confirmed

All other aspects match:
- Permission model (internal actor + purchaseOrders read)
- Account scoping via `owner_account_id`
- Contact filtering by `purchaseOrderSubmission` notification type
- Response field mapping (number, note, is_acknowledgment_sent, scheduled_at from promised_at, etc.)
- All joined sub-resources (supplier with name/number, addresses with geolocation, status, type, priority, carrier, payment/shipping terms, receiving order)
- Line item details (quantity ordered/received, unit price, unit cost)
