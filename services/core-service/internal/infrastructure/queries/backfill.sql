-- name: GetBackfillProgress :one
SELECT cursor_value, completed_at
FROM backfill_progress
WHERE name = ?;

-- name: SaveBackfillProgress :exec
INSERT INTO backfill_progress (name, cursor_value, rows_done, completed_at)
VALUES (sqlc.arg('name'), sqlc.arg('cursor_value'), sqlc.arg('rows_done'), sqlc.narg('completed_at'))
ON DUPLICATE KEY UPDATE
    cursor_value = VALUES(cursor_value),
    rows_done = rows_done + VALUES(rows_done),
    completed_at = VALUES(completed_at),
    updated_at = NOW(3);
