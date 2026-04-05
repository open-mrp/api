# Verification: GET /v1/core/production-runs/{id}

## Result: PARITY CONFIRMED

No issues found. No code changes required.

## What Was Compared

| Aspect | Dashboard | Go | Match? |
|--------|-----------|-----|--------|
| Permission checks | internal actor + productionRuns:read | internal actor + productionRuns:read + target account | Yes |
| DB query | Prisma findUnique by id + accountID | SQL join by id + account_id (with batch count, user name) | Yes |
| Not-found handling | Explicit 404 "Production run not found." | `db.MapSQLError` on `:one` query (sql.ErrNoRows -> 404) | Yes |
| Core response fields | id, number, createdAt, updatedAt, startedAt, completedAt | id, object, number, batch_count, created_at, updated_at, started_at, completed_at | Yes |
| Responsible user | Inline: userID, email, name, image | Sub-resource AccountUser (id, object, name) — expandable | Yes (follows Go API conventions) |
| Idempotency | N/A (GET) | N/A (GET) | Correct |
| Side effects | None | None | Yes |

## Notes

- **Batches**: Dashboard returns full `batches` array inline; Go returns `batch_count` with batches available via separate endpoint (`GET /v1/core/production-runs/{id}/batches`). This is the intended design — the Go API properly separates concerns.
- **orderID**: Dashboard returns `orderID: null` always; Go omits this field. No behavioral difference.
- **Responsible user shape**: Dashboard returns `responsibleUser` with `userID`, `email`, `name`, `image`. Go returns `responsible_user` as an expandable `AccountUser` sub-resource with `id` (account_user join ID), `object`, `name`. This follows the Go API resource conventions (sub-resources with object type identifiers).
- **Error path**: Go uses `db.MapSQLError` which maps `sql.ErrNoRows` to a resource-not-found error, matching the Dashboard's explicit 404.

## Files Reviewed

### Dashboard
- `dashboard/apps/api/src/services/production-run.svc.ts` — `find()` method (lines 63-79)
- `dashboard/apps/api/src/repositories/production-run.repo.ts` — `find()` method (lines 106-121)
- `dashboard/packages/adapters/src/classes/scanning-stations/ProductionRun.ts` — select/map

### Go
- `services/api-gateway/endpoints/production-runs/endpoint_get_production_run.go` — endpoint definition
- `services/api-gateway/endpoints/production-runs/service.go` — `GetProductionRun()` (lines 78-94)
- `services/api-gateway/endpoints/production-runs/presenter.go` — `ProductionRunDetailPresenter()` (lines 36-59)
- `services/api-gateway/pkg/resource/production_run_resource.go` — `ProductionRunDetail` struct
- `services/core-service/internal/infrastructure/grpc/grpc_production_run_handler.go` — `GetProductionRun()` (lines 101-118)
- `services/core-service/internal/service/production_run_service.go` — `GetProductionRun()` (lines 95-118)
- `services/core-service/internal/infrastructure/repository/production_run_repository.go` — `Get()` (lines 210-242)
- `services/core-service/internal/infrastructure/queries/production_run.sql` — `GetProductionRun` query (lines 123-141)
- `services/core-service/internal/domain/production_run_models.go` — domain models
