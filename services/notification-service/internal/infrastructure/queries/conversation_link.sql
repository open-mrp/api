-- name: CreateConversationLink :exec
-- Adds a secondary business-record link to a conversation. Idempotent on the unique
-- (conversation_id, resource_type, resource_id) — a duplicate insert is mapped to a conflict.
INSERT INTO conversation_link (
    id, account_id, conversation_id, resource_type, resource_id, created_by_participant_id, created_at
) VALUES (?, ?, ?, ?, ?, ?, NOW(3));

-- name: DeleteConversationLink :execrows
DELETE FROM conversation_link
WHERE id = ? AND conversation_id = ? AND account_id = ?;

-- name: ListConversationLinks :many
SELECT * FROM conversation_link
WHERE conversation_id = ? AND account_id = ?
ORDER BY created_at ASC, id ASC;
