# Verification: POST /v1/core/account-users/{id}/lock

## Result: Issues found and fixed

## What was compared

- **Permission checks**: Both require internal actor + `TeamUsers:update` permission + target account set. Parity confirmed.
- **Self-lock prevention**: Both prevent locking your own account. Parity confirmed.
- **Caller status check**: Both check that the caller is not disabled/locked. Parity confirmed.
- **Target user existence**: Both check the target user exists. Parity confirmed.
- **Status validations**: Both check for removed, already-disabled, and admin users. Parity confirmed after fix.
- **DB operations**: Both atomically update status to disabled and revoke refresh tokens. SQL queries match Dashboard's Prisma operations.
- **Seat sync side effect**: Dashboard does fire-and-forget `syncSubscriptionSeatCount`. Go publishes `ReportSeatChange` and `SyncSeats` via transactional outbox — functionally equivalent, Go approach is more reliable.
- **Idempotency**: This is a POST endpoint but does not use idempotency keys in the Go implementation. The Dashboard also does not use idempotency keys for this endpoint — it's an idempotent-by-nature action (locking an already-locked user returns an error). Acceptable.

## Issues found and fixed

1. **Validation order mismatch**: Dashboard checks removed → disabled → admin. Go was checking admin → removed → disabled. Fixed to match Dashboard order.
2. **Error message mismatch**: Go said "Restore them first." Dashboard says "Restore the user first." Fixed.
3. **Error message mismatch**: Go said "You cannot perform this action." Dashboard says "You cannot lock other users." Fixed to match.

## Known differences (intentional, not fixed)

- **Response shape**: Dashboard returns the updated account user object (HTTP 200). Go returns empty (HTTP 204 No Content). This is consistent with all Go lock/unlock endpoints and appears to be an intentional API design choice for the `/v1/core` API version.
- **Error types**: Dashboard uses `HttpError.forbidden` / `HttpError.badRequest`. Go uses `apierror.NewAuthorizationError` / `apierror.NewValidationError`. These map to the same HTTP status codes (403/400 respectively).
