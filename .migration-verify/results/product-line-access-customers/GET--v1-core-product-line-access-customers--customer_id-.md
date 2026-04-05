# Verification: GET /v1/core/product-line-access/customers/{customer_id}

## Status: Issues found and fixed

## What was compared

- **Validation**: Path parameter `customerID` required — matches both implementations
- **Permission checks**: Internal actor + `productLineAccess` read permission + target account required — matches
- **DB queries**: Both look up the account relation (owner_account_id + counterparty_account_id + role='customer'), then fetch product lines from `account_relation_product_line` joined to `product_line` — matches
- **Error handling**: 404 if customer relationship not found — matches
- **Response shape**: Go uses `customer` sub-resource with `object` field, `product_lines` list, `created_at`/`updated_at` — follows Go API conventions correctly
- **Side effects**: None — matches (GET endpoint)
- **Idempotency**: Not applicable (GET) — correct

## Issues found and fixed

### 1. Missing 404 when customer has no product line access records

**Dashboard behavior**: The `find` method queries `accountRelationProductLine` rows directly. If zero rows are found for the customer, it returns `null`, and the service throws `HttpError.notFound('Customer product line access not found.')`.

**Go behavior (before fix)**: The `Get` method first verifies the customer relation exists (404 if not), then queries product lines. If the relation exists but has zero product line rows, Go returned a 200 with an empty `product_lines` list.

**Fix**: Added a check in `customer_product_line_access_repository.go` `Get()` method — if `plRows` is empty after querying, return a `ResourceNotFoundError("Customer product line access not found.")`. This matches the Dashboard's behavior of returning 404 when no product line access records exist for a customer.

## No remaining concerns

The endpoint is now at parity with the Dashboard implementation.
