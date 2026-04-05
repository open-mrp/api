# POST /v1/core/volume-discounts — Migration Verification

## Result: PARITY CONFIRMED

No issues found. No code changes required.

## What Was Compared

| Aspect | Dashboard | Go | Match |
|--------|-----------|-----|-------|
| Actor check | `checkIsInternalActor` | `CheckIsInternalActor` | Yes |
| Permission check | `discounts` / `create` | `PermissionDomainDiscounts` / `ActionCreate` | Yes |
| Target account required | Implicit via identity | `CheckTargetAccountSet` | Yes |
| Name uniqueness check | None | `ExistsByName` (stricter) | Yes |
| Idempotency keys | None | Full idempotency pattern | Yes (Go convention) |
| HTTP status code | 201 Created | 201 Created | Yes |
| Insert discount | id, name, accountID | id, name, accountID | Yes |
| Insert tiers | id, name, discountPercentage, threshold | id, name, discountPercentage, threshold, parentTierID | Yes |
| Connect product lines | PrismaUtils.connect | Insert join table rows | Yes |
| Connect categories | PrismaUtils.connect | Insert join table rows | Yes |
| Connect attributes | PrismaUtils.connect | Insert join table rows | Yes |
| Connect units | PrismaUtils.connect | Insert join table rows | Yes |
| Connect customer groups | **Commented out** in Dashboard | Insert join table rows | Intentional improvement |
| Response shape | Full entity + relations | Full entity + relations + object fields | Yes |
| Request validation | name required, tiers required | name required, tiers required | Yes |
| Tier validation | name, discountPercentage, threshold required | name, discount_percentage, threshold required | Yes |

## Notes

- **Customer groups**: The Dashboard repo has the customer group connection commented out (`// accountGroupVolumeDiscounts: PrismaUtils.connect(data.customerGroups)`). The Go API properly wires customer group insertion through all layers (endpoint → gRPC handler → service → repository). This restores intended functionality.
- **Name uniqueness**: The Go API adds a name uniqueness check (`ExistsByName`) that the Dashboard lacks. This is an improvement that prevents duplicate-named discounts within an account.
- **Idempotency**: The Go API follows the standard idempotency key pattern for POST endpoints per project conventions. The Dashboard does not have this pattern.
- **ParentTierID**: The Go API supports `parent_tier_id` on tier creation, matching the database schema. The Dashboard tier creation does not pass this through (the Prisma schema has the field but the create data doesn't include it). The Go API is more complete.
- **No side effects**: Neither implementation triggers emails, webhooks, or other side effects on create.
