# Verification: POST /v1/core/checkout-sessions

## Status: Issues Found and Fixed

The API gateway endpoint and proto definition existed, but the **entire core-service backend implementation was missing**. The gRPC handler fell through to `UnimplementedCoreSalesServiceServer`, returning an "unimplemented" error for every call.

## What Was Compared

| Aspect | Dashboard (Express.js) | Go API (Before) | Go API (After) |
|--------|----------------------|-----------------|----------------|
| Permission checks | Customer actor only (`checkIsAssignedActor` + `isCustomerActor`) | Not implemented | `CheckIsAssignedActor` + `IsCustomerUser` |
| Stripe integration check | `hasIntegration(targetAccountID, 'stripe')` | Not implemented | `HasIntegration` + `GetEncryptedCredentials` |
| Stripe customer resolution | Check `isStripeCustomer`, create via Stripe API if needed, save to DB | Not implemented | `GetStripeCustomerID`, `CreateStripeCustomer`, `SetStripeCustomerID` |
| Account slug for return URL | `accountRepo.findSlug(targetAccountID)` | Not implemented | `accountRepo.GetPortalSlug(targetAccountID)` |
| Checkout session type | Embedded (`ui_mode: 'custom'`) | Not implemented | Embedded (`ui_mode: custom`) |
| Stripe session config | `customer`, `saved_payment_method_options`, `payment_intent_data` with metadata | Not implemented | Same config with `Customer`, `SavedPaymentMethodOptions`, `PaymentIntentData` |
| Line items | Single item: `SO #{orderNumber}`, optional `PO #{customerPO}`, amount in cents | Not implemented | Same structure |
| Payment method types | `['card']` | Not implemented | `['card']` |
| Return URL | `${FRONTEND_URL}/${accountSlug}/dashboard/sales-orders/${orderID}` | Not implemented | Same pattern |
| Response | `{ checkoutSessionClientSecret }` | Not implemented | `{ object, checkout_session_client_secret }` |
| Idempotency | Not idempotent in Dashboard | N/A | Uses idempotency keys (POST endpoint pattern) |
| Error handling | Various HTTP errors | Not implemented | Matching error conditions |

## Issues Found and Fixed

### 1. Missing gRPC Handler (Critical)
The `CreateCustomerCheckoutSession` RPC was defined in `proto/core_sales.proto` and wired in the API gateway, but no handler existed in `grpc_sales_service_handler.go`. Added the handler method.

### 2. Missing Service Method (Critical)
`SalesOrderSvc` had no `CreateCustomerCheckoutSession` method. Implemented the full flow:
- Customer actor validation
- Stripe integration check and credential decryption
- Stripe customer resolution/creation
- Account slug lookup for return URL
- Embedded checkout session creation
- Idempotency key support

### 3. Missing Stripe Client Methods (Critical)
`StripeCheckoutClient` interface only had `CreateOneTimeCheckoutSession` (redirect-based) and `ConstructWebhookEvent`. Added:
- `CreateEmbeddedCheckoutSession` - creates `ui_mode: 'custom'` sessions with `saved_payment_method_options`, `payment_intent_data`, and `Customer`
- `CreateStripeCustomer` - creates Stripe customer with email, name, number, and metadata

### 4. Missing Repository Methods
`CustomerRepo` had no methods for Stripe customer ID management. Added:
- `GetStripeCustomerID` - reads `stripe_customer_id` and `stripe_email` from `account_relation`
- `SetStripeCustomerID` - updates `stripe_customer_id` and `stripe_email` on `account_relation`
- `GetCustomerEmail` - reads customer email from `account_branding`

### 5. Missing SQL Queries
Added three new queries to `customer.sql`:
- `GetCustomerStripeCustomerID`
- `SetCustomerStripeCustomerID`
- `GetCustomerEmail`

### 6. Missing Config
`core-service` had no `FRONTEND_URL` config. Added to `config.go` and wired through `run.go` to the sales order service.

## Files Modified

- `services/core-service/internal/domain/clients.go` - Added interface methods
- `services/core-service/internal/domain/repositories.go` - Added CustomerRepo methods
- `services/core-service/internal/domain/services.go` - Added SalesOrderSvc method
- `services/core-service/internal/infrastructure/queries/customer.sql` - Added SQL queries
- `services/core-service/internal/infrastructure/sqlc/customer.sql.go` - Added generated query code
- `services/core-service/internal/infrastructure/sqlc/db.go` - Added prepared statements
- `services/core-service/internal/infrastructure/repository/customer_repository.go` - Implemented repo methods
- `services/core-service/internal/infrastructure/stripe/checkout_client.go` - Implemented Stripe client methods
- `services/core-service/internal/service/sales_order_service.go` - Implemented service method + added FrontendURL
- `services/core-service/internal/infrastructure/grpc/grpc_sales_service_handler.go` - Added gRPC handler
- `services/core-service/cmd/config.go` - Added FrontendURL config
- `services/core-service/cmd/run.go` - Wired FrontendURL

## Remaining Concerns

1. **Order total format**: Dashboard sends `orderTotal` as dollars (float) and converts to cents via `Math.round(orderTotal * 100)`. The Go API gateway accepts `order_total_cents` (int64) directly. The frontend must send cents, not dollars. This is intentional — the Go API avoids floating-point currency arithmetic.

2. **`formatRecordNumber`**: The Dashboard formats order numbers with `formatRecordNumber()` (pads with leading zeros). The Go implementation uses the raw `orderNumber` in the line item name. This matches what the API gateway receives — formatting is a frontend concern.

3. **Idempotency improvement**: The Dashboard endpoint is NOT idempotent. The Go implementation adds idempotency key support, which is an improvement over the Dashboard behavior, consistent with the project's POST endpoint pattern.

4. **Pre-existing compilation errors**: `sales_order_repo.go` and `shipment_repository.go` have pre-existing errors unrelated to this endpoint (missing `NoteFirstShipAt` method and wrong `MarkShipped`/`MarkVoided` signatures).
