# PUT /v1/core/integrations/{id} — Migration Verification

## Result: PARITY CONFIRMED

No issues found. The Go implementation faithfully reproduces all Dashboard business logic.

## What Was Compared

| Aspect | Dashboard | Go | Match |
|--------|-----------|-----|-------|
| HTTP method & status | PUT, 200 OK | PUT, 200 OK | Yes |
| Auth: internal actor | `checkIsInternalActor` | `CheckIsInternalActor()` | Yes |
| Auth: admin role | `roleTypeCode !== RoleTypes.admin` | `CheckIsAdmin()` | Yes |
| Auth: target account | `identity.targetAccountID` | `CheckTargetAccountSet()` | Yes |
| Updatable fields | `name` (optional), `isActive` (optional) | `name` (*string), `is_active` (*bool) | Yes |
| Protected fields | `code`, `credentials` not in update | Only name/is_active in SQL UPDATE | Yes |
| Account scoping | `where: { id, accountID }` | `WHERE id = ? AND account_id = ?` | Yes |
| Not found handling | Pre-check existence -> 404 | Rows affected == 0 -> 404 | Yes |
| SQL partial update | Prisma partial update | `COALESCE(sqlc.narg, column)` | Yes |
| Side effects | None | None | Yes |
| Response shape | id, name, code, isActive, createdAt, updatedAt | id, object, name, integration_code, is_active, created_at, updated_at | Yes |

## Notes

- Go adds standard `object` field to response (expected convention, not a parity issue).
- Go uses idempotency keys for this PUT endpoint. Per conventions, PUT should be idempotent by default, so this is extra safety rather than a requirement — not a parity concern.
- Field naming differences (camelCase vs snake_case, `code` vs `integration_code`) are expected cross-API convention differences.

## Files Reviewed

- **Dashboard**: `dashboard/apps/api/src/services/account-integration.svc.ts`, `repositories/account-integration.repo.ts`, `controllers/account-integration.ctrl.ts`
- **Go Gateway**: `api-gateway/endpoints/account-integrations/endpoint_update_account_integration.go`, `service.go`, `presenter.go`
- **Go Core**: `core-service/internal/service/account_integration_service.go`, `infrastructure/repository/account_integration_repository.go`, `infrastructure/queries/account_integration.sql`, `infrastructure/grpc/grpc_handler.go`
- **Domain**: `core-service/internal/domain/account_integration_models.go`
- **Resource**: `api-gateway/pkg/resource/account_integration_resource.go`
