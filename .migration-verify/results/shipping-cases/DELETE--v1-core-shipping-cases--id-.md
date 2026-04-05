# DELETE /v1/core/shipping-cases/{id}

## Status: Issues found and fixed

## What was compared

- **Validation rules**: Path parameter `id` required — matches
- **Permission checks**: Both check `isInternalActor` + `hasPermission(shipments, delete)` + `targetAccountID` required — matches
- **DB queries**: Both delete by `id + account_id` — matches
- **Error handling**: See issue below
- **Side effects**: Neither has side effects (no cascade cleanup) — matches
- **Response shape**: Dashboard returns deleted record with HTTP 200; Go returns empty resource with HTTP 204 No Content — this is an intentional Go API convention difference (all Go DELETE endpoints use 204/EmptyResource)
- **Idempotency**: DELETE endpoint, no idempotency keys needed — correct per conventions

## Issues found and fixed

### Missing existence check before delete (FIXED)

**Dashboard behavior**: Prisma's `.delete()` throws a `P2025` (Record not found) error if the shipping case doesn't exist for the given account, resulting in a 404 response.

**Go behavior (before fix)**: The SQL `DELETE FROM shipping_case WHERE id = ? AND account_id = ?` with `:exec` silently succeeds even if no rows match — no error returned to the caller.

**Fix**: Added an `IsInAccount` check before the delete call (matching the pattern used by `DeleteShipment` and other services), returning a `ResourceNotFoundError("Shipping case not found.")` if the record doesn't exist.

## Remaining concerns

None.
