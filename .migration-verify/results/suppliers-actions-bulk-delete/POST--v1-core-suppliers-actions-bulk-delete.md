# Migration Verification: POST /v1/core/suppliers/actions/bulk-delete

## Result: Issues found and fixed

## What was compared

| Aspect | Dashboard | Go | Parity |
|--------|-----------|-----|--------|
| **Request shape** | `{ supplierIDs: string[] }` | `{ supplier_ids: []string }` | OK |
| **Response shape** | Empty object `{}`, 200 | `EmptyResource`, 200 | OK |
| **Actor check** | `checkIsInternalActor` | `CheckIsInternalActor` | OK |
| **Permission check** | `suppliers:delete` | `PermissionDomainSuppliers:ActionDelete` | OK |
| **Target account** | `identity.targetAccountID` | `CheckTargetAccountSet` + `TargetAccountID` | OK |
| **Deletion order** | account_users → account_addresses → account_relations | Same order | OK |
| **Account scoping** | `counterpartyID IN ids, ownerAccountID` | `counterparty_account_id IN ids AND owner_account_id` | OK |
| **Transaction** | `prisma.$transaction()` wraps all deletes | Was missing, now fixed with `withTx` | Fixed |
| **Existence check** | None | None | OK |
| **Idempotency** | None (idempotent by nature) | None | OK |
| **Side effects** | None | None | OK |

## Issues found and fixed

### 1. Missing transaction wrapping (fixed)

The Dashboard wraps all three DELETE operations in a `prisma.$transaction()` to ensure atomicity. The Go implementation was calling `BulkDelete` directly without a transaction, meaning a failure partway through could leave orphaned records.

**Fix:** Wrapped the `BulkDelete` repository call in `s.withTx()` in `supplier_service.go`.

## Minor differences (acceptable)

- **Role filter on relation delete:** Go SQL adds `AND account_relation_role_code = 'supplier'` to the account_relation DELETE. Dashboard doesn't filter by role. This is a safety improvement — since we're deleting supplier relations specifically, the extra filter prevents accidentally deleting non-supplier relations if any exist with the same counterparty IDs. This is strictly safer behavior and doesn't change functional parity.
