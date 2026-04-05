# PUT /v1/core/users/{id}/photo — Migration Verification

## Result: PARITY CONFIRMED — No issues found

## What Was Compared

| Aspect | Dashboard | Go | Match |
|--------|-----------|-----|-------|
| **HTTP Method** | PUT | PUT | Yes |
| **Auth: Internal actor check** | `checkIsInternalActor` | `identity.CheckIsInternalActor()` | Yes |
| **Auth: Self-edit bypass** | `identity.actor.id === userID` skips perm check | `identity.Actor.ID != userID` gates perm check | Yes |
| **Auth: Other-user permission** | `teamUsers` domain, `update` action | `PermissionDomainTeamUsers`, `ActionUpdate` | Yes |
| **Account header required** | Uses `identity.targetAccountID` | Checks `identity.TargetAccountID != nil` | Yes |
| **S3 key format** | `{accountID}/{userID}.png` | `{accountID}/{userID}.png` | Yes |
| **S3 encryption** | AES256 | AES256 | Yes |
| **Content-Type** | Passed from request header (default `image/png`) | Passed from `Content-Type` header | Yes |
| **DB update** | Sets `user.image` to `/v1/users/{userID}/photo` | Sets `user.image_url` to `/v1/core/users/{userID}/photo` | Expected difference (route prefix) |
| **Response shape** | `{ success: true }` | `{ success: true }` | Yes |
| **File validation** | None | None | Yes |
| **Idempotency** | None (PUT is naturally idempotent) | None (PUT is naturally idempotent) | Yes |
| **Side effects** | S3 upload + DB update | S3 upload + DB update | Yes |

## Notes

- The stored image URL uses `/v1/core/users/` prefix instead of `/v1/users/` — this is the expected route prefix change per migration conventions.
- The DB column name difference (`image` vs `image_url`) is a schema naming detail; both store the same relative URL path.
- No idempotency keys needed — PUT is naturally idempotent (repeated uploads overwrite the same S3 key and DB row).
- The Go implementation correctly skips `WithIdempotencyTracking` in the gRPC handler since this is a PUT endpoint.

## No Fixes Required
