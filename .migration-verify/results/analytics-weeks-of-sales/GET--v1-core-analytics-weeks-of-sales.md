# Verification: GET /v1/core/analytics/weeks-of-sales

## Status: Issues found and fixed

## What was compared

- **Permission checks**: Both require internal actor + `PermissionDomainInventory` / `read`. ✅ Match.
- **Request parameters**: Both accept optional `periodInWeeks` (default 4). ✅ Match.
- **Response shape**: Go returns `{ object, data: [...], count }` with `WeeksOfSalesItem` containing `product_line`, `quantity_on_hand`, `average_sales_quantity`, `weeks_of_sales`. Dashboard returns `{ items: [...], count }` with same fields. Field mapping is correct. ✅ Match.
- **Algorithm**: Both follow the same steps: get sale products → group by product line → fetch on-hand inventory → query order demand for period → calculate avg weekly sales → compute weeks of sales. ✅ Match.
- **Product query**: Go adds `AND i.deleted_at IS NULL` filter which Dashboard lacks. This is a reasonable improvement (excludes deleted items from calculations). Minor behavioral difference but correct. ✅ Acceptable.
- **Idempotency**: GET endpoint — no idempotency keys needed. ✅ Correct.
- **Side effects**: None in either implementation. ✅ Match.

## Issues found and fixed

### 1. Wrong date column in SQL query
- **Dashboard**: Filters orders by `issuedAt` (maps to `issued_at` column) — only includes orders that were actually issued
- **Go (before fix)**: Filtered by `so.created_at` — included draft/unissued orders and missed the business intent of using issuance date
- **Fix**: Changed `so.created_at` to `so.issued_at` in `GetOrderQuantityByProductLine` query
- **Also updated**: Repository code to use `sql.NullTime` for the params since `issued_at` is a nullable column

### 2. Incorrect cancelled order exclusion
- **Dashboard**: Does NOT filter by order status — includes all orders with an `issued_at` in the date range (cancelled orders that were issued are included in demand calculations)
- **Go (before fix)**: Had `AND so.sales_order_status_code != 'cancelled'` which excluded cancelled orders from demand
- **Fix**: Removed the cancelled status filter to match Dashboard behavior

### Files modified
- `services/core-service/internal/infrastructure/queries/analytics.sql` — SQL query fix
- `services/core-service/internal/infrastructure/sqlc/analytics.sql.go` — regenerated via `make sqlc core`
- `services/core-service/internal/infrastructure/repository/analytics_repository.go` — updated params to use `sql.NullTime`

## Remaining concerns
- None. The Go implementation now matches the Dashboard behavior for this endpoint.
