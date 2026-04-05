# POST /v1/core/accounts/{account_id}/territories — Verification Result

**Status: PARITY CONFIRMED** — No issues found.

## What Was Compared

| Aspect | Dashboard | Go | Match |
|--------|-----------|-----|-------|
| Permission: internal actor | `checkIsInternalActor` | `identity.CheckIsInternalActor()` | Yes |
| Permission: domain/action | `territories / create` | `PermissionDomainTerritories / ActionCreate` | Yes |
| Target account check | `identity.targetAccountID` (implicit via service constructor) | `identity.CheckTargetAccountSet()` | Yes |
| Required fields | `state`, `salesRep` (object) | `state`, `sales_rep_id` (ID only) | Yes (Go pattern uses IDs) |
| Zipcode validation | Zod: int 501–99999 | Service: 501–99999 check | Yes |
| Zipcode null coercion | If `startZipcode` is null, `endZipcode` forced to null | Same logic in repository | Yes |
| DB insert | Prisma create with FK connects | SQL INSERT with NOW(3) timestamps | Yes |
| Post-insert fetch | Prisma select with nested relations | SQL GET with JOINs (account_user → user, product_line) | Yes |
| Response shape | Full nested `salesRep` (AccountUser), `productLine` | Light sub-resources: `LightSalesRep` (id, name, email), `LightProductLine` (id, name) | Yes (Go pattern) |
| HTTP status | 201 Created | 201 Created | Yes |
| Idempotency | None | Full idempotency key support with recovery points | Yes (Go improvement) |
| Side effects | None | None | Yes |
| Expandable fields | N/A | `sales_rep`, `product_line` via include system | Yes (Go enhancement) |

## Notes

- **Account ID source**: Dashboard uses `identity.targetAccountID` (from header); Go uses `account_id` path parameter. This is consistent with the Go nested-resource URL pattern (`/v1/core/accounts/{account_id}/territories`).
- **Request shape**: Dashboard accepts full salesRep/productLine objects; Go accepts just IDs (`sales_rep_id`, `product_line_id`). This follows Go API conventions.
- **Response shape**: Go returns light sub-resources instead of full objects, with expandable support. This is the standard Go API pattern per `api-resource-conventions.md`.
- **FK constraint handling**: Both implementations rely on DB foreign key constraints to validate that `sales_rep_id` and `product_line_id` exist. Go maps SQL errors via `db.MapSQLError`.

## No Issues Found

All business logic, validation, permission checks, and data flow have been preserved in the Go implementation. The differences are intentional architectural improvements (idempotency, light sub-resources, include system).
