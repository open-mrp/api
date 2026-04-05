# Migration Verification: POST /v1/core/sales-orders/{id}/checkout

**Status: Issues found and fixed**

## What Was Compared

- Validation rules (required fields, email format)
- Permission checks (internal actor, sales_orders update permission)
- Payment status check logic
- Stripe checkout session creation parameters
- Email notification side effects
- Response shape
- Idempotency handling

## Issues Found and Fixed

### 1. CRITICAL: Missing payment intent metadata

**Dashboard:** Passes `payment_intent_data.metadata` with `orderID` and `customerID` to Stripe checkout session.
**Go (before fix):** Did not pass any payment intent metadata.
**Impact:** The Go webhook handler (`handlePaymentIntentSucceeded`) reads `pi.Metadata["orderID"]` and `pi.Metadata["customerID"]` to verify and record payments. Without this metadata, payments made through admin checkout would silently fail to be recorded.

**Fix:** Added `PaymentIntentMetadata` field to `CreateCheckoutSessionParams` domain model. Updated the Stripe checkout client to include `PaymentIntentData` with metadata when provided. Updated the service to pass `order.ID` and `order.BuyerAccountID` as metadata.

Files changed:
- `services/core-service/internal/domain/clients.go`
- `services/core-service/internal/infrastructure/stripe/checkout_client.go`
- `services/core-service/internal/service/sales_order_service.go`

### 2. CRITICAL: Missing saved_payment_method_options

**Dashboard:** Includes `saved_payment_method_options: { payment_method_save: 'enabled' }` in the Stripe checkout session.
**Go (before fix):** Did not include this option.
**Impact:** Returning customers cannot save their payment methods for future use.

**Fix:** Added `SavedPaymentMethodOptions` to the Stripe `CheckoutSessionParams` in `CreateOneTimeCheckoutSession`.

Files changed:
- `services/core-service/internal/infrastructure/stripe/checkout_client.go`

### 3. Payment status check incomplete

**Dashboard:** Considers an order "paid" if `paymentIntents.length > 0 || (invoices.every(isPaidInFull) && statusCode === 'fulfilled')`.
**Go (before fix):** Only checked `EXISTS(SELECT 1 FROM order_payment_intent WHERE sales_order_id = ?)`.
**Impact:** Orders that were fully paid via invoices and marked fulfilled could still have checkout sessions created for them.

**Fix:** Updated the `CheckSalesOrderPaymentStatus` SQL query to also check if the order is fulfilled with all invoices paid. Updated the repository method to handle the new `sql.NullBool` return type.

Files changed:
- `services/core-service/internal/infrastructure/queries/sales_order.sql`
- `services/core-service/internal/infrastructure/sqlc/sales_order.sql.go` (regenerated)
- `services/core-service/internal/infrastructure/repository/sales_order_repo.go`

## Noted Differences (Intentional / Acceptable)

### Line items approach
- **Dashboard:** Single line item with total order amount (`SO #${number}`)
- **Go:** Individual line items per order line with per-product SKU and pricing
- **Assessment:** The Go approach provides more detail on the Stripe checkout page. Both calculate the same total. Acceptable difference.

### Response shape
- **Dashboard:** Returns empty `{}`
- **Go:** Returns `{ "checkout_url": "..." }`
- **Assessment:** Intentional improvement — the caller gets the checkout URL directly.

### Email handling
- **Dashboard:** Sends rich inline HTML email via AWS SES with user/account details
- **Go:** Publishes email via notification service using a template
- **Assessment:** Architectural difference (notification microservice vs direct SES). The notification service approach is the Go convention.

### Customer Stripe ID vs email
- **Dashboard:** Looks up/creates Stripe customer record and uses `stripeCustomerID` in checkout session
- **Go:** Uses `customerEmail` parameter directly
- **Assessment:** The Go has a separate `CreateCustomerCheckoutSession` endpoint for customer-initiated checkout that handles Stripe customer creation. The admin checkout using email is a valid simplification since the checkout link is being emailed to the customer anyway.

### Success URL
- **Dashboard:** Hardcodes success URL from `FRONTEND_URL` + account slug
- **Go:** Accepts `success_url` and `cancel_url` as request parameters
- **Assessment:** More flexible approach in Go. Caller controls the redirect URLs.
