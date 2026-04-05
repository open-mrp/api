# Verification: DELETE /v1/core/suppliers/{id}

## Result: Issue found and fixed

## What was compared

| Aspect | Dashboard | Go | Match? |
|--------|-----------|-----|--------|
| Permission check | `checkIsInternalActor` + `suppliers:update` | `CheckIsInternalActor` + `PermissionDomainSuppliers:ActionUpdate` | Yes |
| Target account required | Yes | Yes (`CheckTargetAccountSet`) | Yes |
| Supplier existence check | `find({ id, ownerAccountID })` → 404 | `Get()` → 404 | Yes |
| Delete account_user | `deleteMany({ accountID: id })` | `DELETE FROM account_user WHERE account_id = ?` | Yes |
| Delete account_address | `deleteMany({ accountID: id })` | `DELETE FROM account_address WHERE account_id = ?` | Yes |
| Delete account_relation | `deleteMany({ counterpartyID: id, ownerAccountID })` | `DELETE FROM account_relation WHERE owner_account_id = ? AND counterparty_account_id = ? AND account_relation_role_code = 'supplier'` | Yes (Go is more precise with role filter) |
| Transaction wrapping | `prisma.$transaction` | **Was missing** | Fixed |
| Response shape | Returns deleted supplier | Returns `SupplierDetail` resource | Yes |
| HTTP status | 200 | 200 | Yes |
| Side effects | None | None | Yes |
| Idempotency keys | Not used (DELETE) | Not used | Yes |

## Issue found and fixed

**Missing transaction wrapping in `DeleteSupplier`**: The Dashboard wraps all three cascading deletes (account_user, account_address, account_relation) in a Prisma transaction for atomicity. The Go `DeleteSupplier` service method was calling the repository directly without `withTx`, meaning a failure partway through could leave the database in an inconsistent state.

**Fix**: Wrapped the `Delete` call in `s.withTx()` in `supplier_service.go:334-343`, matching the pattern already used by `BulkDeleteSuppliers` and other mutation methods in the same file.

Note: `BulkDeleteSuppliers` was already correctly wrapped in `withTx`.

## No remaining concerns

The Go implementation now matches Dashboard behavior for this endpoint.
