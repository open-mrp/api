# Verification: GET /v1/core/production-steps

**Result: Issue found and fixed**

## What was compared

| Aspect | Dashboard | Go | Match? |
|--------|-----------|-----|--------|
| Permission: internal actor check | `checkIsInternalActor` | `types.CheckIsInternalActor` | Yes |
| Permission: domain/action | `productionSteps / read` | `PermissionDomainProductionSteps / ActionRead` | Yes |
| Account scoping | `identity.targetAccountID` | `identity.TargetAccountID` | Yes |
| Filter: query (fulltext search) | `PrismaUtils.getQuery(query)` on name | `MATCH(ps.name) AGAINST(... IN BOOLEAN MODE)` | Yes |
| Filter: itemIDs | OR across productions AND consumptions | **Was production only** — fixed | Fixed |
| Filter: machineIDs | `machines: { some: { id: { in } } }` | `EXISTS (machine m WHERE m.production_step_id = ps.id AND m.id IN ...)` | Yes |
| Filter: scanningStationIDs | `scanningStationID: { in }` | `ps.scanning_station_id IN (...)` | Yes |
| Filter: inputStepIDs | `in: { some: { id: { in } } }` | `EXISTS (_parent_child_production_steps WHERE A = ps.id AND B IN ...)` | Yes |
| Filter: outputStepIDs | `out: { some: { id: { in } } }` | `EXISTS (_parent_child_production_steps WHERE B = ps.id AND A IN ...)` | Yes |
| Filter: startDate/endDate | `createdAt: { gte, lte }` | `ps.created_at >= / <=` | Yes |
| Pagination | Offset-based (take/skip) | Cursor-based | Intentional architectural change |
| Ordering | Relevance (when query), none otherwise | `created_at DESC, id DESC` always | Acceptable (cursor pagination requires deterministic sort) |
| Response shape | `LightProductionStep` (subset of fields) | Full `ProductionStep` (superset) | Go returns more data — acceptable |
| Side effects | None | None | Yes |
| Idempotency | N/A (GET) | N/A (GET) | Yes |

## Issue found and fixed

**Item filter missing consumption check**: The Go SQL `item_ids` filter only matched against `p.item_id` (the produced item). The Dashboard uses an OR clause that also matches items in the `consumptions` table. A production step should match the item filter if the item is either produced OR consumed by that step.

**Fix**: Added an `EXISTS` subquery to both `ListProductionStepsForward` and `ListProductionStepsBackward` SQL queries to also check `consumption.item_id`:

```sql
AND (
    sqlc.arg('include_item_filter') = false
    OR p.item_id IN (sqlc.slice('item_ids'))
    OR EXISTS (
        SELECT 1 FROM consumption c
        WHERE c.production_step_id = ps.id
        AND c.item_id IN (sqlc.slice('item_ids'))
    )
)
```

**Files modified**:
- `services/core-service/internal/infrastructure/queries/production_step_query.sql` — both forward and backward list queries
- `services/core-service/internal/infrastructure/sqlc/production_step_query.sql.go` — regenerated via `make sqlc core`

## Accepted differences (not parity gaps)

1. **Cursor vs offset pagination**: The Go API uses cursor-based pagination as an architectural decision across all list endpoints. This is intentional.
2. **Response richness**: The Go API returns the full `ProductionStep` resource (with notes, department, machines, timestamps, full scanning station object) while the Dashboard returned a `LightProductionStep`. Returning more data is acceptable and not a regression.
3. **Sort order**: The Dashboard sorted by relevance when a search query was provided. The Go API always sorts by `created_at DESC` which is required for deterministic cursor-based pagination.
