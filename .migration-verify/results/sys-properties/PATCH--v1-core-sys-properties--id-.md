# PATCH /v1/core/sys-properties/{id} — Verification Result

## Status: PARITY CONFIRMED

No issues found. The Go implementation correctly matches the Dashboard Express.js behavior.

## What Was Compared

### Permission Checks
- **Dashboard**: `checkIsInternalActor(identity)` + `checkHasPermission(identity, PermissionDomains.systemProperties, 'update')`
- **Go**: `types.CheckIsInternalActor(identity)` + `types.CheckHasPermission(identity, types.PermissionDomainSystemProperties, types.ActionUpdate)` + target account ID check
- **Result**: Match ✓

### Validation
- **Dashboard**: Zod schema validates `propertyID` (string path param) and partial SysProperty body (only `value: z.number()` used)
- **Go**: `SysPropertyID` (path param) and `Value int32` (body field)
- **Result**: Match ✓ — Both accept a numeric value field. Go uses int32 which is appropriate since sys properties are integer counters.

### DB Query & Logic
- **Dashboard**: Prisma `update()` with `WHERE id AND accountID`, sets only `value` field
- **Go**: SQL `UPDATE sys_property SET value = ?, updated_at = NOW(3) WHERE id = ? AND account_id = ?`
- **Result**: Match ✓ — Both scope by account ID (tenant isolation) and only update the value. Go also updates `updated_at` explicitly which is correct.

### Error Handling
- **Dashboard**: Prisma throws if record not found (P2025)
- **Go**: Checks `RowsAffected()` and returns 404 "System property not found" if 0 rows updated
- **Result**: Match ✓ — Both return not-found errors when the record doesn't exist.

### Response Shape
- **Dashboard**: Returns `{ id, type, value }` with Entity base fields (createdAt, updatedAt)
- **Go**: Returns `SysProperty` resource with `id`, `object` ("sys_property"), `type` (sub-resource with id/object/name/code), `value`, `created_at`, `updated_at`
- **Result**: Match ✓ — Go follows the API resource conventions by adding `object` field and making `type` a proper sub-resource.

### Idempotency
- **Dashboard**: No idempotency key support
- **Go**: Full idempotency key support via recovery points (RecoveryPointStarted/Finished)
- **Result**: Go improvement ✓ — PATCH endpoints must support idempotency keys per project conventions.

### Side Effects
- **Dashboard**: None
- **Go**: None
- **Result**: Match ✓

## Issues Found and Fixed
None — no changes were needed.

## Remaining Concerns
None.
