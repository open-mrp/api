# Migration Verification: PUT /v1/core/child-accounts/{child_account_id}

## Result: Issues Found and Fixed

## What Was Compared

- **Permission checks**: Both require internal actor + `customers` domain + `update` action. ✅
- **Target account header**: Dashboard uses `parentAccountID` from route param; Go uses `identity.TargetAccountID` as parent. Equivalent. ✅
- **Circular relationship check**: Both check if parent's parent relation ID equals the child relation ID. ✅
- **Relation resolution**: Both resolve parent and child account IDs to relation IDs via `account_relation` table. ✅
- **Idempotency**: PUT semantics — idempotent via UPDATE. No idempotency keys needed. ✅
- **Error handling**: Both return not-found for missing parent/child relations, conflict for circular relationships. ✅
- **Response shape**: Go returns `ChildAccount` resource with nested `Account` sub-resource, matching conventions. ✅
- **Side effects**: None in either implementation. ✅

## Issues Found and Fixed

### Missing `checkEditAccess` on Child Account

**Dashboard behavior**: Before adding a child account, the dashboard calls `accountRepo.checkEditAccess()` on the child account ID. This verifies:
1. If target is same as actor → allowed
2. If target has an active billing plan (paid) → denied
3. If actor has no relation with target → denied
4. If target has other vendor relations → denied

**Go behavior (before fix)**: This check was entirely missing. Any internal actor with `customers.update` permission could add any account as a child, even paid accounts or accounts with multiple vendor relationships.

**Fix**: Added `MediatorFactory` to `childAccountSvcImpl` and called `meds.EditAccess.CheckEditAccess()` before resolving relations. The existing `EditAccessMed` in the Go codebase already implements the same logic as the dashboard's `hasEditAccess`. Updated wiring in `cmd/run.go` and `svctest/server.go` to pass the mediator factory.

### Files Modified
- `services/core-service/internal/service/child_account_service.go` — Added mediator factory field, validation, wiring, and `CheckEditAccess` call
- `services/core-service/cmd/run.go` — Pass `mediatorFactory` to `ChildAccountSvcConfig`
- `services/core-service/svctest/server.go` — Pass `mediatorFactory` to `ChildAccountSvcConfig`

## Remaining Concerns

None. The Go implementation now matches the dashboard behavior for the `add` (PUT) endpoint.
