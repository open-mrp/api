-- name: ListInventoryChangeLogsForward :many
SELECT
    icl.id,
    icl.action_type_code,
    icl.account_id,
    icl.created_at,
    icl.updated_at,
    i.id AS item_id,
    i.sku AS item_sku,
    i.item_type_code AS item_type_code,
    q.id AS quantity_id,
    q.value AS quantity_value,
    u.id AS quantity_unit_id,
    u.name AS quantity_unit_name,
    u.abbreviation AS quantity_unit_abbreviation,
    u.unit_dimension_code AS quantity_unit_type,
    icl.scanning_station_id,
    ss.name AS scanning_station_name,
    ss.scanning_station_type_code AS scanning_station_type,
    icl.responsible_user_id,
    usr.name AS responsible_user_name
FROM inventory_change_log icl
JOIN item i ON i.id = icl.item_id
JOIN quantity q ON q.id = icl.quantity_id
JOIN unit u ON u.id = q.unit_id
LEFT JOIN scanning_station ss ON ss.id = icl.scanning_station_id
LEFT JOIN user usr ON usr.id = icl.responsible_user_id
WHERE icl.account_id = sqlc.arg('account_id')
AND (
    sqlc.arg('include_item_filter') = false
    OR icl.item_id IN (sqlc.slice('item_ids'))
)
AND (
    sqlc.arg('include_action_type_filter') = false
    OR icl.action_type_code IN (sqlc.slice('action_type_codes'))
)
AND (
    sqlc.arg('include_user_filter') = false
    OR icl.responsible_user_id IN (sqlc.slice('changed_by_user_ids'))
)
AND (
    sqlc.narg('start_date') IS NULL
    OR icl.created_at >= sqlc.narg('start_date')
)
AND (
    sqlc.narg('end_date') IS NULL
    OR icl.created_at <= sqlc.narg('end_date')
)
AND (
    sqlc.narg('search_query') IS NULL
    OR i.sku LIKE sqlc.narg('search_query')
    OR usr.name LIKE sqlc.narg('search_query')
    OR ss.name LIKE sqlc.narg('search_query')
)
AND (
    sqlc.narg('cursor_created_at') IS NULL
    OR icl.created_at < sqlc.narg('cursor_created_at')
    OR (icl.created_at = sqlc.narg('cursor_created_at') AND icl.id < sqlc.narg('cursor_id'))
)
ORDER BY icl.created_at DESC, icl.id DESC
LIMIT ?;

-- name: ListInventoryChangeLogsBackward :many
SELECT
    icl.id,
    icl.action_type_code,
    icl.account_id,
    icl.created_at,
    icl.updated_at,
    i.id AS item_id,
    i.sku AS item_sku,
    i.item_type_code AS item_type_code,
    q.id AS quantity_id,
    q.value AS quantity_value,
    u.id AS quantity_unit_id,
    u.name AS quantity_unit_name,
    u.abbreviation AS quantity_unit_abbreviation,
    u.unit_dimension_code AS quantity_unit_type,
    icl.scanning_station_id,
    ss.name AS scanning_station_name,
    ss.scanning_station_type_code AS scanning_station_type,
    icl.responsible_user_id,
    usr.name AS responsible_user_name
FROM inventory_change_log icl
JOIN item i ON i.id = icl.item_id
JOIN quantity q ON q.id = icl.quantity_id
JOIN unit u ON u.id = q.unit_id
LEFT JOIN scanning_station ss ON ss.id = icl.scanning_station_id
LEFT JOIN user usr ON usr.id = icl.responsible_user_id
WHERE icl.account_id = sqlc.arg('account_id')
AND (
    sqlc.arg('include_item_filter') = false
    OR icl.item_id IN (sqlc.slice('item_ids'))
)
AND (
    sqlc.arg('include_action_type_filter') = false
    OR icl.action_type_code IN (sqlc.slice('action_type_codes'))
)
AND (
    sqlc.arg('include_user_filter') = false
    OR icl.responsible_user_id IN (sqlc.slice('changed_by_user_ids'))
)
AND (
    sqlc.narg('start_date') IS NULL
    OR icl.created_at >= sqlc.narg('start_date')
)
AND (
    sqlc.narg('end_date') IS NULL
    OR icl.created_at <= sqlc.narg('end_date')
)
AND (
    sqlc.narg('search_query') IS NULL
    OR i.sku LIKE sqlc.narg('search_query')
    OR usr.name LIKE sqlc.narg('search_query')
    OR ss.name LIKE sqlc.narg('search_query')
)
AND (
    icl.created_at > sqlc.arg('cursor_created_at')
    OR (icl.created_at = sqlc.arg('cursor_created_at') AND icl.id > sqlc.arg('cursor_id'))
)
ORDER BY icl.created_at ASC, icl.id ASC
LIMIT ?;

-- name: GetInventoryChangeLog :one
SELECT
    icl.id,
    icl.action_type_code,
    icl.account_id,
    icl.created_at,
    icl.updated_at,
    i.id AS item_id,
    i.sku AS item_sku,
    i.item_type_code AS item_type_code,
    q.id AS quantity_id,
    q.value AS quantity_value,
    u.id AS quantity_unit_id,
    u.name AS quantity_unit_name,
    u.abbreviation AS quantity_unit_abbreviation,
    u.unit_dimension_code AS quantity_unit_type,
    icl.scanning_station_id,
    ss.name AS scanning_station_name,
    ss.scanning_station_type_code AS scanning_station_type,
    icl.responsible_user_id,
    usr.name AS responsible_user_name
FROM inventory_change_log icl
JOIN item i ON i.id = icl.item_id
JOIN quantity q ON q.id = icl.quantity_id
JOIN unit u ON u.id = q.unit_id
LEFT JOIN scanning_station ss ON ss.id = icl.scanning_station_id
LEFT JOIN user usr ON usr.id = icl.responsible_user_id
WHERE icl.id = sqlc.arg('id')
AND icl.account_id = sqlc.arg('account_id');

-- name: ListAllInventoryChangeLogs :many
SELECT
    icl.id,
    icl.action_type_code,
    icl.account_id,
    icl.created_at,
    icl.updated_at,
    i.id AS item_id,
    i.sku AS item_sku,
    i.item_type_code AS item_type_code,
    q.id AS quantity_id,
    q.value AS quantity_value,
    u.id AS quantity_unit_id,
    u.name AS quantity_unit_name,
    u.abbreviation AS quantity_unit_abbreviation,
    u.unit_dimension_code AS quantity_unit_type,
    icl.scanning_station_id,
    ss.name AS scanning_station_name,
    ss.scanning_station_type_code AS scanning_station_type,
    icl.responsible_user_id,
    usr.name AS responsible_user_name
FROM inventory_change_log icl
JOIN item i ON i.id = icl.item_id
JOIN quantity q ON q.id = icl.quantity_id
JOIN unit u ON u.id = q.unit_id
LEFT JOIN scanning_station ss ON ss.id = icl.scanning_station_id
LEFT JOIN user usr ON usr.id = icl.responsible_user_id
WHERE icl.account_id = sqlc.arg('account_id')
AND (
    sqlc.arg('include_item_filter') = false
    OR icl.item_id IN (sqlc.slice('item_ids'))
)
AND (
    sqlc.arg('include_action_type_filter') = false
    OR icl.action_type_code IN (sqlc.slice('action_type_codes'))
)
AND (
    sqlc.arg('include_user_filter') = false
    OR icl.responsible_user_id IN (sqlc.slice('changed_by_user_ids'))
)
AND (
    sqlc.narg('start_date') IS NULL
    OR icl.created_at >= sqlc.narg('start_date')
)
AND (
    sqlc.narg('end_date') IS NULL
    OR icl.created_at <= sqlc.narg('end_date')
)
ORDER BY icl.created_at DESC, icl.id DESC;
