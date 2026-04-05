# POST /v1/core/batches/{id}/next-steps

## Status: Issues found and fixed

## What was compared
- Validation rules (required scanning_station_id in body, batch ID in path)
- Permission checks (internal actor, batches read permission, target account ID)
- BFS traversal algorithm for finding possible next production steps
- Consumption counting logic for isMultiPart flag
- Response shape (array of {id, name, is_multi_part})
- No side effects in either implementation (read-only endpoint)
- No idempotency keys needed (POST but read-only action)

## Issues found and fixed

### 1. Algorithm mismatch (fixed)
**Dashboard**: Forward-only BFS from the given batch. For unclosed batches that have been scanned and have a production step, collects child production steps matching the scanning station. For closed batches, follows outgoing batch flow edges and continues traversal. Collects results from ALL qualifying unclosed batches.

**Go (before fix)**: Used `FindFurthestRightBatchInFlow` which traverses the entire flow graph (both directions), then picks the single most recently scanned non-closed batch and gets child steps from only that one batch.

**Fix**: Rewrote `FindPossibleNextSteps` to match the Dashboard's forward-only BFS algorithm. Added a lightweight `GetBatchFlowTraversalInfo` SQL query to avoid expensive joins during traversal.

### 2. Consumption counting filter (fixed)
**Dashboard**: Counts only consumptions where the related item has `typeCode = 'part'` (i.e., `item_type_code = 'part'`). `isMultiPart` is true when this count > 1.

**Go (before fix)**: Counted ALL consumptions regardless of item type via `CountProductionStepConsumptions`.

**Fix**: Added new SQL query `CountProductionStepPartConsumptions` that joins consumption with item and filters by `item_type_code = 'part'`. Updated the repository to use this new query.

## Files modified
- `services/core-service/internal/infrastructure/queries/batch.sql` — added `GetBatchFlowTraversalInfo` query
- `services/core-service/internal/infrastructure/queries/production_step_query.sql` — added `CountProductionStepPartConsumptions` query
- `services/core-service/internal/infrastructure/repository/batch_repository.go` — rewrote `FindPossibleNextSteps` with correct BFS algorithm
- `services/core-service/internal/infrastructure/sqlc/` — regenerated via `make sqlc core`

## Remaining concerns
- None. The endpoint, service layer, gRPC handler, and API gateway all look correct. Permission checks match (internal actor + batches read).
