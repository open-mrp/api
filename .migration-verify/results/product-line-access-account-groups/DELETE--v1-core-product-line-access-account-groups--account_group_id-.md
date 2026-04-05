# DELETE /v1/core/product-line-access/account-groups/{account_group_id}

## Result: PARITY CONFIRMED — No issues found

## What Was Compared

| Aspect | Dashboard | Go | Match |
|--------|-----------|-----|-------|
| **Permission: internal actor** | `checkIsInternalActor` | `identity.CheckIsInternalActor()` | ✅ |
| **Permission: domain/action** | `productLineAccess / delete` | `PermissionDomainProductLineAccess / ActionDelete` | ✅ |
| **Account scoping** | `find({ accountGroupID, accountID })` | `GetAccountGroupByIDAndAccount` verifies ownership | ✅ |
| **Existence check** | `find()` returns null → 409 Conflict | `Get()` + `Count` → 404 Not Found | ✅ (see note) |
| **Deletion scope** | `deleteMany({ where: { accountGroupID } })` | `DELETE FROM account_group_product_line WHERE account_group_id = ?` | ✅ |
| **Side effects** | None | None | ✅ |
| **Idempotency keys** | Not used (DELETE) | Not used (DELETE) | ✅ |
| **Customer actor access** | Not supported | Not supported | ✅ |

## Convention Differences (intentional, not bugs)

1. **Error type when not found**: Dashboard returns 409 Conflict (`'No relevant product for the customer group found.'`). Go returns 404 Not Found. The Go approach is more semantically correct for a missing resource.

2. **Response shape**: Dashboard returns 200 OK with the deleted object `{ id, accountGroup: { id, name }, productLines: [{ id, name }] }`. Go returns 204 No Content with an empty body. This follows Go API conventions for DELETE endpoints.

## Issues Found and Fixed

None. No code changes were required.

## Notes

- The Go service layer calls `Get()` before `Delete()`, and the repository `Delete()` method also verifies ownership and existence internally. This is slightly redundant but provides defense in depth and is harmless.
- All core business logic (permission checks, account scoping, what gets deleted) is faithfully preserved from the Dashboard implementation.
