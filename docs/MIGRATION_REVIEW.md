# Migration Review Checklist

This file tracks the review of every endpoint migrated from the dashboard Express.js API to the Go API. Each endpoint must be independently verified for business logic parity before being removed from this list.

## How to Review an Endpoint

1. **Pick** the next unchecked endpoint from the list below.
2. **Read the dashboard code** for that endpoint:
   - Service logic (business rules, validation, side effects)
   - Repository code (DB queries, data transformations)
   - Controller code (request parsing, response shaping)
3. **Read the Go code** for the corresponding endpoint:
   - API gateway endpoint definition (request/response types)
   - Domain service logic (business rules, validation, side effects)
   - Repository/query code (sqlc queries)
4. **Compare** the following aspects:
   - Validation rules (required fields, formats, constraints)
   - Permission checks (actor type, permission domain, action)
   - DB queries and logic (filters, joins, ordering, pagination)
   - Error handling (error types, messages, field-level errors)
   - Side effects (emails, webhooks, messages, inventory changes)
   - Response shape (field names, types, nested resources, expandables)
   - Idempotency (POST/PATCH endpoints must use idempotency keys)
5. **Fix** any discrepancies found in the Go implementation.
6. **Remove** the endpoint line from this file once verified.

## Key File Locations

| Layer | Dashboard | Go |
|-------|-----------|-----|
| Controllers | `dashboard/apps/api/src/controllers/` | `api/services/api-gateway/endpoints/` |
| Services | `dashboard/apps/api/src/services/` | `api/services/core/domain/` |
| Repositories | `dashboard/apps/api/src/repositories/` | `api/services/core/domain/*/repo_*.go` |
| Routes | `dashboard/apps/api/src/index.ts` | Endpoint `Route` field in each `endpoint_*.go` |

## Checklist

Format: `METHOD /v1/core/route` — Go: `endpoint_file.go` | Dashboard: `ServiceName.methodName`

---

### Accounts

- [ ] `GET /v1/core/accounts/{id}` — Go: `accounts/endpoint_get_account.go` | Dashboard: `AccountSvc.find`
- [ ] `GET /v1/core/accounts/slug/{slug}` — Go: `accounts/endpoint_get_account_by_slug.go` | Dashboard: `AccountSvc.findBySlug`
- [ ] `GET /v1/core/accounts/{id}/logo` — Go: `accounts/endpoint_get_account_logo_url.go` | Dashboard: `AccountSvc.getLogoUrl`
- [ ] `PATCH /v1/core/accounts/{id}` — Go: `accounts/endpoint_update_account.go` | Dashboard: `AccountSvc.update`
- [ ] `PUT /v1/core/accounts/{id}/photo` — Go: `accounts/endpoint_upload_account_photo.go` | Dashboard: `AccountSvc.updatePhoto`

### Account Users

- [ ] `GET /v1/core/account-users` — Go: `account-users/endpoint_list_account_users.go` | Dashboard: `AccountUserSvc.list`
- [ ] `GET /v1/core/account-users/{id}` — Go: `account-users/endpoint_get_account_user.go` | Dashboard: `AccountUserSvc.find`
- [ ] `POST /v1/core/account-users` — Go: `account-users/endpoint_create_account_user.go` | Dashboard: `AccountUserSvc.create`
- [ ] `PATCH /v1/core/account-users/{id}` — Go: `account-users/endpoint_update_account_user.go` | Dashboard: `AccountUserSvc.update`
- [ ] `DELETE /v1/core/account-users/{id}` — Go: `account-users/endpoint_delete_account_user.go` | Dashboard: `AccountUserSvc.delete`
- [ ] `PUT /v1/core/account-users/{id}/notification-preferences` — Go: `account-users/endpoint_update_notification_preferences.go` | Dashboard: `AccountUserSvc.updateNotificationPreferences`
- [ ] `PUT /v1/core/account-users/{id}/password` — Go: `account-users/endpoint_update_account_user_password.go` | Dashboard: `AccountUserSvc.updateAccountUserPassword`
- [ ] `POST /v1/core/account-users/{id}/lock` — Go: `account-users/endpoint_lock_account_user.go` | Dashboard: `AccountUserSvc.lock`
- [ ] `POST /v1/core/account-users/{id}/unlock` — Go: `account-users/endpoint_unlock_account_user.go` | Dashboard: `AccountUserSvc.unlock`
- [ ] `POST /v1/core/account-users/{id}/restore` — Go: `account-users/endpoint_restore_account_user.go` | Dashboard: `AccountUserSvc.restore`

### Account Groups

- [ ] `GET /v1/core/account-groups` — Go: `account-groups/endpoint_list_account_groups.go` | Dashboard: `AccountGroupSvc.list`
- [ ] `GET /v1/core/account-groups/{id}` — Go: `account-groups/endpoint_get_account_group.go` | Dashboard: `AccountGroupSvc.find`
- [ ] `POST /v1/core/account-groups` — Go: `account-groups/endpoint_create_account_group.go` | Dashboard: `AccountGroupSvc.create`
- [ ] `PATCH /v1/core/account-groups/{id}` — Go: `account-groups/endpoint_update_account_group.go` | Dashboard: `AccountGroupSvc.update`
- [ ] `DELETE /v1/core/account-groups/{id}` — Go: `account-groups/endpoint_delete_account_group.go` | Dashboard: `AccountGroupSvc.delete`


### Account Prices

- [ ] `GET /v1/core/account-prices` — Go: `account-prices/endpoint_list_account_prices.go` | Dashboard: `AccountPriceSvc.list`
- [ ] `GET /v1/core/account-prices/{id}` — Go: `account-prices/endpoint_get_account_price.go` | Dashboard: `AccountPriceSvc.find`
- [ ] `POST /v1/core/account-prices` — Go: `account-prices/endpoint_create_account_price.go` | Dashboard: `AccountPriceSvc.create`
- [ ] `PATCH /v1/core/account-prices/{id}` — Go: `account-prices/endpoint_update_account_price.go` | Dashboard: `AccountPriceSvc.update`
- [ ] `DELETE /v1/core/account-prices/{id}` — Go: `account-prices/endpoint_delete_account_price.go` | Dashboard: `AccountPriceSvc.delete`

### Account Integrations

- [ ] `GET /v1/core/integrations` — Go: `account-integrations/endpoint_list_account_integrations.go` | Dashboard: `AccountIntegrationSvc.list`
- [ ] `POST /v1/core/integrations` — Go: `account-integrations/endpoint_create_account_integration.go` | Dashboard: `AccountIntegrationSvc.create`
- [ ] `PUT /v1/core/integrations/{id}` — Go: `account-integrations/endpoint_update_account_integration.go` | Dashboard: `AccountIntegrationSvc.update`
- [ ] `DELETE /v1/core/integrations/{id}` — Go: `account-integrations/endpoint_delete_account_integration.go` | Dashboard: `AccountIntegrationSvc.delete`
- [ ] `GET /v1/core/integrations/stripe/publishable-key` — Go: `account-integrations/endpoint_get_stripe_publishable_key.go` | Dashboard: `AccountIntegrationSvc.findStripePublishableKey`
- [ ] `GET /v1/core/integrations/stripe/status` — Go: `account-integrations/endpoint_get_stripe_status.go` | Dashboard: `AccountIntegrationSvc.hasStripeIntegration`

### Addresses

- [ ] `GET /v1/core/accounts/{account_id}/addresses` — Go: `addresses/endpoint_list_addresses.go` | Dashboard: `AddressSvc.list`
- [ ] `GET /v1/core/accounts/{account_id}/addresses/{id}` — Go: `addresses/endpoint_get_address.go` | Dashboard: `AddressSvc.find`
- [ ] `POST /v1/core/accounts/{account_id}/addresses` — Go: `addresses/endpoint_create_address.go` | Dashboard: `AddressSvc.create`
- [ ] `PATCH /v1/core/accounts/{account_id}/addresses/{id}` — Go: `addresses/endpoint_update_address.go` | Dashboard: `AddressSvc.update`
- [ ] `DELETE /v1/core/accounts/{account_id}/addresses/{id}` — Go: `addresses/endpoint_delete_address.go` | Dashboard: `AddressSvc.delete`

### Address Validation

- [ ] `GET /v1/core/addresses/autocomplete` — Go: `address-validation/endpoint_autocomplete_address.go` | Dashboard: (address validation integration)
- [ ] `GET /v1/core/addresses/details/{id}` — Go: `address-validation/endpoint_get_address_details.go` | Dashboard: (address validation integration)
- [ ] `POST /v1/core/addresses/validate` — Go: `address-validation/endpoint_validate_address.go` | Dashboard: (address validation integration)

### Child Accounts

- [ ] `GET /v1/core/child-accounts` — Go: `child-accounts/endpoint_list_child_accounts.go` | Dashboard: `ChildAccountSvc.list`
- [ ] `PUT /v1/core/child-accounts/{child_account_id}` — Go: `child-accounts/endpoint_add_child_account.go` | Dashboard: `ChildAccountSvc.add`
- [ ] `DELETE /v1/core/child-accounts/{child_account_id}` — Go: `child-accounts/endpoint_remove_child_account.go` | Dashboard: `ChildAccountSvc.remove`

### Territories

- [ ] `GET /v1/core/accounts/{account_id}/territories` — Go: `territories/endpoint_list_territories.go` | Dashboard: `TerritorySvc.list`
- [ ] `GET /v1/core/accounts/{account_id}/territories/{id}` — Go: `territories/endpoint_get_territory.go` | Dashboard: `TerritorySvc.find`
- [ ] `POST /v1/core/accounts/{account_id}/territories` — Go: `territories/endpoint_create_territory.go` | Dashboard: `TerritorySvc.create`
- [ ] `PATCH /v1/core/accounts/{account_id}/territories/{id}` — Go: `territories/endpoint_update_territory.go` | Dashboard: `TerritorySvc.update`
- [ ] `DELETE /v1/core/accounts/{account_id}/territories/{id}` — Go: `territories/endpoint_delete_territory.go` | Dashboard: `TerritorySvc.delete`

### Customers

- [ ] `GET /v1/core/customers` — Go: `customers/endpoint_list_customers.go` | Dashboard: `CustomerSvc.list`
- [ ] `GET /v1/core/customers/{id}` — Go: `customers/endpoint_get_customer.go` | Dashboard: `CustomerSvc.find`
- [ ] `POST /v1/core/customers` — Go: `customers/endpoint_create_customer.go` | Dashboard: `CustomerSvc.create`
- [ ] `PATCH /v1/core/customers/{id}` — Go: `customers/endpoint_update_customer.go` | Dashboard: `CustomerSvc.update`
- [ ] `DELETE /v1/core/customers/{id}` — Go: `customers/endpoint_delete_customer.go` | Dashboard: `CustomerSvc.delete`
- [ ] `POST /v1/core/customers/actions/bulk-delete` — Go: `customers/endpoint_bulk_delete_customers.go` | Dashboard: `CustomerSvc.deleteMany`
- [ ] `POST /v1/core/customers/registration` — Go: `registration-flows/endpoint_register_customer.go` | Dashboard: `CustomerSvc.register`
- [ ] `GET /v1/core/customers/{id}/frequently-ordered-products` — Go: `customers/endpoint_get_frequently_ordered_products.go` | Dashboard: `CustomerSvc.findFrequentlyOrderedProducts`
- [ ] `POST /v1/core/customers/{id}/actions/merge` — Go: `customers/endpoint_merge_customers.go` | Dashboard: `CustomerSvc.merge`

### Customer Product Line Access

- [ ] `GET /v1/core/product-line-access/customers` — Go: `customer-product-line-access/endpoint_list_customer_product_line_access.go` | Dashboard: `CustomerProductLineAccessSvc.list`
- [ ] `GET /v1/core/product-line-access/customers/{customer_id}` — Go: `customer-product-line-access/endpoint_get_customer_product_line_access.go` | Dashboard: `CustomerProductLineAccessSvc.find`
- [ ] `POST /v1/core/product-line-access/customers` — Go: `customer-product-line-access/endpoint_create_customer_product_line_access.go` | Dashboard: `CustomerProductLineAccessSvc.create`
- [ ] `PATCH /v1/core/product-line-access/customers/{customer_id}` — Go: `customer-product-line-access/endpoint_update_customer_product_line_access.go` | Dashboard: `CustomerProductLineAccessSvc.update`
- [ ] `DELETE /v1/core/product-line-access/customers/{customer_id}` — Go: `customer-product-line-access/endpoint_delete_customer_product_line_access.go` | Dashboard: `CustomerProductLineAccessSvc.delete`

### Account Group Product Line Access

- [ ] `GET /v1/core/product-line-access/account-groups` — Go: `account-group-product-line-access/endpoint_list_account_group_product_line_access.go` | Dashboard: `AccountGroupProductLineAccessSvc.list`
- [ ] `GET /v1/core/product-line-access/account-groups/{account_group_id}` — Go: `account-group-product-line-access/endpoint_get_account_group_product_line_access.go` | Dashboard: `AccountGroupProductLineAccessSvc.find`
- [ ] `POST /v1/core/product-line-access/account-groups` — Go: `account-group-product-line-access/endpoint_create_account_group_product_line_access.go` | Dashboard: `AccountGroupProductLineAccessSvc.create`
- [ ] `PATCH /v1/core/product-line-access/account-groups/{account_group_id}` — Go: `account-group-product-line-access/endpoint_update_account_group_product_line_access.go` | Dashboard: `AccountGroupProductLineAccessSvc.update`
- [ ] `DELETE /v1/core/product-line-access/account-groups/{account_group_id}` — Go: `account-group-product-line-access/endpoint_delete_account_group_product_line_access.go` | Dashboard: `AccountGroupProductLineAccessSvc.delete`

### Registration Flows

- [ ] `GET /v1/core/registration-flows` — Go: `registration-flows/endpoint_list_registration_flows.go` | Dashboard: `RegistrationFlowSvc.list`
- [ ] `GET /v1/core/registration-flows/{id}` — Go: `registration-flows/endpoint_get_registration_flow.go` | Dashboard: `RegistrationFlowSvc.find`
- [ ] `GET /v1/core/registration-flows/by-slug/{slug}` — Go: `registration-flows/endpoint_get_registration_flow_by_slug.go` | Dashboard: `AccountSvc.findRegistrationFlowBySlug`
- [ ] `POST /v1/core/registration-flows` — Go: `registration-flows/endpoint_create_registration_flow.go` | Dashboard: `RegistrationFlowSvc.create`
- [ ] `PATCH /v1/core/registration-flows/{id}` — Go: `registration-flows/endpoint_update_registration_flow.go` | Dashboard: `RegistrationFlowSvc.update`
- [ ] `DELETE /v1/core/registration-flows/{id}` — Go: `registration-flows/endpoint_delete_registration_flow.go` | Dashboard: `RegistrationFlowSvc.delete`

### Sales Orders

- [ ] `GET /v1/core/sales-orders` — Go: `sales-orders/endpoint_list_sales_orders.go` | Dashboard: `OrderSvc.list`
- [ ] `GET /v1/core/sales-orders/{id}` — Go: `sales-orders/endpoint_get_sales_order.go` | Dashboard: `OrderSvc.find`
- [ ] `POST /v1/core/sales-orders` — Go: `sales-orders/endpoint_create_sales_order.go` | Dashboard: `OrderSvc.create`
- [ ] `PATCH /v1/core/sales-orders/{id}` — Go: `sales-orders/endpoint_update_sales_order.go` | Dashboard: `OrderSvc.update`
- [ ] `DELETE /v1/core/sales-orders/{id}` — Go: `sales-orders/endpoint_delete_sales_order.go` | Dashboard: `OrderSvc.delete`
- [ ] `POST /v1/core/sales-orders/actions/bulk-delete` — Go: `sales-orders/endpoint_bulk_delete_sales_orders.go` | Dashboard: `OrderSvc.deleteMany`
- [ ] `PUT /v1/core/sales-orders/{id}/actions/change-status` — Go: `sales-orders/endpoint_change_sales_order_status.go` | Dashboard: `OrderSvc.changeStatus`
- [ ] `POST /v1/core/sales-orders/{id}/checkout` — Go: `sales-orders/endpoint_checkout_sales_order.go` | Dashboard: `OrderSvc.checkout`
- [ ] `POST /v1/core/sales-orders/{id}/actions/create-production-run` — Go: `sales-orders/endpoint_create_production_run.go` | Dashboard: `OrderSvc.createProductionRun`

### Sales Order Lines

- [ ] `POST /v1/core/sales-orders/{id}/lines` — Go: `sales-orders/endpoint_create_sales_order_line.go` | Dashboard: `OrderLineSvc.create`
- [ ] `PATCH /v1/core/sales-orders/{id}/lines/{lineId}` — Go: `sales-orders/endpoint_update_sales_order_line.go` | Dashboard: `OrderLineSvc.update`
- [ ] `DELETE /v1/core/sales-orders/{id}/lines/{lineId}` — Go: `sales-orders/endpoint_delete_sales_order_line.go` | Dashboard: `OrderLineSvc.delete`

### Sales Order Statuses

- [ ] `GET /v1/core/sales-orders/statuses` — Go: `sales-order-statuses/endpoint_list_sales_order_statuses.go` | Dashboard: `OrderStatusSvc.list`

### Purchase Orders

- [ ] `GET /v1/core/purchase-orders` — Go: `purchase-orders/endpoint_list_purchase_orders.go` | Dashboard: `PurchaseOrderSvc.list`
- [ ] `GET /v1/core/purchase-orders/{id}` — Go: `purchase-orders/endpoint_get_purchase_order.go` | Dashboard: `PurchaseOrderSvc.find`
- [ ] `POST /v1/core/purchase-orders` — Go: `purchase-orders/endpoint_create_purchase_order.go` | Dashboard: `PurchaseOrderSvc.create`
- [ ] `PATCH /v1/core/purchase-orders/{id}` — Go: `purchase-orders/endpoint_update_purchase_order.go` | Dashboard: `PurchaseOrderSvc.update`
- [ ] `DELETE /v1/core/purchase-orders/{id}` — Go: `purchase-orders/endpoint_delete_purchase_order.go` | Dashboard: `PurchaseOrderSvc.delete`
- [ ] `POST /v1/core/purchase-orders/actions/bulk-delete` — Go: `purchase-orders/endpoint_bulk_delete_purchase_orders.go` | Dashboard: `PurchaseOrderSvc.deleteMany`
- [ ] `PUT /v1/core/purchase-orders/{id}/actions/change-status` — Go: `purchase-orders/endpoint_change_purchase_order_status.go` | Dashboard: `PurchaseOrderSvc.changeStatus`
- [ ] `GET /v1/core/purchase-orders/statuses` — Go: `purchase-orders/endpoint_list_purchase_order_statuses.go` | Dashboard: (static list)

### Purchase Order Lines

- [ ] `POST /v1/core/purchase-orders/{id}/lines` — Go: `purchase-orders/endpoint_create_purchase_order_line.go` | Dashboard: `PurchaseOrderLineSvc.create`
- [ ] `PATCH /v1/core/purchase-orders/{id}/lines/{lineId}` — Go: `purchase-orders/endpoint_update_purchase_order_line.go` | Dashboard: `PurchaseOrderLineSvc.update`
- [ ] `DELETE /v1/core/purchase-orders/{id}/lines/{lineId}` — Go: `purchase-orders/endpoint_delete_purchase_order_line.go` | Dashboard: `PurchaseOrderLineSvc.delete`

### Products

- [ ] `GET /v1/core/products` — Go: `products/endpoint_list_products.go` | Dashboard: `ProductSvc.list`
- [ ] `GET /v1/core/products/{id}` — Go: `products/endpoint_get_product.go` | Dashboard: `ProductSvc.find`
- [ ] `POST /v1/core/products` — Go: `products/endpoint_create_product.go` | Dashboard: `ProductSvc.create`
- [ ] `PATCH /v1/core/products/{id}` — Go: `products/endpoint_update_product.go` | Dashboard: `ProductSvc.update`
- [ ] `DELETE /v1/core/products/{id}` — Go: `products/endpoint_delete_product.go` | Dashboard: `ProductSvc.delete`
- [ ] `PUT /v1/core/products/{id}/product-line/{product_line_id}` — Go: `products/endpoint_change_product_product_line.go` | Dashboard: `ProductSvc.changeProductLine`
- [ ] `PUT /v1/core/products/actions/validate` — Go: `products/endpoint_validate_products.go` | Dashboard: `ProductSvc.validateProducts`

### Product Lines

- [ ] `GET /v1/core/product-lines` — Go: `product-lines/endpoint_list_product_lines.go` | Dashboard: `ProductLineSvc.list`
- [ ] `GET /v1/core/product-lines/{id}` — Go: `product-lines/endpoint_get_product_line.go` | Dashboard: `ProductLineSvc.find`
- [ ] `POST /v1/core/product-lines` — Go: `product-lines/endpoint_create_product_line.go` | Dashboard: `ProductLineSvc.create`
- [ ] `PATCH /v1/core/product-lines/{id}` — Go: `product-lines/endpoint_update_product_line.go` | Dashboard: `ProductLineSvc.update`
- [ ] `DELETE /v1/core/product-lines/{id}` — Go: `product-lines/endpoint_delete_product_line.go` | Dashboard: `ProductLineSvc.delete`

### Product Types

- [ ] `GET /v1/core/product-types` — Go: `product-types/endpoint_list_product_types.go` | Dashboard: (product type CRUD)
- [ ] `GET /v1/core/product-types/{id}` — Go: `product-types/endpoint_get_product_type.go` | Dashboard: (product type CRUD)
- [ ] `POST /v1/core/product-types` — Go: `product-types/endpoint_create_product_type.go` | Dashboard: (product type CRUD)
- [ ] `PATCH /v1/core/product-types/{id}` — Go: `product-types/endpoint_update_product_type.go` | Dashboard: (product type CRUD)
- [ ] `DELETE /v1/core/product-types/{id}` — Go: `product-types/endpoint_delete_product_type.go` | Dashboard: (product type CRUD)

### Items

- [ ] `GET /v1/core/items` — Go: `items/endpoint_list_items.go` | Dashboard: `ItemSvc.list`
- [ ] `GET /v1/core/items/{id}` — Go: `items/endpoint_get_item.go` | Dashboard: `ItemSvc.find`
- [ ] `PATCH /v1/core/items/{id}` — Go: `items/endpoint_update_item.go` | Dashboard: `ItemSvc.update`
- [ ] `POST /v1/core/items/actions/bulk-create` — Go: `items/endpoint_bulk_create_items.go` | Dashboard: `ItemSvc.bulkCreate`
- [ ] `POST /v1/core/items/actions/bulk-reconcile` — Go: `items/endpoint_bulk_reconcile_items.go` | Dashboard: `ItemSvc.bulkReconcile`
- [ ] `PUT /v1/core/items/{id}/category/{category_id}` — Go: `items/endpoint_change_item_category.go` | Dashboard: `ItemSvc.changeCategory`
- [ ] `GET /v1/core/items/{id}/inventory` — Go: `items/endpoint_get_item_inventory.go` | Dashboard: `ItemSvc.fetchInventory`
- [ ] `PATCH /v1/core/items/{id}/inventory` — Go: `items/endpoint_update_item_inventory.go` | Dashboard: `ItemSvc.updateInventory`
- [ ] `GET /v1/core/items/{id}/costs` — Go: `items/endpoint_get_item_costs.go` | Dashboard: `ItemSvc.fetchCosts`
- [ ] `GET /v1/core/items/{id}/trends` — Go: `items/endpoint_get_item_trends.go` | Dashboard: `TrendSvc.fetchTrend`
- [ ] `PUT /v1/core/items/{id}/attributes/{attribute_id}` — Go: `items/endpoint_add_item_attribute.go` | Dashboard: `ItemSvc.addAttribute`
- [ ] `DELETE /v1/core/items/{id}/attributes/{attribute_id}` — Go: `items/endpoint_remove_item_attribute.go` | Dashboard: `ItemSvc.removeAttribute`
- [ ] `GET /v1/core/items/actions/export` — Go: `items/endpoint_export_items.go` | Dashboard: `ItemSvc.exportInventory`

### Item Categories

- [ ] `GET /v1/core/item-categories` — Go: `item-categories/endpoint_list_item_categories.go` | Dashboard: `CategorySvc.list`
- [ ] `GET /v1/core/item-categories/{id}` — Go: `item-categories/endpoint_get_item_category.go` | Dashboard: `CategorySvc.find`
- [ ] `POST /v1/core/item-categories` — Go: `item-categories/endpoint_create_item_category.go` | Dashboard: `CategorySvc.create`
- [ ] `PATCH /v1/core/item-categories/{id}` — Go: `item-categories/endpoint_update_item_category.go` | Dashboard: `CategorySvc.update`
- [ ] `DELETE /v1/core/item-categories/{id}` — Go: `item-categories/endpoint_delete_item_category.go` | Dashboard: `CategorySvc.delete`
- [ ] `PUT /v1/core/item-categories/{id}/properties/{property_id}` — Go: `item-categories/endpoint_add_item_category_property.go` | Dashboard: `CategorySvc.addProperty`
- [ ] `DELETE /v1/core/item-categories/{id}/properties/{property_id}` — Go: `item-categories/endpoint_remove_item_category_property.go` | Dashboard: `CategorySvc.removeProperty`
- [ ] `PUT /v1/core/item-categories/{id}/unit-groups/{unit_group_id}` — Go: `item-categories/endpoint_change_item_category_unit_group.go` | Dashboard: `CategorySvc.changeUnitGroup`

### Properties

- [ ] `GET /v1/core/properties` — Go: `properties/endpoint_list_properties.go` | Dashboard: `PropertySvc.list`
- [ ] `GET /v1/core/properties/{id}` — Go: `properties/endpoint_get_property.go` | Dashboard: `PropertySvc.find`
- [ ] `POST /v1/core/properties` — Go: `properties/endpoint_create_property.go` | Dashboard: `PropertySvc.create`
- [ ] `PATCH /v1/core/properties/{id}` — Go: `properties/endpoint_update_property.go` | Dashboard: `PropertySvc.update`
- [ ] `DELETE /v1/core/properties/{id}` — Go: `properties/endpoint_delete_property.go` | Dashboard: `PropertySvc.delete`

### Attributes

- [ ] `GET /v1/core/properties/{property_id}/attributes` — Go: `properties/endpoint_list_attributes.go` | Dashboard: `AttributeSvc.list`
- [ ] `GET /v1/core/properties/{property_id}/attributes/{id}` — Go: `properties/endpoint_get_attribute.go` | Dashboard: `AttributeSvc.find`
- [ ] `POST /v1/core/properties/{property_id}/attributes` — Go: `properties/endpoint_create_attribute.go` | Dashboard: `AttributeSvc.create`
- [ ] `PATCH /v1/core/properties/{property_id}/attributes/{id}` — Go: `properties/endpoint_update_attribute.go` | Dashboard: `AttributeSvc.update`
- [ ] `DELETE /v1/core/properties/{property_id}/attributes/{id}` — Go: `properties/endpoint_delete_attribute.go` | Dashboard: `AttributeSvc.delete`

### Units

- [ ] `GET /v1/core/units` — Go: `units/endpoint_list_units.go` | Dashboard: (unit CRUD)
- [ ] `GET /v1/core/units/{id}` — Go: `units/endpoint_get_unit.go` | Dashboard: (unit CRUD)
- [ ] `POST /v1/core/units` — Go: `units/endpoint_create_unit.go` | Dashboard: (unit CRUD)
- [ ] `PATCH /v1/core/units/{id}` — Go: `units/endpoint_update_unit.go` | Dashboard: (unit CRUD)
- [ ] `DELETE /v1/core/units/{id}` — Go: `units/endpoint_delete_unit.go` | Dashboard: (unit CRUD)
- [ ] `PUT /v1/core/units/actions/validate` — Go: `units/endpoint_validate_units.go` | Dashboard: `UnitSvc.validateUnits`

### Unit Groups

- [ ] `GET /v1/core/unit-groups` — Go: `unit-groups/endpoint_list_unit_groups.go` | Dashboard: `UnitGroupSvc.list`
- [ ] `GET /v1/core/unit-groups/{id}` — Go: `unit-groups/endpoint_get_unit_group.go` | Dashboard: `UnitGroupSvc.find`
- [ ] `POST /v1/core/unit-groups` — Go: `unit-groups/endpoint_create_unit_group.go` | Dashboard: `UnitGroupSvc.create`
- [ ] `PATCH /v1/core/unit-groups/{id}` — Go: `unit-groups/endpoint_update_unit_group.go` | Dashboard: `UnitGroupSvc.update`
- [ ] `DELETE /v1/core/unit-groups/{id}` — Go: `unit-groups/endpoint_delete_unit_group.go` | Dashboard: `UnitGroupSvc.delete`
- [ ] `PUT /v1/core/unit-groups/{unitGroupId}/units/{id}` — Go: `unit-groups/endpoint_upsert_unit_group_unit.go` | Dashboard: `UnitConversionSvc.upsert`
- [ ] `DELETE /v1/core/unit-groups/{unitGroupId}/units/{id}` — Go: `unit-groups/endpoint_delete_unit_group_unit.go` | Dashboard: `UnitConversionSvc.delete`

### Materials

- [ ] `GET /v1/core/materials` — Go: `materials/endpoint_list_materials.go` | Dashboard: `MaterialSvc.list`
- [ ] `GET /v1/core/materials/{id}` — Go: `materials/endpoint_get_material.go` | Dashboard: `MaterialSvc.find`
- [ ] `POST /v1/core/materials` — Go: `materials/endpoint_create_material.go` | Dashboard: `MaterialSvc.create`
- [ ] `PATCH /v1/core/materials/{id}` — Go: `materials/endpoint_update_material.go` | Dashboard: `MaterialSvc.update`
- [ ] `DELETE /v1/core/materials/{id}` — Go: `materials/endpoint_delete_material.go` | Dashboard: `MaterialSvc.delete`

### Parts

- [ ] `GET /v1/core/parts` — Go: `parts/endpoint_list_parts.go` | Dashboard: `PartSvc.list`
- [ ] `GET /v1/core/parts/{id}` — Go: `parts/endpoint_get_part.go` | Dashboard: `PartSvc.find`
- [ ] `POST /v1/core/parts` — Go: `parts/endpoint_create_part.go` | Dashboard: `PartSvc.create`
- [ ] `PATCH /v1/core/parts/{id}` — Go: `parts/endpoint_update_part.go` | Dashboard: `PartSvc.update`
- [ ] `DELETE /v1/core/parts/{id}` — Go: `parts/endpoint_delete_part.go` | Dashboard: `PartSvc.delete`

### Suppliers

- [ ] `GET /v1/core/suppliers` — Go: `suppliers/endpoint_list_suppliers.go` | Dashboard: `SupplierSvc.list`
- [ ] `GET /v1/core/suppliers/{id}` — Go: `suppliers/endpoint_get_supplier.go` | Dashboard: `SupplierSvc.find`
- [ ] `POST /v1/core/suppliers` — Go: `suppliers/endpoint_create_supplier.go` | Dashboard: `SupplierSvc.create`
- [ ] `PATCH /v1/core/suppliers/{id}` — Go: `suppliers/endpoint_update_supplier.go` | Dashboard: `SupplierSvc.update`
- [ ] `DELETE /v1/core/suppliers/{id}` — Go: `suppliers/endpoint_delete_supplier.go` | Dashboard: `SupplierSvc.delete`
- [ ] `POST /v1/core/suppliers/actions/bulk-delete` — Go: `suppliers/endpoint_bulk_delete_suppliers.go` | Dashboard: `SupplierSvc.deleteMany`

### Supplier Materials

- [ ] `GET /v1/core/suppliers/{supplier_id}/materials` — Go: `supplier-materials/endpoint_list_supplier_materials.go` | Dashboard: `SupplierMaterialSvc.list`
- [ ] `GET /v1/core/suppliers/{supplier_id}/materials/{id}` — Go: `supplier-materials/endpoint_get_supplier_material.go` | Dashboard: `SupplierMaterialSvc.find`
- [ ] `POST /v1/core/suppliers/{supplier_id}/materials` — Go: `supplier-materials/endpoint_create_supplier_material.go` | Dashboard: `SupplierMaterialSvc.create`
- [ ] `PATCH /v1/core/suppliers/{supplier_id}/materials/{id}` — Go: `supplier-materials/endpoint_update_supplier_material.go` | Dashboard: `SupplierMaterialSvc.update`
- [ ] `DELETE /v1/core/suppliers/{supplier_id}/materials/{id}` — Go: `supplier-materials/endpoint_delete_supplier_material.go` | Dashboard: `SupplierMaterialSvc.delete`

### Departments

- [ ] `GET /v1/core/departments` — Go: `departments/endpoint_list_departments.go` | Dashboard: `DepartmentSvc.list`
- [ ] `GET /v1/core/departments/{id}` — Go: `departments/endpoint_get_department.go` | Dashboard: `DepartmentSvc.find`
- [ ] `POST /v1/core/departments` — Go: `departments/endpoint_create_department.go` | Dashboard: `DepartmentSvc.create`
- [ ] `PATCH /v1/core/departments/{id}` — Go: `departments/endpoint_update_department.go` | Dashboard: `DepartmentSvc.update`
- [ ] `DELETE /v1/core/departments/{id}` — Go: `departments/endpoint_delete_department.go` | Dashboard: `DepartmentSvc.delete`

### Production Steps

- [ ] `GET /v1/core/production-steps` — Go: `production-steps/endpoint_list_production_steps.go` | Dashboard: `ProductionStepSvc.list`
- [ ] `GET /v1/core/production-steps/{id}` — Go: `production-steps/endpoint_get_production_step.go` | Dashboard: `ProductionStepSvc.find`
- [ ] `POST /v1/core/production-steps` — Go: `production-steps/endpoint_create_production_step.go` | Dashboard: `ProductionStepSvc.create`
- [ ] `PATCH /v1/core/production-steps/{id}` — Go: `production-steps/endpoint_update_production_step.go` | Dashboard: `ProductionStepSvc.update`
- [ ] `DELETE /v1/core/production-steps/{id}` — Go: `production-steps/endpoint_delete_production_step.go` | Dashboard: `ProductionStepSvc.delete`
- [ ] `POST /v1/core/production-steps/actions/bulk-create` — Go: `production-steps/endpoint_bulk_create_production_steps.go` | Dashboard: `ProductionStepSvc.bulkCreate`

### Productions

- [ ] `GET /v1/core/production-steps/{production_step_id}/productions/{id}` — Go: `production-steps/endpoint_get_production.go` | Dashboard: `ProductionSvc.find`
- [ ] `PATCH /v1/core/production-steps/{production_step_id}/productions/{id}` — Go: `production-steps/endpoint_update_production.go` | Dashboard: `ProductionSvc.update`

### Production Runs

- [ ] `GET /v1/core/production-runs` — Go: `production-runs/endpoint_list_production_runs.go` | Dashboard: `ProductionRunSvc.list`
- [ ] `GET /v1/core/production-runs/{id}` — Go: `production-runs/endpoint_get_production_run.go` | Dashboard: `ProductionRunSvc.find`
- [ ] `POST /v1/core/production-runs` — Go: `production-runs/endpoint_create_production_run.go` | Dashboard: `ProductionRunSvc.create`
- [ ] `PATCH /v1/core/production-runs/{id}` — Go: `production-runs/endpoint_update_production_run.go` | Dashboard: `ProductionRunSvc.update`
- [ ] `DELETE /v1/core/production-runs/{id}` — Go: `production-runs/endpoint_delete_production_run.go` | Dashboard: `ProductionRunSvc.delete`
- [ ] `GET /v1/core/production-runs/{id}/batches` — Go: `production-runs/endpoint_list_batches_by_production_run.go` | Dashboard: `ProductionRunSvc.getBatchesByProductionRun`
- [ ] `POST /v1/core/production-runs/{id}/batches` — Go: `production-runs/endpoint_add_batches_to_production_run.go` | Dashboard: `ProductionRunSvc.addBatchesToProductionRun`

### Production Flows

- [ ] `GET /v1/core/production-flows/by-item/{item_id}` — Go: `production-flows/endpoint_get_production_flow.go` | Dashboard: `ProductionFlowSvc.findByItem`
- [ ] `POST /v1/core/production-flows/actions/connect-steps` — Go: `production-flows/endpoint_connect_steps.go` | Dashboard: `ProductionFlowSvc.connectSteps`

### Consumptions

- [ ] `GET /v1/core/production-steps/{production_step_id}/consumptions/{id}` — Go: `consumptions/endpoint_get_consumption.go` | Dashboard: `ConsumptionSvc.find`
- [ ] `POST /v1/core/production-steps/{production_step_id}/consumptions` — Go: `consumptions/endpoint_create_consumption.go` | Dashboard: `ConsumptionSvc.create`
- [ ] `PATCH /v1/core/production-steps/{production_step_id}/consumptions/{id}` — Go: `consumptions/endpoint_update_consumption.go` | Dashboard: `ConsumptionSvc.update`
- [ ] `DELETE /v1/core/production-steps/{production_step_id}/consumptions/{id}` — Go: `consumptions/endpoint_delete_consumption.go` | Dashboard: `ConsumptionSvc.delete`

### Batches

- [ ] `POST /v1/core/batches/actions/initialize` — Go: `batches/endpoint_initialize_batch.go` | Dashboard: `BatchSvc.initializeBatch`
- [ ] `GET /v1/core/scanning-stations/{id}/batches` — Go: `batches/endpoint_list_batches_by_scanning_station.go` | Dashboard: `BatchSvc.fetchByScanningStation`
- [ ] `GET /v1/core/batches/{id}/flow` — Go: `batches/endpoint_get_batch_flow.go` | Dashboard: `BatchSvc.fetchBatchFlow`
- [ ] `POST /v1/core/scanning-stations/{id}/consumptions` — Go: `batches/endpoint_get_scanning_station_consumption.go` | Dashboard: `BatchSvc.getBatchScanningStationConsumption`
- [ ] `POST /v1/core/batches/actions/split` — Go: `batches/endpoint_split_batch.go` | Dashboard: `BatchSvc.splitBatch`
- [ ] `POST /v1/core/batches/actions/merge` — Go: `batches/endpoint_merge_batches.go` | Dashboard: `BatchSvc.mergeBatches`
- [ ] `POST /v1/core/batches/actions/move` — Go: `batches/endpoint_move_batches.go` | Dashboard: `BatchSvc.moveBatches`
- [ ] `POST /v1/core/batches/remaining-quantities` — Go: `batches/endpoint_get_remaining_quantity_to_split.go` | Dashboard: `BatchSvc.findRemainingQuantityToSplit`
- [ ] `POST /v1/core/batches/{id}/next-steps` — Go: `batches/endpoint_get_possible_next_steps.go` | Dashboard: `BatchSvc.fetchPossibleNextSteps`
- [ ] `POST /v1/core/batches/actions/close` — Go: `batches/endpoint_close_batch.go` | Dashboard: `BatchSvc.close`
- [ ] `DELETE /v1/core/batches/{id}` — Go: `batches/endpoint_delete_batch.go` | Dashboard: `BatchSvc.deleteBatch`
- [ ] `POST /v1/core/batches/actions/bulk-delete` — Go: `batches/endpoint_bulk_delete_batches.go` | Dashboard: `BatchSvc.deleteManyBatches`

### Scanning Stations

- [ ] `GET /v1/core/scanning-stations` — Go: `scanning_stations/endpoint_list_scanning_stations.go` | Dashboard: `ScanningStationSvc.list`
- [ ] `GET /v1/core/scanning-stations/{id}` — Go: `scanning_stations/endpoint_get_scanning_station.go` | Dashboard: `ScanningStationSvc.find`
- [ ] `POST /v1/core/scanning-stations` — Go: `scanning_stations/endpoint_create_scanning_station.go` | Dashboard: `ScanningStationSvc.create`
- [ ] `PATCH /v1/core/scanning-stations/{id}` — Go: `scanning_stations/endpoint_update_scanning_station.go` | Dashboard: `ScanningStationSvc.update`
- [ ] `DELETE /v1/core/scanning-stations/{id}` — Go: `scanning_stations/endpoint_delete_scanning_station.go` | Dashboard: `ScanningStationSvc.delete`
- [ ] `PUT /v1/core/scanning-stations/{id}/production-steps` — Go: `scanning_stations/endpoint_connect_production_steps.go` | Dashboard: `ScanningStationSvc.connectStepsByName`

### Picks

- [ ] `GET /v1/core/picks` — Go: `picks/endpoint_list_picks.go` | Dashboard: `PickSvc.list`
- [ ] `GET /v1/core/picks/{id}` — Go: `picks/endpoint_get_pick.go` | Dashboard: `PickSvc.find`
- [ ] `PUT /v1/core/picks/{id}/actions/pick` — Go: `picks/endpoint_pick_all_lines.go` | Dashboard: `PickSvc.pick`
- [ ] `POST /v1/core/picks/{id}/actions/pack` — Go: `picks/endpoint_pack_pick.go` | Dashboard: `PickSvc.pack`
- [ ] `PATCH /v1/core/picks/{id}` — Go: `picks/endpoint_update_pick.go` | Dashboard: `PickSvc.update`
- [ ] `PUT /v1/core/picks/{id}/actions/void` — Go: `picks/endpoint_void_pick.go` | Dashboard: `PickSvc.void`
- [ ] `GET /v1/core/picks/{id}/shipments` — Go: `picks/endpoint_get_pick_shipments.go` | Dashboard: `ShipmentSvc.fetchNumbersByPick`

### Pick Lines

- [ ] `PATCH /v1/core/picks/{pickId}/lines/{id}` — Go: `picks/endpoint_update_pick_line.go` | Dashboard: `PickLineSvc.update`
- [ ] `PUT /v1/core/picks/{pickId}/lines/{id}/actions/pick` — Go: `picks/endpoint_pick_pick_line.go` | Dashboard: `PickLineSvc.pickRemainingQuantity`
- [ ] `PUT /v1/core/picks/{pickId}/lines/{id}/actions/void` — Go: `picks/endpoint_void_pick_line.go` | Dashboard: `PickLineSvc.voidLine`

### Shipments

- [ ] `GET /v1/core/shipments` — Go: `shipments/endpoint_list_shipments.go` | Dashboard: `ShipmentSvc.list`
- [ ] `GET /v1/core/shipments/{id}` — Go: `shipments/endpoint_get_shipment.go` | Dashboard: `ShipmentSvc.find`
- [ ] `PATCH /v1/core/shipments/{id}` — Go: `shipments/endpoint_update_shipment.go` | Dashboard: `ShipmentSvc.update`
- [ ] `DELETE /v1/core/shipments/{id}` — Go: `shipments/endpoint_delete_shipment.go` | Dashboard: `ShipmentSvc.deleteShipment`
- [ ] `POST /v1/core/shipments/{id}/actions/void` — Go: `shipments/endpoint_void_shipment.go` | Dashboard: `ShipmentSvc.void`
- [ ] `POST /v1/core/shipments/{id}/actions/ship` — Go: `shipments/endpoint_ship_shipment.go` | Dashboard: `ShipmentSvc.ship`
- [ ] `POST /v1/core/shipments/actions/estimate-rate` — Go: `shipments/endpoint_estimate_rate.go` | Dashboard: `ShipmentSvc.estimateRate`
- [ ] `POST /v1/core/shipments/actions/rate-shop` — Go: `shipments/endpoint_rate_shop.go` | Dashboard: `ShipmentSvc.rateShop`

### Shipment Lines

- [ ] `GET /v1/core/shipments/{shipment_id}/lines` — Go: `shipments/endpoint_list_shipment_lines.go` | Dashboard: (shipment lines)
- [ ] `GET /v1/core/shipments/{shipment_id}/lines/{id}` — Go: `shipments/endpoint_get_shipment_line.go` | Dashboard: (shipment lines)
- [ ] `POST /v1/core/shipments/{shipment_id}/lines` — Go: `shipments/endpoint_create_shipment_line.go` | Dashboard: (shipment lines)
- [ ] `PATCH /v1/core/shipments/{shipment_id}/lines/{id}` — Go: `shipments/endpoint_update_shipment_line.go` | Dashboard: (shipment lines)
- [ ] `DELETE /v1/core/shipments/{shipment_id}/lines/{id}` — Go: `shipments/endpoint_delete_shipment_line.go` | Dashboard: (shipment lines)

### Shipping Cases

- [ ] `GET /v1/core/shipping-cases/{id}` — Go: `shipping-cases/endpoint_get_shipping_case.go` | Dashboard: `ShippingCaseSvc.find`
- [ ] `PATCH /v1/core/shipping-cases/{id}` — Go: `shipping-cases/endpoint_update_shipping_case.go` | Dashboard: `ShippingCaseSvc.update`
- [ ] `DELETE /v1/core/shipping-cases/{id}` — Go: `shipping-cases/endpoint_delete_shipping_case.go` | Dashboard: `ShippingCaseSvc.delete`
- [ ] `GET /v1/core/shipping-cases/{id}/label` — Go: `shipping-cases/endpoint_get_shipping_case_label.go` | Dashboard: `ShippingCaseSvc.fetchLabel`

### Shipping Terms

- [ ] `GET /v1/core/shipping-terms` — Go: `shipping-terms/endpoint_list_shipping_terms.go` | Dashboard: `ShippingTermSvc.list`
- [ ] `GET /v1/core/shipping-terms/{id}` — Go: `shipping-terms/endpoint_get_shipping_term.go` | Dashboard: `ShippingTermSvc.find`
- [ ] `POST /v1/core/shipping-terms` — Go: `shipping-terms/endpoint_create_shipping_term.go` | Dashboard: `ShippingTermSvc.create`
- [ ] `PATCH /v1/core/shipping-terms/{id}` — Go: `shipping-terms/endpoint_update_shipping_term.go` | Dashboard: `ShippingTermSvc.update`
- [ ] `DELETE /v1/core/shipping-terms/{id}` — Go: `shipping-terms/endpoint_delete_shipping_term.go` | Dashboard: `ShippingTermSvc.delete`

### Carriers

- [ ] `GET /v1/core/carriers` — Go: `carriers/endpoint_list_carriers.go` | Dashboard: `CarrierSvc.list`
- [ ] `GET /v1/core/carriers/{id}` — Go: `carriers/endpoint_get_carrier.go` | Dashboard: `CarrierSvc.find`
- [ ] `POST /v1/core/carriers` — Go: `carriers/endpoint_create_carrier.go` | Dashboard: `CarrierSvc.create`
- [ ] `PATCH /v1/core/carriers/{id}` — Go: `carriers/endpoint_update_carrier.go` | Dashboard: `CarrierSvc.update`
- [ ] `DELETE /v1/core/carriers/{id}` — Go: `carriers/endpoint_delete_carrier.go` | Dashboard: `CarrierSvc.delete`
- [ ] `POST /v1/core/carriers/{id}/actions/initiate-oauth` — Go: `carriers/endpoint_initiate_oauth.go` | Dashboard: `CarrierSvc.initiateOAuth`
- [ ] `GET /v1/core/carriers/{id}/oauth-status` — Go: `carriers/endpoint_get_oauth_status.go` | Dashboard: `CarrierSvc.getOAuthStatus`
- [ ] `POST /v1/core/carriers/{id}/actions/sync-options` — Go: `carriers/endpoint_sync_options.go` | Dashboard: `CarrierSvc.syncOptions`

### Carrier Options

- [ ] `GET /v1/core/carriers/{carrier_id}/options` — Go: `carrier-options/endpoint_list_carrier_options.go` | Dashboard: `CarrierOptionSvc.list`
- [ ] `GET /v1/core/carriers/{carrier_id}/options/{id}` — Go: `carrier-options/endpoint_get_carrier_option.go` | Dashboard: `CarrierOptionSvc.find`
- [ ] `POST /v1/core/carriers/{carrier_id}/options` — Go: `carrier-options/endpoint_create_carrier_option.go` | Dashboard: `CarrierOptionSvc.create`
- [ ] `PATCH /v1/core/carriers/{carrier_id}/options/{id}` — Go: `carrier-options/endpoint_update_carrier_option.go` | Dashboard: `CarrierOptionSvc.update`
- [ ] `DELETE /v1/core/carriers/{carrier_id}/options/{id}` — Go: `carrier-options/endpoint_delete_carrier_option.go` | Dashboard: `CarrierOptionSvc.delete`

### Receiving Orders

- [ ] `GET /v1/core/receiving-orders` — Go: `receiving-orders/endpoint_list_receiving_orders.go` | Dashboard: `ReceivingOrderSvc.list`
- [ ] `GET /v1/core/receiving-orders/{id}` — Go: `receiving-orders/endpoint_get_receiving_order.go` | Dashboard: `ReceivingOrderSvc.find`
- [ ] `PUT /v1/core/receiving-orders/{id}/actions/receive` — Go: `receiving-orders/endpoint_receive_receiving_order.go` | Dashboard: `ReceivingOrderSvc.receive`
- [ ] `POST /v1/core/receiving-orders/{id}/actions/stock` — Go: `receiving-orders/endpoint_stock_receiving_order.go` | Dashboard: `ReceivingOrderSvc.stock`
- [ ] `PUT /v1/core/receiving-orders/{id}/actions/void` — Go: `receiving-orders/endpoint_void_receiving_order.go` | Dashboard: `ReceivingOrderSvc.void`

### Receiving Order Lines

- [ ] `PATCH /v1/core/receiving-orders/{receivingOrderId}/lines/{id}` — Go: `receiving-orders/endpoint_update_receiving_order_line.go` | Dashboard: `ReceivingOrderLineSvc.update`
- [ ] `PUT /v1/core/receiving-orders/{receivingOrderId}/lines/{id}/actions/receive` — Go: `receiving-orders/endpoint_receive_receiving_order_line.go` | Dashboard: `ReceivingOrderLineSvc.receiveLine`
- [ ] `PUT /v1/core/receiving-orders/{receivingOrderId}/lines/{id}/actions/void` — Go: `receiving-orders/endpoint_void_receiving_order_line.go` | Dashboard: `ReceivingOrderLineSvc.voidLine`

### Deliveries

- [ ] `GET /v1/core/deliveries` — Go: `deliveries/endpoint_list_deliveries.go` | Dashboard: `DeliverySvc.list`
- [ ] `GET /v1/core/deliveries/{id}` — Go: `deliveries/endpoint_get_delivery.go` | Dashboard: `DeliverySvc.find`

### Inventories

- [ ] `GET /v1/core/inventories` — Go: `inventories/endpoint_list_inventories.go` | Dashboard: (inventory listing)

### Transactions

- [ ] `GET /v1/core/transactions` — Go: `transactions/endpoint_list_transactions.go` | Dashboard: `TransactionSvc.list`
- [ ] `GET /v1/core/accounts/{account_id}/transactions` — Go: `transactions/endpoint_list_account_transactions.go` | Dashboard: `TransactionSvc.listByCustomer`
- [ ] `GET /v1/core/transactions/{id}` — Go: `transactions/endpoint_get_transaction.go` | Dashboard: `TransactionSvc.find`
- [ ] `POST /v1/core/transactions` — Go: `transactions/endpoint_create_transaction.go` | Dashboard: `TransactionSvc.create`
- [ ] `PATCH /v1/core/transactions/{id}` — Go: `transactions/endpoint_update_transaction.go` | Dashboard: `TransactionSvc.update`
- [ ] `DELETE /v1/core/transactions/{id}` — Go: `transactions/endpoint_delete_transaction.go` | Dashboard: `TransactionSvc.delete`
- [ ] `GET /v1/core/transaction-methods` — Go: `transactions/endpoint_list_transaction_methods.go` | Dashboard: `TransactionMethodSvc.list`
- [ ] `GET /v1/core/transaction-types` — Go: `transactions/endpoint_list_transaction_types.go` | Dashboard: `TransactionTypeSvc.list`

### Transaction Allocations

- [ ] `GET /v1/core/open-credits` — Go: `transaction-allocations/endpoint_list_open_credits.go` | Dashboard: `TransactionAllocationSvc.fetchOpenCredits`
- [ ] `GET /v1/core/transaction-allocations` — Go: `transaction-allocations/endpoint_list_allocation_entries.go` | Dashboard: `TransactionAllocationSvc.fetchEntries`
- [ ] `PATCH /v1/core/transaction-allocations/{id}` — Go: `transaction-allocations/endpoint_update_transaction_allocation.go` | Dashboard: `TransactionAllocationSvc.update`
- [ ] `DELETE /v1/core/transaction-allocations/{id}` — Go: `transaction-allocations/endpoint_delete_transaction_allocation.go` | Dashboard: `TransactionAllocationSvc.delete`

### Payment Terms

- [ ] `GET /v1/core/payment-terms` — Go: `payment-terms/endpoint_list_payment_terms.go` | Dashboard: `PaymentTermSvc.list`
- [ ] `GET /v1/core/payment-terms/{id}` — Go: `payment-terms/endpoint_get_payment_term.go` | Dashboard: `PaymentTermSvc.find`
- [ ] `POST /v1/core/payment-terms` — Go: `payment-terms/endpoint_create_payment_term.go` | Dashboard: `PaymentTermSvc.create`
- [ ] `PATCH /v1/core/payment-terms/{id}` — Go: `payment-terms/endpoint_update_payment_term.go` | Dashboard: `PaymentTermSvc.update`
- [ ] `DELETE /v1/core/payment-terms/{id}` — Go: `payment-terms/endpoint_delete_payment_term.go` | Dashboard: `PaymentTermSvc.delete`

### Settlements

- [ ] `GET /v1/core/settlements` — Go: `settlements/endpoint_list_settlements.go` | Dashboard: `SettlementSvc.list`
- [ ] `GET /v1/core/settlements/{id}` — Go: `settlements/endpoint_get_settlement.go` | Dashboard: `SettlementSvc.find`
- [ ] `POST /v1/core/settlements` — Go: `settlements/endpoint_create_settlement.go` | Dashboard: `SettlementSvc.create`
- [ ] `PATCH /v1/core/settlements/{id}` — Go: `settlements/endpoint_update_settlement.go` | Dashboard: `SettlementSvc.update`
- [ ] `DELETE /v1/core/settlements/{id}` — Go: `settlements/endpoint_delete_settlement.go` | Dashboard: `SettlementSvc.delete`

### Invoices

- [ ] `GET /v1/core/invoices` — Go: `invoices/endpoint_list_invoices.go` | Dashboard: `InvoiceSvc.list`
- [ ] `GET /v1/core/invoices/{id}` — Go: `invoices/endpoint_get_invoice.go` | Dashboard: `InvoiceSvc.find`
- [ ] `PATCH /v1/core/invoices/{id}` — Go: `invoices/endpoint_update_invoice.go` | Dashboard: `InvoiceSvc.summaryUpdate`
- [ ] `GET /v1/core/accounts/{account_id}/invoices` — Go: `invoices/endpoint_list_customer_invoices.go` | Dashboard: `InvoiceSvc.fetchByCustomer`

### Receivables

- [ ] `GET /v1/core/receivables` — Go: `receivables/endpoint_list_receivables.go` | Dashboard: `ReceivableSvc.getAllReceivables`
- [ ] `GET /v1/core/receivables/accounts/{account_id}` — Go: `receivables/endpoint_list_receivables_by_customer.go` | Dashboard: `ReceivableSvc.getReceivablesByAccount`
- [ ] `GET /v1/core/receivables/accounts/{account_id}/actions/export` — Go: `receivables/endpoint_export_receivables_by_customer.go` | Dashboard: `ReceivableSvc.getReceivablesReportByAccount`
- [ ] `POST /v1/core/accounts/{account_id}/actions/email-receivables` — Go: `receivables/endpoint_email_receivables_for_customer.go` | Dashboard: `ReceivableSvc.emailReceivablesForAccount`

### Analytics

- [ ] `PUT /v1/core/analytics/inventory-receipts` — Go: `analytics/endpoint_analyze_inventory_receipts.go` | Dashboard: `AnalyticsSvc.analyzeInventoryReceipts`
- [ ] `PUT /v1/core/analytics/deliveries` — Go: `analytics/endpoint_analyze_deliveries.go` | Dashboard: `AnalyticsSvc.getDeliveryAnalytics`
- [ ] `PUT /v1/core/analytics/demand-forecast` — Go: `analytics/endpoint_analyze_demand_forecast.go` | Dashboard: `AnalyticsSvc.getDemandForecast`
- [ ] `PUT /v1/core/analytics/manufacturing` — Go: `analytics/endpoint_analyze_manufacturing.go` | Dashboard: `AnalyticsSvc.getManufacturingAnalytics`
- [ ] `PUT /v1/core/analytics/manufacturing-batch` — Go: `analytics/endpoint_analyze_manufacturing_batch.go` | Dashboard: `AnalyticsSvc.getManufacturingAnalyticsBatch`
- [ ] `PUT /v1/core/analytics/materials` — Go: `analytics/endpoint_analyze_materials.go` | Dashboard: `AnalyticsSvc.getMaterialAnalytics`
- [ ] `PUT /v1/core/analytics/oee` — Go: `analytics/endpoint_analyze_oee.go` | Dashboard: `AnalyticsSvc.getOeeAnalytics`
- [ ] `PUT /v1/core/analytics/quarterly-orders` — Go: `analytics/endpoint_analyze_quarterly_orders.go` | Dashboard: `AnalyticsSvc.getQuarterlyOrders`
- [ ] `GET /v1/core/analytics/weeks-of-sales` — Go: `analytics/endpoint_analyze_weeks_of_sales.go` | Dashboard: `AnalyticsSvc.getWeeksOfSales`
- [ ] `PUT /v1/core/analytics/orders` — Go: `analytics/endpoint_analyze_orders.go` | Dashboard: (order analytics)
- [ ] `PUT /v1/core/analytics/sales` — Go: `analytics/endpoint_analyze_sales.go` | Dashboard: (sales analytics)
- [ ] `PUT /v1/core/analytics/new-customers` — Go: `analytics/endpoint_analyze_new_customers.go` | Dashboard: `CustomerSvc.getNewCustomerReport`
- [ ] `PUT /v1/core/analytics/open-batches` — Go: `analytics/endpoint_analyze_open_batches.go` | Dashboard: `BatchSvc.fetchOpenBatches`
- [ ] `PUT /v1/core/analytics/production-costs` — Go: `analytics/endpoint_analyze_production_costs.go` | Dashboard: `BatchSvc.fetchProductionCosts`

### Order Discounts

- [ ] `GET /v1/core/order-discounts` — Go: `order-discounts/endpoint_list_order_discounts.go` | Dashboard: `OrderDiscountSvc.list`
- [ ] `GET /v1/core/order-discounts/{id}` — Go: `order-discounts/endpoint_get_order_discount.go` | Dashboard: `OrderDiscountSvc.find`
- [ ] `POST /v1/core/order-discounts` — Go: `order-discounts/endpoint_create_order_discount.go` | Dashboard: `OrderDiscountSvc.create`
- [ ] `PATCH /v1/core/order-discounts/{id}` — Go: `order-discounts/endpoint_update_order_discount.go` | Dashboard: `OrderDiscountSvc.update`
- [ ] `DELETE /v1/core/order-discounts/{id}` — Go: `order-discounts/endpoint_delete_order_discount.go` | Dashboard: `OrderDiscountSvc.delete`
- [ ] `POST /v1/core/order-discounts/actions/find-by-code` — Go: `order-discounts/endpoint_find_order_discount_by_code.go` | Dashboard: `OrderDiscountSvc.findByCode`

### Volume Discounts

- [ ] `GET /v1/core/volume-discounts` — Go: `volume-discounts/endpoint_list_volume_discounts.go` | Dashboard: `VolumeDiscountSvc.list`
- [ ] `GET /v1/core/volume-discounts/{id}` — Go: `volume-discounts/endpoint_get_volume_discount.go` | Dashboard: `VolumeDiscountSvc.find`
- [ ] `POST /v1/core/volume-discounts` — Go: `volume-discounts/endpoint_create_volume_discount.go` | Dashboard: `VolumeDiscountSvc.create`
- [ ] `PATCH /v1/core/volume-discounts/{id}` — Go: `volume-discounts/endpoint_update_volume_discount.go` | Dashboard: `VolumeDiscountSvc.update`
- [ ] `DELETE /v1/core/volume-discounts/{id}` — Go: `volume-discounts/endpoint_delete_volume_discount.go` | Dashboard: `VolumeDiscountSvc.delete`

### Quantities

- [ ] `PATCH /v1/core/quantities/{id}` — Go: `quantities/endpoint_update_quantity.go` | Dashboard: `QuantitySvc.update`

### Rates

- [ ] `PATCH /v1/core/rates/{id}` — Go: `rates/endpoint_update_rate.go` | Dashboard: `RateSvc.update`

### Machines

- [ ] `GET /v1/core/machines` — Go: `machines/endpoint_list_machines.go` | Dashboard: `MachineSvc.list`
- [ ] `GET /v1/core/machines/{id}` — Go: `machines/endpoint_get_machine.go` | Dashboard: `MachineSvc.find`
- [ ] `POST /v1/core/machines` — Go: `machines/endpoint_create_machine.go` | Dashboard: `MachineSvc.create`
- [ ] `PATCH /v1/core/machines/{id}` — Go: `machines/endpoint_update_machine.go` | Dashboard: `MachineSvc.update`
- [ ] `DELETE /v1/core/machines/{id}` — Go: `machines/endpoint_delete_machine.go` | Dashboard: `MachineSvc.delete`

### Storage Locations

- [ ] `GET /v1/core/storage-locations` — Go: `storage-locations/endpoint_list_storage_locations.go` | Dashboard: `StorageLocationSvc.list`
- [ ] `GET /v1/core/storage-locations/{id}` — Go: `storage-locations/endpoint_get_storage_location.go` | Dashboard: `StorageLocationSvc.find`
- [ ] `POST /v1/core/storage-locations` — Go: `storage-locations/endpoint_create_storage_location.go` | Dashboard: `StorageLocationSvc.create`
- [ ] `PATCH /v1/core/storage-locations/{id}` — Go: `storage-locations/endpoint_update_storage_location.go` | Dashboard: `StorageLocationSvc.update`
- [ ] `DELETE /v1/core/storage-locations/{id}` — Go: `storage-locations/endpoint_delete_storage_location.go` | Dashboard: `StorageLocationSvc.delete`
- [ ] `GET /v1/core/storage-location-types` — Go: `storage-locations/endpoint_list_storage_location_types.go` | Dashboard: `StorageLocationSvc.getStorageLocationTypes`

### Catalog

- [ ] `GET /v1/core/catalog/product-lines` — Go: `catalog/endpoint_list_catalog_product_lines.go` | Dashboard: `CatalogSvc.listProductLines`
- [ ] `GET /v1/core/catalog/product-lines/{id}/products` — Go: `catalog/endpoint_list_catalog_products.go` | Dashboard: `CatalogSvc.listProducts`

### Checkout Sessions

- [ ] `POST /v1/core/checkout-sessions` — Go: `checkout-sessions/endpoint_create_checkout_session.go` | Dashboard: `StripeSvc.createCustomCheckoutSession`

### EDI

- [ ] `PUT /v1/core/edi/actions/pull-orders` — Go: `edi/endpoint_pull_edi_orders.go` | Dashboard: `EdiSvc.getEdiOrders`
- [ ] `POST /v1/core/edi/actions/resubmit-invoice` — Go: `edi/endpoint_resubmit_edi_invoice.go` | Dashboard: `EdiSvc.resubmitInvoice`

### EDI DC Locations

- [ ] `GET /v1/core/dc-locations` — Go: `edi-dc-locations/endpoint_list_dc_locations.go` | Dashboard: `EdiDcLocationSvc.list`
- [ ] `GET /v1/core/dc-locations/{id}` — Go: `edi-dc-locations/endpoint_get_dc_location.go` | Dashboard: `EdiDcLocationSvc.find`
- [ ] `POST /v1/core/dc-locations` — Go: `edi-dc-locations/endpoint_create_dc_location.go` | Dashboard: `EdiDcLocationSvc.create`
- [ ] `PATCH /v1/core/dc-locations/{id}` — Go: `edi-dc-locations/endpoint_update_dc_location.go` | Dashboard: `EdiDcLocationSvc.update`
- [ ] `DELETE /v1/core/dc-locations/{id}` — Go: `edi-dc-locations/endpoint_delete_dc_location.go` | Dashboard: `EdiDcLocationSvc.delete`

### EDI Runs

- [ ] `GET /v1/core/edi-runs` — Go: `edi-runs/endpoint_list_edi_runs.go` | Dashboard: `EdiRunSvc.list`
- [ ] `GET /v1/core/edi-runs/{id}` — Go: `edi-runs/endpoint_get_edi_run.go` | Dashboard: `EdiRunSvc.find`

### Email Logs

- [ ] `GET /v1/core/email-logs` — Go: `email-logs/endpoint_list_email_logs.go` | Dashboard: `EmailLogSvc.list`
- [ ] `GET /v1/core/email-logs/{id}` — Go: `email-logs/endpoint_get_email_log.go` | Dashboard: `EmailLogSvc.find`

### Inventory Change Logs

- [ ] `GET /v1/core/inventory-change-logs` — Go: `inventory-change-logs/endpoint_list_inventory_change_logs.go` | Dashboard: `InventoryChangeLogSvc.list`
- [ ] `GET /v1/core/inventory-change-logs/{id}` — Go: `inventory-change-logs/endpoint_get_inventory_change_log.go` | Dashboard: `InventoryChangeLogSvc.find`
- [ ] `GET /v1/core/inventory-change-logs/actions/export` — Go: `inventory-change-logs/endpoint_export_inventory_change_logs.go` | Dashboard: `InventoryChangeLogSvc.exportInventoryChangeLogs`

### Request Logs

- [ ] `GET /v1/core/request-logs` — Go: `request_logs/endpoint_list_request_logs.go` | Dashboard: `RequestLogSvc.list`
- [ ] `GET /v1/core/request-logs/{id}` — Go: `request_logs/endpoint_get_request_log.go` | Dashboard: `RequestLogSvc.find`

### Roles

- [ ] `GET /v1/core/roles` — Go: `roles/endpoint_list_roles.go` | Dashboard: `RoleSvc.list`
- [ ] `GET /v1/core/roles/{id}` — Go: `roles/endpoint_get_role.go` | Dashboard: `RoleSvc.find`
- [ ] `POST /v1/core/roles` — Go: `roles/endpoint_create_role.go` | Dashboard: `RoleSvc.create`
- [ ] `PATCH /v1/core/roles/{id}` — Go: `roles/endpoint_update_role.go` | Dashboard: `RoleSvc.update`
- [ ] `DELETE /v1/core/roles/{id}` — Go: `roles/endpoint_delete_role.go` | Dashboard: `RoleSvc.delete`

### Permission Groups

- [ ] `GET /v1/core/permission-groups` — Go: `permission-groups/endpoint_list_permission_groups.go` | Dashboard: `PermissionGroupSvc.list`

### Tenancy

- [ ] `GET /v1/core/me` — Go: `tenancy/endpoint_get_current_user.go` | Dashboard: `TenancySvc.findCurrentUser`
- [ ] `GET /v1/core/me/tenancy` — Go: `tenancy/endpoint_get_tenancy.go` | Dashboard: `TenancySvc.getMyTenancy`
- [ ] `GET /v1/core/me/tenancy/customer-accounts/{vendor_account_id}` — Go: `tenancy/endpoint_list_customer_accounts.go` | Dashboard: `TenancySvc.getMyCustomerAccounts`
- [ ] `PUT /v1/core/me/tenancy` — Go: `tenancy/endpoint_switch_account.go` | Dashboard: `TenancySvc.switchAccount`

### Users

- [ ] `GET /v1/core/users/{id}` — Go: `users/endpoint_get_user.go` | Dashboard: `UserSvc.find`
- [ ] `PATCH /v1/core/users/{id}` — Go: `users/endpoint_update_user.go` | Dashboard: `UserSvc.update`
- [ ] `GET /v1/core/users/{id}/photo` — Go: `users/endpoint_get_user_photo_url.go` | Dashboard: `UserSvc.getPhotoUrl`
- [ ] `PUT /v1/core/users/{id}/photo` — Go: `users/endpoint_upload_user_photo.go` | Dashboard: `UserSvc.updatePhoto`

### Sales Targets

- [ ] `GET /v1/core/account-users/{id}/sales-targets` — Go: `sales-targets/endpoint_list_sales_targets.go` | Dashboard: `AccountUserSvc.fetchSalesTargets`
- [ ] `POST /v1/core/account-users/{id}/sales-targets` — Go: `sales-targets/endpoint_create_sales_target.go` | Dashboard: `AccountUserSvc.createSalesTarget`
- [ ] `PUT /v1/core/account-users/{id}/sales-targets/{target_id}` — Go: `sales-targets/endpoint_upsert_sales_target.go` | Dashboard: `AccountUserSvc.upsertSalesTarget`

### System Properties

- [ ] `GET /v1/core/sys-properties` — Go: `sys_properties/endpoint_list_sys_properties.go` | Dashboard: `SysPropertySvc.list`
- [ ] `GET /v1/core/sys-properties/{id}` — Go: `sys_properties/endpoint_get_sys_property.go` | Dashboard: `SysPropertySvc.find`
- [ ] `PATCH /v1/core/sys-properties/{id}` — Go: `sys_properties/endpoint_update_sys_property.go` | Dashboard: `SysPropertySvc.update`
- [ ] `GET /v1/core/sys-properties/{type_code}/latest-value` — Go: `sys_properties/endpoint_get_latest_sys_property_value.go` | Dashboard: `SysPropertySvc.getSysPropertyValue`

### Adjustment Types

- [ ] `GET /v1/core/adjustment-types` — Go: `adjustment-types/endpoint_list_adjustment_types.go` | Dashboard: `AdjustmentTypeSvc.list`

### Priorities

- [ ] `GET /v1/core/priorities` — Go: `priorities/endpoint_list_priorities.go` | Dashboard: `PrioritySvc.list`
- [ ] `GET /v1/core/priorities/{id}` — Go: `priorities/endpoint_get_priority.go` | Dashboard: `PrioritySvc.find`

### Utils

- [ ] `PUT /v1/core/actions/check-duplicates` — Go: `utils/endpoint_check_duplicate.go` | Dashboard: `UtilsSvc.checkDuplicates`
- [ ] `POST /v1/core/actions/email-record` — Go: `utils/endpoint_email_record.go` | Dashboard: `UtilsSvc.emailRecord`
- [ ] `POST /v1/core/actions/request-demo` — Go: `utils/endpoint_request_demo.go` | Dashboard: `AccountSvc.sendDemoRequestEmail`
- [ ] `POST /v1/core/actions/submit-feedback` — Go: `utils/endpoint_submit_feedback.go` | Dashboard: `AccountSvc.sendFeedbackEmail`

### Stripe Webhook

- [ ] `POST /v1/webhooks/stripe` — Go: `webhooks/endpoint_process_webhook.go` | Dashboard: `StripeSvc.handleEvent`
- [ ] `POST /v1/webhooks/stripe/{accountID}` — Go: `webhooks/endpoint_process_account_webhook.go` | Dashboard: `StripeSvc.handleEvent`
