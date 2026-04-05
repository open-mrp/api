# GET /v1/core/machines — Migration Verification

## Result: Parity Confirmed

No issues found. The Go implementation faithfully reproduces the Dashboard business logic.

## What Was Compared

### Permission Checks
- **Dashboard**: `checkIsInternalActor` + `checkHasPermission(machines, read)`
- **Go**: `CheckIsInternalActor()` + `CheckHasPermission(PermissionDomainMachines, ActionRead)` + `CheckTargetAccountSet()`
- **Verdict**: Match. Go adds an explicit target-account check (standard pattern).

### Search/Query Filter
- **Dashboard**: `MachineAdapter.fetchInput` applies `{ OR: [{ name: { startsWith: query } }, { name: { equals: query } }] }` — effectively a prefix match on `name`.
- **Go**: SQL applies `m.name LIKE query + '%'` — also a prefix match on `name`.
- **Verdict**: Match.

### Account Scoping
- **Dashboard**: Filters via `department: { accountID }` (Prisma relation filter).
- **Go**: `JOIN department d ON d.id = m.department_id WHERE d.account_id = ?`
- **Verdict**: Match.

### Response Shape / Fields
- **Dashboard**: `{ items: Machine[], count }` — Machine has: id, name, serialNumber, notes, department (id, name), createdAt, updatedAt.
- **Go**: Standard list resource with `data` array + `page_info` — Machine has: id, object, name, serial_number, notes, department (id, object, name, expandable), created_at, updated_at.
- **Verdict**: Match. Go adds `object` type field per API conventions. Department is a proper sub-resource.

### Pagination
- **Dashboard**: Offset-based (take/skip) with total count.
- **Go**: Cursor-based with page_info.
- **Verdict**: Intentional architectural change — Go API uses cursor-based pagination across all list endpoints.

### Sort Order
- **Dashboard**: `ORDER BY name ASC`
- **Go**: `ORDER BY created_at DESC, id DESC`
- **Verdict**: Expected change — cursor-based pagination requires sorting by a stable sequential field (created_at). This is the standard Go API pattern.

### Customer Actor Access
- **Dashboard**: Internal actors only.
- **Go**: Internal actors only.
- **Verdict**: Match.

### Side Effects
- Neither implementation has side effects for the list endpoint.
- **Verdict**: Match.

### Idempotency
- GET endpoint — inherently idempotent, no idempotency keys needed.
- **Verdict**: Correct.

## No Issues Found
