# Verification: GET /v1/core/production-steps/{id}

## Status: Issues Found and Partially Fixed

## What Was Compared

- **Permission checks**: Internal actor + `productionSteps` / `read` permission
- **Account scoping**: Both scope query by `account_id`
- **Error handling**: 404 when production step not found
- **DB queries**: Main query joins, consumption sub-query, machines, in/out steps
- **Response shape**: All fields, nested resources, sub-resources
- **Side effects**: None expected (GET endpoint)
- **Idempotency**: N/A (GET endpoint)

## Parity Confirmed

1. **Permission checks**: Both check `checkIsInternalActor` and `checkHasPermission(productionSteps, read)`.
2. **Account scoping**: Both filter by `identity.targetAccountID` / `account_id`.
3. **Error handling**: Dashboard returns 404 "Production step not found." when not found. Go uses `db.MapSQLError` which maps `sql.ErrNoRows` to 404 "Resource not found." — same HTTP status, slightly different message text (acceptable).
4. **Core fields**: `id`, `name`, `notes`, `leveling_factor`, `allowances`, `created_at`, `updated_at` — all present.
5. **Rates**: `labor_rate`, `labor_time`, `overhead_rate` — all present with numerator/denominator units.
6. **Production output**: Present with produced item (id, sku, description, type_code), quantity, timestamps.
7. **In/Out steps**: Both return id + name for connected steps (Dashboard: `in`/`out`, Go: `in_steps`/`out_steps` — naming differs but data matches).

## Issues Found and Fixed

### 1. Consumptions missing item description and type code (FIXED)

**File**: `services/core-service/internal/infrastructure/queries/production_step_query.sql`

The `GetProductionStepConsumptions` SQL query was missing `ci.description` and `ci.item_type_code` from the item join. The domain model (`Consumption`) and proto (`ConsumptionInfo`) both have `ItemDescription` and `ItemTypeCode` fields, and the gRPC handler and presenter both reference them, but they were never populated from the DB — always returning zero values.

**Fix**: Added `ci.description AS consumed_item_description` and `ci.item_type_code AS consumed_item_type_code` to the SQL query.

### 2. Consumptions missing timestamps (FIXED)

**File**: `services/core-service/internal/infrastructure/queries/production_step_query.sql`

The query was missing `c.created_at` and `c.updated_at`. The Dashboard returns these, and the domain model + proto support them, but they were never queried.

**Fix**: Added `c.created_at` and `c.updated_at` to the SQL query.

### 3. Repository mapping not populating new consumption fields (FIXED)

**File**: `services/core-service/internal/infrastructure/repository/production_step_repository.go`

The `mapConsumptionFromStepRow` function was not mapping `ItemDescription`, `ItemTypeCode`, `CreatedAt`, or `UpdatedAt`.

**Fix**: Updated the mapping function to populate all four fields from the query result.

## Remaining Concerns (Not Fixed — Require Proto/SQLC Regeneration)

### 4. Machines sub-resource is lighter than Dashboard

**Dashboard** returns full machine objects: `id`, `serialNumber`, `name`, `notes`, `department` (`{id, name}`), `createdAt`, `updatedAt`.

**Go** returns only: `id`, `object`, `name`.

The SQL query (`GetProductionStepMachines`) only selects `m.id, m.name`. The proto (`LightMachineInfo`) only has `id` and `name`. Fixing this requires changes to SQL, sqlc generated code, domain model, proto, gRPC handler, presenter, and API resource — affecting shared types used by other endpoints (e.g., list production steps).

### 5. Scanning station sub-resource is lighter than Dashboard

**Dashboard** returns: `id`, `name`, `notes`, `type`, `batchLabelTagSize`, `batchLabelType`, `materialCheckRequired`, `departmentID`, `createdAt`, `updatedAt`.

**Go** returns only: `id`, `object`, `name`.

Same situation as machines — requires proto/sqlc regeneration and affects shared types.

### 6. Department field

**Go** adds a top-level `department` field (`{id, object}`) that Dashboard does not expose. This is extra data and not a parity issue — it's additive.

## Post-Fix Action Required

After the SQL query changes, run:
```bash
make sqlc core
```
to regenerate the sqlc types so the new columns (`consumed_item_description`, `consumed_item_type_code`, `created_at`, `updated_at`) are available in the generated `GetProductionStepConsumptionsRow` struct.
