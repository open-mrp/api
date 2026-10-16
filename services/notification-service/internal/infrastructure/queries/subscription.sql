-- name: GetSubscription :one
SELECT * FROM subscription
WHERE id = ? AND account_user_id = ?;

-- name: GetSubscriptionByRecord :one
SELECT * FROM subscription
WHERE account_user_id = ? AND record_id = ?;

-- name: InsertSubscription :exec
-- An existing row for the person and record wins: an unsubscribed row keeps automatic subscription out.
INSERT IGNORE INTO subscription (id, account_id, account_user_id, record_type, record_id, reason, status, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, NOW(3), NOW(3));

-- name: SetSubscriptionStatus :exec
UPDATE subscription
SET status = ?, reason = ?, updated_at = NOW(3)
WHERE id = ? AND account_user_id = ?;

-- name: ListRecordSubscribers :many
SELECT account_user_id FROM subscription
WHERE account_id = ? AND record_type = ? AND record_id = ? AND status = 'subscribed';

-- name: RecordHasSubscriptions :one
SELECT EXISTS (
  SELECT 1 FROM subscription WHERE account_id = ? AND record_type = ? AND record_id = ?
) AS has_rows;
