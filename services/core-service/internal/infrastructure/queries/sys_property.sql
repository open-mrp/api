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
AND sp.account_id = sqlc.arg('account_id');

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

-- name: CheckDuplicateTransactionNumber :one
SELECT COUNT(*) FROM transaction
WHERE number = sqlc.arg('value')
AND account_id = sqlc.arg('account_id');

-- name: CheckDuplicateSettlementNumber :one
SELECT COUNT(*) FROM settlement
WHERE number = sqlc.arg('value')
AND account_id = sqlc.arg('account_id');

-- name: CheckDuplicateSalesOrderNumber :one
SELECT COUNT(*) FROM sales_order
WHERE number = sqlc.arg('value')
AND owner_account_id = sqlc.arg('account_id')
AND seller_account_id = sqlc.arg('account_id')
AND sales_order_type_code = 'sales_order';

-- name: CheckDuplicatePurchaseOrderNumber :one
SELECT COUNT(*) FROM sales_order
WHERE number = sqlc.arg('value')
AND owner_account_id = sqlc.arg('account_id')
AND buyer_account_id = sqlc.arg('account_id')
AND sales_order_type_code = 'purchase_order';

-- name: CheckDuplicateSupplierNumber :one
SELECT COUNT(*) FROM account_relation
WHERE external_number = sqlc.arg('value')
AND owner_account_id = sqlc.arg('account_id')
AND account_relation_role_code = 'supplier';

-- name: CheckDuplicateCustomerNumber :one
SELECT COUNT(*) FROM account_relation
WHERE external_number = sqlc.arg('value')
AND owner_account_id = sqlc.arg('account_id')
AND account_relation_role_code = 'customer';

-- name: CheckDuplicateProductionRunNumber :one
SELECT COUNT(*) FROM production_run
WHERE number = sqlc.arg('value')
AND account_id = sqlc.arg('account_id');
