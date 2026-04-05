-- name: CreateOutboxMessage :one
INSERT INTO message_outbox (
    message_id, service_name, message_type, destination, routing_key,
    headers, payload, status, max_attempts, next_run_at, request_id, parent_message_id
) VALUES ($1, $2, $3, $4, $5, $6, $7, 'pending', $8, now(), $9, $10)
RETURNING id;

-- name: AcquireOutboxMessages :exec
WITH rows AS (
    SELECT id FROM message_outbox
    WHERE status = 'pending'
      AND next_run_at <= now()
      AND (locked_at IS NULL OR lock_expires_at < now())
      AND attempts < max_attempts
    ORDER BY next_run_at ASC
    LIMIT $3
)
UPDATE message_outbox
SET locked_at = now(),
    lock_owner = $1,
    lock_expires_at = now() + ($2 || ' seconds')::interval,
    updated_at = now()
WHERE id IN (SELECT id FROM rows);

-- name: GetLockedOutboxMessages :many
SELECT * FROM message_outbox
WHERE lock_owner = $1 AND lock_expires_at > now()
ORDER BY next_run_at ASC;

-- name: MarkOutboxMessagePublished :exec
UPDATE message_outbox
SET status = 'published', published_at = now(),
    locked_at = NULL, lock_owner = NULL, lock_expires_at = NULL,
    updated_at = now()
WHERE id = $1;

-- name: MarkOutboxMessageFailed :exec
UPDATE message_outbox
SET attempts = attempts + 1,
    last_error = $1,
    locked_at = NULL,
    lock_owner = NULL,
    lock_expires_at = NULL,
    next_run_at = now() + ($2 || ' seconds')::interval,
    status = CASE WHEN attempts + 1 >= max_attempts THEN 'failed' ELSE 'pending' END,
    updated_at = now()
WHERE id = $3;

-- name: CleanupExpiredOutboxLocks :execrows
WITH rows AS (
    SELECT id FROM message_outbox
    WHERE lock_expires_at < now()
    LIMIT $1
)
UPDATE message_outbox
SET locked_at = NULL, lock_owner = NULL, lock_expires_at = NULL, updated_at = now()
WHERE id IN (SELECT id FROM rows);

-- name: PurgePublishedOutboxMessages :execrows
WITH rows AS (
    SELECT id FROM message_outbox
    WHERE status = 'published' AND published_at < now() - ($1 || ' hours')::interval
    LIMIT $2
)
DELETE FROM message_outbox
WHERE id IN (SELECT id FROM rows);
