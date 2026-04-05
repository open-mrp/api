-- name: ListCustomerProductLineAccessForward :many
SELECT
    arpl.id,
    ar.counterparty_account_id AS customer_id,
    a.name AS customer_name,
    ar.external_number AS customer_number,
    arpl.product_line_id,
    pl.name AS product_line_name,
    arpl.created_at,
    arpl.updated_at
FROM account_relation_product_line arpl
JOIN account_relation ar ON ar.id = arpl.account_relation_id
JOIN account a ON a.id = ar.counterparty_account_id
JOIN product_line pl ON pl.id = arpl.product_line_id
WHERE ar.owner_account_id = sqlc.arg('owner_account_id')
AND ar.account_relation_role_code = 'customer'
AND (
    sqlc.narg('search_query') IS NULL
    OR a.name LIKE sqlc.narg('search_query')
    OR ar.external_number LIKE sqlc.narg('search_query')
)
AND (
    sqlc.narg('cursor_created_at') IS NULL
    OR arpl.created_at < sqlc.narg('cursor_created_at')
    OR (arpl.created_at = sqlc.narg('cursor_created_at') AND ar.counterparty_account_id < sqlc.narg('cursor_id'))
)
ORDER BY arpl.created_at DESC, ar.counterparty_account_id DESC
LIMIT ?;

-- name: ListCustomerProductLineAccessBackward :many
SELECT
    arpl.id,
    ar.counterparty_account_id AS customer_id,
    a.name AS customer_name,
    ar.external_number AS customer_number,
    arpl.product_line_id,
    pl.name AS product_line_name,
    arpl.created_at,
    arpl.updated_at
FROM account_relation_product_line arpl
JOIN account_relation ar ON ar.id = arpl.account_relation_id
JOIN account a ON a.id = ar.counterparty_account_id
JOIN product_line pl ON pl.id = arpl.product_line_id
WHERE ar.owner_account_id = sqlc.arg('owner_account_id')
AND ar.account_relation_role_code = 'customer'
AND (
    sqlc.narg('search_query') IS NULL
    OR a.name LIKE sqlc.narg('search_query')
    OR ar.external_number LIKE sqlc.narg('search_query')
)
AND (
    arpl.created_at > sqlc.arg('cursor_created_at')
    OR (arpl.created_at = sqlc.arg('cursor_created_at') AND ar.counterparty_account_id > sqlc.arg('cursor_id'))
)
ORDER BY arpl.created_at ASC, ar.counterparty_account_id ASC
LIMIT ?;

-- name: GetCustomerProductLineAccess :many
SELECT
    arpl.id,
    arpl.product_line_id,
    pl.name AS product_line_name,
    arpl.created_at,
    arpl.updated_at
FROM account_relation_product_line arpl
JOIN product_line pl ON pl.id = arpl.product_line_id
WHERE arpl.account_relation_id = sqlc.arg('account_relation_id');

-- name: GetAccountRelationForCustomer :one
SELECT ar.id, a.name, ar.external_number, ar.created_at, ar.updated_at
FROM account_relation ar
JOIN account a ON a.id = ar.counterparty_account_id
WHERE ar.owner_account_id = sqlc.arg('owner_account_id')
AND ar.counterparty_account_id = sqlc.arg('counterparty_account_id')
AND ar.account_relation_role_code = 'customer';

-- name: InsertAccountRelationProductLine :exec
INSERT INTO account_relation_product_line (
    id,
    account_relation_id,
    product_line_id,
    created_at,
    updated_at
) VALUES (
    sqlc.arg('id'),
    sqlc.arg('account_relation_id'),
    sqlc.arg('product_line_id'),
    NOW(3),
    NOW(3)
);

-- name: DeleteAccountRelationProductLinesByRelationID :execresult
DELETE FROM account_relation_product_line
WHERE account_relation_id = sqlc.arg('account_relation_id');

-- name: CountAccountRelationProductLinesByRelationID :one
SELECT COUNT(*) FROM account_relation_product_line
WHERE account_relation_id = sqlc.arg('account_relation_id');
