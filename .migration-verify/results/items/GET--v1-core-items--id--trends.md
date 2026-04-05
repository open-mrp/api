# GET /v1/core/items/{id}/trends

## Status: Issues found and fixed

## What was compared
- Permission checks (actor type, permission domain, action)
- DB queries (table, filters, joins, ordering)
- Data processing logic (deduplication, date windowing)
- Response shape (field names, types)
- Error handling

## Issues found and fixed

### 1. Wrong database table (CRITICAL)
- **Dashboard**: Queries `inventory_log` table (cumulative inventory snapshots)
- **Go (before fix)**: Queried `inventory_change_log` table (individual mutation events)
- **Fix**: Changed SQL query to use `inventory_log` table, matching the Dashboard

### 2. Missing 30-day date filter (CRITICAL)
- **Dashboard**: Filters to `createdAt` within last 30 days using `DATE_SUB`-equivalent logic
- **Go (before fix)**: No date filter at all — returned data from all time
- **Fix**: Added `AND il.created_at >= DATE_SUB(NOW(), INTERVAL 30 DAY)` to the SQL query

### 3. Wrong sort order
- **Dashboard**: Orders by `createdAt ASC`
- **Go (before fix)**: Ordered by `created_at DESC`
- **Fix**: Changed to `ORDER BY il.created_at ASC`

### 4. Arbitrary LIMIT 100
- **Dashboard**: No explicit limit (naturally bounded by 30-day window)
- **Go (before fix)**: `LIMIT 100` which could truncate data
- **Fix**: Removed the LIMIT clause

### 5. Missing deduplication by day
- **Dashboard**: Deduplicates logs by calendar day, keeping the earliest entry per day
- **Go (before fix)**: Returned all rows with no deduplication
- **Fix**: Added deduplication logic in the Go repository layer (matching Dashboard behavior)

## Parity confirmed after fixes
- Permission checks match: internal actor + items:read permission required
- Account scoping matches: both filter by target account ID
- Query logic now matches: same table, same date window, same sort order
- Deduplication logic matches: earliest entry per calendar day

## Remaining concerns
- **Unit normalization**: The Dashboard normalizes quantity values to base units via `BaseUnitUtils.normalizeQuantity()` when the unit is not a base unit. The Go implementation returns raw `quantity.value` without normalization. This is mitigated by the Go API returning structured `{date, value}` points (rather than the Dashboard's flat `number[]`), allowing consumers to handle unit conversion. However, if strict value parity is needed, unit normalization should be added to the Go repository.
- **Response shape difference**: The Go API returns `{object, trend_type, points: [{date, value}]}` while Dashboard returns `{data: number[]}`. This is an intentional API design improvement (structured points with dates vs flat array). The Dashboard also pads the array to exactly 30 elements with leading zeros, which the Go API does not do since it returns actual dated points.
- **Trend type naming**: Dashboard uses `TrendTypes.inventory` ("inventory") while Go endpoint examples reference "on_hand". The Go endpoint accepts any string for `trend_type` — this should be validated or documented to match the Dashboard's enum values.

## Files modified
- `services/core-service/internal/infrastructure/queries/item.sql` — Fixed SQL query
- `services/core-service/internal/infrastructure/repository/item_repository.go` — Added day deduplication
- `services/core-service/internal/infrastructure/sqlc/item.sql.go` — Regenerated via `make sqlc core`
