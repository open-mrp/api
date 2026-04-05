# PATCH /v1/core/rates/{id} — Migration Verification

**Status: Issues found and fixed**

## What was compared

| Aspect | Dashboard | Go | Match? |
|--------|-----------|-----|--------|
| Permission checks | `checkIsInternalActor` + item/productionStep update | `CheckIsInternalActor` + item/productionStep update | ✅ |
| Target account validation | Via identity | `CheckTargetAccountSet` | ✅ |
| Object type validation | `ObjectTypes` enum (item, productionStep) | `constants.ObjectType` switch (item, production_step) | ✅ |
| Object existence check | `itemRepo.checkExistence` / `productionStepRepo.checkExistence` | `ItemRepo.Get` / `ProductionStepRepo.Get` | ✅ |
| Partial update support | Prisma partial via `RateUtils.schema.partial()` | SQL `COALESCE(sqlc.narg(...), field)` | ✅ |
| Fields updated | `measure`, `numeratorUnit`, `denominatorUnit` | `value`, `numerator_unit_id`, `denominator_unit_id` | ✅ |
| Idempotency | N/A (Dashboard) | Idempotency keys with recovery points | ✅ |
| Error handling | Prisma not-found, permission errors | `RowsAffected` check, permission errors, resource not found | ✅ |
| Response shape | id, measure, numeratorUnit, denominatorUnit, createdAt, updatedAt | id, object, value, numerator_unit, denominator_unit, created_at, updated_at | ✅ (after fix) |
| Side effects | None | None | ✅ |

## Issues found and fixed

### 1. Missing `created_at` and `updated_at` in Rate API resource

The Dashboard returns `createdAt` and `updatedAt` on the Rate response. The Go domain model and proto already had these fields, but they were missing from:

- **`services/api-gateway/pkg/resource/rate_resource.go`** — Added `CreatedAt time.Time` and `UpdatedAt time.Time` fields to the `Rate` struct, plus sample data.
- **`services/api-gateway/endpoints/rates/presenter.go`** — Added mapping of `CreatedAt` and `UpdatedAt` from proto `RateInfo` timestamps.

## Notes

- The Go API uses `value` (decimal string) where Dashboard uses `measure` (number). This is consistent with Go API conventions for decimal precision.
- The Go API uses expandable sub-resources for `numerator_unit` and `denominator_unit` (default: `{id, object}`, expandable via `?include=`). Dashboard always returns full `BaseUnit` objects inline. This is a deliberate Go API convention difference, not a parity issue.
- The Go API adds the `object` field to the response per Go API conventions.
