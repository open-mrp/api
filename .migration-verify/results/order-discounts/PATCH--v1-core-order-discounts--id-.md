# PATCH /v1/core/order-discounts/{id} — Verification Result

**Status:** PARITY CONFIRMED — No issues found.

## What Was Compared

| Aspect | Dashboard | Go | Match |
|--------|-----------|-----|-------|
| Permission: internal actor | `checkIsInternalActor` | `identity.CheckIsInternalActor()` | ✅ |
| Permission: domain/action | `discounts` / `update` | `PermissionDomainDiscounts` / `ActionUpdate` | ✅ |
| Account scoping | `identity.targetAccountID` | `identity.TargetAccountID` | ✅ |
| Updatable: name | ✅ | ✅ | ✅ |
| Updatable: code | ✅ | ✅ | ✅ |
| Updatable: percentage | ✅ | ✅ | ✅ |
| Updatable: amount | ✅ | ✅ | ✅ |
| Updatable: type/discount_type | `type` → `typeCode` | `discount_type` → `discount_type_code` | ✅ |
| Partial update (COALESCE / Prisma partial) | Prisma partial update | SQL COALESCE on nullable args | ✅ |
| DB WHERE clause | `id + accountID` | `id + account_id` | ✅ |
| Not-found handling | Prisma throws | `rowsAffected == 0` → 404 | ✅ |
| Response shape | OrderDiscount fields | OrderDiscount resource + `object` field | ✅ |
| Side effects | None | None | ✅ |
| Idempotency | None | Full idempotency key support | ✅ (improvement) |

## Notes

- **Code uniqueness check:** The Go implementation adds an `ExistsByCode` check when updating the `code` field, which the Dashboard lacks. This is strictly additive — it prevents creating duplicate codes on update, which is arguably a bug fix. The Dashboard's `create` method validates code uniqueness but `update` does not.
- **Idempotency:** Go correctly uses idempotency keys per project conventions for PATCH endpoints. The Dashboard did not have this.
- **Field naming:** Dashboard uses `type` in the request body mapped to `typeCode` DB column; Go uses `discount_type` mapped to `discount_type_code`. Both map to the same DB column.

## No Fixes Required

The Go implementation faithfully reproduces all Dashboard business logic and adds appropriate improvements (idempotency, code uniqueness validation) per project conventions.
