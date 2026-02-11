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
