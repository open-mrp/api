# Verification: PUT /v1/core/receiving-orders/{receivingOrderId}/lines/{id}/actions/void

## Result: PARITY CONFIRMED

No issues found. The Go implementation faithfully reproduces the Dashboard business logic.

## What Was Compared

| Aspect | Dashboard | Go | Match |
|--------|-----------|-----|-------|
| Permission: internal actor | checkIsInternalActor | CheckIsInternalActor | Yes |
| Permission: domain/action | receivingOrders:update | ReceivingOrders:Update | Yes |
| Validation: line in order | checkIsInReceivingOrder | IsLineInReceivingOrder | Yes |
| Validation: account ownership | WHERE ownerAccountID | IsInAccount + WHERE account_id | Yes (Go is stricter) |
| DB: quantity zeroed | quantity.measure = 0 | quantity.value = '0' | Yes |
| DB: stocked_at cleared | stockedAt = null | stocked_at = NULL | Yes |
| Transaction | Prisma $transaction | withTx | Yes |
| Response status | 200 OK | 200 OK | Yes |
| Response shape | ReceivingOrderLine object | ReceivingOrderLine resource | Yes |
| Idempotency keys | None (PUT) | None (PUT) | Yes |
| Inventory side effects | None | None | Yes |

## Notes

- The Go implementation adds an extra validation step (`IsInAccount`) to verify the receiving order belongs to the target account before checking the line, which is slightly more thorough than the Dashboard's approach of only checking line-to-order membership and relying on the WHERE clause for account filtering. This is a strict improvement.
- No idempotency key handling is needed since PUT endpoints are idempotent by design.
- No inventory mutations or other side effects occur — this is a simple void (set quantity to 0, clear stocked_at).
