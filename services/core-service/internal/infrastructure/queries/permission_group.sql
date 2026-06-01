-- name: ListPermissionGroupsForward :many
SELECT
    permission_group.id,
    permission_group.code,
    permission_group.name,
    permission_group.description,
    permission_group.created_at,
    permission_group.updated_at
FROM permission_group
WHERE (
    sqlc.narg('search_query') IS NULL
    OR permission_group.name LIKE sqlc.narg('search_query')
)
AND (
    sqlc.narg('cursor_created_at') IS NULL
    OR permission_group.created_at < sqlc.narg('cursor_created_at')
    OR (permission_group.created_at = sqlc.narg('cursor_created_at') AND permission_group.id < sqlc.narg('cursor_id'))
)
ORDER BY permission_group.created_at DESC, permission_group.id DESC
LIMIT ?;

-- name: ListPermissionGroupsBackward :many
SELECT
    permission_group.id,
    permission_group.code,
    permission_group.name,
    permission_group.description,
    permission_group.created_at,
    permission_group.updated_at
FROM permission_group
WHERE (
    sqlc.narg('search_query') IS NULL
    OR permission_group.name LIKE sqlc.narg('search_query')
)
AND (
    permission_group.created_at > sqlc.arg('cursor_created_at')
    OR (permission_group.created_at = sqlc.arg('cursor_created_at') AND permission_group.id > sqlc.arg('cursor_id'))
)
ORDER BY permission_group.created_at ASC, permission_group.id ASC
LIMIT ?;

-- name: GetPermissionGroupsByIDs :many
SELECT
    permission_group.id,
    permission_group.code,
    permission_group.name,
    permission_group.description,
    permission_group.created_at,
    permission_group.updated_at
FROM permission_group
WHERE permission_group.id IN (sqlc.slice('ids'));

-- name: ListPermissionsByGroupCodes :many
SELECT
    permission.id,
    permission.code,
    permission.name,
    permission.description,
    permission.permission_group_code,
    permission.created_at,
    permission.updated_at
FROM permission
WHERE permission.permission_group_code IN (sqlc.slice('permission_group_codes'))
ORDER BY permission.name ASC;
