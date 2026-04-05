# Verification: GET /v1/core/integrations/stripe/publishable-key

**Status:** Issues found and fixed

## What was compared

- **Validation rules:** No request parameters in either implementation — matches.
- **Permission checks:** Actor type restrictions and read access checks.
- **DB queries:** Both query `account_integration` by `account_id` + `integration_code = 'stripe'`, selecting `credentials` and `is_active` — matches.
- **Error handling:** Not-found, inactive integration, and decryption errors — matches.
- **Side effects:** None in either implementation — matches.
- **Response shape:** Dashboard returns `{ publishableKey }`, Go returns `{ object, publishable_key }`. The `object` field and snake_case are expected per Go API conventions.
- **Idempotency:** GET endpoint, not applicable — matches.

## Issue found and fixed

**Supplier actor access control was too permissive in Go.**

- **Dashboard:** Explicitly branches on actor type — internal actors proceed directly, customer actors get a read-access check, and all other actors (including suppliers) are rejected with `403 Forbidden: "User does not have permission to access this resource."`
- **Go (before fix):** Used `identity.IsExternalTarget()` to decide whether to run the read-access check. This would allow supplier actors through if they passed the read-access check, since `IsExternalTarget()` returns true for any actor whose account differs from the target account.
- **Go (after fix):** Now explicitly branches: `IsInternalUser()` → allowed directly, `IsCustomerUser()` → read-access check, else → `NewAuthorizationError("User does not have permission to access this resource.")`. This matches the Dashboard behavior exactly.

**File changed:** `services/core-service/internal/service/account_integration_service.go` (lines ~407-417)

## Remaining concerns

- No unit tests exist for `GetStripePublishableKey` in the Go service layer. Consider adding tests covering internal, customer, supplier, and unassigned actor scenarios.
