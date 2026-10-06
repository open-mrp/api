-- name: InsertDeletedRecord :exec
INSERT INTO deleted_record (
    resource_type,
    resource_id,
    data
) VALUES (
    sqlc.arg('resource_type'),
    sqlc.arg('resource_id'),
    sqlc.arg('data')
);

-- name: CountDeletedRecordsInAccount :one
-- Only a snapshot that records its owner under account_id can match.
SELECT COUNT(*)
FROM deleted_record
WHERE resource_type = sqlc.arg('resource_type')
AND resource_id = sqlc.arg('resource_id')
AND data->>'account_id' = sqlc.arg('account_id')::text;
