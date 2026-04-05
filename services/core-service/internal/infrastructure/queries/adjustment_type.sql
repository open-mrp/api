-- name: ListAdjustmentTypesForward :many
SELECT
    adjustment_type.id,
    adjustment_type.name,
    adjustment_type.code,
    adjustment_type.created_at,
    adjustment_type.updated_at
FROM adjustment_type
WHERE (
    sqlc.narg('search_query') IS NULL
    OR adjustment_type.name LIKE sqlc.narg('search_query')
)
AND (
    sqlc.narg('cursor_created_at') IS NULL
    OR adjustment_type.created_at < sqlc.narg('cursor_created_at')
    OR (adjustment_type.created_at = sqlc.narg('cursor_created_at') AND adjustment_type.id < sqlc.narg('cursor_id'))
)
ORDER BY adjustment_type.created_at DESC, adjustment_type.id DESC
LIMIT ?;

-- name: ListAdjustmentTypesBackward :many
SELECT
    adjustment_type.id,
    adjustment_type.name,
    adjustment_type.code,
    adjustment_type.created_at,
    adjustment_type.updated_at
FROM adjustment_type
WHERE (
    sqlc.narg('search_query') IS NULL
    OR adjustment_type.name LIKE sqlc.narg('search_query')
)
AND (
    adjustment_type.created_at > sqlc.arg('cursor_created_at')
    OR (adjustment_type.created_at = sqlc.arg('cursor_created_at') AND adjustment_type.id > sqlc.arg('cursor_id'))
)
ORDER BY adjustment_type.created_at ASC, adjustment_type.id ASC
LIMIT ?;
