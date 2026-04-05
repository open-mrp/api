# Verification: DELETE /v1/core/departments/{id}

## Result: Parity Confirmed — No fixes needed

## What was compared

| Aspect | Dashboard | Go | Match |
|--------|-----------|-----|-------|
| Permission check | `checkIsInternalActor` + `checkHasPermission(departments, delete)` | `CheckIsInternalActor()` + `CheckHasPermission(PermissionDomainDepartments, ActionDelete)` | Yes |
| Account isolation | Filters by `accountID` in Prisma query | Filters by `account_id` in SQL WHERE clause | Yes |
| Existence check | `checkExistence({ id, accountID })` before delete | `Get()` with same params before delete | Yes |
| Not-found error | `HttpError.notFound('Department not found')` | `apierror.NewResourceNotFoundError("Department not found.")` | Yes |
| Delete query | `prisma.department.delete({ where: { id, accountID } })` | `DELETE FROM department WHERE id = ? AND account_id = ?` | Yes |
| FK constraint handling | Prisma throws on FK violation (scanning stations/machines) | MySQL FK error mapped via `db.MapSQLError` | Yes |
| Response status | HTTP 200 with deleted department object | HTTP 204 with empty body | Intentional difference |
| Idempotency keys | N/A (DELETE) | N/A (DELETE — idempotent by design) | Yes |
| Side effects | None | None | Yes |

## Notes

- **Response shape difference** (HTTP 200 + body vs HTTP 204 + empty): This is consistent with all other Go delete endpoints and is an intentional architectural decision, not a parity gap.
- The Go endpoint description mentions "Deletion will fail if the department still has associated scanning stations or machines" — this is enforced at the database level via foreign key constraints in both implementations, not via explicit application-level checks.
- The Go implementation wraps the delete in a transaction, which is a stricter guarantee than the Dashboard's single Prisma call, but functionally equivalent.
