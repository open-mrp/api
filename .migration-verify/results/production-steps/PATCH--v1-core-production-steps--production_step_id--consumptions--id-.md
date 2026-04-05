# PATCH /v1/core/production-steps/{production_step_id}/consumptions/{id}

## Status: Issues found and fixed

## What was compared

- **Permission checks**: Internal actor + `productionSteps:update` permission + target account header — matches Dashboard
- **Validation**: Production step in account, new item in account (when changing) — matches Dashboard
- **Updatable fields**: item_id, quantity (value+unit), waste_quantity (value+unit), instructions — matches Dashboard
- **Production flow side effects**: When item changes: disconnect old downstream step, update item, re-link flow (all in transaction) — matches Dashboard
- **Idempotency**: PATCH uses idempotency keys with recovery points — correct per Go conventions
- **Response shape**: Consumption resource with expandable consumed_item, quantity, waste_quantity — matches Dashboard
- **Error handling**: Not-found for missing production step or consumption — matches Dashboard
- **Transaction management**: Both item-change and non-item-change paths are transactional — matches Dashboard (Dashboard only uses transaction for item-change path, Go wraps both which is equivalent or safer)

## Issues found and fixed

### 1. Instructions cleared when only ItemID changes (Bug)

**Problem**: When `item_id` was provided but `instructions` was not, the Go code passed `nil` instructions to `UpdateItem`, which mapped to `sql.NullString{}` (NULL), clearing existing instructions. The Dashboard's Prisma update preserves fields not included in the partial update.

**Fix**: Added logic to fetch the current instructions via a new `GetInstructions` repository method when `params.Instructions` is nil before calling `UpdateItem`, preserving the existing value.

**Files changed**:
- `services/core-service/internal/service/consumption_service.go` — fetch current instructions when not provided
- `services/core-service/internal/domain/repositories.go` — added `GetInstructions` to `ConsumptionRepo` interface
- `services/core-service/internal/infrastructure/repository/consumption_repository.go` — implemented `GetInstructions`
- `services/core-service/internal/infrastructure/queries/consumption.sql` — added `GetConsumptionInstructions` query

### 2. Waste quantity could not be updated independently of main quantity (Bug)

**Problem**: The waste quantity update was nested inside the main quantity update condition (`if params.QuantityValue != nil && params.QuantityUnitID != nil`), making it impossible to update waste quantity without also providing main quantity. In the Dashboard, `quantity` and `wasteQuantity` are independent fields in the Prisma update.

**Fix**: Extracted a shared `updateConsumptionQuantities` helper that handles both quantity and waste quantity independently. Both are updated only when their respective value+unit pair is provided, but neither depends on the other.

**Files changed**:
- `services/core-service/internal/service/consumption_service.go` — extracted `updateConsumptionQuantities` helper, used in both code paths

## Remaining concerns

None. All generated code (sqlc, mocks) has been regenerated and tests pass.
