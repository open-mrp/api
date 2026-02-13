-- name: FindAccountUserWithRoleByAccountIDAndUserID :one
SELECT 
    account_user.id,
    account_user.user_id,
    account_user.department_id,
    account_user.last_used_at,
    account_user.created_at,
    account_user.updated_at,
    account_user.role_id,
    account_user.account_id,
    role.role_type_code
FROM account_user 
LEFT JOIN role ON account_user.role_id = role.id
WHERE account_user.account_id = ? AND account_user.user_id = ? 
    AND (account_user.status_code = 'active' OR account_user.status_code IS NULL);

-- name: UpdateAccountUserLastUsedAt :exec
UPDATE account_user SET last_used_at = ? WHERE id = ?;

-- name: FindAccountAffiliationsByUserID :many
SELECT 
    account.id as account_id,
    account.name as account_name,
    role.id as role_id,
    role.name as role_name,
    account_user.last_used_at as last_used_at
FROM account_user 
JOIN account ON account_user.account_id = account.id
JOIN role ON account_user.role_id = role.id
WHERE account_user.user_id = ?
    AND (account_user.status_code = 'active' OR account_user.status_code IS NULL);

-- name: FindLastUsedAccountID :one
SELECT account_user.account_id
FROM account_user 
WHERE account_user.user_id = ?
    AND (account_user.status_code = 'active' OR account_user.status_code IS NULL)
ORDER BY account_user.last_used_at DESC
LIMIT 1;

-- name: CreateAccountUserForRegistration :exec
INSERT INTO account_user (
    id,
    account_id,
    user_id,
    role_id,
    status_code,
    created_at,
    updated_at
) VALUES (?, ?, ?, ?, 'active', NOW(), NOW());

-- name: GetAdminRoleID :one
SELECT r.id
FROM role r
WHERE r.role_type_code = 'admin' AND r.account_id IS NULL
LIMIT 1;

