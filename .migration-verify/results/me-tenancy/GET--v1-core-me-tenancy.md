# GET /v1/core/me/tenancy — Migration Verification

## Status: Issues found and partially fixed

## What was compared

- **Permission checks**: Dashboard uses `checkIsValidActor(identity)`. Go relies on identity propagation through the API gateway (valid actor check happens at the gateway level). Equivalent.
- **Account filtering logic**: Both filter accounts where user status != "active" and filter sandbox accounts where user is disabled/removed on the owner account. Match confirmed.
- **Current account selection**: Both prioritize paid plans over free, then sort by `lastUsedAt` descending. Match confirmed.
- **Sandbox/owner account resolution**: Both correctly handle sandbox → owner lookup and production → sandbox listing (admin-only). Match confirmed.
- **Other accounts filtering**: Both exclude current account, sandboxes, owner account, inactive accounts, and (for non-admins) sandbox accounts. Match confirmed.
- **Error handling**: Go uses `apierror.NewAuthorizationError` for forbidden cases; Dashboard uses `HttpError.forbidden`. Equivalent mapping.
- **Idempotency**: GET endpoint — no idempotency keys needed. Correct.
- **Response shape**: See discrepancies below.

## Issues found and fixed

### 1. Repository stubs not implemented (CRITICAL)
`FindTenancyAccountsByUserID` and `MarkUsedByAccountAndUser` in `account_user_repository.go` were TODO stubs that returned errors, making the endpoint non-functional.

**Fix**: Added SQL queries to `account_user.sql` and implemented the repository methods:
- `FindTenancyAccountsByUserID`: Joins `account_user`, `account`, `role`, `account_billing`, `account_plan`, and `sandbox_account` to return all fields needed for tenancy resolution.
- `MarkUsedByAccountAndUser`: Updates `last_used_at` and `updated_at` by account_id + user_id.

Ran `make sqlc core` to generate the sqlc bindings.

## Remaining concerns

### 1. Missing `pendingRegistration` in response
Dashboard returns `pendingRegistration: { sessionId, planCode, step, createdAt }` from the `registration_session` table (via auth-service). The Go API does not include this field at all. This requires:
- A gRPC call to auth-service (registration sessions live there)
- New proto fields on `GetTenancyResponse`
- New domain model fields
- Presenter/resource changes

### 2. Missing `accountPlan` in currentAccount
Dashboard returns `accountPlan: { typeID, name, planTypeCode, version, pricePerSeat, pricePerMonth, seatMinimum, limits, features }` with full plan details including limits and features records. The Go API only returns the `plan` (plan type code string). This requires:
- New SQL query to fetch plan limits and features
- New proto messages for account plan data
- New domain and API resource models

### 3. Missing `internalStripeCustomerID` in currentAccount
Dashboard returns `internalStripeCustomerID` from `account_billing`. The Go API omits this. This may be intentional (security — it's a Stripe internal ID), but breaks parity. If needed, it would require adding the field to the SQL query, proto, domain model, and API resource.

### 4. Dashboard returns `null` for no tenancy; Go returns `{ has_tenancy: false }`
Minor structural difference. Dashboard returns `null` when no accounts are found; Go returns a `Tenancy` object with `has_tenancy: false`. The frontend likely needs to handle both cases, but this is a minor difference in how "no tenancy" is represented.

## Files modified
- `services/core-service/internal/infrastructure/queries/account_user.sql` — Added `FindTenancyAccountsByUserID` and `MarkUsedByAccountAndUser` queries
- `services/core-service/internal/infrastructure/repository/account_user_repository.go` — Implemented the two repository methods
- `services/core-service/internal/infrastructure/sqlc/account_user.sql.go` — Regenerated via `make sqlc core`
