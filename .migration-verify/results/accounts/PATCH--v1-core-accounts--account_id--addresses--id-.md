# PATCH /v1/core/accounts/{account_id}/addresses/{id}

**Status: Issues found and fixed**

## What was compared

| Aspect | Dashboard | Go (before fix) | Match? |
|--------|-----------|-----------------|--------|
| **Actor check** | `checkIsAssignedActor` (internal + external) | `CheckIsInternalActor` (internal only) | NO — fixed |
| **Permission domain** | `PermissionDomains.customers` | `PermissionDomainAddresses` | NO — fixed |
| **Permission action** | `'update'` | `ActionUpdate` | Yes |
| **Permission conditional** | Only checked for internal actors | Checked unconditionally | NO — fixed |
| **Edit access check** | `accountRepo.checkEditAccess` for external targets | Missing | NO — fixed |
| **Address ownership** | `addressRepo.checkIsInAccount` | `txRepo.IsInAccount` (inside tx) | Yes |
| **Existing fetch** | `addressRepo.find` to detect changes | `txRepo.Get` to detect changes | Yes |
| **Core geo change detection** | Checks line1, city, state, postalCode, country | Checks StreetLine1, Locality, State, PostalCode, Country | Yes |
| **Shared geolocation: create new + relink** | Creates new geo, connects via Prisma | `CreateGeolocation` + `RelinkGeolocation` | Yes |
| **Non-shared geolocation: update in-place** | Updates geo via Prisma update | `UpdateGeolocation` | Yes |
| **google_place_id cleared on geo change** | Set to null | Cleared via UpdateGeolocation (google_place_id = NULL) | Yes |
| **Metadata-only path** | Updates name, is_drop_ship, phone, email, line2 | Updates same fields + line2 on geo | Yes |
| **Partial update semantics** | `AddressUtils.schema.partial()` — all fields optional | All request fields are `*string`/`*bool` pointers | Yes |
| **Idempotency** | Not implemented in Dashboard | Properly implemented with recovery points | Yes (Go improvement) |
| **Response shape** | Address with flat geo fields | Address with nested geolocation sub-resource | Follows Go API conventions |
| **HTTP status** | 200 OK | 200 OK | Yes |
| **gRPC idempotency tracking** | N/A | `contracts.WithIdempotencyTracking(ctx)` present | Yes |
| **Side effects** | None | None | Yes |

## Issues found and fixed

### 1. Actor type restriction (address_service.go:228)
- **Before:** `identity.CheckIsInternalActor()` — blocked external/customer actors
- **After:** `identity.CheckIsAssignedActor()` — allows both internal and external actors, matching Dashboard behavior

### 2. Permission domain and conditionality (address_service.go:231-235)
- **Before:** `identity.CheckHasPermission(types.PermissionDomainAddresses, types.ActionUpdate)` — unconditional, wrong domain
- **After:** Conditional on `identity.IsInternalUser()`, using `types.PermissionDomainCustomers` + `types.ActionUpdate` to match Dashboard's `PermissionDomains.customers, 'update'`

### 3. Missing edit access check (address_service.go:240-245)
- **Before:** No edit access check for external targets
- **After:** Added `meds.EditAccess.CheckEditAccess(ctx, ...)` for external targets, matching Dashboard's `accountRepo.checkEditAccess`

## Remaining concerns

- **Email format validation**: Dashboard validates email format via Zod's `.email()`. The Go API does not validate email format at the service level. This is a minor gap — if email format validation is enforced at the API gateway request validation layer, this is acceptable.
- These are the exact same three issues found in the POST create address endpoint verification, indicating a systematic pattern in the original migration.
