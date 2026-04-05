-- name: ListOrderDiscountsForward :many
SELECT
    od.id,
    od.name,
    od.code,
    od.percentage,
    od.value,
    od.discount_type_code,
    od.account_id,
    (SELECT COUNT(*) FROM sales_order so WHERE so.order_discount_id = od.id) AS order_count,
    od.created_at,
    od.updated_at
FROM order_discount od
WHERE od.account_id = sqlc.arg('account_id')
AND (
    sqlc.narg('search_query') IS NULL
    OR od.name LIKE sqlc.narg('search_query')
    OR od.code LIKE sqlc.narg('search_query')
)
AND (
    sqlc.narg('cursor_created_at') IS NULL
    OR od.created_at < sqlc.narg('cursor_created_at')
    OR (od.created_at = sqlc.narg('cursor_created_at') AND od.id < sqlc.narg('cursor_id'))
)
ORDER BY od.created_at DESC, od.id DESC
LIMIT ?;

-- name: ListOrderDiscountsBackward :many
SELECT
    od.id,
    od.name,
    od.code,
    od.percentage,
    od.value,
    od.discount_type_code,
    od.account_id,
    (SELECT COUNT(*) FROM sales_order so WHERE so.order_discount_id = od.id) AS order_count,
    od.created_at,
    od.updated_at
FROM order_discount od
WHERE od.account_id = sqlc.arg('account_id')
AND (
    sqlc.narg('search_query') IS NULL
    OR od.name LIKE sqlc.narg('search_query')
    OR od.code LIKE sqlc.narg('search_query')
)
AND (
    od.created_at > sqlc.arg('cursor_created_at')
    OR (od.created_at = sqlc.arg('cursor_created_at') AND od.id > sqlc.arg('cursor_id'))
)
ORDER BY od.created_at ASC, od.id ASC
LIMIT ?;

-- name: GetOrderDiscount :one
SELECT
    od.id,
    od.name,
    od.code,
    od.percentage,
    od.value,
    od.discount_type_code,
    od.account_id,
    (SELECT COUNT(*) FROM sales_order so WHERE so.order_discount_id = od.id) AS order_count,
    od.created_at,
    od.updated_at
FROM order_discount od
WHERE od.id = sqlc.arg('id')
AND od.account_id = sqlc.arg('account_id');

-- name: InsertOrderDiscount :exec
INSERT INTO order_discount (id, name, code, percentage, value, discount_type_code, account_id, created_at, updated_at)
VALUES (sqlc.arg('id'), sqlc.arg('name'), sqlc.arg('code'), sqlc.arg('percentage'), sqlc.arg('value'), sqlc.arg('discount_type_code'), sqlc.arg('account_id'), NOW(), NOW());

-- name: UpdateOrderDiscount :execresult
UPDATE order_discount
SET
    name = COALESCE(sqlc.narg('name'), name),
    code = COALESCE(sqlc.narg('code'), code),
    percentage = COALESCE(sqlc.narg('percentage'), percentage),
    value = COALESCE(sqlc.narg('value'), value),
    discount_type_code = COALESCE(sqlc.narg('discount_type_code'), discount_type_code),
    updated_at = NOW()
WHERE id = sqlc.arg('id')
AND account_id = sqlc.arg('account_id');

-- name: DeleteOrderDiscount :execresult
DELETE FROM order_discount
WHERE id = sqlc.arg('id')
AND account_id = sqlc.arg('account_id');

-- name: CountOrderDiscountsByCode :one
SELECT COUNT(*) AS count
FROM order_discount
WHERE account_id = sqlc.arg('account_id')
AND code = sqlc.arg('code')
AND (sqlc.narg('exclude_id') IS NULL OR id != sqlc.narg('exclude_id'));

-- name: FindOrderDiscountByCode :one
SELECT
    od.id,
    od.name,
    od.code,
    od.percentage,
    od.value,
    od.discount_type_code,
    od.account_id,
    (SELECT COUNT(*) FROM sales_order so WHERE so.order_discount_id = od.id) AS order_count,
    od.created_at,
    od.updated_at
FROM order_discount od
WHERE od.code = sqlc.arg('code')
AND od.account_id = sqlc.arg('account_id');

-- name: CheckOrderDiscountDuplicateUsage :one
SELECT COUNT(*) AS count
FROM sales_order
WHERE order_discount_id = sqlc.arg('order_discount_id')
AND buyer_account_id = sqlc.arg('buyer_account_id')
AND seller_account_id = sqlc.arg('seller_account_id')
AND owner_account_id = sqlc.arg('owner_account_id')
AND (sqlc.narg('exclude_order_id') IS NULL OR id != sqlc.narg('exclude_order_id'));
