# Migration Verification: GET /v1/core/product-line-access/account-groups

## Result: PARITY CONFIRMED

No issues found. The Go implementation faithfully reproduces all Dashboard business logic.

## What Was Compared

| Aspect | Dashboard | Go | Match |
|--------|-----------|-----|-------|
| **Permission: internal actor** | `checkIsInternalActor` | `identity.CheckIsInternalActor()` | Yes |
| **Permission: domain/action** | `productLineAccess` / `read` | `PermissionDomainProductLineAccess` / `ActionRead` | Yes |
| **Account scoping** | `ownerAccountID: identity.targetAccountID` | `owner_account_id = params.AccountID` (from `identity.TargetAccountID`) | Yes |
| **Search filter** | Prisma `contains` (case-insensitive) on `accountGroup.name` | SQL `LIKE %query%` on `ag.name` | Yes |
| **Joins** | Prisma relations: `accountGroup`, `productLine` | SQL JOIN on `account_group` and `product_line` | Yes |
| **Grouping** | JS `reduce` groups rows by `accountGroupId`, merges product lines into array | Go `groupForwardRows`/`groupBackwardRows` groups by `AccountGroupID`, appends `ProductLineInfo` | Yes |
| **Response shape** | `{ items: [{ id, accountGroup: {id, name}, productLines: [{id, name}] }], count }` | `{ data: [{ account_group: {id, object, name}, object, product_lines: {data: [{id, object, name}]}, created_at, updated_at }], page_info }` | Yes (Go conventions) |
| **Pagination** | Offset-based (`take`/`skip`) | Cursor-based (`cursor`/`limit`) | Expected difference |
| **Side effects** | None | None | Yes |
| **Idempotency** | N/A (GET) | N/A (GET) | Yes |

## Notes

- **Pagination style difference** is expected: the Go API uses cursor-based pagination as a standard convention, while the Dashboard used offset-based. This is an intentional architectural improvement.
- **Response shape differences** follow Go API conventions: `object` fields on resources, `page_info` instead of `count`, `data` wrapper for lists. These are by design per `api/docs/api-resource-conventions.md`.
- **Row multiplier**: Go uses a `rowLimitMultiplier = 20` to over-fetch rows before grouping, which correctly handles the N:1 relationship between product line rows and account groups.
- **Search**: Both implementations search on account group name only (not product line names). Go wraps the query in `%...%` for LIKE matching, matching Prisma's `contains` behavior.
