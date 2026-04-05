# POST /v1/core/shipments/actions/estimate-rate

## Status: Issues found and fixed

## What was compared
- Authorization checks (actor types, permission domains)
- Freight exemption logic (product line, customer/group, shipping term)
- Shipping term flat rate and minimum order value logic
- Shippo integration checks and error handling
- Shipping rate markup
- Response shape and field types
- Request validation (required fields)

## Issues found and fixed

### 1. Missing customer actor authorization (FIXED)
**Dashboard:** Supports both internal actors (permission check) and customer actors (validates `actor.accountID === customerID`).
**Go (before):** Only checked internal actor permissions; no customer actor validation.
**Fix:** Added `else if identity.IsCustomerUser()` branch that validates the customer actor's account ID matches the `customerID` parameter. Non-internal/non-customer actors now get a validation error.

### 2. Missing product line freight exemption check (FIXED)
**Dashboard:** Calls `productLineRepo.isFreightExempt()` — if any product line in the request is freight exempt, returns 0.
**Go (before):** No product line freight check.
**Fix:** Added loop over `params.ProductLineIDs` using existing `ProductLineRepo.Get()` to check `IsFreightExempt`.

### 3. Missing customer/group freight exemption check (FIXED)
**Dashboard:** Calls `CustomerUtils.isCustomerOrGroupFreightExempt(customer)` which checks customer, type group, and price group freight status.
**Go (before):** No customer freight check.
**Fix:** Added check for `customer.FreightPolicy == constants.FreightPolicyFree` (the Go model consolidates the Dashboard's multi-check into a computed status).

### 4. Missing shipping term logic (FIXED)
**Dashboard:** Checks shipping term for free freight, flat rate, and minimum order value threshold.
**Go (before):** No shipping term checks.
**Fix:** Added full shipping term logic:
- Free freight type → return 0
- Flat rate type → return flat rate value
- Minimum order value → return 0 if order total exceeds threshold

### 5. Error instead of 0 for missing Shippo config (FIXED)
**Dashboard:** Returns 0 when carrier has no `shippoCarrierAccountId` or when no Shippo integration exists.
**Go (before):** Returned validation errors for missing Shippo carrier account ID and missing service level token.
**Fix:** Changed to return 0 gracefully, matching Dashboard behavior.

### 6. Missing Shippo integration check (FIXED)
**Dashboard:** Checks `hasIntegration(integrationCode: 'shippo')` before attempting to fetch credentials.
**Go (before):** Directly fetched credentials without checking integration existence.
**Fix:** Added `HasIntegration()` check before `GetEncryptedCredentials()`.

### 7. Missing shipping rate markup (FIXED)
**Dashboard:** Applies `applyShippingMarkup()` which multiplies by 1.1 (10% markup).
**Go (before):** Returned raw Shippo rate.
**Fix:** Added `shippingRateMarkup = 1.1` constant and applied it to non-zero rates.

### 8. Service level token treated as required (FIXED)
**Dashboard:** Service level token is optional (`carrierOption?.serviceLevelToken ?? undefined`).
**Go (before):** Returned validation error if service level token was nil/empty.
**Fix:** Made service level token optional — defaults to empty string if not configured.

## Remaining concerns
- `FetchShippingRate` on the Shippo client is still unimplemented (returns placeholder error). This is a known infrastructure gap, not specific to this endpoint.
- The Dashboard's `billing` parameter (carrier billing type/account/country/zip) is not passed through to the Shippo client. This appears to be a feature gap in the Go Shippo client interface (`FetchShippingRateParams` doesn't include billing fields).
