-- name: GetAccountUserIDByUserAndAccount :one
-- Resolves the account_user id (the notification recipient key) for a user within an
-- account. identity.Actor.ID is the user id (us_), not the account_user id (acus_), so
-- read/mark requests resolve the recipient via the unique (user_id, account_id) key.
SELECT id FROM account_user
WHERE user_id = ? AND account_id = ?;

-- name: GetUserIDByAccountUserID :one
-- Resolves the user id (us_) for an account_user. The WS user-topic is keyed by user id
-- (the gateway subscribes user:<user_id> from the validated identity), but notification
-- rows are keyed by account_user id, so the realtime push reverses the mapping.
SELECT user_id FROM account_user
WHERE id = ?;

-- name: GetUserContactByAccountUserID :one
-- Resolves a recipient's email + display name for the email bridge (chat-notification emails).
SELECT u.email AS email, u.name AS name
FROM account_user au
JOIN user u ON u.id = au.user_id
WHERE au.id = ?;

-- name: CountUnseenNotificationsByUserAccounts :many
-- Aggregates the user's unseen, non-dismissed notifications per account across every account
-- they belong to. Powers the cross-account unread hint (the bell dot while viewing another
-- account).
SELECT au.account_id AS account_id, COUNT(n.id) AS unread
FROM account_user au
LEFT JOIN notification n
    ON n.recipient_account_user_id = au.id AND n.seen_at IS NULL AND n.dismissed_at IS NULL
WHERE au.user_id = ?
GROUP BY au.account_id;

-- name: CountUnseenAnnouncementsByUserAccounts :many
-- Aggregates the user's unseen, non-dismissed account-scoped announcements per account across
-- every account they belong to.
SELECT au.account_id AS account_id, COUNT(a.id) AS unread
FROM account_user au
JOIN announcement a
    ON a.scope = 'account' AND a.account_id = au.account_id
    AND a.publish_at <= NOW(3) AND (a.expires_at IS NULL OR a.expires_at > NOW(3))
LEFT JOIN announcement_receipt r
    ON r.announcement_id = a.id AND r.account_user_id = au.id
WHERE au.user_id = ?
  AND r.seen_at IS NULL
  AND r.dismissed_at IS NULL
GROUP BY au.account_id;

-- name: ListMessagingContacts :many
-- Returns active account_users in an account whose display name matches the (case-insensitive)
-- substring filter, for the messaging directory. The caller passes name_like = "%" + query + "%".
SELECT au.id AS account_user_id, u.name AS name
FROM account_user au
JOIN user u ON u.id = au.user_id
WHERE au.account_id = ? AND au.status_code = 'active' AND u.name LIKE ?
ORDER BY u.name ASC, au.id ASC
LIMIT 100;
