# Verification: GET /v1/core/suppliers/{id}

**Status: Issues found and fixed**

## What was compared

- **Validation**: Both accept a supplier ID string from the path. Parity confirmed.
- **Permission checks**: Both check `isAssignedActor`, then branch on actor type:
  - Internal actors: require `suppliers:read` permission
  - Supplier actors: can only access their own record (ID match)
  - Customer actors: rejected (Dashboard returns 400 "Invalid actor type"; Go returns 403 via `CheckIsInternalActor`). Behavioral parity confirmed; error code difference (400 vs 403) is acceptable.
- **DB query**: Both query `account_relation` joined with `account`, filtered by `owner_account_id`, `counterparty_account_id`, and `role='supplier'`. Both LEFT JOIN addresses and include a `supplier_material` count subquery.
- **Response shape**: Both return `id`, `name`, `number`, `note`, `bill_to_address`, `ship_to_address`, `material_count`, `created_at`, `updated_at`. Parity confirmed.
- **Error handling**: Both return 404 when supplier not found. Parity confirmed.
- **Side effects**: None (GET endpoint). Parity confirmed.
- **Idempotency**: Not applicable (GET). Parity confirmed.

## Issues found and fixed

### 1. Supplier name not using alias fallback
- **Dashboard**: Uses `accountRelation.alias || account.name` (prefers the relation alias, falls back to account name)
- **Go (before fix)**: Used `a.name` (account name only, ignoring alias)
- **Fix**: Changed SQL to `COALESCE(ar.alias, a.name) AS account_name`
- **File**: `services/core-service/internal/infrastructure/queries/supplier.sql`

### 2. Address fallback to account-level defaults missing
- **Dashboard**: Falls back to `account.defaultBillingAddress` / `account.defaultShippingAddress` when the relation-level addresses are null
- **Go (before fix)**: Only joined on `ar.default_billing_address_id` / `ar.default_shipping_address_id` (relation addresses only)
- **Fix**: Changed address JOINs to use `COALESCE(ar.default_billing_address_id, a.default_billing_address_id)` and same for shipping
- **File**: `services/core-service/internal/infrastructure/queries/supplier.sql`

## Notes

- The Dashboard throws runtime errors (500) when billing or shipping addresses are missing after fallback. The Go code gracefully returns nil addresses. This is a Go improvement over the Dashboard behavior and was intentionally not replicated.
- sqlc was regenerated and all tests pass.
