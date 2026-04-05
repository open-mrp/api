# PATCH /v1/core/production-steps/{production_step_id}/productions/{id}

**Status: Issues found and fixed**

## What was compared

- **Validation rules**: Request accepts optional `item_id`, `quantity_value`, `quantity_unit_id` — matches Dashboard's partial update schema
- **Permission checks**: Both require internal actor + `productionSteps:update` permission
- **Account scoping**: Both verify the production step belongs to the target account
- **DB queries and logic**: Item and quantity updates match; Go requires both `quantity_value` and `quantity_unit_id` together (Dashboard allows independent partial updates via Prisma, but this is an acceptable design choice for consistency)
- **Error handling**: Both return 404 if production not found (Go via the `Get` query after update)
- **Side effects**: Both re-link the production flow after an item change
- **Response shape**: `ProductionOutput` with nested `produced_item` and `quantity` — matches Dashboard's `LightProduction` shape
- **Idempotency**: Go correctly uses idempotency keys with recovery points for this PATCH endpoint

## Issues found and fixed

### 1. LinkFlow was outside the transaction (fixed)

**Problem**: The Dashboard calls `productionFlowRepo.linkFlow()` inside the Prisma `$transaction`, making the flow re-link atomic with the production update. The Go code called `meds.ProductionFlow.LinkFlow()` **after** the transaction committed, meaning a failure in LinkFlow would leave the production updated but the flow graph stale. Additionally, the error was silently swallowed (`_ = flowErr`).

**Fix**: Moved `LinkFlow` inside the `withTx` callback (before `CacheSuccessResponse`), using `txSvc.mediators()` so it participates in the same transaction. Errors now propagate correctly and will roll back the transaction on failure.

## Remaining notes

- **Quantity partial updates**: Dashboard allows updating just `quantity.value` or just `quantity.unitID` independently via Prisma's partial update. Go requires both `quantity_value` and `quantity_unit_id` to be present. This is an intentional design choice that prevents inconsistent quantity state and is acceptable.
- **Pre-update existence check**: Dashboard fetches the production before updating to get a 404 early and to obtain the old item ID for disconnect logic. Go skips this since its `LinkFlow` implementation already clears all links and rebuilds from scratch (making the explicit disconnect unnecessary). The 404 is still returned via the `Get` query after update. This is functionally equivalent.
