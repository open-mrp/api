-- name: CreateNotification :exec
INSERT INTO notification (
    id, account_id, recipient_account_user_id, category,
    source_message_id, conversation_id, title, body,
    template_key, template_params, link_resource_type, link_resource_id,
    sender_type, sender_id, sender_name,
    priority, seen_at, read_at, dismissed_at, metadata, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, NOW(3), NOW(3));

-- name: UpsertCoalescedNotification :execrows
-- Folds repeated activity on one resource onto a single rolling bell row instead of a new row per
-- event. The caller derives the id deterministically from a rolling dedupe key (e.g. one row per order
-- per day per recipient), so the first event of the window inserts and every later event lands on the
-- same PK. On conflict the row is refreshed to the latest title/body, resurfaced as unread (seen_at /
-- read_at cleared) and its change_count bumped, while an explicit dismiss is left in place so a
-- recipient can mute the resource for the rest of the window. The rows-affected result tells the two
-- cases apart — 1 = inserted, 2 = updated — so the caller raises a realtime alert only on the first
-- event. change_count always changes on conflict, so an update never collapses to a 0-row no-op.
INSERT INTO notification (
    id, account_id, recipient_account_user_id, category,
    source_message_id, conversation_id, title, body,
    template_key, template_params, link_resource_type, link_resource_id,
    sender_type, sender_id, sender_name,
    priority, seen_at, read_at, dismissed_at, metadata, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, NOW(3), NOW(3))
ON DUPLICATE KEY UPDATE
    title = VALUES(title),
    body = VALUES(body),
    seen_at = NULL,
    read_at = NULL,
    updated_at = NOW(3),
    metadata = JSON_SET(
        COALESCE(metadata, JSON_OBJECT()),
        '$.change_count',
        COALESCE(CAST(JSON_UNQUOTE(JSON_EXTRACT(metadata, '$.change_count')) AS UNSIGNED), 1) + 1
    );

-- name: UpsertCoalescedNotificationQuietly :execrows
-- UpsertCoalescedNotification without the resurfacing: on conflict the row takes the latest title/body
-- and bumps change_count but keeps its seen/read/dismissed state, so a window alerts its recipient once.
INSERT INTO notification (
    id, account_id, recipient_account_user_id, category,
    source_message_id, conversation_id, title, body,
    template_key, template_params, link_resource_type, link_resource_id,
    sender_type, sender_id, sender_name,
    priority, seen_at, read_at, dismissed_at, metadata, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, NOW(3), NOW(3))
ON DUPLICATE KEY UPDATE
    title = VALUES(title),
    body = VALUES(body),
    updated_at = NOW(3),
    metadata = JSON_SET(
        COALESCE(metadata, JSON_OBJECT()),
        '$.change_count',
        COALESCE(CAST(JSON_UNQUOTE(JSON_EXTRACT(metadata, '$.change_count')) AS UNSIGNED), 1) + 1
    );

-- name: GetNotificationByID :one
SELECT * FROM notification
WHERE id = ? AND recipient_account_user_id = ?;

-- name: CountUnseenNotifications :one
SELECT COUNT(*) FROM notification
WHERE recipient_account_user_id = ? AND seen_at IS NULL AND dismissed_at IS NULL;

-- name: MarkNotificationSeen :exec
UPDATE notification
SET seen_at = COALESCE(seen_at, NOW(3)), updated_at = NOW(3)
WHERE id = ? AND recipient_account_user_id = ?;

-- name: MarkNotificationRead :exec
UPDATE notification
SET seen_at = COALESCE(seen_at, NOW(3)), read_at = COALESCE(read_at, NOW(3)), updated_at = NOW(3)
WHERE id = ? AND recipient_account_user_id = ?;

-- name: MarkNotificationDismissed :exec
UPDATE notification
SET dismissed_at = COALESCE(dismissed_at, NOW(3)), updated_at = NOW(3)
WHERE id = ? AND recipient_account_user_id = ?;

-- name: MarkAllNotificationsSeen :execrows
UPDATE notification
SET seen_at = NOW(3), updated_at = NOW(3)
WHERE recipient_account_user_id = ? AND seen_at IS NULL;

-- name: DismissNotificationsByConversation :execrows
-- Auto-withdraw a recipient's bell notifications for a conversation once they read it in-thread, so
-- already-seen chat alerts don't stack in the bell. Marks seen+read+dismissed in one pass.
UPDATE notification
SET seen_at = COALESCE(seen_at, NOW(3)),
    read_at = COALESCE(read_at, NOW(3)),
    dismissed_at = COALESCE(dismissed_at, NOW(3)),
    updated_at = NOW(3)
WHERE recipient_account_user_id = ? AND conversation_id = ? AND dismissed_at IS NULL;

-- name: DismissNotificationsBySourceMessage :exec
-- Withdraws (dismisses) every bell notification that projected a now-deleted message (§12.7), across
-- all recipients in the account.
UPDATE notification
SET dismissed_at = COALESCE(dismissed_at, NOW(3)), updated_at = NOW(3)
WHERE account_id = ? AND source_message_id = ? AND dismissed_at IS NULL;
