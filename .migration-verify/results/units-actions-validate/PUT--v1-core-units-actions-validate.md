# Migration Verification: PUT /v1/core/units/actions/validate

## Result: Parity Confirmed

No issues found. The Go implementation correctly replicates the Dashboard business logic.

## What Was Compared

| Aspect | Dashboard | Go | Verdict |
|--------|-----------|-----|---------|
| Permission checks | `checkIsAssignedActor` | `CheckIsAssignedActor` + `units:read` for internal users | Go is stricter (intentional improvement) |
| Account filtering | `OR: [{accountID}, {accountID: null}]` | `account_id = ? OR account_id IS NULL` | Parity |
| Case-insensitive matching | MySQL collation + JS `.toLowerCase()` | SQL `LOWER()` + Go `strings.ToLower()` | Parity |
| Missing keys handling | Omitted from response | Omitted from response | Parity |
| Empty input handling | Prisma handles empty `in` array | Early return with empty map | Parity |
| Side effects | None | None | Parity |
| Response shape | `Record<string, Unit>` | `{ "object": "map", "units": { ... } }` | Go convention (object field wrapping) |

## Intentional Design Changes (Not Parity Issues)

- **HTTP method**: POST (Dashboard) -> PUT (Go) — semantically correct for a read-only idempotent action
- **Route prefix**: `/v1/units/` -> `/v1/core/units/` — expected migration prefix
- **Request body**: Direct map (Dashboard) -> wrapped in `{ "unit_map": ... }` (Go) — Go API convention
- **Response body**: Direct map (Dashboard) -> wrapped with `object` field (Go) — Go API resource convention
- **Permission check**: Go adds `units:read` permission check for internal users — security improvement

## Files Reviewed

### Dashboard
- `dashboard/apps/api/src/services/unit.svc.ts` — `validateUnits` method (line 16)
- `dashboard/apps/api/src/repositories/unit.repo.ts` — `validateUnits` method (line 111)
- `dashboard/apps/api/src/controllers/unit.ctrl.ts` — `validateUnits` controller (line 8)

### Go
- `services/api-gateway/endpoints/units/endpoint_validate_units.go` — endpoint definition
- `services/core-service/internal/service/unit_service.go` — `ValidateUnits` method (line 375)
- `services/core-service/internal/infrastructure/repository/unit_repository.go` — `FindByAbbreviations`
- `services/core-service/internal/infrastructure/queries/unit.sql` — `FindUnitsByAbbreviations` query
- `services/api-gateway/pkg/resource/unit_resource.go` — `ValidateUnitsResponse` resource

## Issues Found and Fixed

None — no fixes were needed.
