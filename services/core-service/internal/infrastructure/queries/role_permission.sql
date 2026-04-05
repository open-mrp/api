-- name: FindRolePermissionStrings :many
SELECT CONCAT(role_permission.permission_code, ':create') as permission_string
FROM role_permission WHERE role_permission.role_id = sqlc.arg('role_id') AND role_permission.`create` = 1
UNION ALL
SELECT CONCAT(role_permission.permission_code, ':read')
FROM role_permission WHERE role_permission.role_id = sqlc.arg('role_id') AND role_permission.`read` = 1
UNION ALL
SELECT CONCAT(role_permission.permission_code, ':update')
FROM role_permission WHERE role_permission.role_id = sqlc.arg('role_id') AND role_permission.`update` = 1
UNION ALL
SELECT CONCAT(role_permission.permission_code, ':delete')
FROM role_permission WHERE role_permission.role_id = sqlc.arg('role_id') AND role_permission.`delete` = 1
ORDER BY 1;

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

