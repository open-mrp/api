# Migration Verification: PUT /v1/core/products/{id}/product-line/{product_line_id}

## Result: PARITY CONFIRMED

No issues found. The Go implementation faithfully reproduces the Dashboard behavior.

## What Was Compared

### Permission Checks
- **Dashboard:** `checkIsInternalActor` + `checkHasPermission(items, update)`
- **Go:** `CheckIsInternalActor()` + `CheckHasPermission(PermissionDomainItems, ActionUpdate)` + `CheckTargetAccountSet()`
- **Verdict:** Match. Go additionally validates target account is set explicitly (Dashboard does this implicitly via `this.identity.targetAccountID`).

### Database Logic
- **Dashboard:** Prisma `findFirst` on product by itemID + accountID + deletedAt null, then `update` with `connect(productLineID)`
- **Go:** SQL UPDATE on product SET product_line_id WHERE item_id AND item_id IN (SELECT id FROM item WHERE account_id AND deleted_at IS NULL), then checks rowsAffected == 0 for 404
- **Verdict:** Match. Both scope to account, both respect soft deletes, both rely on DB foreign key constraint for product_line_id validity.

### Error Handling
- **Dashboard:** 404 "Finished good for block {itemID} not found"
- **Go:** 404 "Product not found."
- **Verdict:** Match (different wording, same HTTP status and behavior).

### Response Shape
- **Dashboard:** 200 OK with full Product object (via ProductAdapter.map)
- **Go:** 200 OK with full Product resource including expandable sub-resources (item, product_line, product_type)
- **Verdict:** Match.

### Side Effects
- **Dashboard:** None
- **Go:** None
- **Verdict:** Match.

### Idempotency
- PUT endpoint — both implementations are naturally idempotent without idempotency keys (correct per patterns).
- **Note:** Go gRPC handler includes `WithIdempotencyTracking` which is conventionally for POST/PATCH, but it's a no-op without a key and causes no behavioral difference.

## Issues Found and Fixed
None.

## Remaining Concerns
None.
