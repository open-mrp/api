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

