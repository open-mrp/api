# POST /v1/core/transactions — Migration Verification

## Result: Issues found and fixed

## What was compared

- **Validation rules**: Required fields (customer_id, transaction_type_code, amount) match between Dashboard and Go
- **Permission checks**: Both check `checkIsInternalActor` + `transactions:create` permission + target account set
- **DB queries and logic**: INSERT query, transaction number generation, dollar unit lookup
- **Error handling**: Error types and messages are consistent
- **Side effects**: Both increment sys_property for transaction number; no external side effects
- **Response shape**: TransactionDetail resource with nested customer, amount, transaction_type, etc.
- **Idempotency**: Go properly uses idempotency keys with recovery points for the POST endpoint

## Issues found and fixed

### 1. Missing `adjustment_type_code` in INSERT SQL (Fixed)

The `InsertTransaction` SQL query did not include the `adjustment_type_code` column. The Dashboard creates transactions with `adjustmentType` connected by code, but the Go INSERT omitted this column entirely. Transactions created via the Go API would always have a NULL adjustment type regardless of what was provided.

**Fix**: Added `adjustment_type_code` to the INSERT columns and VALUES in `transaction.sql`.

### 2. Missing `responsible_user_id` in INSERT SQL (Fixed)

The `InsertTransaction` SQL query did not include the `responsible_user_id` column. The Dashboard always stores the authenticated user's AccountUser ID as the responsible user, but the Go INSERT omitted this column.

**Fix**: Added `responsible_user_id` to the INSERT columns and VALUES in `transaction.sql`.

### 3. Updated repo Create method signature (Fixed)

The `TransactionRepo.Create` interface and implementation did not accept `adjustmentTypeCode` or `responsibleUserID` parameters, so even if the service had these values, they couldn't be passed to the INSERT.

**Fix**: Added `adjustmentTypeCode *string` and `responsibleUserID *string` parameters to both the interface (`repositories.go`) and implementation (`transaction_repo.go`). Also updated the `InsertTransactionParams` struct usage to pass these values.

### 4. No auto-population of responsible user (Fixed)

The Dashboard always looks up the AccountUser for the authenticated user (`accountUserRepo.find({ userID, accountID })`) and uses `accountUser.id` as `responsibleAccountUserID`. The Go API accepted `responsible_user_id` as an optional request field but never auto-populated it from the identity.

**Fix**: Added logic in `transaction_service.go` to auto-populate `ResponsibleUserID` from the authenticated user's AccountUser record when not explicitly provided, matching Dashboard behavior.

### 5. Wrong JOIN for responsible_user_id in FindTransactionByID (Fixed)

The `FindTransactionByID` query joined `LEFT JOIN user usr ON usr.id = t.responsible_user_id`, but per the Prisma schema, the `responsible_user_id` column stores `account_user.id` values (not `user.id`). This meant the responsible user name would never be resolved correctly.

**Fix**: Changed the JOIN to go through `account_user` first: `LEFT JOIN account_user au ON au.id = t.responsible_user_id LEFT JOIN user usr ON usr.id = au.user_id`.

### 6. Updated stripe webhook caller (Fixed)

The `stripe_webhook_service.go` also calls `txRepo.Create` and needed the new parameters added (nil for both `adjustmentTypeCode` and `responsibleUserID` since Stripe webhooks don't have these).

## Remaining concerns

- The same `responsible_user_id` JOIN issue exists in the list transaction queries (`ListTransactionsForward`, `ListTransactionsBackward`, `ListAccountTransactionsForward`, `ListAccountTransactionsBackward`). These should be fixed separately as they affect other endpoints.
- The Go API field name `responsible_user_id` is slightly misleading since the column stores `account_user` IDs, not `user` IDs. The Dashboard uses `responsibleAccountUserID` internally. This is a naming concern, not a functional issue.

## Files modified

- `services/core-service/internal/infrastructure/queries/transaction.sql` — Added columns to INSERT, fixed JOIN in FindTransactionByID
- `services/core-service/internal/domain/repositories.go` — Updated TransactionRepo.Create interface
- `services/core-service/internal/infrastructure/repository/transaction_repo.go` — Updated Create method signature and InsertTransaction params
- `services/core-service/internal/service/transaction_service.go` — Added responsible user auto-population, passed adjustment_type_code
- `services/core-service/internal/service/stripe_webhook_service.go` — Updated Create call with new params
- `services/core-service/internal/infrastructure/sqlc/` — Regenerated via `make sqlc core`
