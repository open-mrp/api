# GET /v1/core/machines/{id} — Migration Verification

## Result: PARITY CONFIRMED

No issues found. The Go implementation matches the Dashboard business logic.

## What Was Compared

| Aspect | Dashboard | Go | Match |
|--------|-----------|-----|-------|
| Permission: internal actor | `checkIsInternalActor` | `identity.CheckIsInternalActor()` | Yes |
| Permission: machines/read | `checkHasPermission(machines, read)` | `CheckHasPermission(PermissionDomainMachines, ActionRead)` | Yes |
| Account scoping | `department: { accountID }` (Prisma) | `JOIN department d ... WHERE d.account_id = ?` | Yes |
| Query fields | id, serialNumber, name, notes, department(id, name), createdAt, updatedAt | id, name, serial_number, notes, department_id, department_name, created_at, updated_at | Yes |
| Not-found error | `HttpError.notFound('Machine not found.')` | `db.MapSQLError` (maps sql.ErrNoRows to not-found) | Yes |
| Response: department sub-resource | `BasicInfo { id, name }` | `Department { id, object, name }` (expandable) | Yes |
| Response: notes optional | `notes: string \| null` | `Notes *string` | Yes |
| Idempotency | N/A (GET) | N/A (GET) | Yes |
| Side effects | None | None | Yes |

## Details

- **Dashboard flow**: Controller extracts `machineID` from params → `MachineSvc.find({ id })` → checks internal actor + machines/read permission → `MachineRepo.find({ id, accountID })` → Prisma `findFirst` with `department: { accountID }` filter → `MachineAdapter.map()` → returns Machine with department as `BasicInfo`
- **Go flow**: API Gateway endpoint extracts ID from path → gRPC call to core-service → `MachineSvc.GetMachine(ctx, machineID)` → checks internal actor + machines/read + target account → `MachineRepo.Get()` → SQL query with JOIN on department for account scoping → presenter maps to API resource with Department sub-resource

No fixes needed.
