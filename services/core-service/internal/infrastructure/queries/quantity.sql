-- name: GetQuantityWithUnit :one
SELECT
    q.id,
    q.value,
    q.unit_id,
    u.name AS unit_name,
    u.abbreviation AS unit_abbreviation,
    u.unit_dimension_code AS unit_type,
    q.created_at,
    q.updated_at
FROM quantity q
JOIN unit u ON q.unit_id = u.id
WHERE q.id = sqlc.arg('id');

-- name: UpdateQuantityByID :execresult
UPDATE quantity SET
    value = COALESCE(sqlc.narg('value'), value),
    unit_id = COALESCE(sqlc.narg('unit_id'), unit_id),
    updated_at = NOW(3)
WHERE id = sqlc.arg('id');

-- name: ListQuantityOwnerTypes :many
-- The kinds of resource in the account a quantity belongs to: a material's order point and lead time are its item's, and a production's or consumption's quantities are its production step's. Each branch is a unique-key lookup.
SELECT 'item' AS owner_type
FROM material m
JOIN item i ON i.id = m.item_id
WHERE m.order_point_id = sqlc.arg('id') AND i.account_id = sqlc.arg('account_id')
UNION ALL
SELECT 'item'
FROM material m
JOIN item i ON i.id = m.item_id
WHERE m.lead_time_id = sqlc.arg('id') AND i.account_id = sqlc.arg('account_id')
UNION ALL
SELECT 'production_step'
FROM production p
JOIN production_step ps ON ps.id = p.production_step_id
WHERE p.quantity_id = sqlc.arg('id') AND ps.account_id = sqlc.arg('account_id')
UNION ALL
SELECT 'production_step'
FROM consumption c
JOIN production_step ps ON ps.id = c.production_step_id
WHERE c.quantity_id = sqlc.arg('id') AND ps.account_id = sqlc.arg('account_id')
UNION ALL
SELECT 'production_step'
FROM consumption c
JOIN production_step ps ON ps.id = c.production_step_id
WHERE c.waste_quantity_id = sqlc.arg('id') AND ps.account_id = sqlc.arg('account_id');
