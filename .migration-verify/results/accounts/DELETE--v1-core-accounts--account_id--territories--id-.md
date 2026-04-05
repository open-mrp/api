# DELETE /v1/core/accounts/{account_id}/territories/{id}

## Result: PARITY CONFIRMED

No issues found. The Go implementation correctly mirrors the Dashboard business logic.

## Comparison

| Aspect | Dashboard | Go | Match |
|--------|-----------|-----|-------|
| Permission: internal actor | `checkIsInternalActor` | `CheckIsInternalActor` | Yes |
| Permission: domain + action | territories / delete | PermissionDomainTerritories / ActionDelete | Yes |
| Account scoping | `identity.targetAccountID` | `params.AccountID` (path) + `CheckTargetAccountSet` | Yes |
| DB query | `DELETE WHERE accountID = ? AND id = ?` | `DELETE WHERE id = ? AND account_id = ?` | Yes |
| Not-found handling | Prisma throws RecordNotFound | Explicit `IsInAccount` check → 404 | Yes |
| Delete type | Hard delete | Hard delete | Yes |
| Side effects | None | None | Yes |
| Idempotency keys | N/A (DELETE) | N/A (DELETE) | Yes |
| Transaction | Supported via context | `withTx` wrapper | Yes |

## Minor Differences (Acceptable)

- **Response code:** Dashboard returns 200 with the deleted territory object; Go returns 204 No Content. This is consistent with the Go API's established convention for DELETE endpoints and is a valid REST pattern.
- **Account ID source:** Dashboard uses `identity.targetAccountID` directly; Go uses `account_id` from the URL path but validates via `CheckTargetAccountSet` and `IsInAccount`. Both prevent cross-account access.

## Files Reviewed

- `dashboard/apps/api/src/services/territory.svc.ts`
- `dashboard/apps/api/src/repositories/territory.repo.ts`
- `dashboard/apps/api/src/controllers/territory.ctrl.ts`
- `api/services/api-gateway/endpoints/territories/endpoint_delete_territory.go`
- `api/services/api-gateway/endpoints/territories/service.go`
- `api/services/core-service/internal/infrastructure/grpc/grpc_territory_handler.go`
- `api/services/core-service/internal/service/territory_service.go`
- `api/services/core-service/internal/infrastructure/repository/territory_repository.go`
- `api/services/core-service/internal/infrastructure/queries/territory.sql`
