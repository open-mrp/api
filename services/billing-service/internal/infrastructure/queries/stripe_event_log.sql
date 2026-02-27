-- name: InsertStripeEventLog :exec
INSERT INTO stripe_event_log (id, event_id, event_type, object_id)
VALUES (?, ?, ?, ?);

-- name: StripeEventLogExists :one
SELECT COUNT(*) > 0 FROM stripe_event_log WHERE event_id = ? AND object_id = ?;
