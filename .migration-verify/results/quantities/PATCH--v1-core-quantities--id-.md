# PATCH /v1/core/quantities/{id} — Migration Verification

## Result: PARITY CONFIRMED

No issues found. The Go implementation correctly mirrors all Dashboard business logic.

## What Was Compared

### Permission & Auth Checks
- **Dashboard**: `checkIsInternalActor`, then switches on `objectType` — `item` requires `items:update`, `productionStep` requires `productionSteps:update`, else throws bad request.
- **Go**: `CheckIsInternalActor()`, `checkObjectPermission()` with same mapping, `CheckTargetAccountSet()`. **Match.**

### Object Existence Verification
- **Dashboard**: Calls `itemRepo.checkExistence` or `productionStepRepo.checkExistence` based on object type.
- **Go**: `verifyObjectExists()` calls `ItemRepo.Get` or `ProductionStepRepo.Get`. **Match.**

### Update Logic
- **Dashboard**: `QuantityRepo.update()` updates `measure` and `unit` (via Prisma connect on unit ID). Partial — only provided fields change.
- **Go**: SQL uses `COALESCE(sqlc.narg('value'), value)` and `COALESCE(sqlc.narg('unit_id'), unit_id)` to preserve unchanged fields. After update, re-fetches with unit JOIN. **Match.**

### Validation
- **Dashboard**: Zod validates `data` as `QuantityUtils.schema.partial()` (measure: finite number, unit: BaseUnit object), `objectID` and `objectType` required.
- **Go**: `value` and `unit_id` are optional pointer fields; `object_id` and `object_type` are `validate:"required"`. **Match** (Go accepts `unit_id` directly rather than full unit object — correct since only the ID is needed for the foreign key).

### Error Handling
- **Dashboard**: Invalid object type → `HttpError.badRequest('Invalid object type')`. Nonexistent parent → not found from `checkExistence`. Nonexistent quantity → Prisma throws not found.
- **Go**: Invalid object type → `NewValidationErrorWithParam("Invalid object type.", "object_type")`. Nonexistent parent → not found from `verifyObjectExists`. Nonexistent quantity → `rowsAffected == 0` returns `NewResourceNotFoundError`. **Match.**

### Idempotency
- **Go** correctly implements idempotency keys with recovery points for this PATCH endpoint (required by conventions). Dashboard did not have this — Go is an improvement.

### Side Effects
- **Dashboard**: None.
- **Go**: None. **Match.**

### Response Shape
- **Dashboard** returns: `{ id, measure (number), unit (full BaseUnit), createdAt, updatedAt }`
- **Go** returns: `{ id, object, value (decimal string), display_value, unit (expandable sub-resource) }`
- Field mapping changes (`measure` → `value` as decimal string, `object` field added, `display_value` added) follow Go API conventions.
- Go omits `created_at`/`updated_at` from the Quantity resource — this is consistent with how Quantity is treated as a value-type sub-resource across the Go API. Timestamps are still stored and could be exposed if needed.

## Files Reviewed

**Dashboard:**
- `dashboard/apps/api/src/controllers/quantity.ctrl.ts`
- `dashboard/apps/api/src/services/quantity.svc.ts`
- `dashboard/apps/api/src/repositories/quantity.repo.ts`
- `dashboard/packages/dtos/src/sections/measures.ts`
- `dashboard/packages/objects/src/classes/measures/BaseQuantity.ts`
- `dashboard/packages/adapters/src/classes/measures/Quantity.ts`

**Go:**
- `services/api-gateway/endpoints/quantities/endpoint_update_quantity.go`
- `services/api-gateway/endpoints/quantities/service.go`
- `services/api-gateway/endpoints/quantities/presenter.go`
- `services/api-gateway/pkg/resource/quantity_resource.go`
- `services/core-service/internal/service/measure_service.go`
- `services/core-service/internal/infrastructure/grpc/grpc_measure_handler.go`
- `services/core-service/internal/infrastructure/repository/quantity_repository.go`
- `services/core-service/internal/infrastructure/queries/quantity.sql`
- `services/core-service/internal/domain/quantity_models.go`
