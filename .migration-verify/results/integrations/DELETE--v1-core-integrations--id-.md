# DELETE /v1/core/integrations/{id} — Verification Result

## Status: PARITY CONFIRMED

No issues found. The Go implementation correctly matches the Dashboard behavior.

## What Was Compared

| Aspect | Dashboard | Go | Match |
|--------|-----------|-----|-------|
| **Auth: internal actor** | `checkIsInternalActor()` | `identity.CheckIsInternalActor()` | Yes |
| **Auth: admin role** | `roleTypeCode !== RoleTypes.admin` | `identity.CheckIsAdmin()` | Yes |
| **Auth: target account** | `this.identity.targetAccountID` | `identity.CheckTargetAccountSet()` | Yes |
| **DB: existence check** | `checkExistence({id, accountID})` | `r.Get(ctx, accountID, id)` before delete | Yes |
| **DB: delete query** | `delete WHERE {id, accountID}` | `DELETE WHERE id = ? AND account_id = ?` | Yes |
| **Error: not found** | `HttpError.notFound('Account integration not found.')` | `apierror.NewResourceNotFoundError("Account integration not found.")` | Yes |
| **Response: status** | HTTP 200 | HTTP 200 | Yes |
| **Response: body** | Deleted integration object | Deleted integration object | Yes |
| **Response: fields** | id, name, integrationCode, isActive, createdAt, updatedAt | id, name, integration_code, is_active, created_at, updated_at | Yes |
| **Idempotency** | N/A (DELETE) | No idempotency keys (correct) | Yes |
| **Side effects** | None | None | Yes |

## Notes

- Go fetches the record before deleting (MySQL has no RETURNING clause), then verifies rows affected. This is functionally equivalent to Dashboard's `checkExistence()` + `delete()` pattern.
- No side effects in either implementation — straightforward delete and return.
