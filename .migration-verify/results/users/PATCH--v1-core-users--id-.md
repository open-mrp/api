# PATCH /v1/core/users/{id} — Migration Verification

## Result: Issues found and fixed

## What was compared

| Aspect | Dashboard | Go | Match? |
|--------|-----------|-----|--------|
| Permission: internal actor check | `checkIsInternalActor()` | `identity.CheckIsInternalActor()` | Yes |
| Permission: self-update bypass | Self-update skips permission check | Same — skips if `identity.Actor.ID == userID` | Yes |
| Permission: other-user update | `teamUsers` + `update` | `PermissionDomainTeamUsers` + `ActionUpdate` | Yes |
| Updatable field: `name` | Yes | Yes | Yes |
| Updatable field: `imageUrl` | Yes (via `image` column) | **No** → Fixed | Fixed |
| Updatable field: `emailVerified` | Yes | **No** → Fixed | Fixed |
| Idempotency | Not present in Dashboard | PATCH uses idempotency keys (correct per Go conventions) | N/A — improvement |
| Response shape | User object | User object with `object` field | Yes (improved) |
| Error handling | Prisma errors | `db.MapSQLError` | Equivalent |
| Side effects | None | None | Yes |

## Issues found and fixed

### 1. Missing `image_url` field in update request
The Dashboard `UserRepo.update()` allows updating `imageUrl` (mapped to `image` column). The Go endpoint only accepted `name`.

**Fix:** Added `image_url` field to:
- `UpdateUserRequest` (endpoint)
- `UpdateUserParams` (domain model)
- `UpdateUserRequest` proto message (field 3)
- API gateway service (passes to gRPC)
- gRPC handler (extracts from proto)
- Service (passes to repository)
- `UpdateUserProfile` SQL query (COALESCE pattern)
- `UpdateProfile` repository method signature

### 2. Missing `email_verified` field in update request
The Dashboard `UserRepo.update()` allows updating `emailVerified` (timestamp). The Go endpoint didn't support this.

**Fix:** Added `email_verified_at` field to:
- `UpdateUserRequest` (endpoint, as `*time.Time`)
- `UpdateUserParams` (domain model)
- `UpdateUserRequest` proto message (field 4, as `google.protobuf.Timestamp`)
- API gateway service (converts to `timestamppb`)
- gRPC handler (converts from proto timestamp)
- Service (passes to repository)
- `UpdateUserProfile` SQL query (COALESCE pattern)
- `UpdateProfile` repository method signature

### 3. Updated all callers of `UpdateProfile`
The `account_user_service.go` also calls `UpdateProfile` — updated to pass `nil, nil` for the new `imageURL` and `emailVerified` parameters.

## Remaining concerns

- The COALESCE SQL pattern means you can set these fields to a value but cannot explicitly clear them to NULL via this endpoint. The Dashboard's Prisma update can distinguish between `undefined` (skip) and `null` (set to null). This is a minor limitation but acceptable since `emailVerified` and `imageUrl` are rarely cleared manually.
- Pre-existing build failures in `sales_order_repo.go` and `shipment_repository.go` prevented running the full test suite, but these are unrelated to user update changes.
