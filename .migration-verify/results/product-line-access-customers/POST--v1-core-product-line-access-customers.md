# Verification: POST /v1/core/product-line-access/customers

## Status: PARITY CONFIRMED — No issues found

## What Was Compared

| Aspect | Dashboard | Go | Match? |
|--------|-----------|-----|--------|
| **Permission check** | `productLineAccess` / `create` | `PermissionDomainProductLineAccess` / `ActionCreate` | ✅ |
| **Actor type** | `checkIsInternalActor` | `CheckIsInternalActor` | ✅ |
| **Target account required** | `identity.targetAccountID` | `CheckTargetAccountSet` | ✅ |
| **Customer relation validation** | Queries `accountRelation` where `ownerAccountID` + `counterpartyID` + role=`customer` | `GetAccountRelationForCustomer` with same filters | ✅ |
| **Duplicate check** | `findFirst` on `accountRelationProductLine` by relation ID → 409 conflict | `ExistsByCustomerID` (service) + `CountAccountRelationProductLinesByRelationID` (repo) → 409 conflict | ✅ |
| **Insert logic** | `createMany` one row per product line with generated ID | Loop inserting one row per product line with `id.GenID` | ✅ |
| **Success status** | 201 Created | 201 Created | ✅ |
| **Idempotency** | None | Idempotency keys with recovery points | ✅ (improvement) |
| **Error: customer not found** | 404 "Customer relationship not found" | 404 via `db.MapSQLError` on no rows | ✅ |
| **Error: already exists** | 409 conflict with customer name/number in message | 409 conflict "Product line access for this customer already exists." | ✅ |

## Go Improvements Over Dashboard (Not Regressions)

1. **Product line validation**: Go validates each product line ID exists and belongs to the account before inserting. Dashboard does not — it would fail silently or hit a FK constraint.
2. **Idempotency**: Go implements full idempotency key handling with recovery points. Dashboard has no idempotency support.
3. **Re-fetch after create**: Go re-fetches the created record to return actual DB state (including `created_at`/`updated_at`). Dashboard returns the input data as-is.
4. **Transaction support**: Go wraps the duplicate check + insert in a single transaction. Dashboard does not use an explicit transaction.

## Expected Differences

- **Request shape**: Go accepts `{ customer_id, product_line_ids[] }` (IDs only). Dashboard accepts full objects `{ customer: { id, name, number }, productLines: [{ id, name }] }`. This is appropriate since Go looks up details by ID.
- **Route prefix**: `/v1/core/product-line-access/customers` vs `/v1/product-line-access/customers`.
- **Response shape**: Go returns a proper API resource with `object` field, nested `customer` sub-resource, `product_lines` list wrapper, and timestamps. Dashboard returns the input data structure.

## Files Reviewed

### Dashboard
- `dashboard/apps/api/src/services/customer-product-line-access.svc.ts`
- `dashboard/apps/api/src/repositories/customer-product-line-access.repo.ts`
- `dashboard/apps/api/src/controllers/customer-product-line-access.ctrl.ts`

### Go
- `services/api-gateway/endpoints/customer-product-line-access/endpoint_create_customer_product_line_access.go`
- `services/api-gateway/endpoints/customer-product-line-access/service.go`
- `services/api-gateway/endpoints/customer-product-line-access/presenter.go`
- `services/api-gateway/pkg/resource/customer_product_line_access_resource.go`
- `services/core-service/internal/infrastructure/grpc/grpc_customer_product_line_access_handler.go`
- `services/core-service/internal/service/customer_product_line_access_service.go`
- `services/core-service/internal/infrastructure/repository/customer_product_line_access_repository.go`
- `services/core-service/internal/infrastructure/queries/customer_product_line_access.sql`
- `services/core-service/internal/domain/customer_product_line_access_models.go`

## Conclusion

Full business-logic parity confirmed. The Go implementation faithfully reproduces all Dashboard behavior and adds several improvements (product line validation, idempotency, transactional safety, proper re-fetch). No fixes needed.
