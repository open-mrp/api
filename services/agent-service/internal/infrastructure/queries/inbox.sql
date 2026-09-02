-- name: TryInsertInboxRecord :one
INSERT INTO message_inbox (message_id, service_name, handler, message_type, request_id, parent_message_id, lock_owner, lock_expires_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, now() + ($8 || ' seconds')::interval)
RETURNING id;

-- name: GetInboxRecordByMessageAndHandler :one
SELECT id, message_id, service_name, handler, message_type, request_id, parent_message_id, status, attempts, last_error, received_at, processed_at, failed_at, lock_owner, lock_expires_at
FROM message_inbox
WHERE message_id = $1 AND handler = $2;

-- Conditional on the lease still being free, so two consumers racing the same abandoned record cannot both proceed to the handler.
-- name: ClaimInboxRecord :execrows
UPDATE message_inbox
SET lock_owner = $2, lock_expires_at = now() + ($3 || ' seconds')::interval
WHERE id = $1
  AND status = 'received'
  AND (lock_expires_at IS NULL OR lock_expires_at <= now());

-- The status guard is what makes this safe to call inside a handler's own transaction: a second attempt's UPDATE blocks on the row lock, then matches zero rows once the winner commits, so the loser's work rolls back instead of double-applying.
-- name: CompleteInboxRecord :execrows
UPDATE message_inbox
SET status = 'processed', processed_at = now(), lock_owner = NULL, lock_expires_at = NULL
WHERE id = $1 AND status = 'received';

-- name: MarkInboxRecordFailed :exec
UPDATE message_inbox
SET attempts = attempts + 1, last_error = $1, failed_at = now(), lock_owner = NULL, lock_expires_at = NULL
WHERE id = $2;

-- processed_at is stamped so the existing retention purge and its index cover discarded rows too; status is what distinguishes work that was dropped from work that was applied.
-- name: MarkInboxRecordDiscarded :exec
UPDATE message_inbox
SET status = 'discarded', last_error = $1, failed_at = now(), processed_at = now(), lock_owner = NULL, lock_expires_at = NULL
WHERE id = $2;

-- name: PurgeProcessedInboxMessages :execrows
WITH rows AS (
    SELECT id FROM message_inbox
    WHERE status IN ('processed', 'discarded') AND processed_at < now() - ($1 || ' hours')::interval
    LIMIT $2
)
DELETE FROM message_inbox
WHERE id IN (SELECT id FROM rows);
