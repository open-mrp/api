# Verification: GET /v1/core/account-users/{id}

**Result: Parity confirmed — no code changes required.**

## What was compared

| Area | Dashboard | Go | Match |
|------|-----------|-----|-------|
| **Permission: assigned actor** | `checkIsAssignedActor` | `identity.CheckIsAssignedActor()` | Yes |
| **Permission: internal read** | `checkHasPermission(customers, read)` | `identity.CheckHasPermission(PermissionDomainCustomers, ActionRead)` | Yes |
| **Cross-account read access** | `accountRepo.checkReadAccess()` (always called; short-circuits if same account) | `meds.ReadAccess.CheckReadAccess()` (only for external targets) | Functionally equivalent |
| **Target account required** | Implicit via route param `accountID` | `identity.CheckTargetAccountSet()` | Yes |
| **DB query** | `findFirst({ userID, accountID })` — no status filter | `GetAccountUserDetail` — `WHERE au.account_id = ? AND au.user_id = ?` — no status filter | Yes |
| **Joins** | Prisma includes user, role, department | SQL JOINs user (inner), role (left), department (left) | Yes |
| **Not-found handling** | 404 "Account user not found." | 404 via `MapSQLError(sql.ErrNoRows)` → "Resource not found." | Yes (minor message difference, consistent with Go convention) |
| **Response fields** | id, statusCode, name, email, username, photoUrl, emailVerified, role, department, createdAt, updatedAt | id, object, name, email, username, image_url, is_verified, status, role, department, last_used_at, created_at, updated_at | Yes (Go adds `object` and `last_used_at` per API conventions) |
| **Expandable sub-resources** | role, department (via Prisma includes) | role, department (via IncludeConfig) | Yes |
| **Idempotency** | N/A (GET endpoint) | N/A (GET endpoint) | Yes |
| **Side effects** | None | None | Yes |

## Notes

1. **Photo URL handling**: The Dashboard generates signed S3 URLs on-the-fly by checking for `${accountID}/${userID}.png` in S3, while Go returns the `image_url` column from the `user` table directly. This is an architectural difference — the Go API stores/serves image URLs differently and does not sign URLs per-request. This is consistent with how Go handles image URLs across all endpoints.

2. **Read access check**: The Dashboard always calls `checkReadAccess()`, which returns `true` immediately if `authAccountID === targetAccountID`. The Go implementation only calls `CheckReadAccess` when `identity.IsExternalTarget()` is true. These are functionally equivalent — internal (same-account) users skip the check in both cases.

3. **Error message**: Go uses the generic "Resource not found." message via `MapSQLError`, while Dashboard uses "Account user not found." This is consistent with Go's convention of using `MapSQLError` for all not-found cases.
