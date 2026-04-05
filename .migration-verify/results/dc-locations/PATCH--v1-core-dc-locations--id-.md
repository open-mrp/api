# PATCH /v1/core/dc-locations/{id}

**Status: Parity confirmed — no issues found.**

## What was compared

- **Validation rules:** Both accept optional `location` (string) and customer/account reference. Dashboard uses `schema.partial()`, Go uses nullable fields with COALESCE SQL pattern.
- **Permission checks:** Both require internal actor + `ediRuns:update` permission + target account header.
- **DB queries:** Both update `location` and `account_id` with partial-update semantics (only provided fields change), scoped by `id` and `owner_account_id`.
- **Error handling:** Dashboard relies on Prisma P2025 for not-found. Go pre-verifies existence with `GetDCLocation` and checks `RowsAffected == 0` — functionally equivalent.
- **Side effects:** None in either implementation.
- **Response shape:** Dashboard returns a richer customer summary (id, name, number, email, etc.). Go returns a lightweight `DCLocationCustomer` sub-resource (id, object, name), following Go API conventions for sub-resources. Core data fields (id, location, customer association, timestamps) are preserved.
- **Idempotency:** Go correctly implements idempotency keys with recovery points for the PATCH endpoint, which the Dashboard did not have.
- **Transaction handling:** Go wraps the update in a transaction with idempotency caching, matching architectural patterns.

## Issues found

None.

## Notes

- The Go API's `customer_id` request field maps to `account_id` in the database (via the `AccountID` domain param), which is consistent with how the Dashboard's `customer` field connects via `PrismaUtils.connect(data.customer)`.
- The Go customer sub-resource is intentionally lighter than the Dashboard's `CustomerAccountSummary`, following Go API resource conventions for sub-objects.
