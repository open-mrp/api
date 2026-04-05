# POST /v1/core/sales-orders — Migration Verification

## Result: Issues found and partially fixed

## What was compared

- Permission checks and actor type handling
- Request fields and validation
- DB operations (address creation, order insert, line creation, order number generation)
- Duplicate checking (order number, customer PO)
- Side effects (shipping lines, discount lines, email contacts, change log, plan limits)
- Response shape and includes
- Idempotency handling

## Bug fixed

### SellerAccountID and OwnerAccountID not set (CRITICAL)

**File:** `services/core-service/internal/service/sales_order_service.go` (line ~327-328)

The service was passing `params.SellerAccountID` and `params.OwnerAccountID` from the input params to the create operation, but neither the gRPC handler nor the service ever set these fields. They would be empty strings, causing the SQL insert to write empty values for `seller_account_id` and `owner_account_id`.

**Fix:** Set both to `params.AccountID` (the target account), matching Dashboard behavior where `ownerAccountID = this.identity.targetAccountID` and seller is the same account.

## Remaining discrepancies (not fixed — require design decisions)

### 1. Customer actor access not supported

**Dashboard:** Allows customer actors to create orders for their own customer account (with a check that `data.customer.id === identity.actor.accountID`).

**Go:** Only allows internal actors (`CheckIsInternalActor`). Customer actors are rejected.

**Impact:** Customer-facing order creation won't work through the Go API.

### 2. No shipping line auto-creation

**Dashboard:** Automatically creates a shipping line for every order by:
- Looking up the org's bill-to address
- Calculating shipping rate via Shippo or applying freight exemptions/flat rates/minimum order value rules
- Adding a shipping product line with the calculated rate

**Go:** Does not create shipping lines. Only user-provided lines are inserted.

**Impact:** Orders created via Go API will be missing the shipping line that the Dashboard automatically adds.

### 3. No order discount line auto-creation

**Dashboard:** When `orderDiscountID` is provided, fetches the discount, calculates the discount total from the ordered amount, and adds a credit product line with a negative unit price.

**Go:** Stores `order_discount_id` on the order record but does not create a discount line.

**Impact:** Discount amounts won't be reflected in order line totals.

### 4. No email contact creation

**Dashboard:** Creates `orderEmailContact` records for acknowledgement and invoice email notifications.

**Go:** Does not accept or create email contact associations.

**Impact:** Email notifications for order acknowledgement and invoicing won't be configured.

### 5. No change log entry

**Dashboard:** Creates an audit change log entry (`Created order {number}`).

**Go:** Does not create a change log entry.

**Impact:** Order creation won't appear in audit logs.

### 6. No plan/invoice limit check

**Dashboard:** Checks `AccountPlanSvc.canCreateInvoice()` to enforce subscription plan limits before allowing order creation.

**Go:** Does not check plan limits.

**Impact:** Orders could be created beyond subscription limits.

### 7. No sales rep auto-resolution

**Dashboard:** Auto-resolves the sales rep from territory mapping (based on customer, shipping state/zip).

**Go:** Accepts `sales_rep_id` as an optional input but does not auto-resolve it.

**Impact:** If the frontend doesn't send `sales_rep_id`, the order won't have a sales rep assigned. This may be acceptable if the frontend handles the lookup.

## Notes

- The Go implementation correctly handles idempotency with recovery points
- Address creation approach differs (Go creates new addresses; Dashboard upserts and links existing ones) but this aligns with Go API patterns
- The Go API accepts `sales_order_type_code` and `sales_order_status_code` which gives more flexibility than the Dashboard
- Order number generation and duplicate checking logic is correctly implemented
