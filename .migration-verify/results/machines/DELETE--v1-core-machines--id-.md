# DELETE /v1/core/machines/{id} — Migration Verification

## Result: PARITY CONFIRMED

No issues found. The Go implementation correctly replicates all Dashboard business logic.

## What Was Compared

| Aspect | Dashboard | Go | Match |
|--------|-----------|-----|-------|
| Permission: internal actor | `checkIsInternalActor()` | `identity.CheckIsInternalActor()` | Yes |
| Permission: domain/action | `machines` / `delete` | `PermissionDomainMachines` / `ActionDelete` | Yes |
| Account scoping | `department: { accountID }` | SQL JOIN department ON account_id | Yes |
| Delete type | Hard delete | Hard delete | Yes |
| Account ID source | `identity.targetAccountID` | `*identity.TargetAccountID` | Yes |
| Side effects | None | None | Yes |
| Idempotency keys | N/A (DELETE) | N/A (DELETE) | Yes |

## Acceptable Convention Differences

- **Response**: Dashboard returns HTTP 200 with deleted machine resource; Go returns HTTP 204 No Content. This follows the Go API convention for DELETE endpoints and is intentional.
- **Existence check**: Go explicitly verifies the machine exists before deleting (via `Get()`), while Dashboard relies on Prisma throwing on delete of non-existent record. Both produce a not-found error if the machine doesn't exist.
- **Transaction**: Go wraps the delete in a transaction; Dashboard does not. This is a robustness improvement, not a behavioral difference.

## No Issues Found
