# DELETE /v1/core/child-accounts/{child_account_id}

## Result: Issue found and fixed

## What was compared

- **Permission checks**: Both check `checkIsInternalActor` + `checkHasPermission(customers, update)` + target account set ✅
- **Edit access check**: Dashboard calls `accountRepo.checkEditAccess(authAccountID, childAccountID)` — Go was **missing** this check ❌ → Fixed
- **DB queries**: Both resolve parent and child relation IDs via owner+counterparty lookup, then clear `parent_account_relation_id` ✅
- **Idempotency**: Go's SQL uses `WHERE parent_account_relation_id = ?` so clearing an already-cleared relation is a no-op ✅
- **Error handling**: Both return 404 if parent or child relation not found ✅
- **Side effects**: None in either implementation ✅

## Issues found and fixed

### 1. Missing edit access check (FIXED)

**Dashboard**: `accountRepo.checkEditAccess({ authAccountID, targetAccountID: childAccountID })` — verifies the actor can edit the child account (checks: same account, no active billing plan, relation exists, no other owner relations).

**Go (before fix)**: No edit access check. The `AddChildAccount` method already used `meds.EditAccess.CheckEditAccess()` but `RemoveChildAccount` did not.

**Fix**: Added `meds.EditAccess.CheckEditAccess(ctx, ownerAccountID, childAccountID)` to `RemoveChildAccount` in `child_account_service.go`, matching the pattern used by `AddChildAccount`.

## Acceptable differences

- **Response shape**: Dashboard returns HTTP 200 with the updated ChildCustomer object. Go returns HTTP 204 No Content with empty body. This is a deliberate design choice — DELETE endpoints in the Go API consistently use 204.
- **Route structure**: Dashboard uses `DELETE /v1/identity/:accountID/children/:childAccountID` (parent ID in path). Go uses `DELETE /v1/core/child-accounts/{child_account_id}` (parent derived from Augno-Account header). This is consistent with the Go API's header-based account targeting pattern.
