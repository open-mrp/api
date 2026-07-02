-- name: CreateAnnouncement :exec
INSERT INTO announcement (
    id, scope, account_id, category, template_key, template_params,
    title, body, link_resource_type, link_resource_id, priority,
    audience, publish_at, expires_at, created_by, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, NOW(3), NOW(3));

-- name: GetActiveAnnouncementByID :one
-- Fetches a single active announcement visible to the caller, joined with the caller's
-- per-user receipt (seen/read/dismissed) state.
SELECT
    a.*,
    r.seen_at AS receipt_seen_at,
    r.read_at AS receipt_read_at,
    r.dismissed_at AS receipt_dismissed_at
FROM announcement a
LEFT JOIN announcement_receipt r
    ON r.announcement_id = a.id AND r.account_user_id = sqlc.arg('account_user_id')
WHERE a.id = sqlc.arg('id')
  AND (a.scope = 'platform' OR (a.scope = 'account' AND a.account_id = sqlc.arg('account_id')))
  AND a.publish_at <= NOW(3)
  AND (a.expires_at IS NULL OR a.expires_at > NOW(3));

-- name: ListActiveAnnouncements :many
-- Lists active announcements visible to the caller (platform-wide plus this account's),
-- excluding ones the caller has dismissed, joined with per-user receipt state. Keyset
-- paginated by (publish_at, id) descending.
SELECT
    a.*,
    r.seen_at AS receipt_seen_at,
    r.read_at AS receipt_read_at,
    r.dismissed_at AS receipt_dismissed_at
FROM announcement a
LEFT JOIN announcement_receipt r
    ON r.announcement_id = a.id AND r.account_user_id = sqlc.arg('account_user_id')
WHERE (a.scope = 'platform' OR (a.scope = 'account' AND a.account_id = sqlc.arg('account_id')))
  AND a.publish_at <= NOW(3)
  AND (a.expires_at IS NULL OR a.expires_at > NOW(3))
  AND r.dismissed_at IS NULL
  AND (
    sqlc.narg('cursor_publish_at') IS NULL
    OR a.publish_at < sqlc.narg('cursor_publish_at')
    OR (a.publish_at = sqlc.narg('cursor_publish_at') AND a.id < sqlc.narg('cursor_id'))
  )
ORDER BY a.publish_at DESC, a.id DESC
LIMIT ?;

-- name: CountUnseenAnnouncements :one
-- Counts active announcements the caller has neither seen nor dismissed.
SELECT COUNT(*)
FROM announcement a
LEFT JOIN announcement_receipt r
    ON r.announcement_id = a.id AND r.account_user_id = sqlc.arg('account_user_id')
WHERE (a.scope = 'platform' OR (a.scope = 'account' AND a.account_id = sqlc.arg('account_id')))
  AND a.publish_at <= NOW(3)
  AND (a.expires_at IS NULL OR a.expires_at > NOW(3))
  AND r.seen_at IS NULL
  AND r.dismissed_at IS NULL;

-- name: UpsertAnnouncementSeen :exec
INSERT INTO announcement_receipt (id, announcement_id, account_user_id, seen_at, created_at)
VALUES (?, ?, ?, NOW(3), NOW(3))
ON DUPLICATE KEY UPDATE seen_at = COALESCE(seen_at, NOW(3));

-- name: UpsertAnnouncementRead :exec
INSERT INTO announcement_receipt (id, announcement_id, account_user_id, seen_at, read_at, created_at)
VALUES (?, ?, ?, NOW(3), NOW(3), NOW(3))
ON DUPLICATE KEY UPDATE seen_at = COALESCE(seen_at, NOW(3)), read_at = COALESCE(read_at, NOW(3));

-- name: UpsertAnnouncementDismissed :exec
INSERT INTO announcement_receipt (id, announcement_id, account_user_id, dismissed_at, created_at)
VALUES (?, ?, ?, NOW(3), NOW(3))
ON DUPLICATE KEY UPDATE dismissed_at = COALESCE(dismissed_at, NOW(3));
