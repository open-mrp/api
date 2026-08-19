-- name: GetBatch :one
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
FROM batch b
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
WHERE b.id = sqlc.arg('id')
AND b.account_id = sqlc.arg('account_id');

-- name: GetBatchBase :one
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
FROM batch b
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
WHERE b.id = sqlc.arg('id')
AND b.account_id = sqlc.arg('account_id');

-- GetBatchFlowOutgoing returns the downstream batches the given batch feeds.
--
-- _batch_flow follows the Prisma implicit self-m2m mapping (same rule as
-- _parent_child_production_steps, see docs/patterns/production-step-graph-patterns.md):
-- row (A, B) means B is in A's `in` relation and A is in B's `out` relation, so A = the
-- downstream (consuming) batch and B = the upstream (source) batch.
-- name: GetBatchFlowOutgoing :many
SELECT A AS batch_id FROM _batch_flow WHERE B = sqlc.arg('batch_id');

-- GetBatchFlowIncoming returns the upstream batches that feed the given batch.
-- name: GetBatchFlowIncoming :many
SELECT B AS batch_id FROM _batch_flow WHERE A = sqlc.arg('batch_id');

-- name: GetBatchFlowTraversalInfo :one
SELECT id, closed_at, scanned_at, production_step_id
FROM batch
WHERE id = sqlc.arg('id') AND account_id = sqlc.arg('account_id');

-- name: ListBatchesByScanningStationForward :many
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
    ps.id AS production_step_id,
    ps.name AS production_step_name,
    pr.id AS production_run_id_2,
    pr.number AS production_run_number
FROM batch b
JOIN item i ON b.item_id = i.id
JOIN quantity q ON b.quantity_id = q.id
JOIN unit qu ON q.unit_id = qu.id
LEFT JOIN quantity sq ON b.seconds_quantity_id = sq.id
LEFT JOIN unit su ON sq.unit_id = su.id
LEFT JOIN quantity wq ON b.waste_quantity_id = wq.id
LEFT JOIN unit wu ON wq.unit_id = wu.id
LEFT JOIN scanning_station ss ON b.scanning_station_id = ss.id
LEFT JOIN production_step ps ON b.production_step_id = ps.id
LEFT JOIN production_run pr ON b.production_run_id = pr.id
WHERE b.account_id = sqlc.arg('account_id')
AND b.scanning_station_id = sqlc.arg('scanning_station_id')
AND b.scanned_at IS NOT NULL
AND (
    sqlc.narg('search_query') IS NULL
    OR i.sku LIKE sqlc.narg('search_query')
)
AND (
    sqlc.narg('cursor_scanned_at') IS NULL
    OR b.scanned_at < sqlc.narg('cursor_scanned_at')
    OR (b.scanned_at = sqlc.narg('cursor_scanned_at') AND b.id < sqlc.narg('cursor_id'))
)
ORDER BY b.scanned_at DESC, b.id DESC
LIMIT ?;

-- name: ListBatchesByScanningStationBackward :many
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
    ps.id AS production_step_id,
    ps.name AS production_step_name,
    pr.id AS production_run_id_2,
    pr.number AS production_run_number
FROM batch b
JOIN item i ON b.item_id = i.id
JOIN quantity q ON b.quantity_id = q.id
JOIN unit qu ON q.unit_id = qu.id
LEFT JOIN quantity sq ON b.seconds_quantity_id = sq.id
LEFT JOIN unit su ON sq.unit_id = su.id
LEFT JOIN quantity wq ON b.waste_quantity_id = wq.id
LEFT JOIN unit wu ON wq.unit_id = wu.id
LEFT JOIN scanning_station ss ON b.scanning_station_id = ss.id
LEFT JOIN production_step ps ON b.production_step_id = ps.id
LEFT JOIN production_run pr ON b.production_run_id = pr.id
WHERE b.account_id = sqlc.arg('account_id')
AND b.scanning_station_id = sqlc.arg('scanning_station_id')
AND b.scanned_at IS NOT NULL
AND (
    sqlc.narg('search_query') IS NULL
    OR i.sku LIKE sqlc.narg('search_query')
)
AND (
    b.scanned_at > sqlc.arg('cursor_scanned_at')
    OR (b.scanned_at = sqlc.arg('cursor_scanned_at') AND b.id > sqlc.arg('cursor_id'))
)
ORDER BY b.scanned_at ASC, b.id ASC
LIMIT ?;

-- name: CountBatchesByScanningStation :one
SELECT COUNT(*) FROM batch b
WHERE b.account_id = sqlc.arg('account_id')
AND b.scanning_station_id = sqlc.arg('scanning_station_id')
AND b.scanned_at IS NOT NULL
AND (
    sqlc.narg('search_query') IS NULL
    OR b.item_id IN (SELECT it.id FROM item it WHERE it.sku LIKE sqlc.narg('search_query') AND it.account_id = b.account_id)
);

-- name: ListOpenBatches :many
-- The product-line filter matches via EXISTS rather than a join so an item with several products on matching lines is not double-counted by the SUM.
SELECT
    d.name AS department_name,
    i.sku AS item_name,
    i.id AS item_id,
    b.scanning_station_id,
    SUM(q.value - COALESCE((
        -- A batch's outputs are the downstream (A) side of _batch_flow rows where it is the upstream (B) batch, per the Prisma orientation of the table.
        --
        -- Correlated per batch: a grouped derived table cannot take the account filter and so aggregates the whole flow graph on every call.
        SELECT SUM(oq.value)
        FROM _batch_flow bf
        JOIN batch ob ON bf.A = ob.id
        JOIN quantity oq ON ob.quantity_id = oq.id
        WHERE bf.B = b.id
    ), 0)) AS total_count,
    qu.abbreviation AS unit_abbreviation
FROM batch b
JOIN item i ON b.item_id = i.id
JOIN quantity q ON b.quantity_id = q.id
JOIN unit qu ON q.unit_id = qu.id
LEFT JOIN scanning_station ss ON b.scanning_station_id = ss.id
LEFT JOIN department d ON ss.department_id = d.id
WHERE b.account_id = sqlc.arg('account_id')
AND b.closed_at IS NULL
AND b.scanned_at IS NOT NULL
AND b.scanning_station_id IS NOT NULL
AND (
    sqlc.arg('include_item_filter') = false
    OR b.item_id IN (sqlc.slice('item_ids'))
)
AND (
    sqlc.arg('include_product_line_filter') = false
    OR EXISTS (
        SELECT 1 FROM product p
        WHERE p.item_id = i.id
          AND p.product_line_id IN (sqlc.slice('product_line_ids'))
    )
)
GROUP BY d.name, i.sku, i.id, b.scanning_station_id, qu.abbreviation;

-- name: InsertBatchQuantity :exec
INSERT INTO quantity (id, value, unit_id, created_at, updated_at)
VALUES (sqlc.arg('id'), sqlc.arg('value'), sqlc.arg('unit_id'), NOW(3), NOW(3));

-- InsertBatch creates a batch as unscanned work.
--
-- scanned_at stays NULL: a batch added to a production run is a ticket the floor has yet to run, and stamping it as scanned on creation both fakes production that never happened and closes the run immediately, since a run completes once every batch is scanned. The legacy create does not set it either.
-- name: InsertBatch :exec
INSERT INTO batch (
    id, account_id, item_id, quantity_id,
    seconds_quantity_id, waste_quantity_id,
    production_step_id, scanning_station_id, production_run_id,
    created_at, updated_at
) VALUES (
    sqlc.arg('id'), sqlc.arg('account_id'), sqlc.arg('item_id'), sqlc.arg('quantity_id'),
    sqlc.narg('seconds_quantity_id'), sqlc.narg('waste_quantity_id'),
    sqlc.narg('production_step_id'), sqlc.narg('scanning_station_id'), sqlc.narg('production_run_id'),
    NOW(3), NOW(3)
);

-- LinkBatchMachine attaches a batch to the machine that will run it.
--
-- Attainment reads the machine through this table, so a batch with no link is production nobody can attribute. The legacy create connects machines; this is the same edge.
-- name: LinkBatchMachine :exec
INSERT IGNORE INTO _batches_machines (A, B) VALUES (sqlc.arg('batch_id'), sqlc.arg('machine_id'));

-- InsertBatchFlow records that the source batch feeds the target batch. Per the Prisma orientation of _batch_flow, the downstream (target) batch is column A and the upstream (source) batch is column B.
-- name: InsertBatchFlow :exec
INSERT INTO _batch_flow (A, B) VALUES (sqlc.arg('target_batch_id'), sqlc.arg('source_batch_id'));

-- name: UpdateBatchScannedAt :exec
UPDATE batch SET scanned_at = NOW(3), updated_at = NOW(3)
WHERE id = sqlc.arg('id') AND account_id = sqlc.arg('account_id');

-- name: UpdateBatchProductionStepID :exec
UPDATE batch SET production_step_id = sqlc.arg('production_step_id'), updated_at = NOW(3)
WHERE id = sqlc.arg('id') AND account_id = sqlc.arg('account_id');

-- name: UpdateBatchScanningStationID :exec
UPDATE batch SET scanning_station_id = sqlc.arg('scanning_station_id'), updated_at = NOW(3)
WHERE id = sqlc.arg('id') AND account_id = sqlc.arg('account_id');

-- name: UpdateBatchClosedAt :exec
UPDATE batch SET closed_at = NOW(3), updated_at = NOW(3)
WHERE id = sqlc.arg('id') AND account_id = sqlc.arg('account_id');

-- name: DeleteBatch :exec
DELETE FROM batch WHERE id = sqlc.arg('id') AND account_id = sqlc.arg('account_id');

-- name: DeleteBatchesByIDs :exec
DELETE FROM batch WHERE id IN (sqlc.slice('ids')) AND account_id = sqlc.arg('account_id');

-- name: DeleteBatchFlowByBatchID :exec
DELETE FROM _batch_flow WHERE A = sqlc.arg('batch_id') OR B = sqlc.arg('batch_id');

-- name: DeleteBatchesMachinesByBatchID :exec
DELETE FROM _batches_machines WHERE A = sqlc.arg('batch_id');

-- UnlinkBatchMachinesExcept drops every machine link on a batch other than the one it should now have.
--
-- Used when a ticket is moved to a campaign on a different machine. Attainment attributes production through this table, so leaving the old link would credit a machine the work is no longer assigned to.
-- name: UnlinkBatchMachinesExcept :exec
DELETE FROM _batches_machines WHERE A = sqlc.arg('batch_id') AND B != sqlc.arg('machine_id');

-- name: FindBatchProductionRunID :one
SELECT production_run_id FROM batch
WHERE id = sqlc.arg('id') AND account_id = sqlc.arg('account_id');

-- name: GetBatchMachines :many
SELECT
    m.id,
    m.name
FROM _batches_machines bm
JOIN machine m ON bm.B = m.id
WHERE bm.A = sqlc.arg('batch_id');

-- name: ExportBatchMachines :many
-- The bulk form of GetBatchMachines, so an export naming many batches' machines
-- does not issue one query per batch.
SELECT
    bm.A AS batch_id,
    m.name
FROM _batches_machines bm
JOIN machine m ON bm.B = m.id
WHERE bm.A IN (sqlc.slice('batch_ids'))
ORDER BY bm.A, m.name;

-- name: ExportProductionRunBatches :many
-- The batches of the given runs, one row each, for the production run export.
-- Department comes via the scanning station, as it does on a batch read.
SELECT
    b.id,
    b.production_run_id,
    b.scanned_at,
    i.sku AS item_sku,
    q.value AS quantity_value,
    qu.abbreviation AS quantity_unit_abbreviation,
    d.name AS department_name
FROM batch b
JOIN item i ON b.item_id = i.id
JOIN quantity q ON b.quantity_id = q.id
JOIN unit qu ON q.unit_id = qu.id
LEFT JOIN scanning_station ss ON b.scanning_station_id = ss.id
LEFT JOIN department d ON ss.department_id = d.id
WHERE b.production_run_id IN (sqlc.slice('production_run_ids'))
AND b.account_id = sqlc.arg('account_id')
ORDER BY b.production_run_id, b.created_at, b.id;

-- name: IsBatchInAccount :one
SELECT COUNT(*) FROM batch WHERE id = sqlc.arg('id') AND account_id = sqlc.arg('account_id');

-- name: GetBatchLots :many
SELECT DISTINCT l.lot_number, 'material' AS lot_type
FROM inventory_issue ii
JOIN lot l ON ii.lot_id = l.id
WHERE ii.batch_id = sqlc.arg('batch_id')
AND l.lot_number IS NOT NULL
UNION
SELECT DISTINCT l.lot_number, 'material' AS lot_type
FROM inventory_issue ii
JOIN inventory_allocation ia ON ia.inventory_issue_id = ii.id
JOIN inventory_receipt ir ON ia.inventory_receipt_id = ir.id
JOIN lot l ON ir.lot_id = l.id
WHERE ii.batch_id = sqlc.arg('batch_id')
AND l.lot_number IS NOT NULL;
