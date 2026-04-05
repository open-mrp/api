# POST /v1/core/production-runs — Migration Verification

**Status: PARITY CONFIRMED — No issues found**

## What Was Compared

| Aspect | Dashboard | Go | Match? |
|--------|-----------|-----|--------|
| Permission: internal actor check | `checkIsInternalActor` | `identity.CheckIsInternalActor()` | Yes |
| Permission: domain/action | `productionRuns / create` | `PermissionDomainProductionRuns / ActionCreate` | Yes |
| Target account required | Implicit via `identity.targetAccountID` | `identity.CheckTargetAccountSet()` | Yes |
| Responsible user validation | Looks up account_user by userID + accountID, 404 if not found | Same — `FindByAccountAndUserID`, returns not-found error | Yes |
| Stores account_user ID (not user ID) | `responsibleAccountUserID: accountUser.id` | `params.ResponsibleUserID = accountUser.ID` | Yes |
| Auto-generated number | Via `SysPropertyRepo.findAndIncrementProductionRunNumber` | Via `MAX(CAST(number AS UNSIGNED)) + 1` | Equivalent |
| ID generation | Prisma auto-generates with `pr_` prefix | `id.GenID(id.ProductionRunIDPrefix)` | Yes |
| Idempotency | Not implemented | Full idempotency key support with recovery points | Improvement |
| Response: production run fields | id, number, responsibleUser, batches[], timestamps | id, number, responsible_user, batch_count, timestamps | See notes |
| HTTP status | 201 Created | 201 Created | Yes |

## Design Differences (Intentional, Not Bugs)

1. **Batch creation separated**: Dashboard allows passing `batches[]` in the create request body for inline creation. Go intentionally separates this into `POST /v1/core/production-runs/{id}/batches`. Both approaches work — the Dashboard frontend typically creates production runs with an empty batches array and adds batches separately via the scanning station workflow.

2. **Response shape follows Go API conventions**: Dashboard returns a `LightUser` object (`userID`, `name`, `email`, `image`) and inline `batches[]` array. Go returns an `AccountUser` sub-resource (`id`, `object`, `name`) marked as expandable, and a `batch_count` integer. Batches are fetched via a separate list endpoint. This follows the Go API's sub-resource and expandable conventions.

3. **Number generation mechanism**: Dashboard uses a `sys_property` table with a persistent counter (`productionRunNumber` type code). Go uses `SELECT COALESCE(MAX(CAST(number AS UNSIGNED)), 0) + 1`. Both produce sequential numbers. The Go approach works correctly within the transaction boundary and handles the edge case where numbers may have been manually edited (via PATCH) by computing from actual data.

## No Issues Found — No Fixes Applied

The core business logic is fully preserved:
- Authentication and authorization checks are identical
- Responsible user validation logic matches exactly
- The account_user ID (not user ID) is correctly stored
- Auto-numbering produces equivalent results
- Error cases are handled equivalently (not found, forbidden, etc.)
