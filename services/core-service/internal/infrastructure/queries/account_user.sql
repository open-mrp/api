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
    role.role_type_code as role_type_code,
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

-- name: ListAccountUsersForwardBase :many
SELECT
    au.id,
    au.user_id,
    u.name,
    u.email,
    u.username,
    u.image_url,
    u.email_verified,
    au.role_id,
    au.department_id,
    au.status_code,
    au.last_used_at,
    au.is_commission_eligible,
    au.created_at,
    au.updated_at
FROM account_user au
JOIN `user` u ON au.user_id = u.id
WHERE au.account_id = ?
    AND (CASE WHEN sqlc.arg(include_removed) = true THEN true ELSE au.status_code != 'removed' END)
    AND (
        sqlc.narg(role_type) IS NULL
        OR EXISTS (SELECT 1 FROM role r WHERE r.id = au.role_id AND r.role_type_code = sqlc.narg(role_type))
    )
    AND (
        sqlc.narg(is_commission_eligible) IS NULL
        OR (
            sqlc.narg(is_commission_eligible) = true
            AND (
                au.is_commission_eligible = true
                OR EXISTS (
                    SELECT 1 FROM role r_comm
                    WHERE r_comm.id = au.role_id AND r_comm.role_type_code = 'sales_rep'
                )
            )
        )
        OR (
            sqlc.narg(is_commission_eligible) = false
            AND au.is_commission_eligible = false
            AND NOT EXISTS (
                SELECT 1 FROM role r_comm
                WHERE r_comm.id = au.role_id AND r_comm.role_type_code = 'sales_rep'
            )
        )
    )
    AND (sqlc.narg(query) IS NULL OR (
        MATCH(u.name) AGAINST(sqlc.narg(query) IN BOOLEAN MODE)
        OR u.username LIKE CONCAT('%', sqlc.narg(query_like), '%')
        OR u.email LIKE CONCAT('%', sqlc.narg(query_like), '%')
    ))
    AND (
        sqlc.narg(cursor_created_at) IS NULL
        OR au.created_at < sqlc.narg(cursor_created_at)
        OR (au.created_at = sqlc.narg(cursor_created_at) AND au.id < sqlc.narg(cursor_id))
    )
ORDER BY au.created_at DESC, au.id DESC
LIMIT ?;

-- name: ListAccountUsersBackwardBase :many
SELECT
    au.id,
    au.user_id,
    u.name,
    u.email,
    u.username,
    u.image_url,
    u.email_verified,
    au.role_id,
    au.department_id,
    au.status_code,
    au.last_used_at,
    au.is_commission_eligible,
    au.created_at,
    au.updated_at
FROM account_user au
JOIN `user` u ON au.user_id = u.id
WHERE au.account_id = ?
    AND (CASE WHEN sqlc.arg(include_removed) = true THEN true ELSE au.status_code != 'removed' END)
    AND (
        sqlc.narg(role_type) IS NULL
        OR EXISTS (SELECT 1 FROM role r WHERE r.id = au.role_id AND r.role_type_code = sqlc.narg(role_type))
    )
    AND (
        sqlc.narg(is_commission_eligible) IS NULL
        OR (
            sqlc.narg(is_commission_eligible) = true
            AND (
                au.is_commission_eligible = true
                OR EXISTS (
                    SELECT 1 FROM role r_comm
                    WHERE r_comm.id = au.role_id AND r_comm.role_type_code = 'sales_rep'
                )
            )
        )
        OR (
            sqlc.narg(is_commission_eligible) = false
            AND au.is_commission_eligible = false
            AND NOT EXISTS (
                SELECT 1 FROM role r_comm
                WHERE r_comm.id = au.role_id AND r_comm.role_type_code = 'sales_rep'
            )
        )
    )
    AND (sqlc.narg(query) IS NULL OR (
        MATCH(u.name) AGAINST(sqlc.narg(query) IN BOOLEAN MODE)
        OR u.username LIKE CONCAT('%', sqlc.narg(query_like), '%')
        OR u.email LIKE CONCAT('%', sqlc.narg(query_like), '%')
    ))
    AND (
        au.created_at > sqlc.arg(cursor_created_at)
        OR (au.created_at = sqlc.arg(cursor_created_at) AND au.id > sqlc.arg(cursor_id))
    )
ORDER BY au.created_at ASC, au.id ASC
LIMIT ?;

-- name: ListAccountUsersForward :many
SELECT
    au.id,
    au.user_id,
    u.name,
    u.email,
    u.username,
    u.image_url,
    u.email_verified,
    au.role_id,
    r.name AS role_name,
    r.role_type_code,
    au.department_id,
    d.name AS department_name,
    d.created_at AS department_created_at,
    d.updated_at AS department_updated_at,
    au.status_code,
    au.last_used_at,
    au.is_commission_eligible,
    au.created_at,
    au.updated_at
FROM account_user au
JOIN `user` u ON au.user_id = u.id
LEFT JOIN role r ON au.role_id = r.id
LEFT JOIN department d ON au.department_id = d.id
WHERE au.account_id = ?
    AND (CASE WHEN sqlc.arg(include_removed) = true THEN true ELSE au.status_code != 'removed' END)
    AND (CASE WHEN sqlc.narg(role_type) IS NOT NULL THEN r.role_type_code = sqlc.narg(role_type) ELSE true END)
    AND (
        sqlc.narg(is_commission_eligible) IS NULL
        OR (
            sqlc.narg(is_commission_eligible) = true
            AND (au.is_commission_eligible = true OR r.role_type_code = 'sales_rep')
        )
        OR (
            sqlc.narg(is_commission_eligible) = false
            AND au.is_commission_eligible = false
            AND (r.role_type_code IS NULL OR r.role_type_code <> 'sales_rep')
        )
    )
    AND (sqlc.narg(query) IS NULL OR (
        MATCH(u.name) AGAINST(sqlc.narg(query) IN BOOLEAN MODE)
        OR u.username LIKE CONCAT('%', sqlc.narg(query_like), '%')
        OR u.email LIKE CONCAT('%', sqlc.narg(query_like), '%')
        OR r.name LIKE CONCAT('%', sqlc.narg(query_like), '%')
        OR d.name LIKE CONCAT('%', sqlc.narg(query_like), '%')
    ))
    AND (
        sqlc.narg(cursor_created_at) IS NULL
        OR au.created_at < sqlc.narg(cursor_created_at)
        OR (au.created_at = sqlc.narg(cursor_created_at) AND au.id < sqlc.narg(cursor_id))
    )
ORDER BY au.created_at DESC, au.id DESC
LIMIT ?;

-- name: ListAccountUsersBackward :many
SELECT
    au.id,
    au.user_id,
    u.name,
    u.email,
    u.username,
    u.image_url,
    u.email_verified,
    au.role_id,
    r.name AS role_name,
    r.role_type_code,
    au.department_id,
    d.name AS department_name,
    d.created_at AS department_created_at,
    d.updated_at AS department_updated_at,
    au.status_code,
    au.last_used_at,
    au.is_commission_eligible,
    au.created_at,
    au.updated_at
FROM account_user au
JOIN `user` u ON au.user_id = u.id
LEFT JOIN role r ON au.role_id = r.id
LEFT JOIN department d ON au.department_id = d.id
WHERE au.account_id = ?
    AND (CASE WHEN sqlc.arg(include_removed) = true THEN true ELSE au.status_code != 'removed' END)
    AND (CASE WHEN sqlc.narg(role_type) IS NOT NULL THEN r.role_type_code = sqlc.narg(role_type) ELSE true END)
    AND (
        sqlc.narg(is_commission_eligible) IS NULL
        OR (
            sqlc.narg(is_commission_eligible) = true
            AND (au.is_commission_eligible = true OR r.role_type_code = 'sales_rep')
        )
        OR (
            sqlc.narg(is_commission_eligible) = false
            AND au.is_commission_eligible = false
            AND (r.role_type_code IS NULL OR r.role_type_code <> 'sales_rep')
        )
    )
    AND (sqlc.narg(query) IS NULL OR (
        MATCH(u.name) AGAINST(sqlc.narg(query) IN BOOLEAN MODE)
        OR u.username LIKE CONCAT('%', sqlc.narg(query_like), '%')
        OR u.email LIKE CONCAT('%', sqlc.narg(query_like), '%')
        OR r.name LIKE CONCAT('%', sqlc.narg(query_like), '%')
        OR d.name LIKE CONCAT('%', sqlc.narg(query_like), '%')
    ))
    AND (
        au.created_at > sqlc.arg(cursor_created_at)
        OR (au.created_at = sqlc.arg(cursor_created_at) AND au.id > sqlc.arg(cursor_id))
    )
ORDER BY au.created_at ASC, au.id ASC
LIMIT ?;

-- name: CountAccountUsersFiltered :one
SELECT COUNT(*) AS cnt
FROM account_user au
JOIN `user` u ON au.user_id = u.id
WHERE au.account_id = ?
    AND (CASE WHEN sqlc.arg(include_removed) = true THEN true ELSE au.status_code != 'removed' END)
    AND (
        sqlc.narg(role_type) IS NULL
        OR EXISTS (SELECT 1 FROM role r WHERE r.id = au.role_id AND r.role_type_code = sqlc.narg(role_type))
    )
    AND (
        sqlc.narg(is_commission_eligible) IS NULL
        OR (
            sqlc.narg(is_commission_eligible) = true
            AND (
                au.is_commission_eligible = true
                OR EXISTS (
                    SELECT 1 FROM role r_comm
                    WHERE r_comm.id = au.role_id AND r_comm.role_type_code = 'sales_rep'
                )
            )
        )
        OR (
            sqlc.narg(is_commission_eligible) = false
            AND au.is_commission_eligible = false
            AND NOT EXISTS (
                SELECT 1 FROM role r_comm
                WHERE r_comm.id = au.role_id AND r_comm.role_type_code = 'sales_rep'
            )
        )
    )
    AND (sqlc.narg(query) IS NULL OR (
        MATCH(u.name) AGAINST(sqlc.narg(query) IN BOOLEAN MODE)
        OR u.username LIKE CONCAT('%', sqlc.narg(query_like), '%')
        OR u.email LIKE CONCAT('%', sqlc.narg(query_like), '%')
    ));

-- name: GetAccountUserDetailBase :one
SELECT
    au.id,
    au.user_id,
    u.name,
    u.email,
    u.username,
    u.image_url,
    u.email_verified,
    au.role_id,
    au.department_id,
    au.status_code,
    au.last_used_at,
    au.is_commission_eligible,
    au.created_at,
    au.updated_at
FROM account_user au
JOIN `user` u ON au.user_id = u.id
WHERE au.account_id = ?
    AND au.user_id = ?
    AND au.status_code != 'removed';

-- name: GetAccountUserDetailBaseByID :one
SELECT
    au.id,
    au.user_id,
    u.name,
    u.email,
    u.username,
    u.image_url,
    u.email_verified,
    au.role_id,
    r.role_type_code,
    au.department_id,
    d.name AS department_name,
    d.created_at AS department_created_at,
    d.updated_at AS department_updated_at,
    au.status_code,
    au.last_used_at,
    au.is_commission_eligible,
    au.created_at,
    au.updated_at
FROM account_user au
JOIN `user` u ON au.user_id = u.id
LEFT JOIN role r ON au.role_id = r.id
LEFT JOIN department d ON au.department_id = d.id
WHERE au.account_id = ?
    AND au.id = ?;

-- name: GetAccountUserDetail :one
SELECT
    au.id,
    au.user_id,
    u.name,
    u.email,
    u.username,
    u.image_url,
    u.email_verified,
    au.role_id,
    r.name AS role_name,
    r.role_type_code,
    au.department_id,
    d.name AS department_name,
    d.created_at AS department_created_at,
    d.updated_at AS department_updated_at,
    au.status_code,
    au.last_used_at,
    au.is_commission_eligible,
    au.created_at,
    au.updated_at
FROM account_user au
JOIN `user` u ON au.user_id = u.id
LEFT JOIN role r ON au.role_id = r.id
LEFT JOIN department d ON au.department_id = d.id
WHERE au.account_id = ? AND au.user_id = ?;

-- name: GetAccountUserDetailByAccountAndID :one
SELECT
    au.id,
    au.user_id,
    u.name,
    u.email,
    u.username,
    u.image_url,
    u.email_verified,
    au.role_id,
    r.name AS role_name,
    r.role_type_code,
    au.department_id,
    d.name AS department_name,
    d.created_at AS department_created_at,
    d.updated_at AS department_updated_at,
    au.status_code,
    au.last_used_at,
    au.is_commission_eligible,
    au.created_at,
    au.updated_at
FROM account_user au
JOIN `user` u ON au.user_id = u.id
LEFT JOIN role r ON au.role_id = r.id
LEFT JOIN department d ON au.department_id = d.id
WHERE au.account_id = ? AND au.id = ?;

-- name: ResolveAccountUserID :one
-- Resolves either an account_user id or a user id to the account_user id in
-- the given account. Used by write paths that accept both id forms.
SELECT au.id
FROM account_user au
WHERE au.account_id = sqlc.arg('account_id')
AND (au.id = sqlc.arg('user_or_account_user_id') OR au.user_id = sqlc.arg('user_or_account_user_id'))
AND (au.status_code = 'active' OR au.status_code IS NULL)
LIMIT 1;

-- name: GetAccountUserDetailsByIDs :many
SELECT
    au.id,
    au.user_id,
    u.name,
    u.email,
    u.username,
    u.image_url,
    u.email_verified,
    au.role_id,
    au.department_id,
    au.status_code,
    au.last_used_at,
    au.is_commission_eligible,
    au.created_at,
    au.updated_at
FROM account_user au
JOIN `user` u ON au.user_id = u.id
WHERE au.id IN (sqlc.slice('ids'))
AND au.account_id = sqlc.arg('account_id');

-- name: InsertAccountUser :exec
INSERT INTO account_user (id, account_id, user_id, role_id, department_id, is_commission_eligible, status_code, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, NOW(3), NOW(3));

-- name: UpdateAccountUserRoleAndDepartment :exec
UPDATE account_user
SET role_id = sqlc.narg(role_id),
    department_id = sqlc.narg(department_id),
    is_commission_eligible = sqlc.arg(is_commission_eligible),
    updated_at = NOW(3)
WHERE id = ?;

-- name: FindRemovedAccountUserIDByAccountAndUserID :one
SELECT id
FROM account_user
WHERE account_id = ? AND user_id = ? AND status_code = 'removed'
LIMIT 1;

-- name: ReactivateRemovedAccountUser :exec
UPDATE account_user
SET status_code = 'active',
    role_id = sqlc.narg(role_id),
    department_id = sqlc.narg(department_id),
    is_commission_eligible = sqlc.arg(is_commission_eligible),
    updated_at = NOW(3)
WHERE account_id = ? AND user_id = ? AND status_code = 'removed';

-- name: SoftDeleteAccountUser :execresult
UPDATE account_user
SET status_code = 'removed', updated_at = NOW(3)
WHERE id = ?;

-- name: UpdateAccountUserStatus :exec
UPDATE account_user
SET status_code = ?, updated_at = NOW(3)
WHERE id = ?;

-- name: RevokeRefreshTokensByUserID :exec
UPDATE refresh_token
SET revoked_at = NOW(3)
WHERE user_id = ? AND revoked_at IS NULL;

-- name: FindFirstAccountIDByUserID :one
SELECT account_id FROM account_user WHERE user_id = ? LIMIT 1;

-- name: FindTenancyAccountsByUserID :many
SELECT
    a.id AS account_id,
    a.name AS account_name,
    a.account_type_code,
    a.onboarding_status_code,
    COALESCE(ap.plan_type_code, 'free') AS plan_code,
    au.id AS account_user_id,
    au.status_code AS account_user_status_code,
    au.last_used_at,
    au.role_id,
    r.name AS role_name,
    r.role_type_code,
    r.created_at AS role_created_at,
    r.updated_at AS role_updated_at,
    sa.owner_account_id,
    ab.internal_stripe_customer_id,
    ap.type_id AS plan_type_id,
    ap.name AS plan_name,
    ap.plan_type_code AS plan_plan_type_code,
    ap.version AS plan_version,
    ap.price_per_seat AS plan_price_per_seat,
    ap.price_per_month AS plan_price_per_month,
    ap.seat_minimum AS plan_seat_minimum
FROM account_user au
JOIN account a ON au.account_id = a.id
LEFT JOIN role r ON au.role_id = r.id
LEFT JOIN account_billing ab ON a.account_billing_id = ab.id
LEFT JOIN account_plan ap ON ab.account_plan_id = ap.type_id
LEFT JOIN sandbox_account sa ON sa.account_id = a.id
WHERE au.user_id = ?;


-- name: MarkUsedByAccountAndUser :exec
UPDATE account_user
SET last_used_at = NOW(3), updated_at = NOW(3)
WHERE account_id = ? AND user_id = ?;

-- name: CountAccountUsersByRoleID :one
SELECT COUNT(*) AS cnt
FROM account_user
WHERE role_id = ? AND account_id = ? AND status_code != 'removed';

