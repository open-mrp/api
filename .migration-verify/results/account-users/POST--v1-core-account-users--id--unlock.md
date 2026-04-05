# POST /v1/core/account-users/{id}/unlock

## Result: Parity confirmed (with one structural difference noted)

## What was compared

| Aspect | Dashboard | Go | Match |
|--------|-----------|-----|-------|
| Actor check | `checkIsInternalActor` | `identity.CheckIsInternalActor()` | Yes |
| Permission check | `teamUsers`, `update` | `PermissionDomainTeamUsers`, `ActionUpdate` | Yes |
| Target account validation | Compares route param to identity header | `identity.CheckTargetAccountSet()` | Yes (equivalent) |
| Caller disabled check | Fetches caller, checks `disabled` status | Fetches caller via `GetDetail`, checks `AccountUserStatusDisabled` | Yes |
| Target user existence | `accountUserRepo.find()` → 404 | `GetDetail()` → error if not found | Yes |
| Removed user guard | `badRequest('Cannot unlock a removed user...')` | `NewValidationError('Cannot unlock a removed user...')` | Yes |
| Already active guard | `badRequest('User is already active.')` | `NewValidationError('User is already active.')` | Yes |
| Seat limit check | `AccountPlanSvc.canAddUser()` → counts active users vs plan max | `checkSeatLimit()` → counts active users vs plan max | Yes |
| Status update | `prisma.accountUser.updateMany({ statusCode: active })` | `UpdateStatus(ctx, accountID, userID, AccountUserStatusActive)` | Yes |
| Billing sync | `syncSubscriptionSeatCount()` (fire-and-forget) | `PublishReportSeatChange` + `PublishSyncSeats` via outbox | Yes (Go is more reliable) |
| Transaction wrapping | No explicit transaction | `withTx` wrapping update + billing publish | Yes (Go improvement) |

## Structural difference (not a bug)

**Response shape**: The Dashboard returns the updated `AccountUser` object with HTTP 200. The Go endpoint returns HTTP 204 No Content (`EmptyResource`). This is consistent with the Go `LockAccountUser` endpoint which also returns 204. To achieve full parity, the Go endpoint would need:

1. A new `UnlockAccountUserResponse` proto message containing `AccountUserDetail`
2. Changed RPC return type from `google.protobuf.Empty` to `UnlockAccountUserResponse`
3. Service method returning `*domain.AccountUserDetail`
4. API gateway returning `*apiresource.AccountUser` with HTTP 200

This is a deliberate design pattern in the Go API (lock/unlock/restore all return 204), so it may be an intentional simplification. Flagged for review but not changed.

## No issues fixed

All business logic is correctly preserved. The validation rules, permission checks, seat limit enforcement, status transitions, and billing sync are all functionally equivalent.
