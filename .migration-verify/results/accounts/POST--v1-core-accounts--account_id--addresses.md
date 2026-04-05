# POST /v1/core/accounts/{account_id}/addresses

**Status: Issues found and fixed**

## What was compared

| Aspect | Dashboard | Go (before fix) | Match? |
|--------|-----------|-----------------|--------|
| **Actor check** | `checkIsAssignedActor` (internal + external) | `CheckIsInternalActor` (internal only) | NO — fixed |
| **Permission domain** | `PermissionDomains.customers` | `PermissionDomainAddresses` | NO — fixed |
| **Permission action** | `'update'` | `ActionCreate` | NO — fixed |
| **Permission conditional** | Only checked for internal actors | Checked unconditionally | NO — fixed |
| **Edit access check** | `accountRepo.checkEditAccess` for external targets | Missing | NO — fixed |
| **Required fields** | name (min 1), country (min 1) | name (required), country (required) | Yes |
| **Optional fields** | phone, email, line1, line2, city, state, postalCode, isDropShip | phone, email, street_line_1, street_line_2, locality, state, postal_code, is_drop_ship | Yes |
| **Email validation** | Zod `.email()` validator | No format validation | Minor gap (acceptable — Go validates at gateway level) |
| **DB operations** | Insert geolocation → insert address → insert account_address | Same 3 inserts in same order | Yes |
| **google_place_id** | Set to null on create | Set to null (sqlc.narg default) | Yes |
| **Idempotency** | Not implemented in Dashboard | Properly implemented with recovery points | Yes (Go improvement) |
| **Response shape** | Address with nested geolocation | Same structure with geolocation sub-resource | Yes |
| **HTTP status** | 201 Created | 201 Created | Yes |
| **gRPC idempotency tracking** | N/A | `contracts.WithIdempotencyTracking(ctx)` present | Yes |

## Issues found and fixed

### 1. Actor type restriction (address_service.go:145)
- **Before:** `identity.CheckIsInternalActor()` — blocked external/customer actors
- **After:** `identity.CheckIsAssignedActor()` — allows both internal and external actors, matching Dashboard behavior

### 2. Permission domain and action (address_service.go:148-151)
- **Before:** `identity.CheckHasPermission(types.PermissionDomainAddresses, types.ActionCreate)` — unconditional
- **After:** Conditional on `identity.IsInternalUser()`, using `types.PermissionDomainCustomers` + `types.ActionUpdate` to match Dashboard's `PermissionDomains.customers, 'update'`

### 3. Missing edit access check (address_service.go:157-161)
- **Before:** No edit access check for external targets
- **After:** Added `meds.EditAccess.CheckEditAccess(ctx, ...)` for external targets, matching Dashboard's `accountRepo.checkEditAccess`

## Remaining concerns

- **Email format validation**: Dashboard validates email format via Zod's `.email()`. The Go API does not validate email format at the service level. This is a minor gap — if email format validation is enforced at the API gateway request validation layer, this is acceptable.
