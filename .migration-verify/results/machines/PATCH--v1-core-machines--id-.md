# PATCH /v1/core/machines/{id} — Migration Verification

## Result: PARITY CONFIRMED

No issues found. The Go implementation faithfully reproduces the Dashboard business logic.

## What Was Compared

| Aspect | Dashboard | Go | Match |
|--------|-----------|-----|-------|
| Permission: internal actor | `checkIsInternalActor` | `identity.CheckIsInternalActor()` | Yes |
| Permission: machines:update | `checkHasPermission(machines, update)` | `identity.CheckHasPermission(PermissionDomainMachines, ActionUpdate)` | Yes |
| Target account required | Via `identity.targetAccountID` | `identity.CheckTargetAccountSet()` | Yes |
| Updatable fields | name, serialNumber, notes (all optional) | name, serial_number, notes (all optional via COALESCE) | Yes |
| Account scoping | WHERE department.accountID = ? | JOIN department ON ... WHERE d.account_id = ? | Yes |
| Machine existence check | Prisma throws if not found | Explicit Get + RowsAffected check | Yes |
| Response shape | {id, serialNumber, name, notes, department: {id, name}, createdAt, updatedAt} | Same fields + `object` field per Go conventions | Yes |
| Department sub-resource | BasicInfo {id, name} | Department {id, object, name, notes: null, location: null} | Yes (convention) |
| Idempotency | Not applicable (Dashboard) | Idempotency key support via recovery points | Yes (Go convention) |
| HTTP status | 200 OK | 200 OK | Yes |

## Notable Differences (Non-Issues)

1. **Name uniqueness check (Go addition):** Go checks `ExistsByName` with `excludeID` before updating. Dashboard does not have this check. This is extra strictness consistent with the Go Create endpoint — it prevents updating to a duplicate name but won't reject updates that don't change the name. This is a reasonable data integrity guardrail.

2. **`object` field:** Go responses include `"object": "machine"` per API resource conventions. Expected difference.

3. **Department sub-resource shape:** Dashboard returns BasicInfo `{id, name}`. Go returns full Department resource `{id, object, name, notes, location}` with null for unpopulated fields. This follows Go API sub-resource conventions.

## Files Reviewed

### Dashboard
- `dashboard/apps/api/src/services/machine.svc.ts` — Service update method
- `dashboard/apps/api/src/repositories/machine.repo.ts` — Repository update method
- `dashboard/apps/api/src/controllers/machine.ctrl.ts` — Controller handler
- `dashboard/packages/adapters/src/classes/manufacturing/Machine.ts` — MachineAdapter (select/map)
- `dashboard/packages/dtos/src/sections/machines.ts` — Zod request/response schemas
- `dashboard/packages/objects/src/classes/manufacturing/Machine.ts` — Machine schema definition

### Go
- `api/services/api-gateway/endpoints/machines/endpoint_update_machine.go` — Endpoint definition
- `api/services/api-gateway/endpoints/machines/service.go` — Gateway service
- `api/services/api-gateway/endpoints/machines/presenter.go` — Response presenter
- `api/services/api-gateway/pkg/resource/machine_resource.go` — API resource
- `api/services/core-service/internal/infrastructure/grpc/grpc_fulfillment_service_handler.go` — gRPC handler
- `api/services/core-service/internal/service/machine_service.go` — Business logic
- `api/services/core-service/internal/infrastructure/repository/machine_repository.go` — Repository
- `api/services/core-service/internal/infrastructure/queries/machine.sql` — SQL queries
- `api/services/core-service/internal/domain/machine_models.go` — Domain models
