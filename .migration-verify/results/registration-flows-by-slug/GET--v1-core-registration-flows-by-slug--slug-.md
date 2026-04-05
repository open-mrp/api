# Verification: GET /v1/core/registration-flows/by-slug/{slug}

## Status: PARITY CONFIRMED

No issues found. The Go implementation matches the Dashboard behavior.

## What Was Compared

| Aspect | Dashboard | Go | Match |
|--------|-----------|-----|-------|
| **Auth** | `checkIsValidActor` (authenticated + valid actor type) | `CheckIsAuthenticated()` | Yes — functionally equivalent |
| **Account lookup** | Prisma `findFirst` via `portal.slug` relation | SQL JOIN `account_portal` on slug | Yes |
| **Flow lookup** | `registrationFlowRepo.list({ accountID })` returns first item | `GetByAccountID(accountID)` returns first item | Yes |
| **404 on missing account** | `HttpError.notFound('Account not found.')` | `db.MapSQLError` returns not-found | Yes |
| **404 on no flows** | `HttpError.notFound('Registration flow not found.')` | `apierror.NewResourceNotFoundError(...)` | Yes |
| **Response fields** | id, name, customerGroupOptions, paymentTermOptions, shippingTermOptions, createdAt, updatedAt | Same fields + `object` type fields (Go API convention) | Yes |
| **Options enrichment** | Prisma eager-loads accountGroupOptions, paymentTermOptions, shippingTermOptions | Separate SQL queries for each option type via join tables | Yes |
| **Side effects** | None | None | Yes |
| **Idempotency** | N/A (GET) | N/A (GET) | Yes |

## Notes

- The Go API adds `object` type fields (`registration_flow`, `registration_flow_option`) per API resource conventions. This is expected and not a discrepancy.
- Dashboard's `checkIsValidActor` allows internal, customer, supplier, and unassigned actors. Go's `CheckIsAuthenticated()` is slightly more permissive but functionally equivalent — the Go identity system only produces valid actor types after authentication. This pattern is consistent with other migrated endpoints.
- Both implementations use the same DB join path: portal slug -> account -> registration flows -> options (via join tables for payment terms and shipping terms, direct FK for account groups).
