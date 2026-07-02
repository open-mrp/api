-- name: UpsertSupportRoute :exec
-- Designates (or re-points) the group conversation that handles support for a scope. The unique key
-- is (account_id, relation_account_id), so a second set for the same scope updates the target.
INSERT INTO support_route (
    id, account_id, relation_account_id, group_conversation_id, created_at, updated_at
) VALUES (?, ?, ?, ?, NOW(3), NOW(3))
ON DUPLICATE KEY UPDATE group_conversation_id = VALUES(group_conversation_id), updated_at = NOW(3);

-- name: GetSupportRoute :one
SELECT * FROM support_route
WHERE account_id = ? AND relation_account_id = ?;

-- name: ResolveSupportRoute :one
-- Resolves the route for a customer: a per-relation override (relation_account_id = the customer
-- account) wins over the account-level default (relation_account_id = ''). DESC order puts the
-- non-empty override ahead of the '' default.
SELECT * FROM support_route
WHERE account_id = ? AND relation_account_id IN (?, '')
ORDER BY relation_account_id DESC
LIMIT 1;

-- name: DeleteSupportRoute :execrows
DELETE FROM support_route
WHERE account_id = ? AND relation_account_id = ?;
