# POST /v1/core/account-users/{id}/sales-targets

**Status: Issues found and fixed**

## What was compared

- Permission checks (actor type, permission domain, action)
- Account user (sales rep) validation before creation
- DB queries and logic (inserts, joins)
- Idempotency key support
- Error handling (error types, messages)
- Response shape (field names, types, nested resources)
- Side effects (none expected)

## Issues found and fixed

### 1. Missing account user (sales rep) existence validation

**Dashboard behavior:** Before creating a sales target, the service validates that the account user exists by calling `accountUserRepo.find({ userID, accountID })`. If not found, it returns `HttpError.notFound('Account user not found.')`.

**Go behavior (before fix):** The Go service did not validate that the sales rep existed in the account. An invalid `SalesRepID` would result in a database-level foreign key constraint error with a generic error message, not a clean 404.

**Fix:** Added a `SalesRepExistsInAccount` SQL query to `sales_target.sql` that checks for an active account user by ID and account ID. Added the corresponding method to `SalesTargetRepo` interface and implementation. Added the validation check in `CreateSalesTarget`, `UpsertSalesTarget`, and `ListSalesTargets` service methods — all three match the Dashboard pattern of validating the account user before proceeding.

### Files modified

- `services/core-service/internal/infrastructure/queries/sales_target.sql` — added `SalesRepExistsInAccount` query
- `services/core-service/internal/domain/repositories.go` — added `SalesRepExistsInAccount` to `SalesTargetRepo` interface
- `services/core-service/internal/infrastructure/repository/sales_target_repository.go` — added implementation
- `services/core-service/internal/service/sales_target_service.go` — added validation in Create, Upsert, and List methods

## Confirmed parity

- **Permission checks:** Both require internal actor + `salesTargets` domain + appropriate action (create/update/read)
- **Idempotency:** Go CreateSalesTarget correctly uses idempotency keys with recovery points; UpsertSalesTarget (PUT) is idempotent by design without keys
- **Response shape:** Go returns nested `sales_rep` (User sub-resource) and `amount` (Quantity with `value` and `unit` sub-resource), matching Dashboard's nested `targetAmount` with `measure` and `unit`
- **DB operations:** Both create a quantity record and sales target record atomically; Go does this in a transaction
- **Error handling:** Now returns proper 404 for missing account user

## Notes

- The Go API route `/v1/core/account-users/{id}/sales-targets` takes the account_user ID directly in the path, while the Dashboard route `/v1/identity/:accountID/users/:userID/targets` takes the user ID and looks up the account_user. This is an intentional design difference in the Go API (account-user-centric routing).
- The Dashboard request body accepts the full SalesTarget schema (including `id`), while the Go API accepts flat fields (`start_date`, `end_date`, `amount_value`, `amount_unit_id`) and generates the IDs server-side. This is a better pattern.
- Pre-existing compilation issues in `sales_order_repo.go` and `shipment_repository.go` are unrelated to this endpoint.
