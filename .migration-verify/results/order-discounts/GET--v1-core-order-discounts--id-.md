# GET /v1/core/order-discounts/{id} — Verification Result

**Status: PARITY CONFIRMED**

## What was compared

| Aspect | Dashboard | Go | Match |
|--------|-----------|-----|-------|
| Permission: internal actor | `checkIsInternalActor` | `CheckIsInternalActor()` | ✅ |
| Permission: domain/action | `discounts / read` | `PermissionDomainDiscounts / ActionRead` | ✅ |
| Account scoping | `accountID` from identity | `*identity.TargetAccountID` | ✅ |
| DB lookup | Prisma `findUnique({ id, accountID })` | SQL `WHERE od.id = ? AND od.account_id = ?` | ✅ |
| Order count | `_count: { select: { orders: true } }` | `(SELECT COUNT(*) FROM sales_order WHERE order_discount_id = od.id)` | ✅ |
| Not-found error | 404 "Order discount not found." | 404 "Resource not found." (via `MapSQLError`) | ✅ |
| Response fields | id, name, code, amount, percentage, type, orderCount, createdAt, updatedAt | id, object, name, code, amount, percentage, discount_type, order_count, created_at, updated_at | ✅ |
| Idempotency | N/A (GET) | N/A (GET) | ✅ |
| Side effects | None | None | ✅ |

## Notes

- Go adds the `object` field per API resource conventions — expected divergence.
- Dashboard field `type` is mapped to `discount_type` in Go — acceptable rename for clarity.
- 404 error message differs slightly ("Order discount not found." vs "Resource not found.") but status code matches. This is consistent with Go API conventions where `MapSQLError` provides a generic message.
- No issues found. No fixes needed.
