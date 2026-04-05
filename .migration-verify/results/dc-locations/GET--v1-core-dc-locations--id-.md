# GET /v1/core/dc-locations/{id}

**Status: PARITY CONFIRMED** — No issues found.

## What was compared

| Aspect | Dashboard | Go | Match |
|--------|-----------|-----|-------|
| **Permission check** | `checkIsInternalActor` + `ediRuns:read` | `CheckIsInternalActor` + `PermissionDomainEdiRuns:ActionRead` | Yes |
| **Account scoping** | `ownerAccountID` from identity | `*identity.TargetAccountID` | Yes |
| **DB query** | Prisma `findUnique({ id, ownerAccountID })` with account relation select | SQL `GetDCLocation` with `id` + `owner_account_id`, LEFT JOIN account for customer_name | Yes |
| **Customer sub-resource** | `CustomerAccountSummaryAdapter.map(data.account)` → `{ id, name }` | `DCLocationCustomer{ ID, Object, Name }` (set only when `CustomerId != ""`) | Yes |
| **Response fields** | `id, customer, location, updatedAt, createdAt` | `id, object, location, customer, created_at, updated_at` | Yes (Go adds `object` per convention) |
| **Not-found handling** | Prisma returns `null` → 404 at HTTP layer | `db.MapSQLError` on `sql.ErrNoRows` → not-found error | Yes |
| **Idempotency** | N/A (GET) | N/A (GET) | Yes |
| **Side effects** | None | None | Yes |
| **Validation** | Path param `dcLocationID` (string) | Path param `id` (string) | Yes |

## Notes

- The Go API adds `object` fields on both the DCLocation and its customer sub-resource, which is standard for the Go API convention and not a parity issue.
- Customer is correctly modeled as an expandable sub-resource rather than inlined fields.
- Permission domain `edi_runs` matches between both implementations.
