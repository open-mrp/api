# GET /v1/core/sales-orders/{id} — Migration Verification

## Status: Issues Found and Fixed

## What Was Compared

- **Permission checks**: Both implementations check `CheckIsAssignedActor()`, internal users require `PermissionDomainSalesOrders.ActionRead`, customer users get scoped access via `buyer_account_id`. ✅ Parity confirmed.
- **Customer actor access**: Both implementations have separate query paths for internal vs customer actors. ✅ Parity confirmed.
- **DB queries and joins**: Both query the same tables with the same join structure (account_relation, buyer account, status, type, priority, addresses, carrier, carrier_option, sales_rep, payment_term, shipping_term, order_discount, pick). ✅ Parity confirmed.
- **Response shape**: Core order fields match (id, number, customer_po, note, is_acknowledgment_sent, status, type, priority, customer, addresses, carrier, carrier_option, sales_rep, payment_term, shipping_term, order_discount, production_run, pick, timestamps). ✅ Parity confirmed.
- **Lines expansion**: Both implementations support optional line inclusion. Go uses `include=lines` query parameter. ✅ Parity confirmed.
- **Line detail fields**: quantity_ordered, unit_price, unit_cost, item, product_sku, product_description, edi_line_item_id, line_item_number, timestamps. ✅ Parity confirmed.
- **Error handling**: Both return 404 for not found, 403 for permission denied. ✅ Parity confirmed.
- **Idempotency**: GET endpoint, no idempotency keys needed. ✅ Correct.

## Issues Found and Fixed

### 1. Missing `quantity_picked`, `quantity_packed`, `quantity_invoiced` on line presenter
**File**: `services/api-gateway/endpoints/sales-orders/presenter.go`

The proto and domain model correctly carry these computed aggregate values from the SQL query, and the gRPC handler maps them. However, the `SalesOrderLineDetailPresenter` did not map them to the API resource, meaning the response always returned `null` for these fields even though the data was available.

**Fix**: Added presenter mapping for `QuantityPicked`, `QuantityPacked`, and `QuantityInvoiced` using the same unit as the ordered quantity.

### 2. Missing `owner_account_id` filter in `GetSalesOrderForCustomer` SQL query
**File**: `services/core-service/internal/infrastructure/queries/sales_order.sql`

The `GetSalesOrderForCustomer` query only filtered by `buyer_account_id`, not `owner_account_id`. This meant a customer with orders from multiple sellers could potentially see orders from seller A when authenticated against seller B. The Dashboard's `isOrderForCustomer` properly scopes by both `ownerAccountID` and `buyerAccountID`.

**Fix**: Added `AND so.owner_account_id = sqlc.arg('account_id')` to the query and updated the repository to pass `accountID`.

### 3. Missing `seller_account_id = owner_account_id` filter in GET queries
**File**: `services/core-service/internal/infrastructure/queries/sales_order.sql`

The Dashboard's `find` method filters with `sellerAccountID = ownerAccountID` to ensure only seller orders are returned (not purchase orders in the same table). The list queries already had this filter, but the GET queries did not.

**Fix**: Added `AND so.seller_account_id = so.owner_account_id` to both `GetSalesOrder` and `GetSalesOrderForCustomer` queries.

## Noted Differences (Intentional / Out of Scope)

The Dashboard's GET order response includes several additional fields not present in the Go API:
- `shipments` — array of lightweight shipment records
- `invoices` — array of basic invoice records (id, number)
- `paymentIntentIDs` / `isPaid` — payment status info
- `acknowledgementEmailContacts` / `invoiceEmailContacts` — email contact arrays
- `createdBy` / `isCreatedByInternal` — creation user tracking

These are intentionally not included in the Go API's GET sales order endpoint. They can be fetched via their own dedicated endpoints or added as expandable includes in the future. The Go API's include system (`customer`, `bill_to_address`, `ship_to_address`, `carrier`, `carrier_option`, `payment_term`, `shipping_term`, `order_discount`, `lines`) provides the core data needed.
