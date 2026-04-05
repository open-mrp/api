# Verification: PUT /v1/core/picks/{pickId}/lines/{id}/actions/void

**Status: Issues found and fixed**

## What was compared

| Aspect | Dashboard | Go | Match? |
|--------|-----------|-----|--------|
| HTTP method | PUT | PUT | Yes |
| Permission check | internal actor + picks:update | internal actor + picks:update | Yes |
| Account isolation | ownerAccountID via order→orderLine join | pick IsInAccount + line IsInPick | Yes (equivalent) |
| Packed line guard | `packedAt: null` in WHERE clause | **Missing** → Fixed | Fixed |
| DB mutation | Sets quantity measure to 0 | Sets quantity value to 0 | Yes |
| Response | Returns voided pick line | Returns voided pick line | Yes |
| Side effects | None | None | Yes |
| Idempotency keys | N/A (PUT is idempotent by design) | N/A (PUT) | Yes |

## Issues found and fixed

### 1. Missing packed-line guard (fixed)

**Dashboard behavior:** The Prisma `update` includes `packedAt: null` in the WHERE clause, which means voiding a packed pick line fails (no matching record).

**Go behavior before fix:** No check for `PackedAt` — would happily void already-packed lines.

**Fix:** Added a check in `pick_line_svc.go` VoidPickLine method that fetches the pick line first and returns a validation error if `PackedAt != nil`.

**File:** `services/core-service/internal/service/pick_line_svc.go` (lines ~252-258)

## No remaining concerns

The endpoint is now at parity with the Dashboard implementation.
