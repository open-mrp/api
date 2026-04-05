# POST /v1/core/sales-orders/{id}/actions/create-production-run

## Result: Issues found and fixed

## What was compared

| Aspect | Dashboard | Go (before fix) | Go (after fix) | Status |
|--------|-----------|-----------------|----------------|--------|
| **Actor check** | `checkIsInternalActor` | `CheckIsInternalActor()` | No change | ✅ |
| **Permission** | `productionRuns / create` | `salesOrders / update` | `productionRuns / create` | ✅ Fixed |
| **Target account** | `identity.targetAccountID` | `CheckTargetAccountSet()` | No change | ✅ |
| **Account user lookup** | Finds by userID + accountID | `FindByAccountAndUserID` | No change | ✅ |
| **Order existence check** | `orderRepo.find()` → 404 | `orderRepo.Get()` → 404 | No change | ✅ |
| **Duplicate run check** | Not explicit | Checks `order.ProductionRunID != nil` | No change | ✅ (Go is stricter) |
| **Production run creation** | Auto-incremented number | `GetNextNumber` + `Create` | No change | ✅ |
| **Batch creation** | Batches linked to production run | `production_run_id` was NULL | Passes `ProductionRunID` | ✅ Fixed |
| **Order → run link** | `order.update({ productionRunID })` | `SetProductionRunID` | No change | ✅ |
| **Material demand** | Recursive BOM explosion | Recursive BOM explosion | No change | ✅ |
| **Inventory reservation** | `status_code = 'reserved'`, linked to order | `status_code = 'open'`, no order link | `status_code = 'reserved'`, linked to order | ✅ Fixed |
| **Response shape** | `{ productionRun: { id } }` | `{ production_run_id }` | `{ production_run: { id } }` | ✅ Fixed |
| **HTTP status** | 201 Created | 201 Created | No change | ✅ |
| **Idempotency** | None (Prisma tx only) | Full idempotency key support | No change | ✅ (Go is stricter) |
| **Atomicity** | Prisma `$transaction` | `withTx` + idempotency | No change | ✅ |

## Issues found and fixed

### 1. Permission check (Critical)
- **Dashboard**: `checkHasPermission(identity, PermissionDomains.productionRuns, 'create')`
- **Go (before)**: `CheckHasPermission(types.PermissionDomainSalesOrders, types.ActionUpdate)`
- **Fix**: Changed to `types.PermissionDomainProductionRuns, types.ActionCreate` in `sales_order_service.go`

### 2. Batches not linked to production run (Critical)
- **Dashboard**: Batches are created with `productionRunID` set to the new run
- **Go (before)**: `CreateBatchParams` had no `ProductionRunID` field; repo hardcoded `production_run_id` to NULL
- **Fix**: Added `ProductionRunID` to `CreateBatchParams`, updated `batch_repository.go` to use `db.NullString(params.ProductionRunID)`, and passed `runID` from the service

### 3. Inventory reservation status and order link (Critical)
- **Dashboard**: Creates inventory issues with `status_code = 'reserved'` and `order_id` linked to the sales order via `InventoryIssueRepo.reserveMaterialsForProductionRun`
- **Go (before)**: Used `UpdateInventory` which creates issues with `status_code = 'open'` and no order link
- **Fix**: Added `CreateMaterialReservation` method to `InventoryReservationRepo` that uses the existing `InsertInventoryIssueForReservation` SQL query with `status_code = 'reserved'` and `order_id` set. Added `CreateMaterialReservationParams` domain model. Updated service to use the new method.

### 4. Response shape (Medium)
- **Dashboard**: Returns `{ productionRun: { id: "pr_..." } }`
- **Go (before)**: Returns `{ production_run_id: "pr_..." }`
- **Fix**: Changed response struct to use nested object `{ production_run: { id: "pr_..." } }` per codebase conventions

## Files modified

- `services/core-service/internal/service/sales_order_service.go` — permission check, batch creation params, reservation method
- `services/core-service/internal/domain/batch_models.go` — added `ProductionRunID` to `CreateBatchParams`
- `services/core-service/internal/domain/repositories.go` — added `CreateMaterialReservation` to `InventoryReservationRepo`
- `services/core-service/internal/domain/inventory_mutation_models.go` — added `CreateMaterialReservationParams`
- `services/core-service/internal/infrastructure/repository/batch_repository.go` — use `ProductionRunID` param
- `services/core-service/internal/infrastructure/repository/inventory_reservation_repository.go` — implemented `CreateMaterialReservation`
- `services/api-gateway/endpoints/sales-orders/endpoint_create_production_run.go` — response shape
- `services/api-gateway/endpoints/sales-orders/service.go` — response mapping

## Remaining concerns

1. **Batch source difference**: The Dashboard creates batches from production flow analysis (`getBaseItems` — traverses production flow graph to find material-only production steps), while Go creates batches directly from order lines (`GetLinesForBOM`). This means the batches represent different things: Dashboard batches = work units at leaf production steps; Go batches = ordered product lines. This is a structural difference that may affect downstream production step execution. A deeper investigation of how these batches are consumed by the production flow would be needed to determine impact.

2. **Mock regeneration needed**: The `InventoryReservationRepo` mock will need regeneration via `make mocks core` to include the new `CreateMaterialReservation` method.
