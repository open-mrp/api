# Verification: GET /v1/core/edi-runs

## Status: PARITY CONFIRMED

No issues found. The Go implementation correctly preserves all business logic from the Dashboard Express.js API.

## What Was Compared

| Aspect | Result |
|--------|--------|
| Permission checks | Match — internal actor, ediRuns domain, read action |
| Account scoping | Match — queries filtered by identity's target account ID |
| hasSucceeded filter | Match — optional boolean filter applied identically |
| Ordering | Match — completedAt DESC (Go adds id DESC as tie-breaker for cursor pagination) |
| Response fields | Match — id, completedAt, hasSucceeded, createdAt, updatedAt (Go adds standard `object` field) |
| Error handling | Match — proper not-found and validation errors |
| Side effects | N/A — read-only endpoint, no side effects in either implementation |
| Idempotency | N/A — GET endpoint, idempotent by design |

## Expected Migration Differences (Not Issues)

- **Pagination**: Dashboard uses offset-based (take/skip/count), Go uses cursor-based pagination — this is the standard Go API convention
- **Route prefix**: `/v1/edi-runs` → `/v1/core/edi-runs` — per migration guidelines
- **`query` param**: Dashboard accepts but never uses it in the DB query; Go omits it — correct removal of dead parameter
- **`object` field**: Go adds `"object": "edi_run"` to response — required by Go API resource conventions

## Files Reviewed

### Dashboard
- `dashboard/apps/api/src/controllers/edi-run.ctrl.ts`
- `dashboard/apps/api/src/services/edi-run.svc.ts`
- `dashboard/apps/api/src/repositories/edi-run.repo.ts`
- `dashboard/packages/adapters/src/classes/edi/EdiRun.ts`
- `dashboard/packages/dtos/src/sections/edi.ts`

### Go
- `api/services/api-gateway/endpoints/edi-runs/endpoint_list_edi_runs.go`
- `api/services/api-gateway/endpoints/edi-runs/service.go`
- `api/services/api-gateway/endpoints/edi-runs/presenter.go`
- `api/services/api-gateway/pkg/resource/edi_resource.go`
- `api/services/core-service/internal/service/edi_service.go`
- `api/services/core-service/internal/infrastructure/grpc/grpc_edi_handler.go`
- `api/services/core-service/internal/infrastructure/repository/edi_repository.go`
- `api/services/core-service/internal/infrastructure/queries/edi.sql`
- `api/services/core-service/internal/domain/edi_models.go`
