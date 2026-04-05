# DELETE /v1/core/suppliers/{supplier_id}/materials/{id}

## Result: Parity Confirmed

No issues found. The Go implementation matches the Dashboard behavior.

## What Was Compared

| Aspect | Dashboard | Go | Match |
|--------|-----------|-----|-------|
| **Permission: actor type** | `checkIsInternalActor` | `CheckIsInternalActor()` | Yes |
| **Permission: domain/action** | `suppliers` / `update` | `PermissionDomainSuppliers` / `ActionUpdate` | Yes |
| **Account scoping** | `targetAccountID` | `TargetAccountID` | Yes |
| **Lookup** | Prisma `findFirst` by `supplierAccountID`, `material.itemID`, `ownerAccountID` | SQL join `supplier_material` → `material` on `supplier_account_id`, `item_id`, `owner_account_id` | Yes |
| **Not-found error** | 404 "Supplier material not found." | 404 "Supplier material not found." | Yes |
| **Delete** | By internal `id` | By internal `id` + `owner_account_id` | Yes (Go is stricter, which is fine) |
| **Response** | HTTP 200, deleted record | HTTP 200, deleted record | Yes |
| **Idempotency** | None (DELETE) | None (DELETE) | Yes |
| **Side effects** | None | None | Yes |

## Notes

- The Go delete query includes an additional `owner_account_id` filter beyond what the Dashboard uses (which deletes by `id` alone). This is strictly safer and does not change behavior since the record was already scoped by account in the lookup.
- Response shape includes the full supplier material with nested material/item data, consistent with the adapter mapping in the Dashboard.
