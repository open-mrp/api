# Verification: PUT /v1/core/account-users/{id}/notification-preferences

**Status:** Issues found and fixed

## What was compared

- Permission checks (actor type, permission domain, self-update bypass)
- Account relation lookup logic
- Account user lookup logic
- Notification type validation
- Create/delete preference logic (idempotent toggle behavior)
- Transaction boundaries
- Response shape (AccountUser resource)
- HTTP method and idempotency (PUT, no idempotency key — correct)

## Issues found and fixed

### 1. Wrong permission domain
- **Dashboard:** `checkHasPermission(this.identity, PermissionDomains.customers, 'update')` — uses `customers` domain
- **Go (before fix):** `identity.CheckHasPermission(types.PermissionDomainTeamUsers, types.ActionUpdate)` — used `team` domain
- **Fix:** Changed to `types.PermissionDomainCustomers` (`"customers"`)

### 2. Missing self-update bypass
- **Dashboard:** Skips the permission check when `this.identity.actor.id === userID` (user updating their own preferences)
- **Go (before fix):** Always required the permission check regardless of actor
- **Fix:** Added conditional: only check `customers:update` permission when `identity.Actor.ID != params.UserID`

## File changed

- `services/core-service/internal/service/account_user_service.go` (lines ~893-899)

## Confirmed parity

- Internal actor check: both require internal actor
- Edit access check: both verify edit access between actor account and target account
- Account relation lookup: both find by ownerAccountID + counterpartyAccountID
- Account user existence check: both verify user exists in target account
- Toggle logic: both create preference if enabled+missing, delete if disabled+existing, skip otherwise
- Transaction: both wrap mutations in a transaction
- Notification type validation: Go validates codes explicitly (stricter than Dashboard, which is acceptable)
- Response: both return the updated AccountUser resource

## Remaining concerns

None. Parity is confirmed after the fixes.
