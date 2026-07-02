-- name: ListNotificationPreferences :many
SELECT * FROM notification_preference
WHERE account_user_id = ?
ORDER BY category ASC;

-- name: GetEffectiveNotificationPreference :one
-- Resolves the most specific preference for a recipient + category: a category-specific row wins
-- over the global (category = '') default. Returns no row when neither exists (caller defaults).
SELECT * FROM notification_preference
WHERE account_user_id = sqlc.arg('account_user_id')
  AND category IN (sqlc.arg('category'), '')
ORDER BY (category = sqlc.arg('category')) DESC
LIMIT 1;

-- name: UpsertNotificationPreference :exec
INSERT INTO notification_preference (
    id, account_id, account_user_id, category,
    in_app_enabled, email_enabled, push_enabled, digest, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, NOW(3), NOW(3))
ON DUPLICATE KEY UPDATE
    in_app_enabled = VALUES(in_app_enabled),
    email_enabled = VALUES(email_enabled),
    push_enabled = VALUES(push_enabled),
    digest = VALUES(digest),
    updated_at = NOW(3);

-- name: GetNotificationPreferenceByUserCategory :one
SELECT * FROM notification_preference
WHERE account_user_id = ? AND category = ?;
