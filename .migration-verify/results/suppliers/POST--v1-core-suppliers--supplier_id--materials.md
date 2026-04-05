# Verification: POST /v1/core/suppliers/{supplier_id}/materials

## Result: Issues found and fixed

## What was compared

- **Validation rules**: Required fields (`material_id`, `supplier_part_number`), optional fields (`supplier_description`, `is_active`)
- **Permission checks**: Internal actor + `suppliers:create` permission + target account required
- **Duplicate check**: Same logic — checks `material_id` + `supplier_account_id` + `owner_account_id`
- **DB queries**: INSERT with same columns; re-fetches full record with JOINs on material, item, category, rates, quantities
- **Error handling**: Duplicate returns conflict error (Go uses 409 Conflict vs Dashboard's 400 Bad Request — acceptable improvement)
- **Side effects**: None beyond DB insert in either implementation
- **Response shape**: Matches — `SupplierMaterial` with nested `Material` > `Item` sub-resources
- **Idempotency**: Go properly implements idempotency keys with recovery points (improvement over Dashboard)

## Issues found and fixed

### 1. `is_active` default value mismatch

**Problem**: Dashboard schema defines `isActive: z.boolean().default(true)`, so omitting `is_active` creates an active supplier material. Go used `IsActive bool` which defaults to `false`, creating an inactive supplier material when the field is omitted.

**Fix**: Changed `IsActive` from `bool` to `*bool` in `CreateSupplierMaterialRequest`. When nil (not provided), the gateway service now defaults to `true` before passing to the gRPC layer.

**Files modified**:
- `services/api-gateway/endpoints/supplier-materials/endpoint_create_supplier_material.go` — changed `IsActive` to `*bool`
- `services/api-gateway/endpoints/supplier-materials/service.go` — added nil-check defaulting to `true`

## Acceptable differences

- **Request shape**: Dashboard sends full `SupplierMaterial` object (with supplier.id, item.materialID, etc.); Go accepts only the needed fields (`material_id`, `supplier_part_number`, `supplier_description`, `is_active`) with supplier ID from the path. This is a deliberate API improvement.
- **Duplicate error code**: Dashboard returns 400 Bad Request; Go returns 409 Conflict. The 409 is more semantically correct.
- **Supplier ID mismatch check**: Dashboard validates that path `supplierID` matches `data.supplier.id` in the body. Go doesn't need this check since supplier ID only comes from the path (not duplicated in body).
