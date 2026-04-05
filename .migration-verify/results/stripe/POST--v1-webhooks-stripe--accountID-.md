# Verification: `POST /v1/webhooks/stripe/{accountID}`

**Status: Issues found and fixed**

## What was compared

- **Webhook signature verification flow** — Both use per-account Stripe credentials (fetched from `account_integration`, decrypted, used to construct/verify event)
- **Event deduplication** — Both check `stripe_event_log` by event ID + object ID before processing, and insert a log entry on first occurrence
- **Allowed event types** — Both handle: `payment_intent.succeeded`, `payment_intent.payment_failed`, `payment_intent.canceled`
- **`payment_intent.succeeded` handler** — Order/customer validation, order_payment_intent creation, transaction number generation, payment method mapping, transaction creation
- **`payment_intent.payment_failed` handler** — Transaction lookup by stripe payment ID, note update, allocation deletion, order_payment_intent deletion
- **`payment_intent.canceled` handler** — Transaction lookup, allocation deletion, transaction deletion, quantity deletion, order_payment_intent deletion
- **Error handling** — Both silently log errors and return success to Stripe (never propagate errors)
- **Response shape** — Both return `{ received: true }` equivalent

## Issues found and fixed

### 1. Amount not converted from cents to dollars (CRITICAL)

**Dashboard**: `paymentIntent.amount / 100` — converts Stripe's cent amount to dollars before storing.
**Go (before fix)**: `fmt.Sprintf("%d", pi.Amount)` — stored raw cents, resulting in amounts 100x too large.
**Fix**: Changed to `float64(pi.Amount) / 100.0` with `fmt.Sprintf("%g", amountDollars)`.

### 2. Missing dollar unit ID for quantity record (CRITICAL)

**Dashboard**: Creates quantity with `unit: PrismaUtils.connect(Units.dollar)` — links to the dollar unit record.
**Go (before fix)**: Passed empty string `""` for `amountUnitID`, leaving the quantity without a unit reference.
**Fix**: Added call to `txRepo.GetDollarUnitID(txCtx)` to look up the dollar unit ID and pass it to `Create`.

### 3. Payment method mapping only checked first element

**Dashboard**: Uses `methods.includes('card')` etc. — checks ALL payment method types in the array.
**Go (before fix)**: Only checked `pi.PaymentMethodTypes[0]` — the first element.
**Fix**: Replaced with `mapPaymentMethodFromTypes()` helper that iterates all types and returns the first match, matching Dashboard's `includes` behavior.

## Acceptable differences (not bugs)

1. **Dedup query**: Dashboard checks `eventID + eventType + objectID`; Go checks `eventID + objectID`. The `eventType` check is redundant since Stripe event IDs are globally unique.
2. **Account ID source**: Dashboard uses `order.sellerAccountID` for the transaction's account; Go uses the URL param `accountID`. These are equivalent since the webhook URL is per-seller-account.
3. **Transaction creation scope**: Dashboard creates the transaction record outside the Prisma `$transaction` block (uses `prisma` not `tx`). Go correctly creates everything within `withTx`. This is an improvement.
4. **Canceled handler order**: Dashboard deletes transaction before allocations; Go deletes allocations first then transaction. Go's order is correct for FK constraints.

## Files modified

- `services/core-service/internal/service/stripe_webhook_service.go` — Fixed amount conversion, dollar unit ID lookup, and payment method mapping
