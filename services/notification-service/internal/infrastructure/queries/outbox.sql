-- name: CreateOutboxMessage :execlastid
INSERT INTO message_outbox (
    message_id, service_name, message_type, destination, routing_key,
    headers, payload, status, max_attempts, next_run_at, request_id, parent_message_id
) VALUES (?, ?, ?, ?, ?, ?, ?, 'pending', ?, NOW(3), ?, ?);

-- name: SelectOutboxMessageIDsForLock :many
SELECT id FROM message_outbox
WHERE status = 'pending'
  AND next_run_at <= NOW(3)
  AND (locked_at IS NULL OR lock_expires_at < NOW(3))
  AND attempts < max_attempts
ORDER BY next_run_at ASC, id ASC
LIMIT ?;

-- name: LockOutboxMessagesByIDs :exec
UPDATE message_outbox FORCE INDEX (PRIMARY)
SET locked_at = NOW(3),
    lock_owner = sqlc.arg('lock_owner'),
    lock_expires_at = DATE_ADD(NOW(3), INTERVAL sqlc.arg('lock_duration_seconds') SECOND),
    updated_at = NOW(3)
WHERE id IN (sqlc.slice('ids'))
  AND status = 'pending'
  AND next_run_at <= NOW(3)
  AND (locked_at IS NULL OR lock_expires_at < NOW(3))
  AND attempts < max_attempts
ORDER BY id ASC;

-- name: GetLockedOutboxMessagesByIDs :many
SELECT * FROM message_outbox FORCE INDEX (PRIMARY)
WHERE id IN (sqlc.slice('ids'))
  AND lock_owner = sqlc.arg('lock_owner')
  AND lock_expires_at > NOW(3)
ORDER BY id ASC;

-- name: MarkOutboxMessagesPublished :exec
UPDATE message_outbox
SET status = 'published', published_at = NOW(3),
    locked_at = NULL, lock_owner = NULL, lock_expires_at = NULL,
    updated_at = NOW(3)
WHERE id IN (sqlc.slice('ids'));

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

-- name: SelectExpiredOutboxLockIDs :many
SELECT id FROM message_outbox
WHERE lock_expires_at < NOW(3)
ORDER BY id ASC
LIMIT ?;

-- name: CleanupExpiredOutboxLocksByIDs :execresult
UPDATE message_outbox FORCE INDEX (PRIMARY)
SET locked_at = NULL, lock_owner = NULL, lock_expires_at = NULL, updated_at = NOW(3)
WHERE id IN (sqlc.slice('ids'))
  AND lock_expires_at < NOW(3)
ORDER BY id ASC;

-- name: SelectPublishedOutboxMessageIDsForPurge :many
SELECT id FROM message_outbox
WHERE status = 'published' AND published_at < DATE_SUB(NOW(3), INTERVAL sqlc.arg('retention_hours') HOUR)
ORDER BY published_at ASC, id ASC
LIMIT ?;

-- name: PurgePublishedOutboxMessagesByIDs :execresult
DELETE FROM message_outbox
WHERE id IN (sqlc.slice('ids'))
  AND status = 'published'
  AND published_at < DATE_SUB(NOW(3), INTERVAL sqlc.arg('retention_hours') HOUR)
ORDER BY id ASC;
