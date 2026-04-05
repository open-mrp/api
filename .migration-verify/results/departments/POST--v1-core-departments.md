# Verification: POST /v1/core/departments

**Status: Parity confirmed — no issues found.**

## What was compared

| Aspect | Dashboard | Go | Match |
|--------|-----------|-----|-------|
| Permission: internal actor check | `checkIsInternalActor` | `CheckIsInternalActor()` | Yes |
| Permission: domain/action | `departments / create` | `PermissionDomainDepartments / ActionCreate` | Yes |
| Target account required | Implicit via `identity.targetAccountID` | Explicit `CheckTargetAccountSet()` | Yes |
| Request fields | name (required), notes, location, scanningStations, machines | name (required), notes, location_id, scanning_station_ids, machine_ids | Yes |
| DB insert | Prisma `create` with id, name, notes, account, location | SQL INSERT with id, name, notes, location_id, account_id, NOW(3) timestamps | Yes |
| Scanning station association | Prisma `connect` by station IDs | UPDATE scanning_station SET department_id WHERE id IN (...) AND account_id | Yes (Go adds account_id filter — stricter) |
| Machine association | Prisma `connect` by machine IDs | UPDATE machine SET department_id WHERE id IN (...) | Yes |
| Duplicate name check | None | ExistsByName → 409 conflict | Go is stricter (enhancement) |
| Idempotency | None | Full idempotency key with recovery points | Enhancement per migration pattern |
| Response: status code | 201 Created | 201 Created | Yes |
| Response: fields | id, name, notes, location, scanningStations, machines, createdAt, updatedAt | id, object, name, notes, location, scanning_stations, machines, created_at, updated_at | Yes (new API conventions) |
| Response: sub-resources | location as `{id, name} | null`; stations/machines as arrays of `{id, name}` | location as `{id, object, name} | null`; stations/machines as List with `{id, object, name}` | Yes (new resource conventions add object field + list wrapper) |
| Side effects | None | None | Yes |

## Enhancements in Go (acceptable, not regressions)

1. **Duplicate name check**: Go validates that no department with the same name exists in the account before creating. Dashboard does not. This prevents accidental duplicates.
2. **Idempotency**: Go implements full idempotency key support with recovery points, as required by the migration pattern.
3. **Account-scoped scanning station update**: Go's `SetScanningStationsDepartmentID` SQL includes `AND account_id = ?`, which is stricter than Prisma's `connect` (ID-only). This prevents cross-account association.

## Files reviewed

- Dashboard: `department.ctrl.ts`, `department.svc.ts`, `department.repo.ts`, `Department.ts` (adapter)
- Go: `endpoint_create_department.go`, `service.go` (gateway), `presenter.go`, `department_resource.go`, `grpc_department_handler.go`, `department_service.go`, `department_repository.go`, `department.sql`
