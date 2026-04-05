# PATCH /v1/core/carriers/{carrier_id}/options/{id}

## Result: PARITY CONFIRMED

No issues found. The Go implementation correctly matches the Dashboard business logic.

## What Was Compared

| Aspect | Dashboard | Go | Match |
|--------|-----------|-----|-------|
| Permission checks | internal actor + `carriers.update` | internal actor + `carriers.update` + target account | Yes |
| Default option guard | `CarrierOptionUtils.isDefault(id)` (ID pattern) | `existing.IsDefault` (DB field) | Yes (Go is more robust) |
| Carrier membership check | `isInCarrier({ id, carrierID })` | `IsInCarrier(id, carrierID)` | Yes |
| Existence check | `checkExistence({ id, accountID })` | Via `Get()` + rows affected check | Yes |
| Updatable fields | `name`, `code`, `isPortalEnabled`, `isDefault` | `name`, `code`, `is_portal_enabled`, `is_default` | Yes |
| Account scoping | Prisma `where: { id, accountID }` | SQL `AND account_id = ?` | Yes |
| Error: default option | 403 Forbidden | Validation error | Yes (appropriate mapping) |
| Error: not in carrier | 404 Not Found | Resource not found | Yes |
| Response status | 200 OK | 200 OK | Yes |
| Idempotency | Not implemented | Recovery point pattern | Yes (Go improvement) |

## Go Improvements Over Dashboard

1. **Code uniqueness check**: Go validates `ExistsByCodeInCarrier` before update — Dashboard does not. This is a beneficial addition.
2. **DB-based default check**: Go checks `existing.IsDefault` from DB rather than ID pattern matching, which is more reliable.
3. **Idempotency**: Go uses recovery point pattern for PATCH idempotency as required by conventions.

## No Issues Found
