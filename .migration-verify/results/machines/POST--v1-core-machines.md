# POST /v1/core/machines — Migration Verification

**Status: PARITY CONFIRMED** — No issues found.

## What Was Compared

| Aspect | Dashboard (Express.js) | Go API | Match? |
|--------|----------------------|--------|--------|
| Validation (required fields) | name, serialNumber, department | name, serial_number, department_id | Yes |
| Validation (optional fields) | notes (nullable) | notes (optional pointer) | Yes |
| Permission: internal actor | checkIsInternalActor | CheckIsInternalActor | Yes |
| Permission: domain/action | machines / create | PermissionDomainMachines / ActionCreate | Yes |
| Target account required | Implicit (via department FK) | Explicit CheckTargetAccountSet | Yes |
| DB insert fields | id, name, serialNumber, notes, department | id, name, serial_number, notes, department_id | Yes |
| Department validation | Prisma FK constraint | Explicit department Get + account scope | Yes (stricter) |
| Name uniqueness | Not checked | ExistsByName check | Stricter |
| Response status | 201 Created | 201 Created | Yes |
| Response fields | id, name, serialNumber, notes, department, createdAt, updatedAt | id, object, name, serial_number, notes, department, created_at, updated_at | Yes (+object) |
| Department in response | { id, name } | { id, object, name } | Yes (+object) |
| Side effects | None | None | Yes |
| Idempotency | None | Idempotency keys with recovery points | Enhanced |

## Intentional Differences (Go API Conventions)

1. **`object` field** in response — standard Go API convention per `api-resource-conventions.md`
2. **Snake_case field names** — Go API convention (`serial_number` vs `serialNumber`)
3. **Server-side ID generation** — Go API always generates IDs; Dashboard allowed client-provided IDs
4. **Explicit department ownership check** — Go verifies department belongs to account (defensive)
5. **Name uniqueness check** — Go prevents duplicate machine names per account (defensive)
6. **Idempotency key support** — Go POST endpoints use idempotency keys per architecture patterns

## Issues Found

None. All business logic is preserved with appropriate Go API conventions applied.
