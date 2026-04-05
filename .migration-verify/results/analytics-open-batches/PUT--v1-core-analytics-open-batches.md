# PUT /v1/core/analytics/open-batches

**Status: Issues found and fixed**

## What was compared

- **Validation**: Request schema matches — both accept optional `itemIDs` and `productLineIDs` arrays
- **Permission checks**: Both require internal actor with `batches:read` permission and target account ID — match
- **DB queries and logic**: Discrepancies found and fixed (see below)
- **Error handling**: Standard error propagation — match
- **Side effects**: None in either implementation — match
- **Response shape**: Go returns structured sub-resources (`item`, `scanning_station`) with `object` field per API conventions; Dashboard returns flat fields (`itemName`, `itemID`, `scanningStationID`). This is an intentional API design improvement in Go, not a parity issue.
- **Idempotency**: PUT endpoint — idempotent by design, no idempotency keys needed — match

## Issues found and fixed

### 1. Missing output batch quantity subtraction (FIXED)

**Dashboard behavior**: For each open batch, the net quantity is calculated as `batch.quantity - sum(output batch quantities)`. Output batches are linked via the `_batch_flow` join table (A = parent, B = child). This means the count reflects only the *remaining* quantity at a scanning station after material has flowed out to downstream steps.

**Go behavior (before fix)**: The SQL query used `SUM(q.value)` which counted the *total* quantity without subtracting output flows. This would overcount material at each station.

**Fix**: Added a LEFT JOIN subquery on `_batch_flow` + output batch quantities, changing `SUM(q.value)` to `SUM(q.value - COALESCE(out_totals.out_sum, 0))`.

### 2. Missing scanning station NOT NULL filter (FIXED)

**Dashboard behavior**: Filters `scanningStationID: { not: null }` — only includes batches that are assigned to a scanning station.

**Go behavior (before fix)**: No such filter existed. Batches without a scanning station would be included in results (grouped under NULL scanning_station_id).

**Fix**: Added `AND b.scanning_station_id IS NOT NULL` to the WHERE clause.

### 3. Product line filtering not implemented (PRE-EXISTING, NOT FIXED)

**Dashboard behavior**: When `productLineIDs` are provided, resolves them to part IDs via `ProductionFlowRepo.findPartsByProduct()` (recursive traversal of production flow). These part IDs are then used to filter batches.

**Go behavior**: Has a TODO comment: "Restore product_line_ids filter once product_line table exists in schema." The `productLineIDs` parameter is accepted but ignored (the `_` parameter in the repository method signature). This was a pre-existing gap noted in the original migration.

## Files modified

- `services/core-service/internal/infrastructure/queries/batch.sql` — Updated `ListOpenBatches` query
- `services/core-service/internal/infrastructure/sqlc/batch.sql.go` — Regenerated via `make sqlc core`
