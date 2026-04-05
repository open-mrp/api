# Verification: POST /v1/core/order-discounts/actions/find-by-code

## Result: Issues found and fixed

## What was compared

- **Validation:** Request body fields (`code` required, `buyer_account_id` optional vs Dashboard's required `customerID`)
- **Permission checks:** Both support internal actors (discounts:read) and customer actors (scoped to own account)
- **DB queries:** `FindOrderDiscountByCode` (by code + accountID) and `CheckOrderDiscountDuplicateUsage`
- **Error handling:** 404 "Order discount not found" when not found or duplicate usage detected
- **Response shape:** OrderDiscount resource with id, object, name, code, percentage, amount, discount_type, order_count, created_at, updated_at
- **Idempotency:** N/A — this is effectively a read/lookup endpoint (no mutations)
- **Side effects:** None in either implementation

## Issues found and fixed

### 1. Missing `seller_account_id` filter in duplicate usage check (FIXED)

**Dashboard code:**
```typescript
db.order.findFirst({
    where: {
        orderDiscountID: id,
        buyerAccountID: customerID,
        sellerAccountID: accountID,  // <-- present
        ownerAccountID: accountID,
    },
})
```

**Go code (before fix):**
```sql
WHERE order_discount_id = ? AND buyer_account_id = ? AND owner_account_id = ?
```

The Go `CheckOrderDiscountDuplicateUsage` SQL query was missing the `seller_account_id` filter that the Dashboard includes. Added `AND seller_account_id = sqlc.arg('seller_account_id')` to the query and updated the repository to pass `SellerAccountID: accountID`.

## Acceptable differences (not bugs)

1. **`buyer_account_id` optional in Go vs `customerID` required in Dashboard:** Go intentionally makes this optional for internal actors to support use cases like looking up a discount without a specific customer context. For customer actors, Go auto-sets the buyer account ID from identity. The duplicate check only runs when buyer_account_id is provided, which is always the case for customer actors.

2. **`sales_order_id` parameter (Go only):** Go adds an `exclude_order_id` capability to the duplicate check, allowing an existing order to be excluded (useful when editing an order that already uses the discount). This is an additive enhancement.

3. **Customer actor error messaging:** Dashboard throws a 403 "This discount code is not valid for this customer" if a customer provides a different `customerID`. Go silently overrides `buyer_account_id` to the customer's own account. Same security outcome, different UX.
