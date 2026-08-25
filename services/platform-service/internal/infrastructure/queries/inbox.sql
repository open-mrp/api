-- name: TryInsertInboxRecord :execresult
INSERT INTO message_inbox (message_id, service_name, handler, message_type, request_id, parent_message_id)
VALUES (?, ?, ?, ?, ?, ?);

-- name: GetInboxRecordByMessageAndHandler :one
SELECT id, message_id, service_name, handler, message_type, request_id, parent_message_id, status, attempts, last_error, received_at, processed_at
FROM message_inbox
WHERE message_id = ? AND handler = ?;

-- name: MarkInboxRecordProcessed :exec
UPDATE message_inbox
SET status = 'processed', processed_at = NOW(3)
WHERE id = ?;

-- name: MarkInboxRecordFailed :exec
UPDATE message_inbox
SET attempts = attempts + 1, last_error = ?
WHERE id = ?;

-- name: PurgeProcessedInboxMessages :execresult
DELETE FROM message_inbox
WHERE status = 'processed' AND processed_at < DATE_SUB(NOW(3), INTERVAL ? HOUR)
LIMIT ?;

-- name: ListUnalertedInboxFailures :many
SELECT id, message_id, service_name, handler, message_type, attempts, last_error, received_at
FROM message_inbox
WHERE status = 'received'
  AND alerted_at IS NULL
  AND (last_error IS NOT NULL OR received_at < DATE_SUB(NOW(3), INTERVAL sqlc.arg('crash_stuck_minutes') MINUTE))
ORDER BY id ASC
LIMIT ?;

-- name: MarkInboxRecordsAlerted :exec
UPDATE message_inbox
SET alerted_at = NOW(3)
WHERE id IN (sqlc.slice('ids'));
