# POST /v1/core/products — Verification Result

**Status: Issues found and fixed**

## What was compared

- **Validation rules:** Required fields (SKU, category_id, product_type_code) match. Go validates via struct tags.
- **Permission checks:** Both check `isInternalActor` and `hasPermission(items, create)`. Go also checks target account is set. Match.
- **SKU uniqueness:** Both check SKU uniqueness within the account before creation. Match.
- **Rate initialization:** Both create 3 rates (unit_value, unit_cost, burn_rate) with base unit from category. Go supports `unit_price` for the unit_value rate; Dashboard defaults all to zero. Match.
- **Item creation:** Both create an item record with sku, description, notes, category, rates. Match.
- **Product creation:** Both create a product record linked to item, product_type, optional product_line. Match.
- **Idempotency:** Go correctly uses idempotency keys with recovery points. Dashboard does not have explicit idempotency. Go is correct.
- **Error handling:** SKU conflict returns conflict error in both. Permission errors handled similarly. Match.
- **Response shape:** Go returns ProductFull with nested Item, ProductType, ProductLine sub-resources. Dashboard returns flat Product with nested fields. Shapes differ by design (Go follows api-resource-conventions.md). Acceptable.

## Issues found and fixed

### 1. Missing inventory initialization (FIXED)

**Dashboard behavior:** After creating a product, the Dashboard creates:
1. An inventory log with zero quantity (blank measure in base unit)
2. An inventory change log with zero quantity and `actionType: user_action`

**Go behavior (before fix):** The product service did NOT create these inventory tracking records.

**Fix applied:** Added inventory log and change log creation to `product_service.go` CreateProduct, inside the transaction, after the product record is inserted. This matches the exact pattern already used by `material_service.go` (line 282-302) and `part_service.go`.

## Remaining concerns

None. The inventory initialization was the only missing business logic. All other aspects have parity.
