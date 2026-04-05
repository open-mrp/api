# Verification: POST /v1/core/carriers/{id}/actions/initiate-oauth

## Result: Issues found and fixed

## What was compared

- **Permission checks**: Both require internal actor + `carriers:update` permission. ✅ Match
- **Sandbox guard**: Dashboard throws `HttpError.badRequest`, Go returns `apierror.NewValidationError`. ✅ Equivalent
- **Carrier lookup**: Both fetch by ID + accountID, return not-found if missing. ✅ Match
- **Shippo account validation**: Both check `shippoCarrierAccountId` is non-null. ✅ Match
- **Shippo client construction**: Dashboard uses `AccountIntegrationRepo.buildShippoClient`, Go uses `s.getShippoClient`. ✅ Equivalent
- **OAuth initiation**: Both call Shippo's carrier account OAuth initiation endpoint. ✅ Match
- **Response shape**: Dashboard returns `{ oauthUrl }`, Go returns `{ object, oauth_url }` (added `object` per Go API conventions). ✅ Correct
- **Idempotency**: gRPC handler has `WithIdempotencyTracking` but service doesn't use idempotency keys — appropriate since this endpoint doesn't modify our database. Dashboard doesn't use idempotency either. ✅ Acceptable
- **Error handling**: Error types and messages are equivalent across both implementations. ✅ Match

## Issues found and fixed

### 1. Missing URL encoding in Shippo OAuth URL construction

**File:** `services/core-service/internal/infrastructure/shippo/client.go`

The `redirectURI` and `state` query parameters were being concatenated directly into the URL without URL encoding. Since `redirectURI` is itself a URL (containing `://`, `?`, `=`, `&`, etc.), this would produce a malformed Shippo API request URL.

**Fix:** Added `url.QueryEscape()` for both `redirectURI` and `state` parameters, and added `"net/url"` import.

## Remaining concerns

None. Business logic parity is confirmed.
