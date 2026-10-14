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

-- name: ListConversationLinksPage :many
-- One keyset page of a conversation's links, oldest first.
SELECT * FROM conversation_link
WHERE conversation_id = sqlc.arg('conversation_id') AND account_id = sqlc.arg('account_id')
  AND (
    sqlc.narg('cursor_created_at') IS NULL
    OR created_at > sqlc.narg('cursor_created_at')
    OR (created_at = sqlc.narg('cursor_created_at') AND id > sqlc.narg('cursor_id'))
  )
ORDER BY created_at ASC, id ASC
LIMIT ?;
