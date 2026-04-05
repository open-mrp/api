# Verification: POST /v1/core/account-users

## Result: Issue found and fixed

## What was compared

- **Permission checks:** Dashboard checks `checkIsAssignedActor` + `customers:create` for internal actors. Go matches.
- **Validation:** Email or username required, scanning station password required, non-scanning station password forbidden. Go matches.
- **Scanning station role resolution:** Dashboard always forces scanner role for scanning stations. **Go only assigned scanner role when no role_id was provided — fixed.**
- **Sales rep role resolution:** Dashboard infers from provided role type; Go uses explicit `is_sales_rep` flag. Intentional API contract difference — functionally equivalent.
- **Seat limit enforcement:** Dashboard checks for non-external targets; Go checks for own-account. Semantically equivalent.
- **Edit access check:** Dashboard always calls `checkEditAccess`; Go only calls for external targets (own-account edit access is implied by permission check). Functionally equivalent.
- **User lookup:** Both check for existing user by email or username.
- **Existing user conflict detection:** Both check if user is already linked to the account.
- **New user creation:** Both create user with hashed password (provided or generated).
- **Auto-disable for external target with billing plan:** Both disable newly created users on external accounts with active billing plans.
- **Notification preferences:** Both create preferences for external targets based on boolean flags.
- **Welcome email:** Both send branded email for external targets, generic for own account. Only for new users with generated passwords.
- **Billing events:** Dashboard fires seat sync/change as fire-and-forget; Go publishes via outbox pattern. Functionally equivalent.
- **Idempotency:** Go correctly adds idempotency key support (POST endpoint pattern).
- **Response shape:** Go returns `AccountUserDetail` with role/department as sub-resources. Follows API conventions.

## Issues found and fixed

### 1. Scanning station role not forced when role_id provided

**File:** `services/core-service/internal/service/account_user_service.go:241-248`

**Problem:** The Go code only assigned the scanner role when `params.RoleID == nil`. The Dashboard always forces the scanner role for scanning station users, regardless of what role was provided.

**Fix:** Removed the `params.RoleID == nil` condition so scanning station users always get the scanner role, matching Dashboard behavior.

## Intentional API contract differences (not bugs)

- **Name field:** Go uses single `name` field vs Dashboard's `firstName`/`lastName`
- **Email field:** Go uses `custom_email` JSON tag vs Dashboard's `email`
- **Sales rep:** Go uses explicit `is_sales_rep` boolean vs Dashboard inferring from role type
- **Account ID:** Go uses header-only (`Augno-Account-ID`) vs Dashboard's header + route param

## Remaining concerns

None. The business logic is now in parity with the Dashboard implementation.
