# DELETE /v1/core/dc-locations/{id}

## Result: PARITY CONFIRMED

No issues found. The Go implementation correctly mirrors the Dashboard business logic.

## What Was Compared

| Aspect | Dashboard (Express.js) | Go API | Match? |
|--------|----------------------|--------|--------|
| Actor type check | `checkIsInternalActor` | `CheckIsInternalActor()` | Yes |
| Permission check | `ediRuns` / `delete` | `PermissionDomainEdiRuns` / `ActionDelete` | Yes |
| Ownership filter | Prisma `where: { id, ownerAccountID }` | SQL `WHERE id = ? AND owner_account_id = ?` | Yes |
| Not-found handling | Prisma throws if record missing | Existence check before delete + RowsAffected check | Yes (stricter) |
| Side effects | None | None | Yes |
| Transaction | Implicit (Prisma single op) | Explicit `withTx` wrapper | Yes |
| Response | HTTP 200 with OKAY body | HTTP 204 No Content (EmptyResource) | Acceptable |

## Notes

- The Go API returns 204 No Content instead of 200 OK. This is standard REST convention for DELETE and is an intentional design choice for the new API.
- The Go implementation adds an explicit existence check (`GetDCLocation`) before attempting deletion, providing a clearer not-found error. The Dashboard relies on Prisma to throw when the record doesn't exist. Both produce correct not-found behavior.
- No idempotency key is needed for DELETE (per CLAUDE.md: "All DELETE endpoints must be designed to be idempotent by default without idempotency keys"). The Go implementation correctly omits it.

## Files Reviewed

- **Dashboard**: `dashboard/apps/api/src/controllers/edi-dc-location.ctrl.ts`, `dashboard/apps/api/src/services/edi-dc-location.svc.ts`, `dashboard/apps/api/src/repositories/edi-dc-location.repo.ts`
- **Go API Gateway**: `services/api-gateway/endpoints/edi-dc-locations/endpoint_delete_dc_location.go`, `services/api-gateway/endpoints/edi-dc-locations/service.go`
- **Go Core Service**: `services/core-service/internal/infrastructure/grpc/grpc_edi_handler.go`, `services/core-service/internal/service/edi_service.go`, `services/core-service/internal/infrastructure/repository/edi_repository.go`, `services/core-service/internal/infrastructure/queries/edi.sql`
