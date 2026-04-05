# DELETE /v1/core/accounts/{account_id}/addresses/{id}

**Status: Issues found and fixed**

## What was compared

| Aspect | Dashboard | Go (before fix) | Match? |
|--------|-----------|-----------------|--------|
| **Actor check** | `checkIsAssignedActor` (internal + external) | `CheckIsInternalActor` (internal only) | NO — fixed |
| **Permission domain** | `PermissionDomains.customers` | `PermissionDomainAddresses` | NO — fixed |
| **Permission action** | `'update'` | `ActionDelete` | NO — fixed |
| **Permission conditional** | Only checked for internal actors | Checked unconditionally | NO — fixed |
| **Edit access check** | `accountRepo.checkEditAccess` for external targets | Missing | NO — fixed |
| **Address ownership** | `addressRepo.checkIsInAccount` | `repo.IsInAccount` | Yes |
| **In-use check: sales orders** | Checks billing_address_id and shipping_address_id | Same check via `CheckAddressUsedInSalesOrder` | Yes |
| **In-use check: invoices** | Checks billing_address_id | Same check via `CheckAddressUsedInInvoice` | Yes |
| **In-use check: shipments** | Checks shipping_address_id | Same check via `CheckAddressUsedInShipment` | Yes |
| **In-use check: account defaults** | Checks default billing/shipping address | Same check via `CheckAddressUsedAsAccountDefault` | Yes |
| **In-use error messages** | Descriptive messages with record number/name | Same descriptive messages with record number/name | Yes |
| **Delete: account_address** | `deleteMany` on account_address | `DeleteAccountAddressByAddressID` | Yes |
| **Delete: address** | `delete` on address | `DeleteAddress` | Yes |
| **Transactional delete** | Implicit Prisma transaction | Explicit `withTx` wrapper | Yes |
| **Response shape** | Returns deleted Address object (200 OK) | Returns empty (204 No Content) | Intentional Go convention |
| **Side effects** | None | None | Yes |

## Issues found and fixed

### 1. Actor type restriction (address_service.go:397)
- **Before:** `identity.CheckIsInternalActor()` — blocked external/customer actors
- **After:** `identity.CheckIsAssignedActor()` — allows both internal and external actors, matching Dashboard behavior

### 2. Permission domain, action, and conditionality (address_service.go:400-403)
- **Before:** `identity.CheckHasPermission(types.PermissionDomainAddresses, types.ActionDelete)` — unconditional, wrong domain and action
- **After:** Conditional on `identity.IsInternalUser()`, using `types.PermissionDomainCustomers` + `types.ActionUpdate` to match Dashboard's `PermissionDomains.customers, 'update'`

### 3. Missing edit access check (address_service.go:409-414)
- **Before:** No edit access check for external targets
- **After:** Added `meds.EditAccess.CheckEditAccess(ctx, ...)` for external targets, matching Dashboard's `accountRepo.checkEditAccess`

## Remaining concerns

- **Response shape:** Dashboard returns the deleted address object with 200 OK; Go returns 204 No Content with empty body. This follows Go API conventions for DELETE endpoints and is an intentional design choice.
- These are the exact same three auth issues found in the POST and PATCH address endpoint verifications, confirming a systematic pattern in the original migration.
