# POST /v1/core/production-steps — Migration Verification

## Result: Issues found and fixed

## What Was Compared

- **Authorization**: Internal actor check, permission domain (`productionSteps` / `create`), target account requirement
- **Validation**: Required fields, optional fields, request shape
- **DB mutations**: Rate inserts, step insert, production + quantity insert, consumption creation
- **Side effects**: Production flow linking (auto-connecting steps based on consumed/produced items)
- **Response shape**: Full ProductionStep resource with nested rates, production, consumptions, machines, scanning station, in/out steps, department
- **HTTP status code**: 201 Created
- **Idempotency**: Go properly uses idempotency keys with recovery points (expected improvement over Dashboard)
- **Error handling**: Error types and messages

## Issues Found and Fixed

### 1. Name uniqueness check not present in Dashboard (FIXED)

**Dashboard behavior**: The single `create` method in `ProductionStepSvc` does **not** check for duplicate production step names. The DB schema has no unique constraint on `(name, account_id)` for `production_step`. Duplicate names are allowed.

**Go behavior (before fix)**: The Go service had an `ExistsByName` check that would reject creation if a step with the same name already existed, returning a 409 Conflict error.

**Fix**: Removed the `ExistsByName` check from `production_step_service.go` `CreateProductionStep` to match Dashboard behavior. This prevents a breaking change where requests that succeed on Dashboard would fail on Go.

**Note**: The Dashboard's `bulkCreate` does check for existing names (to decide create vs update), but the single `create` does not.

## No Issues (Confirmed Parity)

- **Auth checks**: Both check `isInternalActor` and `hasPermission(productionSteps, create)`. Go additionally checks `targetAccountSet` (standard Go pattern). ✓
- **Transaction scope**: Both wrap the core creation in a transaction. ✓
- **Rate creation**: Both create labor_rate, labor_time, overhead_rate with value + numerator/denominator unit IDs. ✓
- **Production output**: Both create a production record with item_id, quantity (value + unit). ✓
- **Consumptions**: Both create consumption records with item_id, quantity, waste_quantity. Go additionally supports `instructions` (optional, nil by default — backward compatible). ✓
- **Flow linking**: Both call `linkFlow` outside the main transaction. Go marks it as non-fatal. Both implementations find input steps (steps producing consumed parts) and output steps (steps consuming the produced item), clear existing links, then reconnect. ✓
- **Response shape**: Both return the full production step with all nested data (rates, production, consumptions, machines, scanning station, in/out steps). Go additionally returns `notes`, `department`, `scanning_station` which are optional and backward compatible. ✓
- **HTTP 201 Created**: Both return 201. ✓

## Additional Go Features (Backward Compatible)

- `notes`, `scanning_station_id`, `department_id` fields accepted on create (optional, not in Dashboard create but present in the DB schema)
- `instructions` field on consumptions (optional)
- Idempotency key support with recovery points
- These are additive and do not break existing Dashboard behavior.
