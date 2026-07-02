-- name: CreateBlock :exec
-- Idempotent: re-blocking the same pair is a no-op.
INSERT INTO messaging_block (id, account_id, blocker_account_user_id, blocked_account_user_id, created_at)
VALUES (?, ?, ?, ?, NOW(3))
ON DUPLICATE KEY UPDATE blocker_account_user_id = blocker_account_user_id;

-- name: GetBlockByPair :one
SELECT * FROM messaging_block
WHERE blocker_account_user_id = ? AND blocked_account_user_id = ?;

-- name: DeleteBlock :exec
DELETE FROM messaging_block
WHERE blocker_account_user_id = ? AND blocked_account_user_id = ?;

-- name: BlockExistsBetween :one
-- True if either user has blocked the other (DM create and send are blocked in both directions).
SELECT EXISTS(
    SELECT 1 FROM messaging_block
    WHERE (blocker_account_user_id = sqlc.arg('a') AND blocked_account_user_id = sqlc.arg('b'))
       OR (blocker_account_user_id = sqlc.arg('b') AND blocked_account_user_id = sqlc.arg('a'))
) AS blocked;

-- name: ListBlocks :many
SELECT * FROM messaging_block
WHERE blocker_account_user_id = ?
ORDER BY created_at DESC;
