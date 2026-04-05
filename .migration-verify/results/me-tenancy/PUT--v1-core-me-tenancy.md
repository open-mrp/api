# PUT /v1/core/me/tenancy — Migration Verification

## Status: Issues Found and Fixed

## What Was Compared

- **Validation rules**: Request requires `account_id` (string, required) — matches Dashboard's `accountID` body param
- **Permission checks**: Dashboard uses `checkIsValidActor(identity)`; Go service relies on identity being present via gRPC metadata (user_id extracted in API gateway). Equivalent behavior.
- **DB queries and logic**: Dashboard fetches all account_user rows for the user with account, role, plan, and sandbox_account data via Prisma. Go now does the same via a SQL query joining account_user → account → role → account_billing → account_plan → sandbox_account.
- **Status checks**: Both implementations check:
  - User is not disabled/removed on the target account
  - For sandbox accounts, user is not disabled/removed on the owner (production) account
  - Target account is not suspended or deactivated
- **Account filtering**: Both filter the response to only include active accounts (excluding disabled/removed users, and sandbox accounts where the user is locked on the owner)
- **lastUsedAt update**: Both update the `last_used_at` timestamp for the account_user row on switch
- **Response shape**: Both return current_account (with role), sandboxes (admin only, for non-sandbox accounts), owner_account, and other_accounts
- **Error handling**: Dashboard uses `HttpError.forbidden` (403); Go uses `apierror.NewAuthorizationError` (403). Equivalent.
- **Idempotency**: This is a PUT endpoint — designed to be idempotent by default (no idempotency key needed). Matches conventions.

## Issues Found and Fixed

### 1. Unimplemented SQL queries (CRITICAL — was already fixed in working tree)

The repository methods `FindTenancyAccountsByUserID` and `MarkUsedByAccountAndUser` had TODO stubs returning errors. These have been implemented:

- **FindTenancyAccountsByUserID**: SQL query joins account_user → account → role → account_billing → account_plan → sandbox_account to fetch all fields needed for tenancy resolution (account type, onboarding status, plan code, role info, sandbox owner).
- **MarkUsedByAccountAndUser**: Simple UPDATE setting `last_used_at = NOW(3)` and `updated_at = NOW(3)` where account_id and user_id match.

Both were regenerated via `make sqlc core` and the repository methods map the sqlc rows to domain models correctly.

## Remaining Concerns

### pendingRegistration field missing from Go response

The Dashboard's `switchAccount` response includes a `pendingRegistration` field (with `sessionId`, `planCode`, `step`, `createdAt`) fetched from `registrationSessionRepo.findIncompleteByUserID()`. The Go implementation does not include this field.

This is an architectural gap: the `registration_session` table is not currently accessed by core-service (it's managed by auth-service). Adding it would require either:
- A cross-service gRPC call to auth-service to fetch pending registration sessions, or
- Adding a registration_session repository to core-service

This is a known gap that should be tracked separately as it affects both `GET /v1/core/me/tenancy` and `PUT /v1/core/me/tenancy`. It may be acceptable if the dashboard frontend handles the absence of this field gracefully.

## Compilation

All tenancy-related code compiles successfully. Pre-existing compilation errors exist in unrelated files (item_repository.go, sales_order_repo.go, shipment_repository.go).
