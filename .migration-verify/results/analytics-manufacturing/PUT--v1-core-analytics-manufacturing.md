# PUT /v1/core/analytics/manufacturing — Verification Summary

## Status: Issues found and fixed

## What was compared

- **Validation rules**: Request requires `start_date`, `end_date`, `type` — matches Dashboard
- **Permission checks**: Internal actor + `invoices:read` permission — matches Dashboard
- **DB queries and logic**: SQL queries, date filtering, metric computation formulas
- **Error handling**: Invalid type returns validation error (Go) vs throws Error (Dashboard) — equivalent
- **Response shape**: `{ object, value }` — matches Dashboard's `{ value }` (Go adds standard `object` field per conventions)
- **Side effects**: None in either implementation
- **Idempotency**: PUT endpoint, idempotent by design — correct

## Issues found and fixed

### 1. Date field mismatch: `closed_at` → `scanned_at`
- **Dashboard**: Filters batches by `b.scanned_at >= startDate AND b.scanned_at <= endDate`
- **Go (before fix)**: Filtered by `b.closed_at IS NOT NULL AND b.closed_at >= start_date AND b.closed_at <= end_date`
- **Fix**: Changed SQL queries `GetManufacturingProduction` and `GetManufacturingQuality` to use `b.scanned_at` and removed the `closed_at IS NOT NULL` filter

### 2. Missing analytics types: `costsPerUnit`, `margin`, `laborEfficiency`
- **Dashboard**: Supports 5 types: `production`, `costsPerUnit`, `margin`, `quality`, `laborEfficiency`
- **Go (before fix)**: Only supported `production` and `quality`; returned `0` silently for all other types
- **Fix**: Added three new SQL queries:
  - `GetManufacturingCostsPerUnit` — joins invoice_line, invoice, sales_order_line with unit conversion logic to compute totalCost/totalQuantity
  - `GetManufacturingMargin` — same tables to compute (revenue-cost)/revenue
  - `GetManufacturingLaborEfficiency` — joins batch with production_step, labor time rate, and production totals CTE to compute labor-weighted quality ratio
- Updated repository `GetManufacturingMetric` switch to handle all 5 types
- Changed default case from silently returning 0 to returning a validation error

### 3. Batch endpoint missing 3 of 5 metrics
- **Dashboard batch**: Computes all 5 metrics via two optimized queries (batch + invoice) per period
- **Go (before fix)**: Only computed `production` and `quality` per period, returning 0 for the other 3
- **Fix**: Added combined batch queries (`GetManufacturingBatchBatchMetrics`, `GetManufacturingBatchInvoiceMetrics`) and updated `GetManufacturingBatch` to use them, computing all 5 metrics per period

### 4. Labor efficiency formula correction
- **Dashboard single-metric `laborEfficiency`** had a bug: used `+` operator for seconds and quantity columns instead of `*` when applying the labor time ratio
- **Dashboard batch `laborEfficiency`** used the corrected `*` operator (noted in comment: "Bug fix: labor efficiency columns use * instead of + when applying labor time ratio")
- **Go implementation**: Uses the corrected `*` operator consistently in both single-metric and batch queries

## Files modified

- `services/core-service/internal/infrastructure/queries/analytics.sql` — Fixed date columns, added 5 new SQL queries
- `services/core-service/internal/infrastructure/sqlc/analytics.sql.go` — Regenerated via `make sqlc core`
- `services/core-service/internal/infrastructure/sqlc/db.go` — Regenerated (prepared statements)
- `services/core-service/internal/infrastructure/repository/analytics_repository.go` — Updated `GetManufacturingMetric` to handle all 5 types, rewrote `GetManufacturingBatch` to compute all 5 metrics

## Remaining concerns

- The Dashboard's `getManufacturingAnalyticsBatch` service method accepts `customerIDs`, `productLineIDs`, `customerGroupIDs`, `itemIDs` filter params, but the Dashboard repository `getManufacturingAnalyticsBatch` ignores them (only uses accountID, startDate, endDate). The Go implementation is consistent with this behavior — it accepts the params at the gRPC level but doesn't filter by them in SQL.
- The existing `GetManufacturingProduction` query referenced in other analytics endpoints (e.g., open batches) also uses `closed_at`; this was NOT changed since those are separate query names. Only the manufacturing analytics queries were affected.
