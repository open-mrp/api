-- name: GetNotificationSetting :one
SELECT * FROM notification_setting
WHERE account_id = ? AND account_user_id = ?;

-- name: UpsertNotificationSetting :exec
INSERT INTO notification_setting (id, account_id, account_user_id, defaults, auto_subscribe, paused_until, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, NOW(3), NOW(3))
ON DUPLICATE KEY UPDATE
    defaults = VALUES(defaults),
    auto_subscribe = VALUES(auto_subscribe),
    paused_until = VALUES(paused_until),
    updated_at = NOW(3);

-- name: GetAccountUserRoleNotificationDefaults :one
-- The notification defaults of the person's role.
SELECT r.notification_defaults
FROM account_user au
JOIN role r ON r.id = au.role_id
WHERE au.id = ?;
