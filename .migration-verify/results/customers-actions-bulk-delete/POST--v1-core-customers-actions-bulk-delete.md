# Migration Verification: POST /v1/core/customers/actions/bulk-delete

## Status: PARITY CONFIRMED

No issues found. The Go implementation correctly matches the Dashboard behavior.

## What Was Compared

| Aspect | Dashboard | Go | Match |
|--------|-----------|-----|-------|
| Permission check | Internal actor + `customers:delete` | Internal actor + `customers:delete` | ✅ |
| Account scoping | `ownerAccountID` from identity | `OwnerAccountID` from `TargetAccountID` | ✅ |
| DB operation | DELETE `accountRelation` by counterpartyID + ownerAccountID | DELETE `account_relation` by counterparty + owner + role_code='customer' | ✅ (stricter) |
| What's NOT deleted | Account records, users, addresses (commented out) | Same — only relations deleted | ✅ |
| Response shape | Empty `{}` with 200 OK | `EmptyResource` with 200 OK | ✅ |
| Side effects | None | None | ✅ |
| Idempotency | Not used | Not used (naturally idempotent DELETE WHERE) | ✅ |

## Notes

- The Go SQL query includes an additional `account_relation_role_code = 'customer'` filter that the Dashboard does not have. This is a safety improvement — it prevents accidentally deleting non-customer relationships if the same counterparty account has multiple relation types.
- Idempotency key tracking is intentionally omitted. The operation is naturally idempotent (re-running DELETE WHERE on already-deleted rows is a no-op). This is consistent with all other bulk-delete POST endpoints in the Go codebase.
- The Dashboard's `deleteMany` had commented-out lines for deleting `accountUser` and `accountAddress` records, confirming the intentional decision to only remove the relationship.

## Files Reviewed

### Dashboard
- `dashboard/apps/api/src/services/customer.svc.ts` — `deleteMany()` method
- `dashboard/apps/api/src/repositories/customer.repo.ts` — `deleteMany()` method (lines 1192-1212)
- `dashboard/apps/api/src/controllers/customer.ctrl.ts` — route definition

### Go
- `api/services/api-gateway/endpoints/customers/endpoint_bulk_delete_customers.go`
- `api/services/api-gateway/endpoints/customers/service.go` — gRPC call
- `api/services/core-service/internal/infrastructure/grpc/grpc_customer_handler.go` — handler (lines 338-351)
- `api/services/core-service/internal/service/customer_service.go` — `BulkDeleteCustomers()` (lines 428-451)
- `api/services/core-service/internal/infrastructure/repository/customer_repository.go` — `BulkDelete()` (lines 626-639)
- `api/services/core-service/internal/infrastructure/queries/customer.sql` — `BulkDeleteCustomerRelations` query (lines 478-482)
