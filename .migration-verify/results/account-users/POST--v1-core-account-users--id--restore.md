# POST /v1/core/account-users/{id}/restore

## Result: PARITY CONFIRMED — No issues found

## What was compared

| Aspect | Dashboard | Go | Match |
|---|---|---|---|
| **Actor check** | `checkIsInternalActor` | `identity.CheckIsInternalActor()` | ✅ |
| **Permission** | `teamUsers` / `update` | `PermissionDomainTeamUsers` / `ActionUpdate` | ✅ |
| **Caller disabled check** | Fetches caller's account user, checks `statusCode === disabled` | `GetDetail` for caller, checks `StatusCode == AccountUserStatusDisabled` | ✅ |
| **Target user exists** | `accountUserRepo.find()`, throws 404 if not found | `GetDetail()`, returns error if not found | ✅ |
| **Status validation** | Checks `statusCode !== removed`, throws 400 | Checks `StatusCode != AccountUserStatusRemoved`, returns validation error | ✅ |
| **Seat limit check** | `AccountPlanSvc.canAddUser()` — gets plan, gets seat limit, counts active users | `checkSeatLimit()` — gets plan code, gets seat limit, counts active users | ✅ |
| **Status update** | `prisma.accountUser.updateMany` sets status to `active` | `UpdateStatus()` with `AccountUserStatusActive` in transaction | ✅ |
| **Billing sync** | Fire-and-forget `syncSubscriptionSeatCount()` | `PublishReportSeatChange` + `PublishSyncSeats` via outbox pattern (in transaction) | ✅ (improved) |
| **Account ID source** | Route param `targetAccountID` + header check for match | Header only (`identity.TargetAccountID`) — no route param needed | ✅ |

## Notes

- **Response shape**: Dashboard returns the updated user (HTTP 200). Go returns empty (HTTP 204). This is an intentional design choice — the Go route uses a different path (`/v1/core/account-users/{id}/restore` vs `/v1/identity/:accountID/users/:userID/restore`) and follows Go API conventions for action endpoints.
- **Billing sync reliability**: Go version is improved — uses transactional outbox pattern instead of fire-and-forget HTTP, ensuring at-least-once delivery.
- **Atomicity**: Go version wraps the status update and billing event publishing in a single transaction, which is more robust than the dashboard's separate steps.
- **Error messages**: Minor wording differences (e.g., "User is not in removed status" vs "User is not removed") but semantically equivalent error types.

## No fixes required
