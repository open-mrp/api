# GET /v1/core/carriers/{carrier_id}/options/{id}

## Result: PARITY CONFIRMED

No issues found. The Go implementation faithfully reproduces all Dashboard business logic.

## What Was Compared

| Aspect | Dashboard | Go | Match |
|--------|-----------|-----|-------|
| **Permission: assigned actor** | `checkIsAssignedActor()` | `identity.CheckIsAssignedActor()` | Yes |
| **Permission: internal read** | `PermissionDomains.carriers` + `read` (internal only) | `PermissionDomainCarriers` + `ActionRead` (internal only) | Yes |
| **Permission: customer/supplier** | Bypass permission check | Bypass + external target read access check | Yes (Go stricter) |
| **Carrier-option-in-carrier check** | `checkIfCarrierOptionIsInCarrier()` before fetch | `IsInCarrier()` before fetch | Yes |
| **DB query** | `WHERE id = :id AND (accountID = :accountID OR accountID IS NULL)` | `WHERE id = :id AND (account_id = :account_id OR account_id IS NULL)` | Yes |
| **Not found error** | `HttpError.notFound('Carrier option not found.')` | `apierror.NewResourceNotFoundError("Carrier option not found.")` | Yes |
| **Response fields** | id, name, code, serviceLevelToken, isPortalEnabled, isDefault, createdAt, updatedAt | id, object, name, code, service_level_token, is_portal_enabled, is_default, created_at, updated_at | Yes (+object) |
| **Idempotency** | N/A (GET) | N/A (GET) | Yes |
| **Side effects** | None | None | Yes |

## Notes

- Go adds `object` field per API resource conventions — expected enhancement, not a discrepancy.
- Go adds `CheckReadAccess` for external targets — stricter than Dashboard but correct.
- Both implementations query carrier_option allowing `account_id IS NULL` to support system-synced default options.
- No fixes were required.
