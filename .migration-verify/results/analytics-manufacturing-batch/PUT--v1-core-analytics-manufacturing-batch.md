# PUT /v1/core/analytics/manufacturing-batch

## Status: Issues found and fixed

## What was compared

- **Validation rules**: Request fields (startDate, endDate, comparisonStartDate, comparisonEndDate, optional filter IDs) - matches
- **Permission checks**: `checkIsInternalActor` + `checkHasPermission(invoices, read)` - matches
- **DB queries and logic**: Two queries (batch metrics + invoice metrics) with metric calculations
- **Error handling**: Standard error mapping through `db.MapSQLError` and tracing
- **Side effects**: None (read-only analytics endpoint) - matches
- **Response shape**: `{ current: ManufacturingMetrics, comparison: ManufacturingMetrics }` with 5 fields each - matches after fix
- **Idempotency**: PUT endpoint, idempotent by design (no mutation) - correct

## Issues found and fixed

### 1. Date field mismatch (batch queries)
- **Dashboard**: Uses `b.scanned_at` for batch date filtering
- **Go (before)**: Used `b.closed_at` via the `GetManufacturingProduction`/`GetManufacturingQuality` queries
- **Fix**: Created new `GetManufacturingBatchBatchMetrics` SQL query using `b.scanned_at`

### 2. Missing metrics: CostsPerUnit, Margin, LaborEfficiency
- **Dashboard**: Computes all 5 metrics (production, quality, laborEfficiency, costsPerUnit, margin) using two specialized queries
- **Go (before)**: Only computed Production and Quality by reusing individual metric queries; CostsPerUnit, Margin, LaborEfficiency were always 0
- **Fix**: Added `GetManufacturingBatchBatchMetrics` query (with `active_steps` CTE for labor efficiency) and `GetManufacturingBatchInvoiceMetrics` query (with unit conversion logic for costs/margin)

### 3. Missing invoice-based query (Query B)
- **Dashboard**: Has a complex query joining `invoice_line`, `invoice`, `sales_order_line` with full unit conversion logic to compute `totalCost`, `totalQuantity`, `totalRevenue`, `totalProfit`
- **Go (before)**: No equivalent query existed
- **Fix**: Added `GetManufacturingBatchInvoiceMetrics` SQL query matching the dashboard's Query B with identical unit conversion logic

### 4. Missing labor efficiency CTE
- **Dashboard**: Uses a `WITH active_steps AS (...)` CTE to compute production totals per step, then applies labor time ratio (`lt.value / pt.prod_total`) using multiplication
- **Go (before)**: No labor efficiency computation
- **Fix**: Included in the new `GetManufacturingBatchBatchMetrics` query

## Files modified

- `services/core-service/internal/infrastructure/queries/analytics.sql` - Added 2 new SQL queries
- `services/core-service/internal/infrastructure/sqlc/analytics.sql.go` - Regenerated via `make sqlc core`
- `services/core-service/internal/infrastructure/repository/analytics_repository.go` - Rewrote `GetManufacturingBatch` to use new queries and compute all 5 metrics

## Notes

- Filter parameters (customerIDs, productLineIDs, customerGroupIDs, itemIDs) are accepted by both dashboard and Go but are NOT used in SQL queries in either implementation. This is matching behavior - the dashboard repo also ignores these filters.
- The Go implementation now uses `getManufacturingMetricsForPeriod` helper to avoid duplicating the metric computation logic for current vs comparison periods (matching the dashboard's `Promise.all` pattern).
