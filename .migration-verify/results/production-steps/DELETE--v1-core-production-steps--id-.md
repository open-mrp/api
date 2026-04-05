# DELETE /v1/core/production-steps/{id} — Verification Result

**Status: Parity confirmed with one fix applied**

## What was compared

| Aspect | Dashboard | Go | Match |
|--------|-----------|-----|-------|
| Permission: internal actor | `checkIsInternalActor` | `identity.CheckIsInternalActor()` | Yes |
| Permission: domain/action | `productionSteps` / `delete` | `PermissionDomainProductionSteps` / `ActionDelete` | Yes |
| Account scoping | `this.identity.targetAccountID` | `*identity.TargetAccountID` | Yes |
| Existence check | Prisma throws P2025 if not found | Explicit `IsInAccount` check → 404 | Yes (equivalent) |
| Parent-child link cleanup | Prisma ON DELETE CASCADE on junction table | Explicit `DeleteParentChildLinks` | Yes (equivalent) |
| Cascade deletes (consumptions, productions) | Prisma ON DELETE CASCADE | Database FK CASCADE constraints | Yes (equivalent) |
| Response | 200 OK with deleted LightProductionStep | 204 No Content (empty) | Intentional Go API convention |
| Idempotency keys | Not used (DELETE) | Not used (DELETE) | Yes |
| Side effects | None beyond DB delete | None beyond DB delete | Yes |

## Issue found and fixed

**Atomicity of delete operations**: The Go service performed two sequential operations without a transaction:
1. `DeleteParentChildLinks` — removes junction table rows
2. `Delete` — removes the production step row

If step 2 failed (e.g., due to other FK constraints), the parent-child links would already be deleted, causing data integrity loss. The Dashboard's Prisma `delete()` handles cascades atomically in a single DB operation.

**Fix**: Wrapped both operations in a `withTx` transaction so they succeed or fail together.

## Response shape difference (intentional)

The Dashboard returns 200 OK with the deleted `LightProductionStep` object. The Go API returns 204 No Content. This is a deliberate Go API convention — delete endpoints return empty responses. Not a parity issue.

## No remaining concerns

All business logic, permission checks, account scoping, and error handling are aligned.
