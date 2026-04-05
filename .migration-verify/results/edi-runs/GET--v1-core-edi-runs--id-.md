# GET /v1/core/edi-runs/{id}

## Result: PARITY CONFIRMED

No issues found. The Go implementation matches the Dashboard business logic.

## What Was Compared

| Aspect | Dashboard | Go | Match |
|--------|-----------|-----|-------|
| Actor type check | `checkIsInternalActor` | `CheckIsInternalActor()` | Yes |
| Permission check | `ediRuns` / `read` | `PermissionDomainEdiRuns` / `ActionRead` | Yes |
| Account scoping | `where: { id, accountID }` | `WHERE er.id = ? AND er.account_id = ?` | Yes |
| Fields returned | id, completedAt, hasSucceeded, createdAt, updatedAt | id, object, completed_at, has_succeeded, created_at, updated_at | Yes (+object field per convention) |
| Not found handling | 404 "EDI run not found." | `db.MapSQLError` → 404 | Yes |
| Idempotency | N/A (GET) | N/A (GET) | Yes |
| Side effects | None | None | Yes |

## Files Reviewed

**Dashboard:**
- `dashboard/apps/api/src/services/edi-run.svc.ts` — `EdiRunSvc.find()`
- `dashboard/apps/api/src/repositories/edi-run.repo.ts` — `EdiRunRepo.find()`
- `dashboard/apps/api/src/controllers/edi-run.ctrl.ts` — Controller wiring
- `dashboard/packages/objects/src/classes/edi/EdiRun.ts` — Response shape / adapter select

**Go:**
- `services/api-gateway/endpoints/edi-runs/endpoint_get_edi_run.go` — Endpoint definition
- `services/api-gateway/endpoints/edi-runs/service.go` — Gateway service
- `services/api-gateway/endpoints/edi-runs/presenter.go` — Proto → resource conversion
- `services/api-gateway/pkg/resource/edi_resource.go` — API resource definition
- `services/core-service/internal/infrastructure/grpc/grpc_edi_handler.go` — gRPC handler
- `services/core-service/internal/service/edi_service.go` — Business logic / auth checks
- `services/core-service/internal/infrastructure/repository/edi_repository.go` — Data access
- `services/core-service/internal/infrastructure/queries/edi.sql` — SQL query

## Issues Found

None.

## Remaining Concerns

None.
