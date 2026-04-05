# POST /v1/core/customers/registration

**Status: Issues found and fixed**

## What was compared

- Request fields and validation rules
- Authentication/authorization checks
- Database operations (both existing and new customer paths)
- SQL queries (inserts, lookups, sys_property counter)
- Default values for account relation fields
- Customer name trimming behavior
- Error handling
- Response shape (empty 201)
- Idempotency key support
- Transaction boundaries

## Issues found and fixed

### 1. `defaultCarrierOptionID` set to NULL instead of `'ground'`

**File:** `services/core-service/internal/infrastructure/repository/customer_registration_repository.go:197`

The Dashboard sets `defaultCarrierOptionID: CarrierOptions.ground` (which is the string `'ground'`) when creating the account relation for a new customer. The Go code was setting it to `gosql.NullString{}` (NULL).

**Fix:** Changed to `gosql.NullString{String: "ground", Valid: true}`.

### 2. Customer name not trimmed

**File:** `services/core-service/internal/service/registration_flow_service.go:476`

The Dashboard trims the customer name with `.trim()` before using it for both the account name and the account relation alias. The Go code was passing the name directly without trimming.

**Fix:** Added `strings.TrimSpace(*data.Name)` when building the `CreateNewCustomerAccountParams`.

## Confirmed parity

- **Auth checks:** Dashboard uses `checkIsValidActor` (authenticated + valid actor type). Go uses `CheckIsAuthenticated()` + actor nil check. Equivalent for this endpoint's purpose (allows unassigned users).
- **Existing customer path:** Both trim the customer number, look up by owner_account_id + external_number with role='customer', and create an account_user link.
- **New customer path:** Both validate same required fields (name, address, shipping term, payment term, customer group), fetch user email, generate next customer number via sys_property, and create the same set of records (geolocation, address, account, branding, account_relation, account_address, account_user).
- **Account relation defaults:** commission_policy='applied', freight_policy='billed', account_status='normal', priority='normal', default_carrier_id=NULL all match.
- **Response:** Both return empty body with 201 status.
- **Idempotency:** Go implementation uses idempotency keys with recovery points (not present in Dashboard but required by Go conventions).
- **Transaction:** Both wrap all DB operations in a single transaction.
