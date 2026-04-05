# Verification: POST /v1/core/production-steps/{production_step_id}/consumptions

**Status: Parity confirmed — no fixes needed**

## What was compared

### Permission checks
- **Dashboard**: `checkIsInternalActor` + `checkHasPermission(productionSteps, 'create')`
- **Go**: `identity.CheckIsInternalActor()` + `identity.CheckHasPermission(PermissionDomainProductionSteps, ActionCreate)` + `identity.CheckTargetAccountSet()`
- **Result**: Match. Go additionally validates target account is set (standard Go API pattern).

### Account & entity validation
- **Dashboard**: `productionStepRepo.isInAccount()` and `itemRepo.isInAccount()` — returns 404 if either fails
- **Go**: `repos.NewProductionStepQueryRepo().IsInAccount()` and `repos.NewItemRepo().Get()` — returns not-found error if either fails
- **Result**: Match. Both verify production step and item belong to the target account.

### Required fields / validation
- **Dashboard**: Zod schema validates id, quantity (measure + unit), wasteQuantity (measure + unit), consumedItem (with itemID), instructions (nullable, defaults to null). ID is client-generated.
- **Go**: Struct tags require `item_id`, `quantity_value`, `quantity_unit_id`, `waste_quantity_value`, `waste_quantity_unit_id`. `instructions` is optional. IDs are server-generated.
- **Result**: Equivalent. Go uses flat field names (API convention) vs Dashboard's nested objects. Server-side ID generation in Go is an improvement.

### DB operations
- **Dashboard**: Prisma creates consumption with nested quantity and wasteQuantity relations, connecting item and productionStep. Note: `instructions` is NOT passed in `ConsumptionAdapter.fetchCreateInput()` (silently dropped on create).
- **Go**: Inserts two `quantity` rows, then inserts `consumption` row with all fields including `instructions`. Fetches the created consumption afterward.
- **Result**: Match. Go correctly passes `instructions` on create (fixing a Dashboard oversight where instructions are silently dropped).

### Side effects
- **Dashboard**: Within `prisma.$transaction()`: create consumption → `productionFlowRepo.linkFlow()`
- **Go**: Within `s.withTx()`: create consumption → `mediators().ProductionFlow.LinkFlow()`
- **Result**: Match. Both atomically create the consumption and rebuild production flow connections in a single transaction.

### Idempotency
- **Dashboard**: No idempotency key support
- **Go**: Full idempotency key support with recovery points (RecoveryPointStarted → RecoveryPointFinished), response caching, and error caching
- **Result**: Go adds idempotency as required by architecture patterns. This is an expected improvement.

### Response shape
- **Dashboard**: Returns `LightConsumption` (id, quantity, wasteQuantity, consumedItem, instructions) — no timestamps, no object field
- **Go**: Returns `Consumption` resource (id, object, quantity, waste_quantity, consumed_item, instructions, created_at, updated_at) with sub-resources following API conventions
- **Result**: Go response is a superset. Includes `object` type and timestamps per API resource conventions. Sub-resources (consumed_item, quantity) follow the `{id, object, ...}` pattern.

### Error handling
- **Dashboard**: "Production step not found." (step), "Production block not found." (item)
- **Go**: "Production step not found." (step), generic not-found from `db.MapSQLError` (item)
- **Result**: Minor wording difference for item-not-found error message. Both return 404 status. Acceptable.

### HTTP status code
- **Dashboard**: 201 Created
- **Go**: 201 Created (`http.StatusCreated`)
- **Result**: Match.

## Summary

The Go implementation faithfully reproduces all Dashboard business logic for consumption creation. The core flow — permission checks, account scoping, entity validation, atomic creation with production flow linking — is equivalent. Differences are all additive improvements aligned with Go API conventions (server-side IDs, idempotency, richer response shape, instructions on create).
