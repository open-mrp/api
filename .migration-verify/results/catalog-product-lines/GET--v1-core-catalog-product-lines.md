# GET /v1/core/catalog/product-lines

## Result: Issues found and fixed

## What was compared

- **Validation rules**: Both have empty request (no parameters) - matches
- **Permission checks**: Both use `CheckIsAssignedActor()` with `PermissionDomainProducts`/`ActionRead` for internal users - matches
- **DB queries**: Both filter by `account_id`, `is_portal_ready = 1`, `deleted_at IS NULL`, `DISTINCT`, `ORDER BY name` - matches
- **Error handling**: Standard patterns used in both - matches
- **Response shape**: Dashboard returns `{ id, name }[]`; Go returns `{ object: "list", data: [{ id, object, name }] }` - appropriate Go API convention difference
- **Side effects**: None in either implementation - matches
- **Idempotency**: GET endpoint, not applicable - matches

## Issues found and fixed

### 1. Missing customer actor product line filtering (critical)

**Problem**: The Dashboard implementation filters product lines for customer actors through 3 access pathways:
1. Account group membership (`account_group_product_line` -> `account_group` <- `account_relation`)
2. Direct account relation product lines (`account_relation_product_line` -> `account_relation`)
3. Price group assignment (`account_group_product_line` -> `account_group` <- `account_relation_price_group` -> `account_relation`)

The Go implementation returned ALL product lines for the account regardless of actor type, which would expose product lines that customer actors should not have access to.

**Fix**:
- Added `ListCatalogProductLinesForCustomer` SQL query with all 3 customer access filter pathways
- Added `ListProductLinesForCustomer` method to `CatalogRepo` interface and repository implementation
- Updated `catalog_service.go` to detect customer actors via `identity.IsCustomerUser()` and route to the filtered query using `identity.Actor.AccountID` as the counterparty

**Files modified**:
- `services/core-service/internal/infrastructure/queries/catalog.sql`
- `services/core-service/internal/domain/repositories.go`
- `services/core-service/internal/infrastructure/repository/catalog_repository.go`
- `services/core-service/internal/service/catalog_service.go`

## Remaining concerns

None. The endpoint now matches Dashboard behavior for both internal and customer actors.
