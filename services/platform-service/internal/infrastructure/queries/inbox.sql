-- name: TryInsertInboxRecord :execresult
INSERT INTO message_inbox (message_id, service_name, handler, message_type, request_id, parent_message_id, lock_owner, lock_expires_at)
VALUES (
    sqlc.arg('message_id'),
    sqlc.arg('service_name'),
    sqlc.arg('handler'),
    sqlc.arg('message_type'),
    sqlc.arg('request_id'),
    sqlc.arg('parent_message_id'),
    sqlc.arg('lock_owner'),
    DATE_ADD(NOW(3), INTERVAL sqlc.arg('lock_duration_seconds') SECOND)
);

-- name: GetInboxRecordByMessageAndHandler :one
SELECT id, message_id, service_name, handler, message_type, request_id, parent_message_id, status, attempts, last_error, received_at, processed_at, failed_at, lock_owner, lock_expires_at
FROM message_inbox
WHERE message_id = ? AND handler = ?;

-- Conditional on the lease still being free, so two consumers racing the same abandoned record cannot both proceed to the handler.
-- name: ClaimInboxRecord :execrows
UPDATE message_inbox
SET lock_owner = sqlc.arg('lock_owner'),
    lock_expires_at = DATE_ADD(NOW(3), INTERVAL sqlc.arg('lock_duration_seconds') SECOND)
WHERE id = sqlc.arg('id')
  AND status = 'received'
  AND (lock_expires_at IS NULL OR lock_expires_at <= NOW(3));

-- The status guard is what makes this safe to call inside a handler's own transaction: a second attempt's UPDATE blocks on the row lock, then matches zero rows once the winner commits, so the loser's work rolls back instead of double-applying.
-- name: CompleteInboxRecord :execrows
UPDATE message_inbox
SET status = 'processed', processed_at = NOW(3), lock_owner = NULL, lock_expires_at = NULL
WHERE id = ? AND status = 'received';

-- name: MarkInboxRecordFailed :exec
UPDATE message_inbox
SET attempts = attempts + 1,
    last_error = sqlc.arg('last_error'),
    failed_at = NOW(3),
    lock_owner = NULL,
    lock_expires_at = NULL
WHERE id = sqlc.arg('id');

-- processed_at is stamped so the existing retention purge and its index cover discarded rows too; status is what distinguishes work that was dropped from work that was applied.
-- name: MarkInboxRecordDiscarded :exec
UPDATE message_inbox
SET status = 'discarded',
    last_error = sqlc.arg('last_error'),
    failed_at = NOW(3),
    processed_at = NOW(3),
    lock_owner = NULL,
    lock_expires_at = NULL
WHERE id = sqlc.arg('id');

-- name: PurgeProcessedInboxMessages :execresult
DELETE FROM message_inbox
WHERE status IN ('processed', 'discarded') AND processed_at < DATE_SUB(NOW(3), INTERVAL ? HOUR)
LIMIT ?;

-- Two index-friendly branches rather than one OR across statuses, which would not use message_inbox_alert_scan_idx.
-- A 'received' row is only stuck if nothing holds its lease; a live lease means an attempt is legitimately still running.
-- name: ListUnalertedInboxFailures :many
SELECT id, message_id, service_name, handler, message_type, status, attempts, last_error, received_at
FROM (
    SELECT id, message_id, service_name, handler, message_type, status, attempts, last_error, received_at
    FROM message_inbox
    WHERE status = 'received'
      AND alerted_at IS NULL
      AND (lock_expires_at IS NULL OR lock_expires_at <= NOW(3))
      AND (last_error IS NOT NULL OR received_at < DATE_SUB(NOW(3), INTERVAL sqlc.arg('crash_stuck_minutes') MINUTE))
    UNION ALL
    SELECT id, message_id, service_name, handler, message_type, status, attempts, last_error, received_at
    FROM message_inbox
    WHERE status = 'discarded'
      AND alerted_at IS NULL
) failures
ORDER BY id ASC
LIMIT ?;

-- name: MarkInboxRecordsAlerted :exec
UPDATE message_inbox
SET alerted_at = NOW(3)
WHERE id IN (sqlc.slice('ids'));
