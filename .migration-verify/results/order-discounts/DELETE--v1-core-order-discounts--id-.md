# DELETE /v1/core/order-discounts/{id}

## Result: Issues found and fixed

## What was compared

- **Validation rules**: Path parameter extraction (order discount ID)
- **Permission checks**: Internal actor check + `discounts:delete` permission + target account set
- **DB queries**: DELETE by id + account_id for multi-tenancy
- **Error handling**: 404 when not found, SQL error mapping
- **Side effects**: None in either implementation
- **Response shape**: Deleted order discount resource vs empty response
- **Idempotency**: DELETE is idempotent by design — no idempotency keys needed (correct)

## Issues found and fixed

### 1. Response shape mismatch (fixed)

**Dashboard**: Returns the deleted `OrderDiscount` object with HTTP 200.
**Go (before)**: Returned `EmptyResource` with HTTP 204 No Content.

This was inconsistent with both the Dashboard behavior and the established Go API pattern (e.g., `DELETE /v1/core/batches/{id}` returns the deleted batch with HTTP 200).

**Fix**: Changed the Go implementation across all layers to return the deleted `OrderDiscount` resource with HTTP 200:

- **Proto** (`proto/core_sales.proto`): Changed `rpc DeleteOrderDiscount` return type from `google.protobuf.Empty` to `DeleteOrderDiscountResponse` containing `OrderDiscountInfo`.
- **gRPC handler** (`grpc_sales_service_handler.go`): Updated to return `*pb.DeleteOrderDiscountResponse` with the deleted discount.
- **Domain interfaces** (`services.go`, `repositories.go`): Changed `DeleteOrderDiscount` and `Delete` to return `(*OrderDiscount, *apierror.APIError)`.
- **Service** (`order_discount_service.go`): Updated to capture and return the deleted discount from the repository.
- **Repository** (`order_discount_repository.go`): Now fetches the order discount before deleting (matching Prisma's delete behavior that returns the deleted record).
- **API gateway endpoint** (`endpoint_delete_order_discount.go`): Changed to return `*apiresource.OrderDiscount` with `http.StatusOK`.
- **API gateway service** (`service.go`): Updated to use `DeleteOrderDiscountResponse` and present the result via `OrderDiscountPresenter`.

## Parity confirmed for

- Permission model: `checkIsInternalActor` + `discounts:delete` — matches exactly
- Account scoping: Both filter by `account_id` — matches exactly
- SQL query: `DELETE FROM order_discount WHERE id = ? AND account_id = ?` — matches Prisma's composite delete
- No side effects in either implementation
