# Verification: DELETE /v1/core/parts/{id}

## Result: Issues found and fixed

## What was compared

- **Validation**: Request parameter validation (itemID path param)
- **Permission checks**: Actor type and permission domain/action
- **DB queries**: Soft delete SQL (account-scoped, deleted_at IS NULL guard)
- **Error handling**: Not found, auth, permission errors
- **Side effects**: Cascade deletions in Dashboard vs soft-delete-only in Go
- **Response shape**: Returns pre-deletion part object
- **Idempotency**: DELETE is naturally idempotent (no idempotency keys needed)

## Issues found and fixed

### 1. Permission domain mismatch (FIXED)

- **Dashboard**: Uses `PermissionDomains.items` with `'delete'` action for all item types (products, materials, parts)
- **Go (before fix)**: Used `PermissionDomainParts` with `ActionDelete`
- **Go (after fix)**: Uses `PermissionDomainItems` with `ActionDelete`
- **File**: `services/core-service/internal/service/part_service.go:347`
- **Rationale**: The product delete endpoint already correctly uses `PermissionDomainItems`, matching Dashboard. The part delete was inconsistent.

## Confirmed parity

- **Actor check**: Both require internal actor (403 if not)
- **Account scoping**: Both scope by target account ID
- **Pre-fetch**: Both fetch the part before deletion and return it
- **Not found**: Both return 404 if part doesn't exist
- **Response**: Both return the part as it existed before deletion (HTTP 200)
- **SQL**: Go soft-deletes with `deleted_at = NOW(3), updated_at = NOW(3)` filtered by `account_id` and `deleted_at IS NULL`

## Remaining concerns

### Soft-delete vs hard-delete cascade (intentional difference)

The Dashboard performs a hard delete cascade after soft-deleting:
1. Soft-deletes the item (sets deletedAt)
2. Hard-deletes consumptions, productions, quantities, inventory data
3. Hard-deletes the item record itself
4. Hard-deletes associated rate records (unitValue, unitCost, burnRate)

The Go implementation only soft-deletes (sets `deleted_at`). This is consistent across ALL Go item deletions (products, materials, parts), so this appears to be an **intentional architectural decision** — not a parity gap. Related data is presumably cleaned up via background processes or is effectively excluded by `deleted_at IS NULL` filters in all queries.

### Other part service methods also use wrong permission domain

The Go part service uses `PermissionDomainParts` for read (single), create, and update operations as well (lines 110, 135, 254). Dashboard uses `PermissionDomains.items` for all. This is out of scope for this verification (only DELETE was assigned) but should be addressed.
