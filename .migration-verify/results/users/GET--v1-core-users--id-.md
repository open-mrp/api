# GET /v1/core/users/{id} — Migration Verification

## Result: Issues found and fixed

## What was compared

| Aspect | Dashboard | Go | Match? |
|--------|-----------|-----|--------|
| Auth: internal actor check | checkIsInternalActor | identity.CheckIsInternalActor | Yes |
| Auth: self-access bypass | actor.id !== id skips perm check | identity.Actor.ID != identifier skips perm check | Yes |
| Auth: teamUsers.read perm | checkHasPermission(teamUsers, read) | CheckHasPermission(TeamUsers, Read) | Yes |
| Lookup by ID | OR condition: id/email/username | FindByID only | **Fixed** |
| Image URL normalization | S3 URLs → `/v1/users/{id}/photo` | Not present | **Fixed** |
| 404 on not found | HttpError.notFound | db.MapSQLError (sql.ErrNoRows → ResourceNotFoundError) | Yes |
| Response: id | Yes | Yes | Yes |
| Response: object | N/A (not in Dashboard) | Yes (added per Go conventions) | OK |
| Response: email | nullable | *string | Yes |
| Response: name | nullable | *string | Yes |
| Response: username | nullable | *string | Yes |
| Response: emailVerified/email_verified_at | nullable Date | *time.Time | Yes |
| Response: imageUrl/image_url | nullable string | *string | Yes |
| Response: createdAt/created_at | Date | time.Time | Yes |
| Response: updatedAt/updated_at | Date | time.Time | Yes |
| Idempotency | N/A (GET endpoint) | N/A (GET endpoint) | Yes |
| Side effects | None | None | Yes |

## Issues found and fixed

### 1. Flexible identifier lookup (fixed)

**Dashboard behavior:** The `UserRepo.find({ identifier })` method queries the database with an OR condition across `id`, `email`, and `username` fields, allowing users to look up a user by any of these three identifiers.

**Go behavior (before fix):** Only looked up by ID via `userRepo.FindByID()`.

**Fix:** Updated `GetUser` in `user_service.go` to try `FindByID` first, then fall back to `FindByEmail`, then `FindByUsername` — only falling through on not-found errors. This preserves the same flexible lookup behavior.

### 2. Image URL normalization (fixed)

**Dashboard behavior:** The `UserAdapter.map()` method checks if the image URL contains "augno-user-photos" (old S3 signed URLs) and normalizes it to `/v1/users/{id}/photo`.

**Go behavior (before fix):** No normalization. While the `UploadUserPhoto` service already stores normalized URLs (`/v1/core/users/{id}/photo`), legacy records may still contain raw S3 URLs.

**Fix:** Added `normalizeUserImageURL()` helper in `user_service.go` that converts legacy S3 URLs to `/v1/core/users/{id}/photo` before returning the response.

## Files modified

- `services/core-service/internal/service/user_service.go` — Added flexible identifier lookup (ID → email → username fallback) and image URL normalization

## Remaining concerns

- The Dashboard normalizes to `/v1/users/{id}/photo` while the Go API normalizes to `/v1/core/users/{id}/photo` (with the `core` prefix). This is correct — the Go API uses the `v1/core/` route prefix per migration conventions.
- The Dashboard's `UserAdapter.map()` validates that at least email or username must exist (throws if both null). The Go API does not enforce this — this is acceptable since it's a data integrity concern that should be handled at write time, not read time.
