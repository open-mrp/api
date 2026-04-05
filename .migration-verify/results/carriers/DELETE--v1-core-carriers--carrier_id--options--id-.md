# Verification: DELETE /v1/core/carriers/{carrier_id}/options/{id}

**Status: PARITY CONFIRMED** (with minor convention differences noted below)

## What Was Compared

| Aspect | Dashboard (Express.js) | Go API | Match? |
|--------|----------------------|--------|--------|
| Actor check | `checkIsInternalActor` | `identity.CheckIsInternalActor()` | Yes |
| Permission check | `carriers:delete` | `PermissionDomainCarriers, ActionDelete` | Yes |
| Carrier membership check | `checkIfCarrierOptionIsInCarrier` → 404 | `IsInCarrier` → 404 | Yes |
| Default option guard | `CarrierOptionUtils.isDefault(id)` → 403 | `optionRepo.Get` + `IsDefault` → validation error | See note 1 |
| Existence check | `checkExistence` → 404 | Implicit via `Get` + `Delete` rows affected check | Yes |
| DB delete | `DELETE WHERE accountID, id` | `DELETE WHERE account_id, id` | Yes |
| Response | 200 + deleted CarrierOption object | 204 No Content | See note 2 |
| Side effects | None | None | Yes |
| Idempotency | N/A (DELETE is idempotent by design) | No idempotency keys (correct) | Yes |

## Notes

### 1. Default option error type (acceptable difference)

- **Dashboard:** Returns HTTP 403 Forbidden via `HttpError.forbidden('Default carrier option cannot be deleted.')`
- **Go:** Returns validation error via `NewValidationError("Default carrier options cannot be deleted.")`

The Dashboard's use of 403 for a business rule violation is arguably a misuse of the status code (403 = authorization/permission issue). The Go API's use of a validation error is more semantically appropriate. Both prevent deletion of default options. **No change needed.**

Both implementations achieve the same check — Dashboard checks if the ID is in a known set of default IDs (`CarrierOptions` enum), while Go checks the `is_default` DB column. The DB-based approach is more robust.

### 2. Response shape (Go API convention)

- **Dashboard:** Returns the deleted `CarrierOption` object with HTTP 200
- **Go:** Returns HTTP 204 No Content with empty body

This is consistent with the Go API's convention — all other DELETE endpoints return `EmptyResource` with 204. Changing this one endpoint would be inconsistent. **No change needed.**

## Business Logic Parity

All core business logic is preserved:
1. Only internal actors with `carriers:delete` permission can delete
2. Default (system-synced) carrier options cannot be deleted
3. The carrier option must belong to the specified carrier
4. The carrier option must exist in the account
5. No side effects (no emails, webhooks, or messages)
6. Delete is scoped to `account_id` for multi-tenancy

## No Issues Found — No Code Changes Required
