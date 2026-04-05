-- name: GetRoleByID :one
SELECT id, name, role_type_code FROM role WHERE id = ?;

-- name: FindRoleByTypeCode :one
SELECT id, name, role_type_code
FROM role
WHERE role_type_code = ? AND (account_id = ? OR account_id IS NULL)
LIMIT 1;

-- name: GetRoleByIDAndAccount :one
SELECT id, name, role_type_code, account_id, created_at, updated_at
FROM role
WHERE id = ? AND (account_id = ? OR account_id IS NULL);

-- name: ListRolesForward :many
SELECT
    r.id,
    r.name,
    r.role_type_code,
    r.account_id,
    r.created_at,
    r.updated_at
FROM role r
WHERE (r.account_id = sqlc.arg('account_id') OR r.account_id IS NULL)
AND (
    sqlc.narg('search_query') IS NULL
    OR MATCH(r.name) AGAINST (sqlc.narg('search_query') IN BOOLEAN MODE)
)
AND (
    sqlc.arg('include_role_type_filter') = false
    OR r.role_type_code IN (sqlc.slice('role_type_codes'))
)
AND (
    sqlc.narg('cursor_created_at') IS NULL
    OR r.created_at < sqlc.narg('cursor_created_at')
    OR (r.created_at = sqlc.narg('cursor_created_at') AND r.id < sqlc.narg('cursor_id'))
)
ORDER BY r.created_at DESC, r.id DESC
LIMIT ?;

-- name: ListRolesBackward :many
SELECT
    r.id,
    r.name,
    r.role_type_code,
    r.account_id,
    r.created_at,
    r.updated_at
FROM role r
WHERE (r.account_id = sqlc.arg('account_id') OR r.account_id IS NULL)
AND (
    sqlc.narg('search_query') IS NULL
    OR MATCH(r.name) AGAINST (sqlc.narg('search_query') IN BOOLEAN MODE)
)
AND (
    sqlc.arg('include_role_type_filter') = false
    OR r.role_type_code IN (sqlc.slice('role_type_codes'))
)
AND (
    r.created_at > sqlc.arg('cursor_created_at')
    OR (r.created_at = sqlc.arg('cursor_created_at') AND r.id > sqlc.arg('cursor_id'))
)
ORDER BY r.created_at ASC, r.id ASC
LIMIT ?;

-- name: ExistsRoleByName :one
SELECT COUNT(*) > 0 AS `exists`
FROM role
WHERE name = sqlc.arg('name')
AND account_id = sqlc.arg('account_id')
AND (sqlc.narg('exclude_id') IS NULL OR id != sqlc.narg('exclude_id'));

-- name: InsertRole :exec
INSERT INTO role (id, name, role_type_code, account_id, created_at, updated_at)
VALUES (sqlc.arg('id'), sqlc.arg('name'), sqlc.arg('role_type_code'), sqlc.arg('account_id'), NOW(3), NOW(3));

-- name: UpdateRoleName :exec
UPDATE role
SET name = sqlc.arg('name'), updated_at = NOW(3)
WHERE id = sqlc.arg('id') AND account_id = sqlc.arg('account_id');

-- name: DeleteRoleByID :exec
DELETE FROM role WHERE id = sqlc.arg('id') AND account_id = sqlc.arg('account_id');
