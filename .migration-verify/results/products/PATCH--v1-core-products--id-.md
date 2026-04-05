# PATCH /v1/core/products/{id} — Migration Verification

## Result: Issues found and fixed

## What was compared

- **Permission checks**: Internal actor + `items`/`update` permission — matches Dashboard
- **Updatable fields**: `sku`, `description`, `notes`, `is_portal_ready` — matches Dashboard
- **SKU uniqueness check**: Conflict error if duplicate SKU in account — matches Dashboard
- **Product existence check**: 404 if product not found — matches Dashboard
- **Idempotency**: PATCH uses idempotency keys with recovery points — correct
- **Response shape**: Product with nested item, product_line, product_type sub-resources — correct
- **Side effects**: None in either implementation — matches
- **SQL queries**: Two UPDATE statements (item + product tables) with conditional field updates — correct pattern
- **Error handling**: 404/409/401/403 error types — matches Dashboard

## Issues found and fixed

### Bug: `UpdateDescription` and `UpdateNotes` always set to `true`

**File**: `services/api-gateway/endpoints/products/service.go` (lines 132, 134)

**Problem**: The API gateway always set `UpdateDescription: true` and `UpdateNotes: true` when building the gRPC request. This meant if a user sent a PATCH with only `{ "sku": "NEW-SKU" }` (without `description` or `notes`), the SQL would set `description` and `notes` to NULL, clearing existing values.

In the Dashboard, Prisma's `undefined` handling means absent fields are simply not updated. The Go SQL uses `CASE WHEN update_description = true THEN ... ELSE description END` to conditionally update, but the flag was always true.

**Fix**: Changed to `UpdateDescription: req.Description != nil` and `UpdateNotes: req.Notes != nil`, consistent with the materials endpoint which already uses this pattern.

## Remaining concerns

None. The Go implementation now matches Dashboard behavior for all compared aspects.
