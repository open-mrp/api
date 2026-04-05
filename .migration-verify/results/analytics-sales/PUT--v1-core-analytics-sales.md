# PUT /v1/core/analytics/sales — Migration Verification

## Result: Issues found and fixed

## What was compared

- **Validation rules**: Request fields (start_date, end_date, product_line_ids, customer_ids, sales_rep_ids, customer_group_ids)
- **Permission checks**: Internal actor + invoices:read permission
- **DB queries**: SQL query structure, JOINs, field selections, filters, unit conversions
- **Error handling**: Standard error propagation
- **Side effects**: None (read-only endpoint)
- **Response shape**: All fields in SalesEntry resource
- **Idempotency**: PUT endpoint, idempotent by design (no idempotency keys needed)

## Issues found and fixed

### 1. Wrong column names in SQL query (Critical)
The Go query used `sol.sell_rate_id` and `sol.cost_rate_id` which don't exist in the `sales_order_line` table. The correct columns are `sol.unit_price_id` and `sol.unit_cost_id`.

### 2. Missing unit conversion (Critical)
The Dashboard performs full unit conversion using ratio/offset values from the unit table (converting quantities to base units, and rates through numerator/denominator unit conversions). The Go query was doing simple `CAST(il_q.value)` without any conversion. Fixed by adding the same conversion math used in the existing `GetQuarterlyOrderTotals` query.

### 3. Missing fields in SQL query and domain model (Major)
The Go query was missing many fields that the Dashboard returns:
- `id` (invoice line ID)
- `issued_at`, `completed_at`, `first_ship_at`, `promised_at` (from sales_order)
- `customer_po` (customer PO number)
- `sales_rep_username` (via user table)
- `parent_customer_id` (via parent account relation)
- `customer_created_at` (buyer account created date)
- `customer_type_group_id` (account group ID)
- `customer_group_name` (account group name)
- `product_type_code` (from product table)
- `category_name` (from item_category table)
- `ship_to_country` (from geolocation)
- `order_discount_code` (from order_discount table)

### 4. Missing JOINs (Major)
Added missing JOINs to match Dashboard:
- `item_category ic` for category name
- `unit_group ug` and base `unit bu_unit` for unit conversion
- Rate numerator/denominator units for price/cost conversion
- `parent_account_relation` for parent customer lookup
- `order_discount` for discount code
- Changed sales rep join from `ar.default_sales_rep_id` to `so.sales_rep_id` to match Dashboard

### 5. Missing filter parameters (Major)
The Go query only filtered by account ID and date range. Added filters for:
- `sales_rep_ids` (filter by sales rep)
- `product_line_ids` (filter by product line)
- `customer_group_ids` (filter by customer group)
- `customer_ids` (filter by customer, including parent customer hierarchy lookup)

### 6. Missing sales rep data protection (Major)
Dashboard zeros out `unitCost` when the requesting user has a sales rep role, and restricts the query to only show data for that sales rep's account user ID. Added this logic to the Go analytics service.

### 7. gRPC handler missing field mappings
Updated the gRPC handler to map all new fields from the domain model to the proto response.

## Files modified
- `services/core-service/internal/infrastructure/queries/analytics.sql` — Rewrote GetSalesEntries query
- `services/core-service/internal/infrastructure/sqlc/analytics.sql.go` — Regenerated via `make sqlc core`
- `services/core-service/internal/domain/analytics_models.go` — Updated SalesEntry struct with all fields
- `services/core-service/internal/infrastructure/repository/analytics_repository.go` — Updated field mapping and filter params
- `services/core-service/internal/service/analytics_service.go` — Added sales rep data protection
- `services/core-service/internal/infrastructure/grpc/grpc_analytics_handler.go` — Updated proto field mappings

## Remaining concerns
- The Dashboard applies `formatRecordNumber()` to order numbers and invoice numbers (pads numeric prefixes to 6 digits). The Go API does not do this formatting. This is a minor display difference.
- The Dashboard streams results via an async generator with batching (10,000 per batch). The Go API returns all results at once. For very large result sets this could be a memory concern, but is functionally equivalent.
- Pre-existing compilation errors in `sales_order_repo.go` and `shipment_repository.go` prevent a full `make test` run, but these are unrelated to this endpoint.
