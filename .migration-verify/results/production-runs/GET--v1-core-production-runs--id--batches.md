# Verification: GET /v1/core/production-runs/{id}/batches

**Status: Issues found and fixed**

## What was compared

- **Permission checks**: Internal actor + productionRuns read permission — matches Dashboard ✓
- **Request parameters**: Path param `id` only, no query params — matches Dashboard ✓
- **BFS graph traversal algorithm**: Both implementations use identical logic — always traverse downstream, only traverse upstream for active branches (open or leading to open batches) ✓
- **Account scoping**: All queries scoped to `identity.targetAccountID` ✓
- **Response shape**: Compared field-by-field
- **Idempotency**: GET endpoint, no idempotency keys needed ✓
- **Side effects**: None in either implementation ✓

## Issues found and fixed

### 1. Missing `department_name` field
**Dashboard**: Returns `departmentName` derived from `scanningStation.department.name`.
**Go (before)**: Did not include department name.
**Fix**: Added `LEFT JOIN department d ON ss.department_id = d.id` to `GetBatch` and `GetBatchBase` SQL queries. Added `DepartmentName *string` to domain `Batch` model. Propagated through proto (`department_name` field 18), gRPC handler, presenter, and API resource.

### 2. Missing `lots` field
**Dashboard**: Returns `lots` array with `{ lotNumber, type }` objects. Lots are derived from:
  - `inventory_issue.lot.lotNumber` → type "material"
  - `inventory_issue.allocations[].receipt.lot.lotNumber` → type "material"
  - `productionRun.number` → type "productionRun"
  With deduplication by lotNumber.
**Go (before)**: Did not include lots data.
**Fix**: Added `GetBatchLots` SQL query using UNION of inventory_issue lots and allocation receipt lots. Added `BatchLot` struct to domain. Production run lot number added in repository logic. Propagated through proto (`BatchLotInfo` message, `lots` field 19), gRPC handler, presenter, and API resource.

### 3. Missing `input_batch_ids` and `output_batch_ids` fields
**Dashboard**: Returns `inputBatchIDs` and `outputBatchIDs` arrays from the batch flow graph (`_batch_flow` table).
**Go (before)**: Did not include flow IDs on batches returned by this endpoint (though `BatchFlowNode` existed for the flow endpoint).
**Fix**: Added `InputBatchIDs` and `OutputBatchIDs` to domain `Batch` model. Fetched using existing `GetBatchFlowIncoming`/`GetBatchFlowOutgoing` queries in `ListBatchesByRun`. Propagated through proto (fields 20, 21), gRPC handler, presenter, and API resource.

## Files modified

- `services/core-service/internal/infrastructure/queries/batch.sql` — Added department JOIN to GetBatch/GetBatchBase, added GetBatchLots query
- `services/core-service/internal/domain/batch_models.go` — Added BatchLot struct, DepartmentName/Lots/InputBatchIDs/OutputBatchIDs to Batch
- `proto/core.proto` — Added BatchLotInfo message, new fields to BatchInfo
- `services/core-service/internal/infrastructure/repository/batch_repository.go` — Extract department_name in mapBatchRow
- `services/core-service/internal/infrastructure/repository/production_run_repository.go` — Fetch lots and flow IDs in ListBatchesByRun
- `services/core-service/internal/infrastructure/grpc/grpc_batch_handler.go` — Convert new fields in batchToProto
- `services/api-gateway/endpoints/batches/presenter.go` — Convert new proto fields in BatchPresenter
- `services/api-gateway/pkg/resource/batch_resource.go` — Added BatchLot type, new fields to Batch resource

## Remaining notes

- The Go API uses sub-resources (e.g. `production_step: {id, object, name}`) where Dashboard uses flat IDs (e.g. `productionStepID`). This is by design per Go API conventions.
- The Go API wraps the response in `List[Batch]` with pagination info, while Dashboard returns a flat array. This is standard Go API convention.
- The Go API includes `updated_at` which Dashboard does not — this is additional data, not a parity issue.
- Pre-existing build errors in unrelated files (invoice_repo, item_repository, sales_order_repo, shipment_repository) are not related to this endpoint.
