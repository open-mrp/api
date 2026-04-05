# POST /v1/core/suppliers — Verification Result

**Status: Issues found and fixed**

## What was compared

- **Permission checks**: Internal actor + `suppliers:create` permission — matches Dashboard
- **Validation**: Required fields `name`, `number` — matches Dashboard DTO schema
- **Duplicate number check**: `ExistsByNumber` query matches Dashboard's `isDuplicate` (checks `external_number` on `account_relation` with role `supplier`, excludes self) — matches
- **Account creation**: Creates account with `account_type_code = 'company'`, `onboarding_status_code = 'unclaimed'` — matches
- **Relation creation**: Creates `account_relation` with role `supplier`, priority `normal` — matches
- **Address creation**: Creates addresses via address repo (geolocation + address + account_address link) — matches Dashboard's `addressRepo.create` + `linkAccountToAddress`
- **Idempotency**: Go uses idempotency keys with recovery points; Dashboard uses `prisma.$transaction` — Go is stricter (correct per Go API patterns)
- **Response shape**: Returns full `SupplierDetail` with nested `Address` sub-resources — matches Dashboard
- **Status code**: 201 Created — matches Dashboard

## Issues found and fixed

### 1. Missing `alias` on account_relation (SQL)

**Dashboard**: Sets `alias: data.name` on the `account_relation` record during creation.
**Go (before)**: Did not include `alias` in `InsertSupplierRelation`.
**Fix**: Added `alias` column to `InsertSupplierRelation` SQL and passed `params.Name` from the repository.

### 2. Missing address IDs on account record (SQL + repo)

**Dashboard**: Sets `default_billing_address_id` and `default_shipping_address_id` on the `account` record itself (in addition to the relation).
**Go (before)**: Only set address IDs on the `account_relation`, not on the `account` record. `InsertSupplierAccount` only inserted `id, name, type, status`.
**Fix**: Added `default_billing_address_id` and `default_shipping_address_id` to the `InsertSupplierAccount` SQL. Updated the repository to compute address NullStrings before the account insert and pass them to both queries.

### 3. Missing ship-to → bill-to fallback (service)

**Dashboard**: When only a `billToAddress` is provided (no `shipToAddress`), the shipping address defaults to the billing address ID — on both the account and the relation: `defaultShippingAddressID: shippingAddressID ?? billingAddress?.id`.
**Go (before)**: Left `default_shipping_address_id` as NULL when no ship-to address was provided, even if a bill-to address existed.
**Fix**: Added fallback logic in `supplier_service.go`: if `shipToAddressID` is nil but `billToAddressID` is not, set `shipToAddressID = billToAddressID`.

## Files modified

- `services/core-service/internal/infrastructure/queries/supplier.sql` — Added `alias` to relation insert, address IDs to account insert
- `services/core-service/internal/infrastructure/sqlc/supplier.sql.go` — Regenerated
- `services/core-service/internal/infrastructure/repository/supplier_repository.go` — Reordered NullString computation, passed alias and account-level address IDs
- `services/core-service/internal/service/supplier_service.go` — Added ship-to → bill-to fallback

## Remaining concerns

None. All three gaps have been resolved and the Go implementation now matches Dashboard behavior for this endpoint.
