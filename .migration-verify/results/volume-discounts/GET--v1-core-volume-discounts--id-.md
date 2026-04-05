# GET /v1/core/volume-discounts/{id} — Migration Verification

## Result: PARITY CONFIRMED

No issues found. The Go implementation correctly matches the Dashboard business logic.

## What Was Compared

### Permission Checks
- **Dashboard:** `checkIsAssignedActor` (allows both internal and customer actors); if internal actor, checks `read` permission on `discounts` domain.
- **Go:** `CheckIsAssignedActor`; if internal user, checks `PermissionDomainDiscounts` / `ActionRead`; also checks `CheckTargetAccountSet`.
- **Verdict:** Match. Customer actors are allowed without explicit permission check in both implementations.

### DB Query and Logic
- **Dashboard:** `findUnique` with `{ id, accountID }` filter, uses `VolumeDiscountAdapter.select` to include tiers, accountGroupVolumeDiscounts, productLines, categories, attributes, acceptableUnits.
- **Go:** `GetVolumeDiscount` SQL query filters by `id` and `account_id`, then `enrichSingle` makes separate queries for tiers, customer groups, product lines, categories, attributes, and units.
- **Verdict:** Match. Both filter by ID + account ID and return the same set of related data.

### Error Handling
- **Dashboard:** Returns `HttpError.notFound('Quantity discount not found.')` when not found.
- **Go:** `db.MapSQLError` on the SQL query maps `sql.ErrNoRows` to a resource-not-found error.
- **Verdict:** Match. Both return 404 when the volume discount doesn't exist.

### Response Shape
- **Dashboard:** Returns `{ id, name, tiers, customerGroups, productLines, categories, attributes, acceptableUnits, createdAt, updatedAt }`. Tiers include `{ id, name, threshold, discountPercentage, createdAt, updatedAt }`.
- **Go:** Returns `{ id, object, name, tiers, customer_groups, product_lines, categories, attributes, acceptable_units, created_at, updated_at }`. Tiers include `{ id, object, name, discount_percentage, threshold, created_at, updated_at }`. Relations are expandable via the include system.
- **Verdict:** Match. Go adds `object` fields per API conventions and uses the include/expandable system for sub-resources, which is expected for the new API.

### Customer Actor Access
- **Dashboard:** Customer actors can call `find` without additional filtering — the method just looks up by ID + accountID.
- **Go:** Same behavior — `GetVolumeDiscount` accepts `CustomerAccountID` in params but the service/repo don't use it for the get-by-ID path.
- **Verdict:** Match.

### Side Effects
- Neither implementation has any side effects (no emails, webhooks, messages, or inventory changes).

### Idempotency
- GET endpoint — inherently idempotent, no idempotency key needed. Both implementations comply.

## Issues Found and Fixed
None.

## Remaining Concerns
None.
