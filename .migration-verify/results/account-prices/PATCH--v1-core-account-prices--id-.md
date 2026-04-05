# PATCH /v1/core/account-prices/{id} — Migration Verification

## Result: PARITY CONFIRMED

No issues found. The Go implementation faithfully reproduces the Dashboard business logic.

## What Was Compared

### Permission Checks
- **Dashboard:** `checkIsInternalActor` + `checkHasPermission(discounts, update)`
- **Go:** `CheckIsInternalActor()` + `CheckHasPermission(PermissionDomainDiscounts, ActionUpdate)` + `CheckTargetAccountSet()`
- **Match:** Yes. Go additionally validates the target account header is set (standard Go pattern).

### Validation / Request Shape
- **Dashboard:** `CustomerPriceUtils.schema.partial()` — all fields optional (customer, productLine, unitValue, categories, attributes)
- **Go:** All update fields are optional pointers (`*string`, `*[]string`)
- **Match:** Yes. Both support partial updates where only provided fields are changed.

### DB Queries and Logic
- **Dashboard:**
  1. Existence check (`checkExistence` with owner account scope)
  2. If categories provided: delete all existing, recreate from input
  3. If attributes provided: delete all existing, recreate from input
  4. Prisma `update` with COALESCE-equivalent behavior (undefined fields not changed)
  5. Rate update via nested `RateAdapter.updateInput`
- **Go:**
  1. Get rate ID (implicitly checks existence — returns not-found on missing)
  2. Update account_price with COALESCE for recipient_account_id, product_line_id
  3. Update rate with COALESCE for value, numerator_unit_id, denominator_unit_id
  4. If categories provided: delete all existing, recreate from input
  5. If attributes provided: delete all existing, recreate from input
  6. Fetch updated record via Get()
- **Match:** Yes. Same delete-all-then-recreate pattern for categories/attributes. Same COALESCE partial update semantics.

### Error Handling
- **Dashboard:** `HttpError.notFound("Account price not found.")` from explicit existence check
- **Go:** `apierror.NewResourceNotFoundError("Resource not found.")` from `db.MapSQLError` when `GetRateIDByAccountPriceID` returns no rows
- **Match:** Yes (same HTTP status, minor message difference which is acceptable).

### Idempotency
- **Go:** Uses `WithIdempotencyTracking` in gRPC handler + idempotency key recovery points in service layer
- **Dashboard:** No idempotency support
- **Match:** Go correctly adds idempotency for PATCH as required by architecture patterns.

### Transaction Safety
- **Go:** Entire update wrapped in `withTx` transaction
- **Dashboard:** No explicit transaction (deletions + update are separate non-atomic operations)
- **Match:** Go is stricter (better). All mutations are atomic.

### Side Effects
- Neither implementation has side effects beyond DB changes (no emails, webhooks, messages).

### Response Shape
- Both return the full account price object with nested recipient account, product line, rate (with units), categories, and attributes.
- Go uses the `AccountPricePresenter` which maps the proto response to the `apiresource.AccountPrice` resource with expandable sub-resources.

## Issues Found
None.
