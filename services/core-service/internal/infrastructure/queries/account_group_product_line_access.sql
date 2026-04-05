-- name: ListAccountGroupProductLineAccessForward :many
SELECT
    agpl.id,
    agpl.account_group_id,
    ag.name AS account_group_name,
    agpl.product_line_id,
    pl.name AS product_line_name,
    ag.created_at AS account_group_created_at,
    ag.updated_at AS account_group_updated_at
FROM account_group_product_line agpl
JOIN account_group ag ON ag.id = agpl.account_group_id
JOIN product_line pl ON pl.id = agpl.product_line_id
WHERE ag.owner_account_id = sqlc.arg('owner_account_id')
AND (
    sqlc.narg('search_query') IS NULL
    OR ag.name LIKE sqlc.narg('search_query')
)
AND (
    sqlc.narg('cursor_created_at') IS NULL
    OR ag.created_at < sqlc.narg('cursor_created_at')
    OR (ag.created_at = sqlc.narg('cursor_created_at') AND ag.id < sqlc.narg('cursor_id'))
)
ORDER BY ag.created_at DESC, ag.id DESC
LIMIT ?;

-- name: ListAccountGroupProductLineAccessBackward :many
SELECT
    agpl.id,
    agpl.account_group_id,
    ag.name AS account_group_name,
    agpl.product_line_id,
    pl.name AS product_line_name,
    ag.created_at AS account_group_created_at,
    ag.updated_at AS account_group_updated_at
FROM account_group_product_line agpl
JOIN account_group ag ON ag.id = agpl.account_group_id
JOIN product_line pl ON pl.id = agpl.product_line_id
WHERE ag.owner_account_id = sqlc.arg('owner_account_id')
AND (
    sqlc.narg('search_query') IS NULL
    OR ag.name LIKE sqlc.narg('search_query')
)
AND (
    ag.created_at > sqlc.arg('cursor_created_at')
    OR (ag.created_at = sqlc.arg('cursor_created_at') AND ag.id > sqlc.arg('cursor_id'))
)
ORDER BY ag.created_at ASC, ag.id ASC
LIMIT ?;

-- name: GetAccountGroupProductLineAccess :many
SELECT
    agpl.id,
    agpl.product_line_id,
    pl.name AS product_line_name,
    agpl.created_at,
    agpl.updated_at
FROM account_group_product_line agpl
JOIN product_line pl ON pl.id = agpl.product_line_id
WHERE agpl.account_group_id = sqlc.arg('account_group_id');

-- name: InsertAccountGroupProductLine :exec
INSERT INTO account_group_product_line (
    id,
    account_group_id,
    product_line_id,
    created_at,
    updated_at
) VALUES (
    sqlc.arg('id'),
    sqlc.arg('account_group_id'),
    sqlc.arg('product_line_id'),
    NOW(3),
    NOW(3)
);

-- name: DeleteAccountGroupProductLinesByAccountGroupID :execresult
DELETE FROM account_group_product_line
WHERE account_group_id = sqlc.arg('account_group_id');

-- name: CountAccountGroupProductLinesByAccountGroupID :one
SELECT COUNT(*) FROM account_group_product_line
WHERE account_group_id = sqlc.arg('account_group_id');

-- name: GetAccountGroupByIDAndAccount :one
SELECT id, name, created_at, updated_at
FROM account_group
WHERE id = sqlc.arg('id')
AND owner_account_id = sqlc.arg('owner_account_id');

-- name: ProductLineExistsByIDAndAccount :one
SELECT COUNT(*) FROM product_line
WHERE id = sqlc.arg('id')
AND account_id = sqlc.arg('account_id');
