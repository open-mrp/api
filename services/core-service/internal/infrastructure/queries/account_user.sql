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
) VALUES (?, ?, ?, ?, 'active', NOW(3), NOW(3));

-- name: GetAdminRoleID :one
SELECT r.id
FROM role r
WHERE r.role_type_code = 'admin' AND r.account_id IS NULL
LIMIT 1;

-- name: DeactivateAccountUsersExcept :execresult
UPDATE account_user
SET status_code = 'disabled', updated_at = NOW(3)
WHERE account_id = ? AND user_id != ?
    AND (status_code = 'active' OR status_code IS NULL)
ORDER BY last_used_at ASC
LIMIT ?;

-- name: CountActiveAccountUsers :one
SELECT COUNT(*) AS cnt
FROM account_user
WHERE account_id = ?
    AND (status_code = 'active' OR status_code IS NULL);

-- name: ReactivateAccountUsers :execresult
UPDATE account_user
SET status_code = 'active', updated_at = NOW(3)
WHERE account_id = ? AND status_code = 'disabled'
ORDER BY updated_at DESC
LIMIT ?;

-- name: EnsureAccountUserActive :execresult
UPDATE account_user
SET status_code = 'active', updated_at = NOW(3)
WHERE account_id = ? AND user_id = ? AND status_code = 'disabled';

-- name: FindAdminUserIDByAccountID :one
SELECT au.user_id
FROM account_user au
JOIN role r ON au.role_id = r.id
WHERE au.account_id = ? AND r.role_type_code = 'admin'
    AND (au.status_code = 'active' OR au.status_code IS NULL)
LIMIT 1;

