-- name: StripeEventLogExists :one
SELECT COUNT(*) > 0 AS event_exists FROM stripe_event_log
WHERE event_id = sqlc.arg('event_id')
AND object_id = sqlc.arg('object_id');

-- name: InsertStripeEventLog :exec
INSERT INTO stripe_event_log (id, event_id, object_id, event_type, created_at, updated_at)
VALUES (sqlc.arg('id'), sqlc.arg('event_id'), sqlc.arg('object_id'), sqlc.arg('event_type'), NOW(3), NOW(3));
