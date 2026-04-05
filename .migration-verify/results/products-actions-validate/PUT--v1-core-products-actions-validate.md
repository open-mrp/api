# Verification: PUT /v1/core/products/actions/validate

## Result: Issue found and fixed

## What was compared

- **Validation rules**: Request body is a map of string keys to SKU strings — matches in both implementations.
- **Permission checks**: Both check `checkIsAssignedActor`, then for internal users check `PermissionDomainItems` + `ActionRead`. Customer actors are allowed through without explicit permission in both. Both require target account to be set.
- **DB queries**: Both query `product` joined with `item`, `item_category`, rates (`unit_value`, `unit_cost`, `burn_rate`), `product_line` (LEFT JOIN), and `product_type`. Both filter by `account_id`, `deleted_at IS NULL`, and `sku IN (...)`. Parity confirmed.
- **Error handling**: Both silently omit SKUs that don't match any product. Both propagate DB errors.
- **Side effects**: None in either implementation (read-only operation).
- **Response shape**: Dashboard returns a flat `Record<string, Product>`. Go returns `{ object: "map", products: { ... } }` which follows Go API resource conventions. Acceptable difference.
- **Idempotency**: PUT endpoint, idempotent by nature. No idempotency key needed. Both implementations match.

## Issue found

**Case-insensitive SKU matching**: The Dashboard repository performs case-insensitive matching using `.toLowerCase()` on both the stored SKU and the user-provided SKU. The Go repository was using case-sensitive Go map lookups (`skuToProduct[sku]`). While MySQL's `IN` clause is case-insensitive by default (due to collation), the in-memory map in Go used the exact-case SKU from the database as the key and looked it up with the user-provided SKU, which could differ in case.

## Fix applied

In `services/core-service/internal/infrastructure/repository/product_repo.go`:
- Added `strings` import
- Changed the `skuToProduct` map to use `strings.ToLower()` on keys when building and looking up, matching the Dashboard's case-insensitive behavior.
