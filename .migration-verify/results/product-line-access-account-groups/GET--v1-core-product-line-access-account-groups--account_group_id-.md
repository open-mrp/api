# GET /v1/core/product-line-access/account-groups/{account_group_id}

## Result: PARITY CONFIRMED — No issues found

## What Was Compared

| Aspect | Dashboard | Go | Match? |
|--------|-----------|-----|--------|
| **Permission domain** | `productLineAccess` (`relevant_products`) | `PermissionDomainProductLineAccess` | Yes |
| **Permission action** | `read` | `ActionRead` | Yes |
| **Actor type check** | `checkIsInternalActor` | `identity.CheckIsInternalActor()` | Yes |
| **Account scoping** | `identity.targetAccountID` | `*identity.TargetAccountID` (with `CheckTargetAccountSet`) | Yes |
| **Account group lookup** | Repo finds account group by ID + accountID | SQL `GetAccountGroupByIDAndAccount` by ID + ownerAccountID | Yes |
| **Product line fetch** | Prisma `findMany` on join table with productLine select | SQL join `account_group_product_line` → `product_line` | Yes |
| **404 on not found** | Returns null → throws `HttpError.notFound` | `db.MapSQLError` maps `sql.ErrNoRows` → 404 | Yes |
| **Empty product lines** | Returns empty array | Returns empty slice | Yes |
| **Response shape** | `{ id, accountGroup: { id, name }, productLines: [{ id, name }] }` | `{ account_group: { id, name, object }, object, product_lines: { items, count }, created_at, updated_at }` | Yes (Go conventions) |
| **Side effects** | None | None | Yes |
| **Idempotency** | N/A (GET) | N/A (GET) | Yes |

## Notes

- Response shape differences (sub-resource wrapping, `object` fields, `List` wrapper for product lines, timestamps) are intentional Go API conventions, not parity issues.
- The Go implementation includes `created_at` and `updated_at` from the account group record, which the Dashboard does not return. This is additive, not a breaking change.
- Both implementations scope the query to the requesting user's target account, preventing cross-account access.
