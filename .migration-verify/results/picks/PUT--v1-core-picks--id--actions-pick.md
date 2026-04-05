# PUT /v1/core/picks/{id}/actions/pick

**Status: Issues found and fixed**

## What was compared

- Permission checks (actor type, permission domain, action)
- Validation rules (account ownership, pick existence)
- DB queries and logic (pick remaining quantity calculation, filters)
- Error handling (not found, auth errors)
- Side effects (none expected for this endpoint)
- Response shape (pick detail with lines, departments)
- Idempotency (PUT — idempotent by design, no idempotency keys needed)

## Parity assessment

### Permission checks — MATCH
Both Dashboard and Go check:
- Internal actor only
- `picks` permission domain with `update` action
- Target account ID must be set

### Core logic — MATCH (with minor difference)
Both calculate remaining quantity per pick line as: `ordered_qty - SUM(other_pick_lines_qty)`.

The Dashboard processes lines sequentially (one `pickRemainingQuantity` call per line) and includes the current line in the sum. The Go uses a single SQL UPDATE that excludes the current line (`pl2.id != pl.id`).

For the standard flow (lines start at qty=0 before picking), both approaches produce identical results. The Go approach is actually more correct for edge cases where a line already has a non-zero value.

### Filters — MATCH (effectively)
Dashboard filters pick lines by `finishedAt IS NULL` (on pick) AND `packedAt IS NULL`. Go filters only by `packed_at IS NULL`. Since finished picks have all lines packed, the `packed_at IS NULL` filter alone produces the same result.

### Account scoping — MATCH
Dashboard scopes the UPDATE by `ownerAccountID` on the order. Go doesn't scope the UPDATE SQL by account but validates account ownership via `Get()` within the same transaction, so unauthorized updates are rolled back.

## Issues found and fixed

### 1. Missing departments in response
**Problem:** The Go `PickAllLines` method did not fetch departments for the returned pick. The Dashboard's `find()` method (called after picking) returns the full pick including departments via `PickAdapter.select()`.

**Fix:** Added `txRepo.GetDepartments()` call in `PickAllLines` to populate `pick.Departments`, matching the `GetPick` method pattern.

**File:** `services/core-service/internal/service/pick_svc.go`

## No remaining concerns
