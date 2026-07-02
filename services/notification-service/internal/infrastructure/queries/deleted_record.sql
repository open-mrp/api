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

-- name: CountDeletedRecordsByResourceAndResourceID :one
SELECT COUNT(*)
FROM deleted_record
WHERE resource_type = sqlc.arg('resource_type')
AND resource_id = sqlc.arg('resource_id');
