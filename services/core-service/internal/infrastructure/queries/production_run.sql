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

-- name: RunHasScannedBatches :one
-- A scanned batch has moved inventory, which deleting its row does not undo.
SELECT EXISTS (
    SELECT 1 FROM batch b
    WHERE b.account_id = sqlc.arg('account_id')
      AND b.production_run_id = sqlc.arg('production_run_id')
      AND b.scanned_at IS NOT NULL
) AS has_scanned;

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

-- HighestNumericProductionRunNumber is the highest all-digit run number the account uses that the
-- counter (an INT) could also hand out. It is read when the counter is created or turns out to be behind
-- a run renamed or imported ahead of it, never on the usual allocation. A plain read: it locks nothing,
-- where seeding the counter with INSERT ... SELECT MAX share-locked every run of the account on every
-- allocation and deadlocked run creation against scans starting runs.
--
-- Runs imported with a prefixed number ('PR-FC-001') are not part of the series.
-- name: HighestNumericProductionRunNumber :one
SELECT CAST(COALESCE(MAX(CAST(number AS UNSIGNED)), 0) AS SIGNED) AS highest
FROM production_run
WHERE account_id = sqlc.arg('account_id')
AND number REGEXP '^[0-9]{1,10}$'
AND CAST(number AS UNSIGNED) < 2147483647;

-- RaiseProductionRunNumberCounter moves the counter up to value, creating it there if the account has
-- none. It never moves it down.
-- name: RaiseProductionRunNumberCounter :exec
INSERT INTO sys_property (id, account_id, sys_property_type_code, value, created_at, updated_at)
VALUES (sqlc.arg('id'), sqlc.arg('account_id'), 'production_run_number', sqlc.arg('value'), NOW(3), NOW(3))
ON DUPLICATE KEY UPDATE value = GREATEST(value, sqlc.arg('value')), updated_at = NOW(3);

-- name: SetBatchProductionRunID :exec
UPDATE batch SET production_run_id = sqlc.arg('production_run_id'), updated_at = NOW(3)
WHERE id = sqlc.arg('id') AND account_id = sqlc.arg('account_id');

-- name: ListBatchesByIDs :many
-- The bulk form of GetBatch; same columns, so rows convert to GetBatchRow.
-- A search hydrates a run's whole flow, thousands of ids, and past a few hundred the planner scans the
-- table instead; FORCE INDEX keeps each id a primary-key lookup. The flow walk's other by-id reads
-- below are pinned to their keys for the same reason.
SELECT
    b.id,
    b.account_id,
    b.closed_at,
    b.scanned_at,
    b.created_at,
    b.updated_at,
    b.production_run_id,
    i.id AS item_id,
    i.sku AS item_sku,
    i.description AS item_description,
    q.id AS quantity_id,
    q.value AS quantity_value,
    qu.id AS quantity_unit_id,
    qu.abbreviation AS quantity_unit_abbreviation,
    qu.unit_dimension_code AS quantity_unit_type,
    sq.id AS seconds_quantity_id,
    sq.value AS seconds_quantity_value,
    su.id AS seconds_unit_id,
    su.abbreviation AS seconds_unit_abbreviation,
    su.unit_dimension_code AS seconds_unit_type,
    wq.id AS waste_quantity_id,
    wq.value AS waste_quantity_value,
    wu.id AS waste_unit_id,
    wu.abbreviation AS waste_unit_abbreviation,
    wu.unit_dimension_code AS waste_unit_type,
    ss.id AS scanning_station_id,
    ss.name AS scanning_station_name,
    d.id AS department_id,
    d.name AS department_name,
    ps.id AS production_step_id,
    ps.name AS production_step_name,
    pr.id AS production_run_id_2,
    pr.number AS production_run_number
FROM batch b FORCE INDEX (PRIMARY)
JOIN item i ON b.item_id = i.id
JOIN quantity q ON b.quantity_id = q.id
JOIN unit qu ON q.unit_id = qu.id
LEFT JOIN quantity sq ON b.seconds_quantity_id = sq.id
LEFT JOIN unit su ON sq.unit_id = su.id
LEFT JOIN quantity wq ON b.waste_quantity_id = wq.id
LEFT JOIN unit wu ON wq.unit_id = wu.id
LEFT JOIN scanning_station ss ON b.scanning_station_id = ss.id
LEFT JOIN department d ON ss.department_id = d.id
LEFT JOIN production_step ps ON b.production_step_id = ps.id
LEFT JOIN production_run pr ON b.production_run_id = pr.id
WHERE b.id IN (sqlc.slice('ids'))
AND b.account_id = sqlc.arg('account_id');

-- name: ListMachinesForBatches :many
SELECT
    bm.A AS batch_id,
    m.id,
    m.name,
    m.serial_number
FROM _batches_machines bm FORCE INDEX (_batches_machines_AB_unique)
JOIN machine m ON bm.B = m.id
WHERE bm.A IN (sqlc.slice('batch_ids'));

-- name: ListLotsForBatches :many
-- The lot numbers each batch consumed, directly or through allocated receipts.
SELECT DISTINCT ii.batch_id, l.lot_number, 'material' AS lot_type
FROM inventory_issue ii FORCE INDEX (inventory_issue_batch_id_idx)
JOIN lot l ON ii.lot_id = l.id
WHERE ii.batch_id IN (sqlc.slice('issued_batch_ids'))
AND l.lot_number IS NOT NULL
UNION
SELECT DISTINCT ii.batch_id, l.lot_number, 'material' AS lot_type
FROM inventory_issue ii FORCE INDEX (inventory_issue_batch_id_idx)
JOIN inventory_allocation ia ON ia.inventory_issue_id = ii.id
JOIN inventory_receipt ir ON ia.inventory_receipt_id = ir.id
JOIN lot l ON ir.lot_id = l.id
WHERE ii.batch_id IN (sqlc.slice('allocated_batch_ids'))
AND l.lot_number IS NOT NULL;

-- name: ListBatchFlowEdgesForBatches :many
-- Every _batch_flow edge touching the given batches. A is downstream, B upstream.
SELECT bf.A AS downstream_id, bf.B AS upstream_id
FROM _batch_flow bf FORCE INDEX (_batch_flow_AB_unique)
WHERE bf.A IN (sqlc.slice('downstream_ids'))
UNION
SELECT bf2.A AS downstream_id, bf2.B AS upstream_id
FROM _batch_flow bf2 FORCE INDEX (_batch_flow_B_index)
WHERE bf2.B IN (sqlc.slice('upstream_ids'));

-- name: ListProductionRunBatchSummaries :many
-- The runs' planned output, totalled per item and unit.
SELECT
    b.production_run_id,
    i.id AS item_id,
    i.sku AS item_sku,
    u.id AS unit_id,
    u.abbreviation AS unit_abbreviation,
    CAST(SUM(q.value) AS CHAR) AS quantity_value,
    COUNT(*) AS batch_count
FROM batch b
JOIN item i ON b.item_id = i.id
JOIN quantity q ON b.quantity_id = q.id
JOIN unit u ON q.unit_id = u.id
WHERE b.account_id = sqlc.arg('account_id')
AND b.production_run_id IN (sqlc.slice('production_run_ids'))
GROUP BY b.production_run_id, i.id, i.sku, u.id, u.abbreviation
ORDER BY b.production_run_id, i.sku, u.abbreviation;

-- name: ListRunBatchTraversal :many
-- A run's own batches with what the flow walk and pagination need, and nothing more.
SELECT b.id, b.closed_at, b.created_at
FROM batch b
WHERE b.production_run_id = sqlc.arg('production_run_id')
AND b.account_id = sqlc.arg('account_id');

-- name: ListBatchTraversalByIDs :many
-- Closed state and age of the given batches. A batch outside the account is simply absent.
SELECT b.id, b.closed_at, b.created_at
FROM batch b FORCE INDEX (PRIMARY)
WHERE b.id IN (sqlc.slice('ids'))
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

