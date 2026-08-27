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
    i.created_at AS item_created_at,
    i.updated_at AS item_updated_at,
    q.id AS quantity_id,
    q.value AS quantity_value,
    u.id AS quantity_unit_id,
    u.name AS quantity_unit_name,
    u.abbreviation AS quantity_unit_abbreviation,
    u.unit_dimension_code AS quantity_unit_type,
    u.ratio_numerator AS quantity_unit_ratio_numerator,
    u.ratio_denominator AS quantity_unit_ratio_denominator,
    u.offset_numerator AS quantity_unit_offset_numerator,
    u.offset_denominator AS quantity_unit_offset_denominator,
    u.created_at AS quantity_unit_created_at,
    u.updated_at AS quantity_unit_updated_at,
    icl.scanning_station_id,
    ss.name AS scanning_station_name,
    ss.scanning_station_type_code AS scanning_station_type,
    ss.created_at AS scanning_station_created_at,
    ss.updated_at AS scanning_station_updated_at,
    icl.responsible_user_id,
    usr.name AS responsible_user_name,
    usr.created_at AS responsible_user_created_at,
    usr.updated_at AS responsible_user_updated_at
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

-- name: ListConsumptionChangeLogsForBurnRate :many
SELECT
    q.value,
    q.unit_id,
    icl.created_at
FROM inventory_change_log icl
JOIN quantity q ON q.id = icl.quantity_id
WHERE icl.item_id = sqlc.arg('item_id')
AND icl.account_id = sqlc.arg('account_id')
AND icl.created_at >= DATE_SUB(NOW(), INTERVAL 30 DAY)
AND icl.action_type_code IN ('scan', 'user_correction')
AND CAST(q.value AS DECIMAL(65,30)) < 0
ORDER BY icl.created_at ASC;
