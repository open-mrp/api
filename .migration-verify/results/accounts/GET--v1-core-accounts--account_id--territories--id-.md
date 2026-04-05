# GET /v1/core/accounts/{account_id}/territories/{id}

## Result: PARITY CONFIRMED

No issues found. The Go implementation correctly mirrors the Dashboard business logic.

## What Was Compared

- **Permission checks**: Both require internal actor + territories read permission. Go additionally validates target account is set via `CheckTargetAccountSet()`.
- **DB query**: Both filter by territory ID and account ID. Go JOINs account_user → user for sales rep name/email and LEFT JOINs product_line, matching the Dashboard's Prisma select with adapters.
- **Error handling**: Dashboard throws `HttpError.notFound('Territory not found.')`. Go maps SQL "no rows" to a 404 via `db.MapSQLError()`. Equivalent behavior.
- **Response shape**: Dashboard returns full AccountUser and ProductLine objects. Go returns light sub-resources (id, object, name, email for sales rep; id, object, name for product line) with expandable support. This follows Go API conventions correctly.
- **Side effects**: Neither implementation has side effects.
- **Idempotency**: GET endpoint — inherently idempotent, no idempotency key needed. Correct in both.

## Notes

- The Go route uses `/v1/core/accounts/{account_id}/territories/{id}` (account_id from path) while Dashboard uses `identity.targetAccountID`. This is consistent with the Go API's account-scoped route pattern.
- The Go sub-resources are intentionally lighter than Dashboard's full objects, per the API resource conventions (light sub-resources with expandable support).
