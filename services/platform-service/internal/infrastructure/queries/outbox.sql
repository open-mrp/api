-- name: AcquireOutboxMessages :exec
UPDATE message_outbox
SET locked_at = NOW(3),
    lock_owner = ?,
    lock_expires_at = DATE_ADD(NOW(3), INTERVAL ? SECOND),
    updated_at = NOW(3)
WHERE status = 'pending'
  AND next_run_at <= NOW(3)
  AND (locked_at IS NULL OR lock_expires_at < NOW(3))
  AND attempts < max_attempts
ORDER BY next_run_at ASC
LIMIT ?;

-- name: GetLockedOutboxMessages :many
SELECT * FROM message_outbox
WHERE lock_owner = ? AND lock_expires_at > NOW(3)
ORDER BY next_run_at ASC;

-- name: MarkOutboxMessagePublished :exec
UPDATE message_outbox
SET status = 'published', published_at = NOW(3),
    locked_at = NULL, lock_owner = NULL, lock_expires_at = NULL,
    updated_at = NOW(3)
WHERE id = ?;

-- name: MarkOutboxMessageFailed :exec
UPDATE message_outbox
SET attempts = attempts + 1,
    last_error = ?,
    locked_at = NULL,
    lock_owner = NULL,
    lock_expires_at = NULL,
    next_run_at = DATE_ADD(NOW(3), INTERVAL ? SECOND),
    status = CASE WHEN attempts + 1 >= max_attempts THEN 'failed' ELSE 'pending' END,
    updated_at = NOW(3)
WHERE id = ?;

-- name: CleanupExpiredOutboxLocks :execresult
UPDATE message_outbox
SET locked_at = NULL, lock_owner = NULL, lock_expires_at = NULL, updated_at = NOW(3)
WHERE lock_expires_at < NOW(3)
LIMIT ?;

-- name: PurgePublishedOutboxMessages :execresult
DELETE FROM message_outbox
WHERE status = 'published' AND published_at < DATE_SUB(NOW(3), INTERVAL ? HOUR)
LIMIT ?;
