# Verification: PATCH /v1/core/account-users/{id}

**Result: Parity confirmed — no issues found.**

## What was compared

### Permission checks
- **Dashboard**: `checkIsInternalActor` → self-edit allowed, otherwise `checkHasPermission(customers, update)` → `checkEditAccess`
- **Go**: `CheckIsInternalActor` → self-edit allowed, otherwise `CheckHasPermission(customers, update)` → `CheckTargetAccountSet` → `CheckEditAccess` (external targets only)
- **Verdict**: Equivalent. Go correctly skips `CheckEditAccess` for own-account targets since it's a no-op in that case.

### Updatable fields
- **Dashboard**: `firstName`, `lastName`, `email`, `username`, `role` (BasicInfo object), `department` (BasicInfo object)
- **Go**: `name` (single field), `custom_email`, `username`, `role_id`, `department_id`
- **Verdict**: Equivalent capabilities. Dashboard joins firstName+lastName into a single `user.name` DB column (`${firstName} ${lastName}`). Go exposes this directly as `name`. Role/department use ID references per Go API conventions.

### Validation rules
| Rule | Dashboard | Go |
|------|-----------|-----|
| Email cannot be set to null | Explicit check, throws 400 | N/A — `*string` with `omitempty` treats JSON null as "not provided" (safe) |
| firstName/lastName must be together | Explicit check | N/A — single `name` field |
| Duplicate email | `userRepo.checkDuplicateEmail` (excludes self) | `FindByEmail` + ID comparison (excludes self) |
| Duplicate username | `userRepo.checkDuplicateUsername` (excludes self, only if truthy) | `FindByUsername` + ID comparison (excludes self) |
| Account user existence | `checkExistence` by userID+accountID | `FindByAccountAndUserID` |

### Database queries
- **User profile update**: Both use COALESCE to only update provided fields (name, email, username). Go SQL: `UpdateUserProfile` with `COALESCE(sqlc.narg(...), column)`.
- **Account user update**: Both update role_id and department_id with COALESCE. Go SQL: `UpdateAccountUserRoleAndDepartment`.
- **Transaction scope**: Go wraps all updates in a single transaction. Dashboard uses Prisma's update which handles atomicity.

### Error handling
- 404 if account user not found: both
- 409 for duplicate email/username: both (Go uses `NewConflictErrorWithParam`)
- Auth errors for missing permissions: both

### Side effects
- None for either implementation (no emails, webhooks, billing events, or messages on update).

### Idempotency
- Go correctly implements idempotency key support with recovery points (required for PATCH endpoints per conventions).
- Dashboard does not use idempotency keys (legacy behavior).

### Response shape
- Go returns `AccountUser` resource with `role` and `department` as expandable subresources (objects with id, object, name).
- Dashboard returns `AccountUser` with `role` and `department` as BasicInfo objects.
- Both include: id, name, email, username, image_url, is_verified, status, last_used_at, created_at, updated_at.

## Issues found and fixed
None — the Go implementation faithfully reproduces all Dashboard business logic with appropriate API convention adjustments.
