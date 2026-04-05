# GET /v1/core/me — Migration Verification

**Status: Issues found and fixed**

## What was compared

| Aspect | Dashboard | Go | Match? |
|--------|-----------|-----|--------|
| Validation (actor type) | `checkIsValidActor` + `identity.type === user` | None (was missing) | Fixed |
| User lookup | `UserRepo.find()` by ID/email/username | `UserRepo.FindByID()` by ID only | OK (identity always provides ID) |
| Not-found handling | 404 "User not found." | 404 "User not found." | Yes |
| S3 photo check | Only when `targetAccountID` set | Only when `targetAccountID` set | Yes |
| S3 key format | `{accountID}/{userID}.png` | `{accountID}/{userID}.png` | Yes |
| Presigned URL duration | 3600s (1 hour) | 15 min (was wrong) | Fixed |
| Response shape | id, email, name, username, emailVerified, imageUrl, createdAt, updatedAt | id, object, email, name, username, email_verified_at, image_url, created_at, updated_at | Yes (Go adds `object` per convention) |
| Permission checks | No specific permission domain check | None | Yes |
| Idempotency | N/A (GET) | N/A (GET) | Yes |
| Side effects | None | None | Yes |

## Issues found and fixed

### 1. Missing identity validation (gateway service)

**File:** `services/api-gateway/endpoints/tenancy/service.go`

The Dashboard calls `checkIsValidActor(this.identity)` (ensures authentication + valid actor status) and then checks `this.identity.type !== IdentityType.user` to reject API key and agent identities. The Go implementation had no identity validation — it silently extracted the actor ID without checking authentication or identity type.

**Fix:** Added `identity.CheckIsUser()` call before building the gRPC request. This checks both authentication and that the identity type is "user", matching the Dashboard behavior. Also simplified the request building since after the check, `identity.Actor` is guaranteed non-nil.

### 2. Presigned URL duration mismatch (core service)

**File:** `services/core-service/internal/service/tenancy_service.go`

Dashboard generates presigned URLs with 3600 seconds (1 hour) expiry. Go was using 15 minutes.

**Fix:** Changed from `15*time.Minute` to `60*time.Minute`.

## Notes

- The TenancyEndpointGroup exists in code but is not registered in `init_groups.go`. This is an infrastructure wiring issue, not a business logic parity concern.
- The Dashboard's `UserAdapter.map()` normalizes old S3 URLs to endpoint paths — this is irrelevant for the `/me` endpoint since the photo URL is always freshly generated from S3.
- The Go response adds the standard `object: "user"` field per API resource conventions.
