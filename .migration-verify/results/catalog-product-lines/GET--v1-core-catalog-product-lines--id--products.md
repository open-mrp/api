# Verification: GET /v1/core/catalog/product-lines/{id}/products

## Status: Issues found and fixed

## What was compared

- **Validation rules**: Path parameter (productLineID) — matches
- **Permission checks**: `CheckIsAssignedActor()` + internal user permission check — Go adds `PermissionDomainProducts/ActionRead` for internal users (improvement over Dashboard which only checks assigned actor)
- **DB queries and logic**: Filters, joins, ordering, pagination
- **Error handling**: Standard error propagation — matches
- **Side effects**: None (read-only GET) — matches
- **Response shape**: Categories with products grouped by item category
- **Idempotency**: GET endpoint, idempotent by design — correct

## Issues found and fixed

### 1. Missing customer access filtering (CRITICAL)
**Dashboard**: When the actor is a customer, products are filtered by 3-pathway customer access check (account group relations, direct product line relations, and price group relations).
**Go (before)**: No customer filtering at all — all products were returned regardless of actor type.
**Fix**: Added `ListCatalogProductsForCustomer` SQL query with the same 3-pathway EXISTS subqueries used in the existing `ListCatalogProductLinesForCustomer`. Updated the service to route customer actors to this query (matching the pattern already used for `ListCatalogProductLines`). Added `ListProductsForCustomer` to the `CatalogRepo` interface.

### 2. Missing category account filter (MODERATE)
**Dashboard**: Filters categories with `OR: [{ accountID }, { accountID: null }]` — only includes categories owned by the account or public (null account) categories.
**Go (before)**: No category filter — all categories were included.
**Fix**: Added `AND (ic.account_id = ? OR ic.account_id IS NULL)` to both `ListCatalogProducts` and `ListCatalogProductsForCustomer` SQL queries.

### 3. Missing properties on categories (MODERATE)
**Dashboard**: Each category includes `properties: [{id, name}]` from the `_item_categories_properties` join table.
**Go (before)**: No properties returned.
**Fix**: Added `ListCatalogCategoryProperties` SQL query. Updated domain models (`CatalogCategory.Properties`, new `CatalogProperty` type). Updated repository to fetch and attach properties. Updated proto (`CatalogPropertyProto`), gRPC handler, API gateway service, and API resource.

### 4. Missing attributes on products (MODERATE)
**Dashboard**: Each product includes `attributes: [{id, name, propertyID, propertyName}]` from the `_item_attributes` join table with property name lookup.
**Go (before)**: No attributes returned.
**Fix**: Added `ListCatalogProductAttributes` SQL query. Updated domain models (`CatalogProduct.Attributes`, new `CatalogAttribute` type). Updated repository to fetch and attach attributes. Updated proto (`CatalogAttributeProto`), gRPC handler, API gateway service, and API resource.

### 5. Response shape differences (BY DESIGN)
**Dashboard**: Returns a bare array of categories.
**Go**: Wraps in `{object: "list", data: [...]}` with `object` fields on each nested type.
**Decision**: This is intentional per Go API resource conventions. Not a bug.

### 6. Permission check difference (INTENTIONAL IMPROVEMENT)
**Dashboard**: Only checks `checkIsAssignedActor()`.
**Go**: Additionally checks `PermissionDomainProducts/ActionRead` for internal users.
**Decision**: This is a stricter permission model and is consistent with other Go catalog endpoints. Kept as-is.

## Files modified

- `services/core-service/internal/infrastructure/queries/catalog.sql` — Added 3 new queries, updated existing query with category filter
- `services/core-service/internal/infrastructure/repository/catalog_repository.go` — Added `ListProductsForCustomer`, `buildCatalogCategories` helper with properties/attributes loading
- `services/core-service/internal/service/catalog_service.go` — Added customer actor routing
- `services/core-service/internal/domain/catalog_models.go` — Added `CatalogProperty`, `CatalogAttribute` types and fields
- `services/core-service/internal/domain/repositories.go` — Added `ListProductsForCustomer` to `CatalogRepo` interface
- `proto/core.proto` — Added `CatalogPropertyProto`, `CatalogAttributeProto` messages
- `services/core-service/internal/infrastructure/grpc/grpc_catalog_handler.go` — Updated `catalogCategoryToProto` to map properties and attributes
- `services/api-gateway/endpoints/catalog/service.go` — Updated response mapping for properties and attributes
- `services/api-gateway/pkg/resource/catalog_resource.go` — Added `CatalogProperty`, `CatalogAttribute` resource types
- `shared/constants/object_type.go` — Added `ObjectTypeCatalogProperty`, `ObjectTypeCatalogAttribute`

## Remaining concerns

- sqlc generated code and proto generated code were regenerated via `make sqlc core` and `make proto`
- Mocks need regeneration via `make mocks core`
- Pre-existing build errors in core-service (unrelated to catalog: customer_repository, shipment_repository, sales_order_repo, stripe checkout) are not addressed
