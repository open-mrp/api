-- name: ListProductLinesForward :many
SELECT
    pl.id,
    pl.name,
    pl.description,
    pl.notes,
    pl.is_commission_exempt,
    pl.is_freight_exempt,
    pl.unit_group_id,
    pl.account_id,
    pl.created_at,
    pl.updated_at
FROM product_line pl
WHERE (pl.account_id = sqlc.arg('account_id') OR pl.account_id IS NULL)
AND (
    sqlc.narg('search_query') IS NULL
    OR pl.name LIKE sqlc.narg('search_query')
)
AND (
    sqlc.narg('cursor_created_at') IS NULL
    OR pl.created_at < sqlc.narg('cursor_created_at')
    OR (pl.created_at = sqlc.narg('cursor_created_at') AND pl.id < sqlc.narg('cursor_id'))
)
ORDER BY pl.created_at DESC, pl.id DESC
LIMIT ?;

-- name: ListProductLinesBackward :many
SELECT
    pl.id,
    pl.name,
    pl.description,
    pl.notes,
    pl.is_commission_exempt,
    pl.is_freight_exempt,
    pl.unit_group_id,
    pl.account_id,
    pl.created_at,
    pl.updated_at
FROM product_line pl
WHERE (pl.account_id = sqlc.arg('account_id') OR pl.account_id IS NULL)
AND (
    sqlc.narg('search_query') IS NULL
    OR pl.name LIKE sqlc.narg('search_query')
)
AND (
    pl.created_at > sqlc.arg('cursor_created_at')
    OR (pl.created_at = sqlc.arg('cursor_created_at') AND pl.id > sqlc.arg('cursor_id'))
)
ORDER BY pl.created_at ASC, pl.id ASC
LIMIT ?;

-- name: GetProductLinesByIDs :many
SELECT
    pl.id,
    pl.name,
    pl.description,
    pl.notes,
    pl.is_commission_exempt,
    pl.is_freight_exempt,
    pl.unit_group_id,
    pl.account_id,
    pl.created_at,
    pl.updated_at
FROM product_line pl
WHERE pl.id IN (sqlc.slice('ids'));

-- name: GetProductLinesByIDsScoped :many
SELECT
    pl.id,
    pl.name,
    pl.description,
    pl.notes,
    pl.is_commission_exempt,
    pl.is_freight_exempt,
    pl.unit_group_id,
    pl.account_id,
    pl.created_at,
    pl.updated_at
FROM product_line pl
WHERE pl.id IN (sqlc.slice('ids'))
AND (pl.account_id = sqlc.arg('account_id') OR pl.account_id IS NULL);

-- name: GetProductLine :one
SELECT
    pl.id,
    pl.name,
    pl.description,
    pl.notes,
    pl.is_commission_exempt,
    pl.is_freight_exempt,
    pl.unit_group_id,
    pl.account_id,
    pl.created_at,
    pl.updated_at
FROM product_line pl
WHERE pl.id = sqlc.arg('id')
AND (pl.account_id = sqlc.arg('account_id') OR pl.account_id IS NULL);

-- name: InsertProductLine :exec
INSERT INTO product_line (
    id,
    name,
    is_commission_exempt,
    is_freight_exempt,
    unit_group_id,
    account_id,
    created_at,
    updated_at
) VALUES (
    sqlc.arg('id'),
    sqlc.arg('name'),
    sqlc.arg('is_commission_exempt'),
    sqlc.arg('is_freight_exempt'),
    sqlc.arg('unit_group_id'),
    sqlc.arg('account_id'),
    NOW(3),
    NOW(3)
);

-- name: UpdateProductLine :execresult
UPDATE product_line SET
    name = COALESCE(sqlc.narg('name'), name),
    is_commission_exempt = COALESCE(sqlc.narg('is_commission_exempt'), is_commission_exempt),
    is_freight_exempt = COALESCE(sqlc.narg('is_freight_exempt'), is_freight_exempt),
    unit_group_id = COALESCE(sqlc.narg('unit_group_id'), unit_group_id),
    updated_at = NOW(3)
WHERE id = sqlc.arg('id')
AND account_id = sqlc.arg('account_id');

-- name: DeleteProductLine :execresult
DELETE FROM product_line
WHERE id = sqlc.arg('id')
AND account_id = sqlc.arg('account_id');

-- name: CountProductLinesByName :one
SELECT COUNT(*) FROM product_line
WHERE name = ? AND (account_id = ? OR account_id IS NULL)
AND (sqlc.narg('exclude_id') IS NULL OR id != sqlc.narg('exclude_id'));

-- name: GetUnitGroupForProductLine :one
SELECT
    ug.id,
    ug.name,
    ug.base_unit_id,
    ug.unit_type_code,
    ug.created_at,
    ug.updated_at
FROM unit_group ug
WHERE ug.id = sqlc.arg('id');
