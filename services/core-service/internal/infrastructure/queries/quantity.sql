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
