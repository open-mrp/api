# PUT /v1/core/analytics/orders — Migration Verification

## Status: Issues found and fixed

## What was compared

- **SQL query**: Dashboard `findOrderEntries` vs Go `GetOrderEntries`
- **Validation/permissions**: Internal actor check, salesOrders:read permission, target account
- **Filtering**: salesRepIDs, customerIDs (with parent customer support), customerGroupIDs, productLineIDs
- **Response shape**: All 40+ fields in OrderEntry
- **Sales rep handling**: `isSalesRep` detection, sales rep ID override, cost data sanitization
- **Side effects**: None (read-only analytics endpoint)
- **Idempotency**: PUT endpoint, idempotent by design (no idempotency keys needed)

## Issues found and fixed

### 1. SQL query completely rewritten (analytics.sql — GetOrderEntries)

**Before**: Simplified query missing most Dashboard fields, wrong column references, wrong filtering.

**Fixed**:
- Changed entry ID from `so.id` (order ID) to `sol.id` (order line ID) to match Dashboard
- Added `issued_at`, `completed_at`, `first_ship_at`, `promised_at`, `customer_po_number` from sales_order
- Changed sales rep lookup from `ar.default_sales_rep_id` to `so.sales_rep_id` (direct on order)
- Changed sales rep name from `sr_user.name` to `bu.username` to match Dashboard
- Added parent customer ID via `parent_ar` join
- Added customer `created_at`, `account_group_id` (customer type group)
- Added product type code, category name from item_category
- Added full unit conversion math for quantities (ratio_numerator/ratio_denominator offsets) instead of raw values
- Added invoice quantity aggregation with unit normalization
- Added cost calculations: `unit_cost`, `unit_profit`, `total_cost`, `total_profit`, `total_invoiced`
- Added shipping geolocation (state, city, zip, country) via address → geolocation joins
- Added order discount code via `order_discount` join
- Changed price join from `sol.sell_rate_id` (non-existent column) to `sol.unit_price_id` (correct column)
- Added cost rate join via `sol.unit_cost_id`
- Added status filter: `sales_order_status_code = 'issued'`
- Added product type filter: `product_type_code = 'sale'`
- Removed date range filter (Dashboard's order analytics has no date filter)
- Added dynamic filters for salesRepIDs, customerIDs (with parent customer support), customerGroupIDs, productLineIDs using sqlc.slice pattern
- Changed ordering from `so.created_at` to `so.issued_at ASC`

### 2. Domain model expanded (analytics_models.go — OrderEntry)

**Before**: 21 fields. **After**: 42 fields matching Dashboard's OrderEntry.

Added: ID, IssuedAt, CompletedAt, FirstShipAt, PromisedAt, CustomerPO, SalesRepUsername, ParentCustomerID, CustomerCreatedAt, CustomerTypeGroupID, CustomerGroupName, ProductTypeCode, CategoryName, QuantityInvoiced, UnitCost, UnitProfit, TotalInvoiced, TotalCost, TotalProfit, ShipToState, ShipToCity, ShipToZipcode, ShipToCountry, OrderDiscountCode.

Removed: StartDate, EndDate (not used by Dashboard for order analytics), Status (always 'issued').

### 3. Repository updated (analytics_repository.go)

- Passes all dynamic filter parameters with nil-safety
- Maps all new sqlc row fields to domain model including NullString/NullTime handling

### 4. gRPC handler updated (grpc_analytics_handler.go)

- Maps all 40+ domain fields to proto
- Handles nullable timestamps with nil checks
- Removed StartDate/EndDate parsing (no longer in params)

### 5. Service updated (analytics_service.go)

- Added sales rep handling: when `IsSalesRep` is true and identity is a user, looks up their `account_user` ID and restricts `SalesRepIDs` filter to only their own data
- Added cost data sanitization: sets `UnitCost = 0` for sales rep users (matching Dashboard's `OrderAnalyticsUtils.protectTransaction`)

### 6. API gateway updated (service.go)

- Added `IsSalesRep` detection from identity's `RoleTypeCode` (matching Dashboard's controller behavior)
- Added `appctx` and `constants` imports

## Minor differences (intentional/acceptable)

- **Number formatting**: Dashboard applies `formatRecordNumber()` (zero-padding) to order numbers server-side. Go API returns raw numbers. This is consistent with how all other Go endpoints handle numbers — formatting is a frontend concern.
- **Streaming**: Dashboard streams results in batches of 10,000. Go returns all results at once. This is acceptable for the Go API's architecture.

## Pre-existing issues (not related to this endpoint)

- `sales_order_repo.go` missing `NoteFirstShipAt` method
- `shipment_repository.go` `MarkShipped` signature mismatch
- `SalesEntry` mapping issues in `GetSalesEntries`
- These are from other changes on the `feat/agents` branch.
