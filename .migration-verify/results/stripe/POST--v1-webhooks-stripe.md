# POST /v1/webhooks/stripe — Migration Verification

## Status: Issues found and fixed

## Endpoint Mapping

The Dashboard endpoint `POST /webhooks/stripe/:accountID` maps to **two** Go endpoints:
- `POST /v1/webhooks/stripe` — global billing webhook (billing-service), handles subscription/billing events
- `POST /v1/webhooks/stripe/{accountID}` — per-account webhook (core-service), handles payment intent events

The per-account endpoint is the one that corresponds to the Dashboard's `StripeSvc.handleEvent` business logic.

## What was compared

| Area | Dashboard | Go | Parity |
|------|-----------|----|----|
| Route & method | POST /webhooks/stripe/:accountID | POST /v1/webhooks/stripe/{accountID} | Yes |
| Auth | Unauthenticated | Unauthenticated (SkipRequestBodyParsing) | Yes |
| Signature verification | constructEventAsync via Stripe SDK | ConstructEvent via stripe-go SDK | Yes |
| Credential fetching | buildStripeClient → decrypt AES | GetEncryptedCredentials → DecryptAESGCM | Yes |
| Event deduplication | stripe_event_log (eventID, eventType, objectID) | stripe_event_log (event_id, object_id) | Yes (event_id is unique) |
| Handled event types | payment_intent.succeeded/failed/canceled | Same three types | Yes |
| Response | `{ received: true }` always (catches errors) | Always returns nil (200) | Yes |
| payment_intent.succeeded — order verification | Checks order exists, buyer matches | IsOrderForCustomer | Yes |
| payment_intent.succeeded — amount storage | `amount / 100` (dollars), unit = dollar | **Fixed**: was storing raw cents with no unit | Fixed |
| payment_intent.succeeded — payment method | `includes()` scanning all types | **Fixed**: was only checking index [0] | Fixed |
| payment_intent.succeeded — dollar unit | Uses `Units.dollar` | **Fixed**: now calls `GetDollarUnitID` | Fixed |
| payment_intent.succeeded — transaction creation | Creates quantity + transaction in tx | Same pattern | Yes |
| payment_intent.payment_failed — logic | Update note, delete allocations, delete OPI | Same | Yes |
| payment_intent.canceled — logic | Delete transaction, delete allocations, delete OPI | Delete allocations, delete transaction, delete quantity, delete OPI | Yes (correct order in Go) |
| Error handling | Logs to errorLog table | Logs via slog | Acceptable (different logging infra) |

## Issues found and fixed

### 1. Amount stored in cents instead of dollars (CRITICAL)
**File:** `services/core-service/internal/service/stripe_webhook_service.go`

The Go code was storing `pi.Amount` (Stripe's raw cents value) directly. The Dashboard divides by 100 to convert to dollars before storage.

**Fix:** Added `float64(pi.Amount) / 100.0` conversion with `fmt.Sprintf("%g", ...)` formatting.

### 2. Missing dollar unit ID (CRITICAL)
**File:** `services/core-service/internal/service/stripe_webhook_service.go`

The Go code was passing an empty string `""` for `amountUnitID` when creating the transaction quantity record. The Dashboard uses `Units.dollar`.

**Fix:** Added call to `txRepo.GetDollarUnitID(txCtx)` (matching pattern used in `transaction_service.go`) and pass the result.

### 3. Payment method mapping only checked first type (MINOR)
**File:** `services/core-service/internal/service/stripe_webhook_service.go`

The Go code only checked `PaymentMethodTypes[0]`, while the Dashboard scans the entire array with `includes()`.

**Fix:** Extracted `mapPaymentMethodFromTypes` helper that iterates all types and returns the first match.

## Minor behavioral differences (acceptable, not fixed)

1. **Order-customer verification order**: In Dashboard, `addPaymentIntentToOrder` is called before customer verification — so if customerID is missing, the payment intent is still linked. In Go, both checks happen upfront. This is unlikely to cause issues since both metadata fields are always set during checkout session creation.

2. **Error logging**: Dashboard logs to an `errorLog` table; Go uses structured logging (slog). This is an infrastructure difference, not a business logic gap.

3. **Dedup query**: Dashboard checks (eventID, eventType, objectID); Go checks (event_id, object_id). Since event_id is globally unique in Stripe, the Go approach is sufficient.
