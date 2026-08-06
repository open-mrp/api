-- name: ListProductionRunsForward :many
SELECT
    pr.id,
    pr.number,
    pr.responsible_user_id,
    au.id AS responsible_account_user_id,
    COALESCE(u.name, au.id, '') AS responsible_user_name,
    au.status_code AS responsible_user_status_code,
    au.created_at AS responsible_user_created_at,
    au.updated_at AS responsible_user_updated_at,
    pr.started_at,
    pr.completed_at,
    pr.created_at,
    pr.updated_at,
    COUNT(DISTINCT b.id) AS batch_count
FROM production_run pr
-- responsible_user_id may store either an account_user id or a legacy user
-- id; match both, scoped to the run's account.
LEFT JOIN account_user au ON au.account_id = pr.account_id AND (au.id = pr.responsible_user_id OR au.user_id = pr.responsible_user_id)
LEFT JOIN user u ON u.id = au.user_id
LEFT JOIN batch b ON b.production_run_id = pr.id AND b.account_id = pr.account_id
WHERE pr.account_id = sqlc.arg('account_id')
AND (
    sqlc.narg('search_query') IS NULL
    OR pr.number LIKE sqlc.narg('search_query')
    OR EXISTS (
        SELECT 1 FROM batch bq
        WHERE bq.production_run_id = pr.id
        AND bq.account_id = pr.account_id
        AND bq.id LIKE sqlc.narg('batch_id_query')
    )
)
AND (
    sqlc.arg('include_status_filter') = false
    OR (sqlc.arg('status_open') = true AND pr.completed_at IS NULL)
    OR (sqlc.arg('status_closed') = true AND pr.completed_at IS NOT NULL)
)
AND (
    sqlc.arg('include_item_filter') = false
    OR EXISTS (
        SELECT 1 FROM batch b2
        WHERE b2.production_run_id = pr.id
        AND b2.account_id = pr.account_id
        AND b2.item_id IN (sqlc.slice('item_ids'))
    )
)
AND (
    sqlc.arg('include_machine_filter') = false
    OR EXISTS (
        SELECT 1 FROM batch b3
        JOIN _batches_machines bm ON bm.A = b3.id
        WHERE b3.production_run_id = pr.id
        AND b3.account_id = pr.account_id
        AND bm.B IN (sqlc.slice('machine_ids'))
    )
)
AND (
    sqlc.narg('start_date') IS NULL
    OR pr.created_at >= sqlc.narg('start_date')
)
AND (
    sqlc.narg('end_date') IS NULL
    OR pr.created_at <= sqlc.narg('end_date')
)
AND (
    sqlc.narg('cursor_created_at') IS NULL
    OR pr.created_at < sqlc.narg('cursor_created_at')
    OR (pr.created_at = sqlc.narg('cursor_created_at') AND pr.id < sqlc.narg('cursor_id'))
)
GROUP BY pr.id, au.id
ORDER BY pr.created_at DESC, pr.id DESC
LIMIT ?;

-- name: ListProductionRunsBackward :many
SELECT
    pr.id,
    pr.number,
    pr.responsible_user_id,
    au.id AS responsible_account_user_id,
    COALESCE(u.name, au.id, '') AS responsible_user_name,
    au.status_code AS responsible_user_status_code,
    au.created_at AS responsible_user_created_at,
    au.updated_at AS responsible_user_updated_at,
    pr.started_at,
    pr.completed_at,
    pr.created_at,
    pr.updated_at,
    COUNT(DISTINCT b.id) AS batch_count
FROM production_run pr
-- responsible_user_id may store either an account_user id or a legacy user
-- id; match both, scoped to the run's account.
LEFT JOIN account_user au ON au.account_id = pr.account_id AND (au.id = pr.responsible_user_id OR au.user_id = pr.responsible_user_id)
LEFT JOIN user u ON u.id = au.user_id
LEFT JOIN batch b ON b.production_run_id = pr.id AND b.account_id = pr.account_id
WHERE pr.account_id = sqlc.arg('account_id')
AND (
    sqlc.narg('search_query') IS NULL
    OR pr.number LIKE sqlc.narg('search_query')
    OR EXISTS (
        SELECT 1 FROM batch bq
        WHERE bq.production_run_id = pr.id
        AND bq.account_id = pr.account_id
        AND bq.id LIKE sqlc.narg('batch_id_query')
    )
)
AND (
    sqlc.arg('include_status_filter') = false
    OR (sqlc.arg('status_open') = true AND pr.completed_at IS NULL)
    OR (sqlc.arg('status_closed') = true AND pr.completed_at IS NOT NULL)
)
AND (
    sqlc.arg('include_item_filter') = false
    OR EXISTS (
        SELECT 1 FROM batch b2
        WHERE b2.production_run_id = pr.id
        AND b2.account_id = pr.account_id
        AND b2.item_id IN (sqlc.slice('item_ids'))
    )
)
AND (
    sqlc.arg('include_machine_filter') = false
    OR EXISTS (
        SELECT 1 FROM batch b3
        JOIN _batches_machines bm ON bm.A = b3.id
        WHERE b3.production_run_id = pr.id
        AND b3.account_id = pr.account_id
        AND bm.B IN (sqlc.slice('machine_ids'))
    )
)
AND (
    sqlc.narg('start_date') IS NULL
    OR pr.created_at >= sqlc.narg('start_date')
)
AND (
    sqlc.narg('end_date') IS NULL
    OR pr.created_at <= sqlc.narg('end_date')
)
AND (
    sqlc.narg('cursor_created_at') IS NULL
    OR pr.created_at > sqlc.narg('cursor_created_at')
    OR (pr.created_at = sqlc.narg('cursor_created_at') AND pr.id > sqlc.narg('cursor_id'))
)
GROUP BY pr.id, au.id
ORDER BY pr.created_at ASC, pr.id ASC
LIMIT ?;

-- name: GetProductionRun :one
SELECT
    pr.id,
    pr.number,
    pr.responsible_user_id,
    au.id AS responsible_account_user_id,
    COALESCE(u.name, au.id, '') AS responsible_user_name,
    au.status_code AS responsible_user_status_code,
    au.created_at AS responsible_user_created_at,
    au.updated_at AS responsible_user_updated_at,
    pr.account_id,
    pr.started_at,
    pr.completed_at,
    pr.created_at,
    pr.updated_at,
    COUNT(DISTINCT b.id) AS batch_count
FROM production_run pr
-- responsible_user_id may store either an account_user id or a legacy user
-- id; match both, scoped to the run's account.
LEFT JOIN account_user au ON au.account_id = pr.account_id AND (au.id = pr.responsible_user_id OR au.user_id = pr.responsible_user_id)
LEFT JOIN user u ON u.id = au.user_id
LEFT JOIN batch b ON b.production_run_id = pr.id AND b.account_id = pr.account_id
WHERE pr.id = sqlc.arg('id')
AND pr.account_id = sqlc.arg('account_id')
GROUP BY pr.id, au.id;

-- name: InsertProductionRun :exec
INSERT INTO production_run (id, responsible_user_id, number, account_id, created_at, updated_at)
VALUES (sqlc.arg('id'), sqlc.arg('responsible_user_id'), sqlc.arg('number'), sqlc.arg('account_id'), NOW(3), NOW(3));

-- name: UpdateProductionRunNumber :exec
UPDATE production_run SET number = sqlc.arg('number'), updated_at = NOW(3)
WHERE id = sqlc.arg('id') AND account_id = sqlc.arg('account_id');

-- name: UpdateProductionRunResponsibleUser :exec
UPDATE production_run SET responsible_user_id = sqlc.arg('responsible_user_id'), updated_at = NOW(3)
WHERE id = sqlc.arg('id') AND account_id = sqlc.arg('account_id');

-- name: DeleteProductionRunByID :exec
DELETE FROM production_run WHERE id = sqlc.arg('id') AND account_id = sqlc.arg('account_id');

-- name: CountProductionRunsByNumber :one
SELECT COUNT(*) FROM production_run
WHERE account_id = sqlc.arg('account_id')
AND number = sqlc.arg('number')
AND (sqlc.narg('exclude_id') IS NULL OR id != sqlc.narg('exclude_id'));

-- name: IsProductionRunCompleted :one
SELECT CASE WHEN completed_at IS NOT NULL THEN true ELSE false END AS is_completed
FROM production_run
WHERE id = sqlc.arg('id') AND account_id = sqlc.arg('account_id');

-- name: DeleteBatchesByProductionRunID :exec
DELETE FROM batch WHERE production_run_id = sqlc.arg('production_run_id') AND account_id = sqlc.arg('account_id');

-- name: FindSalesOrderIDsByProductionRunID :many
SELECT id FROM sales_order
WHERE production_run_id = sqlc.arg('production_run_id')
AND owner_account_id = sqlc.arg('account_id');

-- name: UnlinkSalesOrdersFromProductionRun :exec
UPDATE sales_order SET production_run_id = NULL, updated_at = NOW(3)
WHERE production_run_id = sqlc.arg('production_run_id')
AND owner_account_id = sqlc.arg('account_id');

-- name: DeleteReservedInventoryIssuesByOrderID :exec
DELETE FROM inventory_issue
WHERE order_id = sqlc.arg('order_id')
AND account_id = sqlc.arg('account_id')
AND status_code = 'reserved';

-- name: GetNextProductionRunNumberFull :one
-- FOR UPDATE serializes concurrent allocators per account: without it two
-- transactions can read the same MAX and collide on the (account_id, number)
-- unique key.
SELECT COALESCE(MAX(CAST(number AS UNSIGNED)), 0) + 1 AS next_number
FROM production_run WHERE account_id = sqlc.arg('account_id')
FOR UPDATE;

-- AllocateNextProductionRunNumber atomically reserves the next run number for the account
-- and returns it via LAST_INSERT_ID.
--
-- The single upsert holds a row lock on the per-account counter, so concurrent creates
-- serialize instead of colliding. The old read-MAX-then-write pattern raced, which two
-- releases issued at once would hit routinely. Mirrors AllocateNextOrderNumber.
-- name: AllocateNextProductionRunNumber :execresult
INSERT INTO sys_property (id, account_id, sys_property_type_code, value, created_at, updated_at)
VALUES (sqlc.arg('id'), sqlc.arg('account_id'), 'production_run_number', LAST_INSERT_ID(1), NOW(3), NOW(3))
ON DUPLICATE KEY UPDATE value = LAST_INSERT_ID(value + 1), updated_at = NOW(3);

-- SeedProductionRunNumberCounter primes the counter from existing rows the first time an
-- account allocates, so a database that already has runs does not restart numbering at 1.
--
-- Only all-digit numbers count toward the series. Runs imported with a prefixed number
-- ('PR-FC-001') are not part of it, and casting them would fail the whole statement under
-- strict mode rather than being ignored the way a bare SELECT's warning is.
-- name: SeedProductionRunNumberCounter :exec
INSERT INTO sys_property (id, account_id, sys_property_type_code, value, created_at, updated_at)
SELECT sqlc.arg('id'), sqlc.arg('account_id'), 'production_run_number',
       COALESCE(MAX(CAST(pr.number AS UNSIGNED)), 0), NOW(3), NOW(3)
FROM production_run pr
WHERE pr.account_id = sqlc.arg('account_id')
AND pr.number REGEXP '^[0-9]+$'
ON DUPLICATE KEY UPDATE id = id;

-- name: SetBatchProductionRunID :exec
UPDATE batch SET production_run_id = sqlc.arg('production_run_id'), updated_at = NOW(3)
WHERE id = sqlc.arg('id') AND account_id = sqlc.arg('account_id');

-- name: GetBatchIDsByProductionRun :many
SELECT b.id, b.closed_at
FROM batch b
WHERE b.production_run_id = sqlc.arg('production_run_id')
AND b.account_id = sqlc.arg('account_id');

-- name: GetBatchClosedAt :one
SELECT b.closed_at
FROM batch b
WHERE b.id = sqlc.arg('id')
AND b.account_id = sqlc.arg('account_id');

-- name: ExportProductionRuns :many
-- Unpaginated by design; the caller passes a row cap as the limit. The sales
-- order is joined rather than counted, so a run without one still exports.
SELECT
    pr.id,
    pr.number,
    COALESCE(u.name, au.id, '') AS responsible_user_name,
    pr.started_at,
    pr.completed_at,
    so.id AS order_id,
    pr.created_at,
    pr.updated_at
FROM production_run pr
LEFT JOIN account_user au ON au.account_id = pr.account_id AND (au.id = pr.responsible_user_id OR au.user_id = pr.responsible_user_id)
LEFT JOIN user u ON u.id = au.user_id
LEFT JOIN sales_order so ON so.production_run_id = pr.id AND so.owner_account_id = pr.account_id
WHERE pr.account_id = sqlc.arg('account_id')
AND (
    sqlc.narg('search_query') IS NULL
    OR pr.number LIKE sqlc.narg('search_query')
)
ORDER BY pr.created_at DESC, pr.id DESC
LIMIT ?;

