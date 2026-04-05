# GET /v1/core/registration-flows/{id} — Verification Result

**Status: PARITY CONFIRMED** — No issues found.

## What Was Compared

| Aspect | Dashboard | Go | Match |
|--------|-----------|-----|-------|
| Permission: actor type | `checkIsInternalActor` | `CheckIsInternalActor` | Yes |
| Permission: domain/action | `account:read` | `PermissionDomainAccount:ActionRead` | Yes |
| Account isolation | `where: { id, accountID }` | `WHERE rf.id = ? AND rf.account_id = ?` | Yes |
| Options: customer groups | Loaded via Prisma adapter select | `ListAccountGroupOptionsByFlowID` query | Yes |
| Options: payment terms | Loaded via Prisma adapter select | `ListPaymentTermOptionsByFlowID` via junction table | Yes |
| Options: shipping terms | Loaded via Prisma adapter select | `ListShippingTermOptionsByFlowID` via junction table | Yes |
| Response shape | id, name, customerGroupOptions, paymentTermOptions, shippingTermOptions, createdAt, updatedAt | Same fields + `object` type on resource and options (per Go conventions) | Yes |
| Error: not found | 404 "Registration flow not found." | SQL error mapped to not-found via `db.MapSQLError()` | Yes |
| Idempotency | N/A (GET) | N/A (GET) | Yes |
| Side effects | None | None | Yes |

## Summary

The Go implementation is a faithful migration of the Dashboard endpoint. All permission checks, DB queries, option loading, error handling, and response shapes match. The Go version correctly adds `object` fields per API resource conventions and uses the `v1/core/` route prefix per migration conventions.

No fixes were required.
