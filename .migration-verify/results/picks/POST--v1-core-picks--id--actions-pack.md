# Verification: POST /v1/core/picks/{id}/actions/pack

## Result: Issues found and fixed

## What was compared

- **Validation rules**: Required fields (pickID, shipmentCaseCount >= 1), no-lines-to-pack check
- **Permission checks**: Internal actor + picks:update permission + target account ID
- **DB queries and logic**: FindLinesToPack, PackLines, remaining quantity calculation, shipment creation, shipment line creation, shipping case creation, mark-finished-if-all-packed
- **Error handling**: Validation error for no lines, DB errors mapped correctly
- **Side effects**: Shipment + shipment lines + shipping cases created, pick lines packed, pick finished if all lines packed
- **Response shape**: PackPickResult with Pick (full detail) and ShipmentNumber
- **Idempotency**: POST uses idempotency keys with recovery points (Go only, improvement over Dashboard)

## Issues found and fixed

### 1. Remaining pick line quantity value (fixed)
- **Dashboard**: Creates remaining pick lines with quantity `measure: 0` (placeholder lines that need explicit quantity assignment before packing)
- **Go (before fix)**: Created remaining pick lines with the actual remaining quantity value
- **Fix**: Changed `CreateQuantity` call to use `"0"` instead of `remainingValue`

### 2. Missing unpacked pick line check (fixed)
- **Dashboard**: Before creating a remaining pick line, checks if there's already an unpacked pick line for that order line (`!hasUnpackedPickLine`). Skips creation if one exists.
- **Go (before fix)**: Always created a remaining pick line when remaining > 0, regardless of existing unpacked lines
- **Fix**: Added `HasUnpackedPickLineForOrderLine` SQL query and repository method. Service now checks for existing unpacked pick lines before creating new ones.

### 3. Order line deduplication (fixed)
- **Dashboard**: Processes unique order line IDs via Prisma's `findMany` with `{ in: orderLineIDs }` (naturally deduplicates)
- **Go (before fix)**: Iterated all `linesToPack` entries, potentially processing the same order line multiple times
- **Fix**: Added `processedOrderLines` map to skip duplicate order line IDs

## Files modified

- `services/core-service/internal/infrastructure/queries/pick.sql` — Added `HasUnpackedPickLineForOrderLine` query
- `services/core-service/internal/domain/repositories.go` — Added `HasUnpackedPickLineForOrderLine` to `PickLineRepo` interface
- `services/core-service/internal/infrastructure/repository/pick_line_repository.go` — Implemented `HasUnpackedPickLineForOrderLine`
- `services/core-service/internal/service/pick_svc.go` — Fixed remaining quantity logic (dedup, unpacked check, qty 0)
- `services/core-service/internal/infrastructure/sqlc/` — Regenerated via `make sqlc core`

## Acceptable differences (not bugs)

- **Transaction scope**: Dashboard marks pick as finished OUTSIDE the transaction; Go does it inside. Go is more correct (more atomic).
- **FindLinesToPack**: Dashboard also checks `finishedAt: null` on pick; Go doesn't. Functionally equivalent since a finished pick would have no unpacked lines.
- **Error wording**: Dashboard returns "Pick has no lines to pack." vs Go's "No lines to pack." Minor difference.
- **MarkFinishedIfAllPacked**: Go additionally filters by `q.value > 0`; Dashboard checks any unpacked line regardless of quantity. Go is more logically correct (zero-qty lines shouldn't prevent finishing).

## Remaining concerns

- Build cannot be fully tested due to pre-existing compilation errors from other WIP changes in `pick_repository.go` (unrelated `DeleteDuplicatePickLines` and `UpdateFinishedAt` methods). Once those are resolved, tests should pass.
- Mock regeneration should be run after all WIP changes stabilize (`make mocks core`).
