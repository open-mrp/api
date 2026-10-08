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

-- name: ListRequestLogIDsAfter :many
-- Walks the narrow occurred_at index, which carries the primary key, so finding the next page never
-- reads the wide rows or their bodies into the buffer pool.
SELECT id, occurred_at
FROM request_log FORCE INDEX (request_log_occurred_at_idx)
WHERE occurred_at < sqlc.arg('before')
  AND (occurred_at > sqlc.arg('after_occurred_at')
       OR (occurred_at = sqlc.arg('after_occurred_at') AND id > sqlc.arg('after_id')))
ORDER BY occurred_at, id
LIMIT ?;

-- name: ListInlineRequestLogPayloads :many
SELECT id, query_json, request_body_json, response_body_json, stack_trace
FROM request_log
WHERE id IN (sqlc.slice('ids'))
  AND payload_key IS NULL;

-- name: PointRequestLogsAtPayloads :execrows
-- The key format matches contracts.RequestLogPayloadKey. The guard skips a row that was already
-- moved, so re-running a batch changes nothing.
UPDATE request_log
SET payload_key = CONCAT('request-logs/', id, '.json.gz'),
    query_json = NULL,
    request_body_json = NULL,
    response_body_json = NULL,
    stack_trace = NULL
WHERE id IN (sqlc.slice('ids'))
  AND payload_key IS NULL;
