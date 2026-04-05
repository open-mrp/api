-- name: ListPrioritiesForward :many
SELECT
    priority.id,
    priority.name,
    priority.code,
    priority.created_at,
    priority.updated_at
FROM priority
WHERE (
    sqlc.narg('search_query') IS NULL
    OR priority.name LIKE sqlc.narg('search_query')
)
AND (
    sqlc.narg('cursor_created_at') IS NULL
    OR priority.created_at < sqlc.narg('cursor_created_at')
    OR (priority.created_at = sqlc.narg('cursor_created_at') AND priority.id < sqlc.narg('cursor_id'))
)
ORDER BY priority.created_at DESC, priority.id DESC
LIMIT ?;

-- name: ListPrioritiesBackward :many
SELECT
    priority.id,
    priority.name,
    priority.code,
    priority.created_at,
    priority.updated_at
FROM priority
WHERE (
    sqlc.narg('search_query') IS NULL
    OR priority.name LIKE sqlc.narg('search_query')
)
AND (
    priority.created_at > sqlc.arg('cursor_created_at')
    OR (priority.created_at = sqlc.arg('cursor_created_at') AND priority.id > sqlc.arg('cursor_id'))
)
ORDER BY priority.created_at ASC, priority.id ASC
LIMIT ?;

-- name: GetPriority :one
SELECT
    priority.id,
    priority.name,
    priority.code,
    priority.created_at,
    priority.updated_at
FROM priority
WHERE priority.id = sqlc.arg('id') OR priority.code = sqlc.arg('code');
