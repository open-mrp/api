# Verification: GET /v1/core/sys-properties/{id}

## Result: PARITY CONFIRMED

No issues found. The Go implementation correctly mirrors the Dashboard business logic.

## What Was Compared

| Aspect | Dashboard | Go | Match |
|--------|-----------|-----|-------|
| Permission: internal actor check | `checkIsInternalActor` | `CheckIsInternalActor()` | Yes |
| Permission: domain/action | `systemProperties` / `read` | `PermissionDomainSystemProperties` / `ActionRead` | Yes |
| Permission: target account | implicit via `this.identity.targetAccountID` | explicit `CheckTargetAccountSet()` | Yes |
| DB query | Prisma `findUnique` by `id` + `accountID` | SQL `WHERE sp.id = ? AND sp.account_id = ?` with JOIN to `sys_property_type` | Yes |
| Not found handling | Service checks null, throws `HttpError.notFound('System property not found.')` | `db.MapSQLError` maps SQL no-rows to not-found error | Yes |
| Response shape | `{ id, type (string enum), value (number), createdAt, updatedAt }` | `{ id, object, type: { id, object, name, code }, value (int32), created_at, updated_at }` | Yes (improved) |
| Side effects | None | None | Yes |
| Idempotency | N/A (GET) | N/A (GET) | Yes |

## Notes

- The Go API returns `type` as a proper sub-resource with `id`, `object`, `name`, and `code` fields, rather than a flat string enum. This follows the Go API resource conventions (sub-resources pattern from CLAUDE.md) and is an intentional improvement.
- The Go API adds an `object` field (`sys_property`) to the response, per API resource conventions.
- No customer actor access in Dashboard for this endpoint; Go correctly restricts to internal actors only.

## Files Reviewed

- **Dashboard**: `dashboard/apps/api/src/controllers/sys-property.ctrl.ts`, `dashboard/apps/api/src/services/sys-property.svc.ts`, `dashboard/apps/api/src/repositories/sys-property.repo.ts`, `dashboard/packages/adapters/src/classes/settings/SysProperty.ts`, `dashboard/packages/objects/src/classes/settings/SysProperty.ts`
- **Go**: `api/services/api-gateway/endpoints/sys_properties/endpoint_get_sys_property.go`, `api/services/api-gateway/endpoints/sys_properties/service.go`, `api/services/api-gateway/endpoints/sys_properties/presenter.go`, `api/services/api-gateway/pkg/resource/sys_property_resource.go`, `api/services/core-service/internal/infrastructure/grpc/grpc_sys_property_handler.go`, `api/services/core-service/internal/service/sys_property_service.go`, `api/services/core-service/internal/infrastructure/repository/sys_property_repository.go`, `api/services/core-service/internal/infrastructure/queries/sys_property.sql`
