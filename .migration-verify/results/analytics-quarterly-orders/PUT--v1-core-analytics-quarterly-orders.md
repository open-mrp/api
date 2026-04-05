# PUT /v1/core/analytics/quarterly-orders

## Status: Issues found and fixed

## What was compared
- Request validation and schema (both accept optional arrays: salesRepIDs, itemIDs, productLineIDs, customerIDs, customerGroupIDs)
- Permission checks (both: internal actor + invoices.read)
- SQL query logic (joins, calculations, grouping, filtering)
- Response shape (map of year string -> {q1, q2, q3, q4, total})
- Side effects (none in either implementation)
- Idempotency (PUT endpoint, no idempotency keys needed — correct)

## Issues found and fixed

### 1. Date column mismatch (critical)
- **Dashboard:** Groups by `QUARTER(so.issued_at)` / `YEAR(so.issued_at)`
- **Go (before):** Used `so.created_at`
- **Fix:** Changed to `so.issued_at`

### 2. Revenue calculation oversimplified (critical)
- **Dashboard:** Full unit conversion math: `(quantity * unit_ratio + unit_offset) * (price * price_unit_ratio + price_unit_offset) / denominator_unit_conversion`. Joins quantity → unit (for order unit conversion), rate → numerator_unit, denominator_unit (for price unit conversion).
- **Go (before):** Simple `quantity.value * sell_rate.value` with no unit conversions
- **Fix:** Rewrote query to match Dashboard's full unit conversion calculation with all required joins

### 3. Wrong rate column (critical)
- **Dashboard:** Joins `sol.unit_price_id` (the unit price rate)
- **Go (before):** Joined `sol.sell_rate_id`
- **Fix:** Changed to `sol.unit_price_id` with proper numerator/denominator unit joins

### 4. Missing product type filter (moderate)
- **Dashboard:** Filters `fg.product_type_code = 'sale'` to only include sale products
- **Go (before):** No product type filter
- **Fix:** Added `fg.product_type_code = 'sale'` filter

### 5. Incorrect cancelled order exclusion (moderate)
- **Dashboard:** Does NOT exclude cancelled orders
- **Go (before):** Had `so.sales_order_status_code != 'cancelled'`
- **Fix:** Removed the cancelled status exclusion to match Dashboard

### 6. Optional filters not applied (critical)
- **Dashboard:** Dynamically applies filters for customerIDs (with child account lookup), salesRepIDs, productLineIDs, itemIDs, customerGroupIDs
- **Go (before):** Accepted all filter params but the SQL query ignored them entirely
- **Fix:** Added all five filters using the `sqlc.arg('include_X_filter')` / `sqlc.slice()` pattern, including the customer child account relationship subquery

### 7. Missing table joins (critical)
- **Dashboard:** Joins `product fg`, `quantity q`, `unit u_ord`, `rate r_price`, `unit u_price_num`, `unit u_price_den`, `account_relation ar`
- **Go (before):** Only joined `sales_order_line`, `quantity`, `rate`
- **Fix:** Added all missing joins

### 8. Result structure change
- **Before:** SQL returned per-quarter rows, Go code aggregated into per-year structures
- **After:** SQL returns per-year rows with q1/q2/q3/q4/total columns (matching Dashboard approach), Go code maps directly

## Files modified
- `services/core-service/internal/infrastructure/queries/analytics.sql` — Rewrote `GetQuarterlyOrderTotals` query
- `services/core-service/internal/infrastructure/repository/analytics_repository.go` — Updated `GetQuarterlyOrders` to use new params and row structure
- `services/core-service/internal/infrastructure/sqlc/analytics.sql.go` — Regenerated via `make sqlc core`

## Remaining concerns
- None. The Go implementation now matches the Dashboard business logic for this endpoint.
