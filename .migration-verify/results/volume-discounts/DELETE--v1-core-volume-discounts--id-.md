# DELETE /v1/core/volume-discounts/{id}

## Result: PARITY CONFIRMED

No issues found. The Go implementation faithfully reproduces all Dashboard business logic.

## What was compared

- **Permission checks:** Both require internal actor + `discounts/delete` permission
- **Account isolation:** Both filter the main delete by `account_id` to enforce multi-tenancy
- **Cascade deletes:** Dashboard explicitly deletes customer group associations and tiers, relying on Prisma for junction tables (product lines, categories, attributes, units). Go explicitly deletes all six related tables before the main record — more thorough and correct for raw SQL.
- **Transaction atomicity:** Both wrap all deletes in a single transaction
- **Not-found handling:** Dashboard relies on Prisma throwing on missing record; Go checks rows affected and returns ResourceNotFoundError
- **Side effects:** Neither implementation has side effects beyond DB deletion

## Deliberate differences (not bugs)

- **Response:** Dashboard returns 200 OK with the deleted object; Go returns 204 No Content with an empty body. This is a deliberate REST convention for the new API, consistent with other Go delete endpoints.
- **Route prefix:** Dashboard uses `/v1/volume-discounts/:volumeDiscountID`; Go uses `/v1/core/volume-discounts/{id}` per migration conventions.

## No changes required
