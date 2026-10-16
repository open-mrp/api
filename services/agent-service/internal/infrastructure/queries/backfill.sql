-- name: GetBackfillProgress :one
SELECT cursor_value, completed_at
FROM backfill_progress
WHERE name = $1;

-- name: SaveBackfillProgress :exec
INSERT INTO backfill_progress (name, cursor_value, rows_done, completed_at)
VALUES (sqlc.arg('name'), sqlc.arg('cursor_value'), sqlc.arg('rows_done'), sqlc.narg('completed_at'))
ON CONFLICT (name) DO UPDATE SET
    cursor_value = EXCLUDED.cursor_value,
    rows_done = backfill_progress.rows_done + EXCLUDED.rows_done,
    completed_at = EXCLUDED.completed_at,
    updated_at = now();

-- ListAgentMemoriesAfter pages every account's memories by primary key, for copying them into notes.
-- name: ListAgentMemoriesAfter :many
SELECT id, account_id, category, content, entity_type, entity_id, importance, expires_at, created_at, updated_at
FROM agent_memory
WHERE id > sqlc.arg('after_id')
ORDER BY id
LIMIT sqlc.arg('row_limit');
