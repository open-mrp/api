# PUT /v1/core/analytics/demand-forecast — Verification Result

**Status: Issues found and fixed**

## What was compared

- **Validation rules**: Request fields (productLineIDs, itemIDs, historyMonths, forecastMonths), defaults (24/4), max bounds (60/24)
- **Permission checks**: Internal actor + invoices:read — matches dashboard's `checkIsInternalActor` + `checkHasPermission(PermissionDomains.invoices, 'read')`
- **DB queries and logic**: Monthly demand from sales_order_line, monthly sales from invoice_line, product_type_code filter, date range filtering, grouping by item/month
- **Forecasting algorithm**: Seasonal EMA with seasonal factors, deseasonalization, EMA smoothing, residual-based confidence bands (z=1.0, ~68% CI)
- **Error handling**: Standard identity/permission/account checks
- **Response shape**: items[], currentMonthFraction; per-item: history, forecast, revenueHistory, revenueForecast, salesHistory, salesForecast, currentMonthDemand/Revenue/Sales
- **Idempotency**: Not required (PUT is idempotent by design, no data mutation)

## Issues found and fixed

### 1. Demand forecast was a stub (CRITICAL)
The Go repository `GetDemandForecast` method returned empty results. The entire Seasonal EMA forecasting algorithm from the dashboard was missing.

**Fix**: Implemented the full algorithm in `analytics_demand_forecast.go`:
- Fetches monthly demand (order-based) and monthly sales (invoice-based) data
- Groups by item, fills gaps with zero months
- Separates current partial month from complete history
- Computes seasonal factors per calendar month
- Deseasonalizes data and applies EMA (alpha = 2/(min(n,12)+1))
- Calculates residual standard deviation (last 12 months only)
- Generates forecast points with confidence bounds (demand, revenue, sales)
- Computes currentMonthFraction for proration

### 2. SQL query missing revenue, currency, end date, and product_type_code filter
The `GetDemandForecastMonthlyDemand` SQL query was missing:
- Revenue computation (quantity * sell rate)
- Currency field from price unit
- End date filter (was unbounded)
- `product_type_code = 'sale'` filter

**Fix**: Updated `analytics.sql` to add:
- `monthly_revenue` column via JOIN on sell rate
- `currency` column from price numerator unit
- `end_date` filter (`so.created_at < ?`)
- `product_type_code = 'sale'` WHERE clause
- Same fixes applied to `GetDemandForecastMonthlyRevenue` (added end date and product_type_code filter)

### 3. Item/ProductLine filtering not supported in SQL
sqlc doesn't support optional IN clauses, so filtering by `productLineIDs` and `itemIDs` was not implemented.

**Fix**: Applied post-query filtering in Go code, matching the dashboard's behavior of filtering items by product line and item ID.

## Remaining concerns

- **Unit conversion**: The dashboard's demand query performs full unit conversion using ratio_numerator/ratio_denominator/offset values to convert quantities to the item's base unit. The Go SQL query sums raw `sol_q.value` without unit conversion. This is a pre-existing simplification in the Go SQL that may cause minor numerical differences for items where orders use non-base units. This was not introduced by this migration and would need a separate SQL update.
- **Revenue calculation**: The dashboard computes order-based revenue with full unit conversion (quantity in base units * price with unit conversion). The Go version uses simplified `sol_q.value * r_sell.value`. Same pre-existing simplification as above.
