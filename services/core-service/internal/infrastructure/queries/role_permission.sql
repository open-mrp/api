-- name: FindRolePermissionFlags :many
-- One scan of the role's permissions; the caller expands each row into its `code:verb`
-- strings. The UNION ALL of four filtered SELECTs this replaced scanned
-- role_permission_role_id_idx once per verb and filesorted the result, reading 4x the rows
-- on every authorized request.
SELECT permission_code, `create`, `read`, `update`, `delete`
FROM role_permission
WHERE role_id = sqlc.arg('role_id');

-- name: ListRolePermissionsByRoleID :many
SELECT id, permission_code, `create`, `read`, `update`, `delete`, role_id, created_at, updated_at
FROM role_permission
WHERE role_id = ?
ORDER BY permission_code;

-- name: InsertRolePermission :exec
INSERT INTO role_permission (id, permission_code, `create`, `read`, `update`, `delete`, role_id, created_at, updated_at)
VALUES (sqlc.arg('id'), sqlc.arg('permission_code'), sqlc.arg('create'), sqlc.arg('read'), sqlc.arg('update'), sqlc.arg('delete'), sqlc.arg('role_id'), NOW(3), NOW(3));

-- name: ListRolePermissionsByRoleIDs :many
SELECT id, permission_code, `create`, `read`, `update`, `delete`, role_id, created_at, updated_at
FROM role_permission
WHERE role_id IN (sqlc.slice('role_ids'))
ORDER BY role_id, permission_code;

-- name: DeleteRolePermissionsByRoleID :exec
DELETE FROM role_permission WHERE role_id = ?;

