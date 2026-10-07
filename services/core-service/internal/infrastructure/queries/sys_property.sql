-- The production_schedule_version counter is excluded from every read the settings endpoints serve:
-- schedule generation allocates versions from it under a unique key, so it is not a number series a
-- user may see or move — moving it below the latest version would make the next generation collide.

-- name: ListSysPropertiesForward :many
SELECT
    sp.id,
    sp.sys_property_type_code AS type_code,
    sp.value,
    sp.account_id,
    sp.created_at,
    sp.updated_at,
    spt.id AS type_id,
    spt.name AS type_name
FROM sys_property sp
JOIN sys_property_type spt ON sp.sys_property_type_code = spt.code
WHERE sp.account_id = sqlc.arg('account_id')
AND sp.sys_property_type_code <> 'production_schedule_version'
AND (
    sqlc.narg('search_query') IS NULL
    OR spt.name LIKE sqlc.narg('search_query')
)
AND (
    sqlc.narg('cursor_created_at') IS NULL
    OR sp.created_at < sqlc.narg('cursor_created_at')
    OR (sp.created_at = sqlc.narg('cursor_created_at') AND sp.id < sqlc.narg('cursor_id'))
)
ORDER BY sp.created_at DESC, sp.id DESC
LIMIT ?;

-- name: ListSysPropertiesBackward :many
SELECT
    sp.id,
    sp.sys_property_type_code AS type_code,
    sp.value,
    sp.account_id,
    sp.created_at,
    sp.updated_at,
    spt.id AS type_id,
    spt.name AS type_name
FROM sys_property sp
JOIN sys_property_type spt ON sp.sys_property_type_code = spt.code
WHERE sp.account_id = sqlc.arg('account_id')
AND sp.sys_property_type_code <> 'production_schedule_version'
AND (
    sqlc.narg('search_query') IS NULL
    OR spt.name LIKE sqlc.narg('search_query')
)
AND (
    sp.created_at > sqlc.arg('cursor_created_at')
    OR (sp.created_at = sqlc.arg('cursor_created_at') AND sp.id > sqlc.arg('cursor_id'))
)
ORDER BY sp.created_at ASC, sp.id ASC
LIMIT ?;

-- name: GetSysProperty :one
SELECT
    sp.id,
    sp.sys_property_type_code AS type_code,
    sp.value,
    sp.account_id,
    sp.created_at,
    sp.updated_at,
    spt.id AS type_id,
    spt.name AS type_name
FROM sys_property sp
JOIN sys_property_type spt ON sp.sys_property_type_code = spt.code
WHERE sp.id = sqlc.arg('id')
AND sp.account_id = sqlc.arg('account_id')
AND sp.sys_property_type_code <> 'production_schedule_version';

-- name: GetSysPropertiesByIDs :many
-- Returns sys properties matching the given IDs that belong to the caller's
-- account. Used by the api-gateway resourcekit resolver.
SELECT
    sp.id,
    sp.sys_property_type_code AS type_code,
    sp.value,
    sp.account_id,
    sp.created_at,
    sp.updated_at,
    spt.id AS type_id,
    spt.name AS type_name
FROM sys_property sp
JOIN sys_property_type spt ON sp.sys_property_type_code = spt.code
WHERE sp.id IN (sqlc.slice('ids'))
AND sp.account_id = sqlc.arg('account_id')
AND sp.sys_property_type_code <> 'production_schedule_version';

-- name: GetSysPropertyByTypeCode :one
SELECT
    sp.id,
    sp.sys_property_type_code AS type_code,
    sp.value,
    sp.account_id,
    sp.created_at,
    sp.updated_at,
    spt.id AS type_id,
    spt.name AS type_name
FROM sys_property sp
JOIN sys_property_type spt ON sp.sys_property_type_code = spt.code
WHERE sp.sys_property_type_code = sqlc.arg('type_code')
AND sp.account_id = sqlc.arg('account_id');

-- name: InsertSysProperty :exec
INSERT INTO sys_property (
    id,
    sys_property_type_code,
    value,
    account_id,
    created_at,
    updated_at
) VALUES (
    sqlc.arg('id'),
    sqlc.arg('type_code'),
    sqlc.arg('value'),
    sqlc.arg('account_id'),
    NOW(3),
    NOW(3)
);

-- name: UpdateSysPropertyValue :execresult
UPDATE sys_property SET
    value = sqlc.arg('value'),
    updated_at = NOW(3)
WHERE id = sqlc.arg('id')
AND account_id = sqlc.arg('account_id');

-- name: IncrementSysPropertyValue :execresult
UPDATE sys_property SET
    value = value + 1,
    updated_at = NOW(3)
WHERE id = sqlc.arg('id')
AND account_id = sqlc.arg('account_id');

-- The ListTaken*Numbers queries return which of the given candidate numbers a record in the series
-- already carries, so the next free number is found a batch of candidates per query.

-- name: ListTakenTransactionNumbers :many
SELECT number FROM transaction
WHERE account_id = sqlc.arg('account_id')
AND number IN (sqlc.slice('numbers'));

-- name: ListTakenSettlementNumbers :many
SELECT number FROM settlement
WHERE account_id = sqlc.arg('account_id')
AND number IN (sqlc.slice('numbers'));

-- name: ListTakenSalesOrderNumbers :many
SELECT number FROM sales_order
WHERE owner_account_id = sqlc.arg('account_id')
AND seller_account_id = sqlc.arg('account_id')
AND sales_order_type_code = 'sales_order'
AND number IN (sqlc.slice('numbers'));

-- name: ListTakenPurchaseOrderNumbers :many
SELECT number FROM sales_order
WHERE owner_account_id = sqlc.arg('account_id')
AND buyer_account_id = sqlc.arg('account_id')
AND sales_order_type_code = 'purchase_order'
AND number IN (sqlc.slice('numbers'));

-- name: ListTakenSupplierNumbers :many
SELECT external_number FROM account_relation
WHERE owner_account_id = sqlc.arg('account_id')
AND account_relation_role_code = 'supplier'
AND external_number IN (sqlc.slice('numbers'));

-- name: ListTakenCustomerNumbers :many
SELECT external_number FROM account_relation
WHERE owner_account_id = sqlc.arg('account_id')
AND account_relation_role_code = 'customer'
AND external_number IN (sqlc.slice('numbers'));

-- name: ListTakenProductionRunNumbers :many
SELECT number FROM production_run
WHERE account_id = sqlc.arg('account_id')
AND number IN (sqlc.slice('numbers'));
