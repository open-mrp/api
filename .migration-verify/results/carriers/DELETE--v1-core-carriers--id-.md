# DELETE /v1/core/carriers/{id} — Migration Verification

## Result: Issues found and fixed

## What was compared

| Aspect | Dashboard | Go | Match? |
|--------|-----------|-----|--------|
| Permission checks | Internal actor + `carriers.delete` | Internal actor + `carriers.delete` | ✅ |
| Carrier existence check | `find` then null check | `Get` returns not-found error | ✅ |
| Shippo deactivation (non-sandbox) | Errors propagate, blocking deletion | ~~Errors swallowed~~ → Fixed to propagate | ✅ (fixed) |
| Sandbox skip | `SandboxGuard.isSandbox` | `accountSvc.GetAccountContext` → `IsSandbox` | ✅ |
| Carrier options deletion | Hard delete (`deleteMany`) | Hard delete (`DELETE FROM`) | ✅ |
| Carrier soft delete | Sets `deletedAt` | Sets `deleted_at` and `updated_at` | ✅ |
| Transaction wrapping | No explicit tx (Prisma individual calls) | Explicit transaction for options + soft delete | ✅ (Go is stricter) |
| Response | HTTP 200 with carrier data | HTTP 204 No Content | Acceptable convention difference |

## Issues found and fixed

### 1. Shippo deactivation error handling (fixed)

**File:** `services/core-service/internal/service/carrier_service.go` line 464

**Problem:** The Go code swallowed the error from `shippoClient.DeactivateCarrierAccount()` with a comment "Log but don't block deletion". The Dashboard code has no try/catch around the equivalent call, so errors propagate and block the deletion.

**Fix:** Changed to `return tracing.Trace(span, apiErr)` so the error propagates, matching Dashboard behavior.

## Notes

- The Go API returns 204 (No Content) vs Dashboard's 200 with data. This is a standard REST convention for DELETE and is an intentional API design choice, not a parity issue.
- The Go implementation wraps the options deletion + carrier soft delete in a transaction, which is actually an improvement over the Dashboard's non-transactional approach.
