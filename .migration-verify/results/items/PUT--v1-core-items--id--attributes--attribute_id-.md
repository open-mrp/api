# PUT /v1/core/items/{id}/attributes/{attribute_id}

## Result: PARITY CONFIRMED — No issues found

## What was compared

- **Validation:** Both endpoints take `itemID` and `attributeID` from path parameters only. No request body.
- **Permissions:** Both check internal actor + `items:update` permission. Go also explicitly checks target account is set.
- **DB logic:** Dashboard uses Prisma `connect` (many-to-many). Go uses `INSERT INTO _item_attributes ON DUPLICATE KEY UPDATE A = A`. Both handle duplicate associations as a no-op.
- **Account scoping:** Dashboard checks `accountID` + `deletedAt: null` in the Prisma `update` where clause. Go checks `account_id` + `deleted_at IS NULL` in the `GetItem` query after the insert. Since both operations are in the same transaction, behavior is identical — if the item doesn't belong to the account, the operation fails.
- **Response shape:** Both return the full Item object with attributes populated.
- **Error handling:** Both surface not-found errors if the item doesn't exist or doesn't belong to the account.
- **Side effects:** Neither implementation has side effects (no emails, webhooks, messages).
- **Idempotency:** Go uses idempotency keys (extra safety). Dashboard does not (PUT is naturally idempotent). This is additive, not a divergence.

## Notes

- The Go implementation uses idempotency keys for this PUT endpoint. Per conventions, PUT endpoints should be idempotent by default without keys. This is extra safety and doesn't affect parity, but could be simplified in a future cleanup pass.
- Account ownership validation in Go happens at the `Get` step rather than the `AddAttribute` step, but since both are in the same transaction, the rollback ensures correctness.
