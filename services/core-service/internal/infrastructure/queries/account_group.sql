-- name: ListAccountGroupsForward :many
SELECT
    account_group.id,
    account_group.owner_account_id,
    account_group.name,
    account_group.description,
    account_group.commission_status_code,
    account_group.freight_status_code,
    account_group.account_group_type_code,
    account_group.registration_flow_id,
    account_group.created_at,
    account_group.updated_at
FROM account_group
WHERE account_group.owner_account_id = sqlc.arg('owner_account_id')
AND (
    sqlc.narg('search_query') IS NULL
    OR account_group.name LIKE sqlc.narg('search_query')
    OR account_group.description LIKE sqlc.narg('search_query')
)
AND (
    sqlc.narg('type_filter') IS NULL
    OR account_group.account_group_type_code = sqlc.narg('type_filter')
)
AND (
    sqlc.narg('cursor_created_at') IS NULL
    OR account_group.created_at < sqlc.narg('cursor_created_at')
    OR (account_group.created_at = sqlc.narg('cursor_created_at') AND account_group.id < sqlc.narg('cursor_id'))
)
ORDER BY account_group.created_at DESC, account_group.id DESC
LIMIT ?;

-- name: ListAccountGroupsBackward :many
SELECT
    account_group.id,
    account_group.owner_account_id,
    account_group.name,
    account_group.description,
    account_group.commission_status_code,
    account_group.freight_status_code,
    account_group.account_group_type_code,
    account_group.registration_flow_id,
    account_group.created_at,
    account_group.updated_at
FROM account_group
WHERE account_group.owner_account_id = sqlc.arg('owner_account_id')
AND (
    sqlc.narg('search_query') IS NULL
    OR account_group.name LIKE sqlc.narg('search_query')
    OR account_group.description LIKE sqlc.narg('search_query')
)
AND (
    sqlc.narg('type_filter') IS NULL
    OR account_group.account_group_type_code = sqlc.narg('type_filter')
)
AND (
    account_group.created_at > sqlc.arg('cursor_created_at')
    OR (account_group.created_at = sqlc.arg('cursor_created_at') AND account_group.id > sqlc.arg('cursor_id'))
)
ORDER BY account_group.created_at ASC, account_group.id ASC
LIMIT ?;

-- name: GetAccountGroup :one
SELECT
    account_group.id,
    account_group.owner_account_id,
    account_group.name,
    account_group.description,
    account_group.commission_status_code,
    account_group.freight_status_code,
    account_group.account_group_type_code,
    account_group.registration_flow_id,
    account_group.created_at,
    account_group.updated_at
FROM account_group
WHERE account_group.id = sqlc.arg('id')
AND account_group.owner_account_id = sqlc.arg('owner_account_id');

-- name: InsertAccountGroup :exec
INSERT INTO account_group (
    id,
    owner_account_id,
    name,
    description,
    commission_status_code,
    freight_status_code,
    account_group_type_code,
    created_at,
    updated_at
) VALUES (
    sqlc.arg('id'),
    sqlc.arg('owner_account_id'),
    sqlc.arg('name'),
    sqlc.narg('description'),
    sqlc.arg('commission_status_code'),
    sqlc.arg('freight_status_code'),
    sqlc.arg('account_group_type_code'),
    NOW(3),
    NOW(3)
);

-- name: UpdateAccountGroup :execresult
UPDATE account_group SET
    name = COALESCE(sqlc.narg('name'), name),
    description = COALESCE(sqlc.narg('description'), description),
    commission_status_code = COALESCE(sqlc.narg('commission_status_code'), commission_status_code),
    freight_status_code = COALESCE(sqlc.narg('freight_status_code'), freight_status_code),
    updated_at = NOW(3)
WHERE id = sqlc.arg('id')
AND owner_account_id = sqlc.arg('owner_account_id');

-- name: DeleteAccountGroup :execresult
DELETE FROM account_group
WHERE id = sqlc.arg('id')
AND owner_account_id = sqlc.arg('owner_account_id');

-- name: CountAccountGroupUsageInAccountRelation :one
SELECT COUNT(*)
FROM account_relation
WHERE owner_account_id = sqlc.arg('owner_account_id')
AND account_group_id = sqlc.narg('account_group_id');

-- name: CountAccountGroupUsageInAccountGroupProductLine :one
SELECT COUNT(*)
FROM account_group_product_line
WHERE account_group_id = sqlc.arg('account_group_id')
;

-- name: CountAccountGroupUsageInAccountGroupQuantityDiscount :one
SELECT COUNT(*)
FROM account_group_quantity_discount
WHERE account_group_id = sqlc.arg('account_group_id')
;

-- name: CountAccountGroupUsageInAccountRelationPriceGroup :one
SELECT COUNT(*)
FROM account_relation_price_group
WHERE account_group_id = sqlc.arg('account_group_id')
;

-- name: DeleteAccountRelationPriceGroupsByAccountGroupID :exec
DELETE FROM account_relation_price_group
WHERE account_group_id = sqlc.arg('account_group_id');

-- name: CountAccountGroupUsageInRegistrationFlow :one
SELECT COUNT(*)
FROM account_group
WHERE id = sqlc.arg('account_group_id')
AND owner_account_id = sqlc.arg('owner_account_id')
AND registration_flow_id IS NOT NULL;

-- name: CountAccountGroupsByName :one
SELECT COUNT(*) FROM account_group
WHERE name = ? AND owner_account_id = ?
AND (sqlc.narg('exclude_id') IS NULL OR id != sqlc.narg('exclude_id'));
