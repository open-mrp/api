-- name: ListSalesOrderStatusesForward :many
SELECT
    sales_order_status.id,
    sales_order_status.code,
    sales_order_status.name,
    sales_order_status.created_at,
    sales_order_status.updated_at
FROM sales_order_status
WHERE (
    sqlc.narg('search_query') IS NULL
    OR sales_order_status.name LIKE sqlc.narg('search_query')
)
AND (
    sqlc.narg('cursor_created_at') IS NULL
    OR sales_order_status.created_at < sqlc.narg('cursor_created_at')
    OR (sales_order_status.created_at = sqlc.narg('cursor_created_at') AND sales_order_status.id < sqlc.narg('cursor_id'))
)
ORDER BY sales_order_status.created_at DESC, sales_order_status.id DESC
LIMIT ?;

-- name: ListSalesOrderStatusesBackward :many
SELECT
    sales_order_status.id,
    sales_order_status.code,
    sales_order_status.name,
    sales_order_status.created_at,
    sales_order_status.updated_at
FROM sales_order_status
WHERE (
    sqlc.narg('search_query') IS NULL
    OR sales_order_status.name LIKE sqlc.narg('search_query')
)
AND (
    sales_order_status.created_at > sqlc.arg('cursor_created_at')
    OR (sales_order_status.created_at = sqlc.arg('cursor_created_at') AND sales_order_status.id > sqlc.arg('cursor_id'))
)
ORDER BY sales_order_status.created_at ASC, sales_order_status.id ASC
LIMIT ?;

-- name: GetSalesOrderStatusesByIDs :many
-- System-wide resource; no per-caller scoping.
SELECT
    sales_order_status.id,
    sales_order_status.code,
    sales_order_status.name,
    sales_order_status.created_at,
    sales_order_status.updated_at
FROM sales_order_status
WHERE sales_order_status.id IN (sqlc.slice('ids'));
