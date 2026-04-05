# POST /v1/core/production-flows/actions/connect-steps

## Result: Parity Confirmed

No issues found. The Go implementation correctly mirrors the Dashboard business logic.

## What Was Compared

### Permission Checks
- **Dashboard:** `checkIsInternalActor` + `checkHasPermission(productionSteps, 'update')`
- **Go:** `identity.CheckIsInternalActor()` + `identity.CheckHasPermission(PermissionDomainProductionSteps, ActionUpdate)` + `identity.CheckTargetAccountSet()`
- **Verdict:** Match. Go additionally explicitly validates target account is set (good practice).

### Validation / Existence Checks
- **Dashboard:** Service calls `productionStepRepo.checkExistence({ id, accountID })` for both source and target steps. This queries `productionStep.findFirst({ where: { id, accountID } })` and throws `HttpError.notFound('Production step not found.')` if not found. The repo also re-checks source step via `doesExist` + `isInAccount`.
- **Go:** Service calls `stepQueryRepo.IsInAccount(ctx, accountID, stepID)` for both source and target. Returns `ResourceNotFoundError("Source production step not found.")` / `"Target production step not found."`.
- **Verdict:** Match. Both validate that both steps belong to the caller's account before proceeding. Go provides more specific error messages (source vs target), which is an improvement.

### Database Operation
- **Dashboard:** `db.productionStep.update({ where: { id: sourceStepID, accountID }, data: { out: PrismaUtils.connect(targetStepID) } })` — Prisma adds a row to the `_parent_child_production_steps` many-to-many junction table.
- **Go:** `INSERT IGNORE INTO _parent_child_production_steps (A, B) VALUES (source_id, target_id)` — directly inserts into the same junction table, silently ignoring duplicates.
- **Verdict:** Match. Both write to the same `_parent_child_production_steps` table with columns A (source/parent) and B (target/child). The Go `INSERT IGNORE` is explicitly idempotent at the SQL level.

### Idempotency
- **Dashboard:** No explicit idempotency key handling (PUT endpoint, naturally idempotent via Prisma connect which is a no-op if the relation already exists).
- **Go:** Full idempotency key support via `contracts.WithIdempotencyTracking` and recovery points (`RecoveryPointStarted` / `RecoveryPointFinished`). Uses `INSERT IGNORE` for SQL-level idempotency. This is a POST endpoint per Go API conventions.
- **Verdict:** Go correctly implements idempotency keys as required for POST endpoints. This is an expected enhancement for the migration.

### Response Shape
- **Dashboard:** Returns empty object `{}`
- **Go:** Returns `EmptyResource{}` with HTTP 200
- **Verdict:** Match.

### Side Effects
- **Dashboard:** None (no emails, webhooks, or messages)
- **Go:** None
- **Verdict:** Match.

### Request Shape
- **Dashboard:** `{ sourceProductionStepID: string, targetProductionStepID: string }`
- **Go:** `{ source_production_step_id: string, target_production_step_id: string }` (with `validate:"required"` on both)
- **Verdict:** Match. Field naming follows Go API snake_case convention.

## Issues Found
None.
