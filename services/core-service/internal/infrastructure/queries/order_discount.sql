-- name: GetOrderDiscount :one
SELECT * FROM order_discount
WHERE id = sqlc.arg('id')
AND account_id = sqlc.arg('account_id');

-- name: InsertOrderDiscount :exec
INSERT INTO order_discount (id, name, code, percentage, value, discount_type_code, percent_off, amount_off_amount,
    amount_off_currency, is_active, account_id, created_at, updated_at)
VALUES (sqlc.arg('id'), sqlc.arg('name'), sqlc.arg('code'), sqlc.arg('percentage'), sqlc.arg('value'),
    sqlc.arg('discount_type_code'), sqlc.narg('percent_off'), sqlc.narg('amount_off_amount'),
    sqlc.narg('amount_off_currency'), sqlc.arg('is_active'), sqlc.arg('account_id'), NOW(), NOW());

-- name: UpdateOrderDiscount :exec
UPDATE order_discount
SET
    name = sqlc.arg('name'),
    code = sqlc.arg('code'),
    percentage = sqlc.arg('percentage'),
    value = sqlc.arg('value'),
    discount_type_code = sqlc.arg('discount_type_code'),
    percent_off = sqlc.narg('percent_off'),
    amount_off_amount = sqlc.narg('amount_off_amount'),
    amount_off_currency = sqlc.narg('amount_off_currency'),
    is_active = sqlc.arg('is_active'),
    updated_at = NOW()
WHERE id = sqlc.arg('id')
AND account_id = sqlc.arg('account_id');

-- name: DeleteOrderDiscount :execresult
DELETE FROM order_discount
WHERE id = sqlc.arg('id')
AND account_id = sqlc.arg('account_id');

-- name: GetOrderDiscountsByIDs :many
-- Order discounts matching the given IDs that belong to the caller's account.
SELECT * FROM order_discount
WHERE id IN (sqlc.slice('ids'))
AND account_id = sqlc.arg('account_id');

-- name: CountOrderDiscountsByCode :one
SELECT COUNT(*) AS count
FROM order_discount
WHERE account_id = sqlc.arg('account_id')
AND code = sqlc.arg('code')
AND (sqlc.narg('exclude_id') IS NULL OR id != sqlc.narg('exclude_id'));

-- name: FindOrderDiscountByCode :one
SELECT * FROM order_discount
WHERE code = sqlc.arg('code')
AND account_id = sqlc.arg('account_id');

-- name: AdjustOrderDiscountRedemptions :exec
UPDATE order_discount
SET times_redeemed = GREATEST(CAST(times_redeemed AS SIGNED) + sqlc.arg('delta'), 0)
WHERE id = sqlc.arg('id');

-- name: CheckOrderDiscountDuplicateUsage :one
SELECT COUNT(*) AS count
FROM sales_order
WHERE order_discount_id = sqlc.arg('order_discount_id')
AND buyer_account_id = sqlc.arg('buyer_account_id')
AND seller_account_id = sqlc.arg('seller_account_id')
AND owner_account_id = sqlc.arg('owner_account_id')
AND (sqlc.narg('exclude_order_id') IS NULL OR id != sqlc.narg('exclude_order_id'));
