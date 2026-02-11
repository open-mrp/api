-- name: FindRolePermissionStrings :many
SELECT 
    CASE 
        WHEN `create` = 1 THEN CONCAT(permission_code, ':create')
        WHEN `read` = 1 THEN CONCAT(permission_code, ':read')
        WHEN `update` = 1 THEN CONCAT(permission_code, ':update')
        WHEN `delete` = 1 THEN CONCAT(permission_code, ':delete')
    END as permission_string
FROM role_permission 
WHERE role_permission.role_id = ? 
    AND (`create` = 1 OR `read` = 1 OR `update` = 1 OR `delete` = 1)
ORDER BY permission_string;

