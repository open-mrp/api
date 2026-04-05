# PATCH /v1/core/product-line-access/customers/{customer_id}

## Result: Issues found and fixed

## What was compared
- **Validation rules**: Request field requirements (product_line_ids required)
- **Permission checks**: Internal actor + productLineAccess domain + update action
- **DB queries and logic**: AccountRelation lookup, delete-and-recreate strategy, product line validation
- **Error handling**: 404 for missing customer relationship, 400 for missing fields
- **Side effects**: None in either implementation
- **Response shape**: Customer sub-resource, product lines list, timestamps
- **Idempotency**: PATCH uses idempotency keys correctly

## Issues found and fixed

### 1. Overly strict "existing access required" check (service layer)
- **File**: `services/core-service/internal/service/customer_product_line_access_service.go`
- **Issue**: The service called `txRepo.Get()` before `txRepo.Update()`, which returns 404 if the customer has no existing product line access records. The Dashboard does NOT require existing product line access — it only requires the customer relationship (AccountRelation) to exist.
- **Fix**: Removed the `Get` call from the `UpdateCustomerProductLineAccess` service method. The repository's `Update` method already verifies the customer relationship exists via `GetAccountRelationForCustomer`.

### 2. Overly strict "existing access required" check (repository layer)
- **File**: `services/core-service/internal/infrastructure/repository/customer_product_line_access_repository.go`
- **Issue**: The repository checked `CountAccountRelationProductLinesByRelationID` and returned 404 if count was 0. The Dashboard simply deletes whatever exists (even zero rows) and creates new ones.
- **Fix**: Removed the count check from the `Update` method. The delete-then-insert approach works correctly even when no prior records exist.

## Parity confirmed for
- Permission checks match: internal actor + `productLineAccess` domain + `update` action
- Replace-all semantics match: delete existing, insert new product line IDs
- Customer relationship validation matches: both check AccountRelation exists with role='customer'
- Product line existence validation: Go additionally validates each product line ID exists and belongs to the account (stricter than Dashboard but provides better error messages; not a breaking change)
- Response shape matches: customer sub-resource with id/name/number, product lines array, timestamps
- Idempotency: Go correctly implements idempotency keys for this PATCH endpoint

## Remaining concerns
- The Go implementation validates that each product line ID exists and belongs to the account before inserting. The Dashboard does not do this (relies on DB constraints). This is strictly better behavior and not a parity concern.
