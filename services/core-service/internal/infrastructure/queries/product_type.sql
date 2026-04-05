-- name: ListProductTypesForward :many
SELECT
    product_type.id,
    product_type.name,
    product_type.code,
    product_type.created_at,
    product_type.updated_at
FROM product_type
WHERE (
    sqlc.narg('search_query') IS NULL
    OR product_type.name LIKE sqlc.narg('search_query')
)
AND (
    sqlc.narg('cursor_created_at') IS NULL
    OR product_type.created_at < sqlc.narg('cursor_created_at')
    OR (product_type.created_at = sqlc.narg('cursor_created_at') AND product_type.id < sqlc.narg('cursor_id'))
)
ORDER BY product_type.created_at DESC, product_type.id DESC
LIMIT ?;

-- name: ListProductTypesBackward :many
SELECT
    product_type.id,
    product_type.name,
    product_type.code,
    product_type.created_at,
    product_type.updated_at
FROM product_type
WHERE (
    sqlc.narg('search_query') IS NULL
    OR product_type.name LIKE sqlc.narg('search_query')
)
AND (
    product_type.created_at > sqlc.arg('cursor_created_at')
    OR (product_type.created_at = sqlc.arg('cursor_created_at') AND product_type.id > sqlc.arg('cursor_id'))
)
ORDER BY product_type.created_at ASC, product_type.id ASC
LIMIT ?;

-- name: GetProductType :one
SELECT
    product_type.id,
    product_type.name,
    product_type.code,
    product_type.created_at,
    product_type.updated_at
FROM product_type
WHERE product_type.id = sqlc.arg('id') OR product_type.code = sqlc.arg('code');

-- name: InsertProductType :exec
INSERT INTO product_type (
    id,
    name,
    code,
    created_at,
    updated_at
) VALUES (
    sqlc.arg('id'),
    sqlc.arg('name'),
    sqlc.arg('code'),
    NOW(3),
    NOW(3)
);

-- name: UpdateProductType :execresult
UPDATE product_type SET
    name = COALESCE(sqlc.narg('name'), name),
    code = COALESCE(sqlc.narg('code'), code),
    updated_at = NOW(3)
WHERE id = sqlc.arg('id');

-- name: DeleteProductType :execresult
DELETE FROM product_type
WHERE id = sqlc.arg('id');

-- name: CountProductTypesByName :one
SELECT COUNT(*) FROM product_type
WHERE name = ?
AND (sqlc.narg('exclude_id') IS NULL OR id != sqlc.narg('exclude_id'));

-- name: CountProductTypesByCode :one
SELECT COUNT(*) FROM product_type
WHERE code = ?
AND (sqlc.narg('exclude_id') IS NULL OR id != sqlc.narg('exclude_id'));

-- name: ProductTypeExistsByID :one
SELECT COUNT(*) FROM product_type
WHERE id = ?;
