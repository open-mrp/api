# POST /v1/core/dc-locations — Migration Verification

**Status: PARITY CONFIRMED** — No issues found.

## What Was Compared

### Dashboard Files
- `dashboard/apps/api/src/services/edi-dc-location.svc.ts` — Service logic
- `dashboard/apps/api/src/repositories/edi-dc-location.repo.ts` — Repository (Prisma)
- `dashboard/apps/api/src/controllers/edi-dc-location.ctrl.ts` — Controller
- `dashboard/packages/dtos/src/sections/edi.ts` + `dc-locations.ts` — DTO schemas
- `dashboard/packages/objects/src/classes/edi/EdiDCLocation.ts` — Object model

### Go Files
- `api/services/api-gateway/endpoints/edi-dc-locations/endpoint_create_dc_location.go` — Endpoint definition
- `api/services/api-gateway/endpoints/edi-dc-locations/service.go` — Gateway service
- `api/services/api-gateway/endpoints/edi-dc-locations/presenter.go` — Response presenter
- `api/services/api-gateway/pkg/resource/edi_resource.go` — API resource
- `api/services/core-service/internal/infrastructure/grpc/grpc_edi_handler.go` — gRPC handler
- `api/services/core-service/internal/service/edi_service.go` — Domain service
- `api/services/core-service/internal/infrastructure/repository/edi_repository.go` — Repository
- `api/services/core-service/internal/infrastructure/queries/edi.sql` — SQL queries

## Comparison Details

| Aspect | Result |
|--------|--------|
| Validation (required fields) | Match — customer_id and location both required |
| Permission checks (internal actor + ediRuns/create) | Match |
| Target account requirement | Match (Go adds explicit check) |
| DB insert (id, location, account_id, owner_account_id, timestamps) | Match |
| No uniqueness constraints checked | Match |
| Response shape (nested customer with id/name) | Match (Go adds object fields per convention) |
| Status code 201 Created | Match |
| No side effects (no emails, webhooks, messages) | Match |
| Idempotency | Go adds full idempotency key support (improvement) |

## Issues Found
None.

## Remaining Concerns
None.
