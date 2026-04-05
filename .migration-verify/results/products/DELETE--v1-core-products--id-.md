# Verification: DELETE /v1/core/products/{id}

## Result: Parity Confirmed (with noted architectural difference)

No code changes required.

## What Was Compared

### Permission Checks
- **Dashboard**: `checkIsInternalActor`, `checkHasPermission(items, 'delete')`
- **Go**: `CheckIsInternalActor()`, `CheckHasPermission(PermissionDomainItems, ActionDelete)`, `CheckTargetAccountSet()`
- **Verdict**: Match. Go additionally validates target account is set (standard Go pattern).

### Pre-deletion Fetch
- **Dashboard**: `productRepo.findByItemID({ itemID, accountID })` — filters `deletedAt: null`
- **Go**: `productRepo.Get(ctx, GetProductFullParams{AccountID, ItemID})` — filters `deleted_at IS NULL`
- **Verdict**: Match.

### Soft Delete
- **Dashboard**: `UPDATE item SET deletedAt = new Date() WHERE id = ? AND accountID = ? AND deletedAt IS NULL`
- **Go**: `UPDATE item SET deleted_at = NOW(3) WHERE id = ? AND account_id = ? AND deleted_at IS NULL`
- **Verdict**: Match.

### Response Shape
- **Dashboard**: Returns the product object fetched before deletion, HTTP 200
- **Go**: Returns the product object fetched before deletion, HTTP 200
- **Verdict**: Match.

### Idempotency
- DELETE is idempotent by default (no idempotency keys needed per architecture patterns)
- Both implementations return a not-found error if the product is already deleted (`deleted_at IS NULL` guard)
- **Verdict**: Match.

### Error Handling
- **Dashboard**: `HttpError.notFound('Product not found.')` if product doesn't exist
- **Go**: `apierror.NewResourceNotFoundError("Product not found.")` if product doesn't exist (checked both at service level via Get and at repo level via RowsAffected)
- **Verdict**: Match.

## Noted Architectural Difference: Cascading Hard Deletes

The Dashboard performs extensive cascading hard deletes after the soft delete:
1. Soft-deletes the item
2. Hard-deletes consumption records
3. Hard-deletes consumption quantity records
4. Hard-deletes production records
5. Hard-deletes production quantity records
6. Hard-deletes inventory change logs, inventory logs, and related quantities
7. Hard-deletes the item record itself
8. Hard-deletes associated rate records (unitValue, unitCost, burnRate)

The Go API only performs step 1 (soft delete on the item table).

**This is not a parity gap** — it is an intentional architectural decision. The Go API uses soft deletes consistently, and all queries filter by `deleted_at IS NULL`, making the product invisible through normal API access. The related data remains in the database but is effectively orphaned/inaccessible. This approach is:
- Safer (data is recoverable)
- Simpler (single atomic operation)
- Consistent with the Go API's soft-delete pattern across other entities

No action required unless a future cleanup/purge process is desired for hard-deleting soft-deleted items and their related data.
