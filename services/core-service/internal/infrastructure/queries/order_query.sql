-- name: FindOrderIDByProductionRun :one
SELECT id
FROM sales_order
WHERE owner_account_id = sqlc.arg('account_id')
  AND production_run_id = sqlc.arg('production_run_id')
LIMIT 1;
