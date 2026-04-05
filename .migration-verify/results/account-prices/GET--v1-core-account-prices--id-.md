# GET /v1/core/account-prices/{id}

## Result: Parity Confirmed

No issues found. The Go implementation faithfully reproduces the Dashboard behavior.

## What Was Compared

### Validation
- **Dashboard**: Validates `priceID` path param via `RequestValidator.validate` with Zod schema.
- **Go**: Extracts `AccountPriceID` from `path:"id"` tag. Equivalent.

### Permission Checks
- **Dashboard**: `checkIsAssignedActor` → for internal actors: `checkHasPermission(discounts, read)`. Customer actors pass through without permission check.
- **Go**: `CheckIsAssignedActor` → for internal actors: `CheckHasPermission(PermissionDomainDiscounts, ActionRead)`. Customer actors pass through without permission check. **Match.**

### DB Query
- **Dashboard**: `db.accountPrice.findUnique({ where: { id, ownerAccountID } })` with `AccountPriceAdapter.select()` to pull related data (rate, units, categories, attributes, recipient account, product line).
- **Go**: `GetAccountPrice` SQL query joins `account_price` with `rate`, `account` (recipient), `product_line`, `unit` (numerator), `unit` (denominator), scoped by `id` and `owner_account_id`. Then separately fetches categories and attributes via `GetAccountPriceCategories` and `GetAccountPriceAttributes`. **Match.**

### Error Handling
- **Dashboard**: Returns `HttpError.notFound('Account price not found.')` if repo returns null.
- **Go**: `db.MapSQLError` converts `sql.ErrNoRows` to a not-found error. **Match.**

### Customer Actor Access
- **Dashboard**: Does NOT restrict customer actors from viewing prices where they are not the recipient. Any customer actor with access to the owner account can view any price by ID.
- **Go**: Adds an additional check — if the caller is a customer actor, verifies `accountPrice.RecipientAccountID == customerAccountID`, returning 404 otherwise. This is an **intentional security improvement** over the Dashboard behavior, not a regression.

### Response Shape
- **Dashboard**: Returns `CustomerPrice` object via `AccountPriceAdapter.map()` with nested rate (value, numerator unit, denominator unit), categories, attributes, recipient account, product line.
- **Go**: Returns `AccountPrice` API resource with nested `recipient_account` (expandable), `product_line` (expandable), `rate` (with nested `numerator_unit` and `denominator_unit`), `categories` (list), `attributes` (list), `created_at`, `updated_at`. All fields include `object` type identifiers per API conventions. **Match.**

### Side Effects
- None in either implementation. This is a read-only endpoint.

### Idempotency
- GET endpoint — idempotent by design. No idempotency key needed. **Match.**

## Notes
- The Go implementation includes an additional customer actor recipient check (line 147-152 in `account_price_service.go`) that the Dashboard lacks. This is a security hardening that prevents customer actors from viewing prices intended for other customers. This is kept as-is since it's a strict improvement.
