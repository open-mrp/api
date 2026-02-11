-- name: AcquireOutboxMessages :exec
UPDATE message_outbox
SET locked_at = NOW(),
    lock_owner = ?,
    lock_expires_at = DATE_ADD(NOW(), INTERVAL ? SECOND),
    updated_at = NOW()
WHERE status = 'pending'
  AND next_run_at <= NOW()
  AND (locked_at IS NULL OR lock_expires_at < NOW())
  AND attempts < max_attempts
ORDER BY next_run_at ASC
LIMIT ?;

-- name: GetLockedOutboxMessages :many
SELECT * FROM message_outbox
WHERE lock_owner = ? AND lock_expires_at > NOW()
ORDER BY next_run_at ASC;

-- name: DeleteOutboxMessage :exec
DELETE FROM message_outbox WHERE id = ?;

-- name: MarkOutboxMessageFailed :exec
UPDATE message_outbox
SET attempts = attempts + 1,
    last_error = ?,
    locked_at = NULL,
    lock_owner = NULL,
    lock_expires_at = NULL,
    next_run_at = DATE_ADD(NOW(), INTERVAL POW(2, attempts) SECOND),
    status = CASE WHEN attempts + 1 >= max_attempts THEN 'failed' ELSE 'pending' END,
    updated_at = NOW()
WHERE id = ?;

-- name: CleanupExpiredOutboxLocks :execresult
UPDATE message_outbox
SET locked_at = NULL, lock_owner = NULL, lock_expires_at = NULL, updated_at = NOW()
WHERE lock_expires_at < NOW();
