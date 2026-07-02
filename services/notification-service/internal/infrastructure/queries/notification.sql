-- name: CreateNotification :exec
INSERT INTO notification (
    id, account_id, recipient_account_user_id, category,
    source_message_id, conversation_id, title, body,
    template_key, template_params, link_resource_type, link_resource_id,
    sender_type, sender_id, sender_name,
    priority, seen_at, read_at, dismissed_at, metadata, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, NOW(3), NOW(3));

-- name: GetNotificationByID :one
SELECT * FROM notification
WHERE id = ? AND recipient_account_user_id = ?;

-- name: ListNotifications :many
-- Keyset-paginated notification feed (created_at DESC, id DESC), driven by
-- nf_recipient_feed_idx for the default (non-dismissed) feed,
-- nf_recipient_category_created_idx for the category filter, and the
-- (created_at DESC, id DESC) tail of nf_recipient_sender_idx for the
-- sender_type/sender_id filters — each carries the sort key inline, so the
-- common paths are sort-free.
-- ACCEPTED RESIDUAL (performant-list-endpoint-patterns.md, residual-documentation):
-- the status filter (unseen/seen/read/dismissed) resolves to nullable-timestamp
-- predicates (seen_at/read_at/dismissed_at IS [NOT] NULL) that no covering index
-- carries as a leading sort-free key, so a sparse-state status page can still
-- filesort. This is bounded by the per-recipient row count and the LIMIT, and is
-- accepted rather than carrying a derived low-cardinality status_code column +
-- index. Revisit if status-filtered pages become hot. See notes for the rewrite.
SELECT * FROM notification
WHERE recipient_account_user_id = sqlc.arg('recipient_account_user_id')
  AND (sqlc.narg('category') IS NULL OR category = sqlc.narg('category'))
  AND (
    (sqlc.narg('status') IS NULL AND dismissed_at IS NULL)
    OR (sqlc.narg('status') = 'unseen' AND seen_at IS NULL AND dismissed_at IS NULL)
    OR (sqlc.narg('status') = 'seen' AND seen_at IS NOT NULL AND read_at IS NULL AND dismissed_at IS NULL)
    OR (sqlc.narg('status') = 'read' AND read_at IS NOT NULL AND dismissed_at IS NULL)
    OR (sqlc.narg('status') = 'dismissed' AND dismissed_at IS NOT NULL)
  )
  AND (sqlc.narg('search') IS NULL OR title LIKE sqlc.narg('search') OR body LIKE sqlc.narg('search'))
  AND (sqlc.arg('include_sender_filter') = 0 OR sender_id IN (sqlc.slice('sender_ids')))
  AND (sqlc.arg('include_sender_type_filter') = 0 OR sender_type IN (sqlc.slice('sender_types')))
  AND (
    sqlc.narg('cursor_created_at') IS NULL
    OR created_at < sqlc.narg('cursor_created_at')
    OR (created_at = sqlc.narg('cursor_created_at') AND id < sqlc.narg('cursor_id'))
  )
ORDER BY created_at DESC, id DESC
LIMIT ?;

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
