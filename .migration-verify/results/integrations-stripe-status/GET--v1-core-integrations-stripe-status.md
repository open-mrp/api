# Verification: GET /v1/core/integrations/stripe/status

**Status: PARITY CONFIRMED** — No issues found.

## What was compared

| Aspect | Dashboard | Go | Match |
|--------|-----------|-----|-------|
| **Request params** | None | None | ✅ |
| **Auth** | `checkIsAssignedActor` | `CheckIsAssignedActor()` + `CheckTargetAccountSet()` | ✅ |
| **Customer actor** | `isCustomerActor` → `checkReadAccess` | `IsExternalTarget()` → `ReadAccess.CheckReadAccess` | ✅ |
| **Permission domain** | None required | None required | ✅ |
| **DB query** | `findFirst({ accountID, integrationCode })` | `COUNT(*) WHERE account_id AND integration_code` | ✅ |
| **Active filter** | Not filtered by `isActive` | Not filtered by `isActive` | ✅ |
| **Response** | `{ hasStripeIntegration: bool }` | `{ object, has_stripe_integration: bool }` | ✅ |
| **Side effects** | None | None | ✅ |
| **Idempotency** | GET (inherent) | GET (inherent) | ✅ |

## Notes

- Go response includes the `object: "stripe_status"` field per API resource conventions — this is expected and correct for the migration.
- Both implementations use simple existence checks (Dashboard via `findFirst`, Go via `COUNT > 0`) without filtering by `isActive` status.
- Go adds `CheckTargetAccountSet()` which is a stricter validation than Dashboard but does not break parity — it ensures the required header is present.
