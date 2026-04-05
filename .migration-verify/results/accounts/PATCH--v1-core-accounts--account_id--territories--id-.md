# PATCH /v1/core/accounts/{account_id}/territories/{id}

## Result: PARITY CONFIRMED

No issues found. The Go implementation correctly matches the Dashboard business logic.

## What Was Compared

- **Permission checks**: Both require internal actor + `territories:update` permission
- **Account scoping**: Both scope the update query by account ID and territory ID
- **Validation**: Zipcode range (501-99999) validated in both implementations
- **Zipcode null cascade**: Dashboard forces `endZipcode = null` when `startZipcode` is set to null; Go repo cascades `ClearStartZipcode` to also set `ClearEndZipcode = true` — equivalent behavior
- **Product line disconnect**: Dashboard uses Prisma `disconnectIfNull`; Go uses `ClearProductLine` flag with CASE SQL — equivalent behavior
- **Partial update fields**: state, salesRep, startZipcode, endZipcode, productLine — all present in both
- **Update SQL**: Uses COALESCE for simple fields, CASE statements for nullable/clearable fields — correct
- **Response shape**: Territory with nested sales_rep (id, name, email) and product_line (id, name), plus Object field and timestamps
- **Idempotency**: Go correctly adds idempotency key handling for this PATCH endpoint (Dashboard had none, but Go conventions require it)
- **Side effects**: None in either implementation
- **Error handling**: Go checks territory exists in account before updating; Dashboard relies on Prisma throwing on missing record

## No Fixes Required
