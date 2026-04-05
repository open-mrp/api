# PATCH /v1/core/transactions/{id} — Migration Verification

## Result: Issues found and fixed

## What was compared

- **Validation rules**: Number uniqueness check (excluding current transaction)
- **Permission checks**: Internal actor + `transactions:update` permission + target account required
- **DB queries**: UPDATE transaction fields, UPDATE quantity for amount changes
- **Error handling**: Conflict on duplicate number, not-found on missing user
- **Side effects**: None in either implementation
- **Response shape**: TransactionDetail resource with all related data
- **Idempotency**: PATCH uses idempotency keys with recovery points (correct)

## Issues found and fixed

### 1. Missing `is_fully_allocated` update support
**Dashboard**: Passes `isFullyAllocated` boolean to the Prisma update, allowing callers to mark a transaction as fully/not fully allocated.
**Go (before fix)**: Did not accept or pass this field.
**Fix**: Added `is_fully_allocated` to proto, endpoint request, gRPC handler, domain params, SQL query, and repository.

### 2. Missing `created_at` update support
**Dashboard**: Passes `createdAt` to the Prisma update, allowing backdating of transaction timestamps.
**Go (before fix)**: Did not accept or pass this field.
**Fix**: Added `created_at` to proto, endpoint request, gRPC handler, domain params, SQL query, and repository.

### 3. Missing responsible user validation
**Dashboard**: Validates the responsible user exists in the account with a two-step lookup:
1. First checks if the provided ID matches an `account_user.id` in the account
2. If not found, checks if it matches a `user.id` and resolves to the corresponding `account_user.id`
3. Returns 404 "Account user not found" if neither matches

**Go (before fix)**: Passed `responsible_user_id` directly to SQL without validation. Since the column has no FK constraint, an invalid ID would be silently stored.
**Fix**: Added `ResolveResponsibleUserID` SQL query and repo method that checks both `account_user.id` and `account_user.user_id`. Added validation in the service layer before calling Update, returning a proper 404 error.

## Post-fix action required

After these changes, run:
```bash
make proto
make sqlc core
make mocks core
```

The proto-generated and sqlc-generated files need regeneration to pick up the new fields.

## Items confirmed as matching

- Number uniqueness validation (both check excluding current ID)
- Transaction method code set/clear logic
- Adjustment type code set/clear logic
- Responsible user clear logic
- Amount update via separate quantity table
- Note update with explicit marker flag
- Permission checks (internal actor, transactions:update)
- Account scoping on all queries
- Idempotency key handling
