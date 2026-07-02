-- name: PurgeActionedNotifications :execrows
-- Deletes notifications the recipient has already read or dismissed, once they are older than
-- the retention window (measured from creation). Served by notification_purge_idx
-- (created_at, read_at, dismissed_at): a leading created_at range bounds the scan to old rows,
-- with the read/dismissed check applied as a covered residual. Batched via LIMIT.
DELETE FROM notification
WHERE created_at < (NOW(3) - INTERVAL ? HOUR)
  AND (read_at IS NOT NULL OR dismissed_at IS NOT NULL)
LIMIT ?;

-- name: PurgeStaleNotifications :execrows
-- Hard cap: deletes any notification older than the maximum retention window regardless of
-- read/dismissed state, so unattended feeds can't grow without bound. Served by the leading
-- created_at column of notification_purge_idx.
DELETE FROM notification
WHERE created_at < (NOW(3) - INTERVAL ? HOUR)
LIMIT ?;

-- name: PurgeExpiredAnnouncements :execrows
-- Deletes announcements that expired longer ago than the retention window.
DELETE FROM announcement
WHERE expires_at IS NOT NULL AND expires_at < (NOW(3) - INTERVAL ? HOUR)
LIMIT ?;

-- name: PurgeOrphanedAnnouncementReceipts :execrows
-- Cleans up receipts whose announcement has been purged (no FK cascade exists).
DELETE FROM announcement_receipt
WHERE announcement_id NOT IN (SELECT id FROM announcement)
LIMIT ?;

-- name: ListPurgeableMessageAttachments :many
-- Attachments belonging to messages tombstoned longer than the retention window. The reaper deletes
-- the S3 object (when present) before deleting the row, so the object delete is re-attempted rather
-- than orphaned if a crash happens between the two steps. Served by mgah_message_idx joining
-- message_purge_idx (deleted_at).
SELECT a.id AS id, a.s3_key AS s3_key
FROM message_attachment a
JOIN message m ON m.id = a.message_id
JOIN conversation c ON c.id = m.conversation_id
WHERE m.deleted_at IS NOT NULL
  AND m.deleted_at < (NOW(3) - INTERVAL ? HOUR)
  AND c.legal_hold = 0
LIMIT ?;

-- name: DeleteMessageAttachmentByID :exec
-- Deletes a single attachment row after its S3 object has been removed.
DELETE FROM message_attachment WHERE id = ?;

-- name: PurgeTombstonedMessages :execrows
-- Hard-deletes messages tombstoned longer than the retention window, but only once all their
-- attachments are gone (attachments are purged first so their S3 objects are deleted). Messages in
-- conversations under legal hold are exempt until the hold clears. Served by message_purge_idx
-- (deleted_at).
DELETE FROM message
WHERE deleted_at IS NOT NULL
  AND deleted_at < (NOW(3) - INTERVAL ? HOUR)
  AND NOT EXISTS (SELECT 1 FROM conversation c WHERE c.id = message.conversation_id AND c.legal_hold = 1)
  AND NOT EXISTS (SELECT 1 FROM message_attachment WHERE message_attachment.message_id = message.id)
LIMIT ?;
