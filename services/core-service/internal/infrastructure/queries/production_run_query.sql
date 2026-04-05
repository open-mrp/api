-- name: StartProductionRun :exec
UPDATE production_run SET started_at = NOW(3), updated_at = NOW(3)
WHERE id = sqlc.arg('id') AND account_id = sqlc.arg('account_id') AND started_at IS NULL;

-- name: CountUnscannedOrUndeletedBatchesByRun :one
SELECT COUNT(*) FROM batch
WHERE production_run_id = sqlc.arg('production_run_id')
AND account_id = sqlc.arg('account_id')
AND scanned_at IS NULL;

-- name: CloseProductionRun :exec
UPDATE production_run SET completed_at = NOW(3), updated_at = NOW(3)
WHERE id = sqlc.arg('id') AND account_id = sqlc.arg('account_id') AND completed_at IS NULL;

-- name: CreateProductionRun :exec
INSERT INTO production_run (id, responsible_user_id, number, account_id, created_at, updated_at)
VALUES (sqlc.arg('id'), sqlc.arg('responsible_user_id'), sqlc.arg('number'), sqlc.arg('account_id'), NOW(3), NOW(3));

-- name: GetNextProductionRunNumber :one
SELECT COALESCE(MAX(CAST(number AS UNSIGNED)), 0) + 1 AS next_number
FROM production_run WHERE account_id = sqlc.arg('account_id');
