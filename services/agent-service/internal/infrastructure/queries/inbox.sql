-- name: TryInsertInboxRecord :one
INSERT INTO message_inbox (message_id, service_name, handler, message_type, request_id, parent_message_id)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id;

-- name: GetInboxRecordByMessageAndHandler :one
SELECT id, message_id, service_name, handler, message_type, request_id, parent_message_id, status, attempts, last_error, received_at, processed_at
FROM message_inbox
WHERE message_id = $1 AND handler = $2;

-- name: MarkInboxRecordProcessed :exec
UPDATE message_inbox
SET status = 'processed', processed_at = now()
WHERE id = $1;

-- name: MarkInboxRecordFailed :exec
UPDATE message_inbox
SET attempts = attempts + 1, last_error = $1
WHERE id = $2;

-- name: PurgeProcessedInboxMessages :execrows
WITH rows AS (
    SELECT id FROM message_inbox
    WHERE status = 'processed' AND processed_at < now() - ($1 || ' hours')::interval
    LIMIT $2
)
DELETE FROM message_inbox
WHERE id IN (SELECT id FROM rows);
