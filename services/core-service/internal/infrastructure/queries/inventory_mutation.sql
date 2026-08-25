-- name: InsertInventoryReceipt :exec
INSERT INTO inventory_receipt (
    id, owner_account_id, holder_account_id, item_id,
    quantity_id, unit_cost_id, status_code,
    batch_id, storage_location_id, lot_id,
    received_at, created_at, updated_at
) VALUES (
    sqlc.arg('id'),
    sqlc.arg('owner_account_id'),
    sqlc.arg('holder_account_id'),
    sqlc.arg('item_id'),
    sqlc.arg('quantity_id'),
    sqlc.arg('unit_cost_id'),
    'available',
    sqlc.narg('batch_id'),
    sqlc.narg('storage_location_id'),
    sqlc.narg('lot_id'),
    NOW(3),
    NOW(3),
    NOW(3)
);

-- name: InsertInventoryIssue :exec
INSERT INTO inventory_issue (
    id, account_id, item_id,
    quantity_id, status_code,
    batch_id, storage_location_id, lot_id,
    issued_at, created_at, updated_at
) VALUES (
    sqlc.arg('id'),
    sqlc.arg('account_id'),
    sqlc.arg('item_id'),
    sqlc.arg('quantity_id'),
    'open',
    sqlc.narg('batch_id'),
    sqlc.narg('storage_location_id'),
    sqlc.narg('lot_id'),
    NOW(3),
    NOW(3),
    NOW(3)
);

-- name: InsertQuantityForInventory :exec
INSERT INTO quantity (id, value, unit_id)
VALUES (sqlc.arg('id'), sqlc.arg('value'), sqlc.arg('unit_id'));

-- name: InsertRateForInventory :exec
INSERT INTO rate (id, value, numerator_unit_id, denominator_unit_id)
VALUES (sqlc.arg('id'), sqlc.arg('value'), sqlc.arg('numerator_unit_id'), sqlc.arg('denominator_unit_id'));

-- name: GetItemUnitCost :one
SELECT r.id, r.value, r.numerator_unit_id, r.denominator_unit_id
FROM item i
JOIN rate r ON r.id = i.unit_cost_id
WHERE i.id = sqlc.arg('item_id')
  AND i.account_id = sqlc.arg('account_id')
  AND i.deleted_at IS NULL;

-- name: FindBatchProductionRunIDAncestry :many
SELECT b.id, b.production_run_id,
       bf.B as parent_id
FROM batch b
LEFT JOIN _batch_flow bf ON bf.A = b.id
WHERE b.id IN (sqlc.slice('batch_ids'));

-- name: InsertInventoryLog :exec
INSERT INTO inventory_log (id, item_id, quantity_id, account_id, created_at, updated_at)
VALUES (sqlc.arg('id'), sqlc.arg('item_id'), sqlc.arg('quantity_id'), sqlc.arg('account_id'), NOW(3), NOW(3));

-- name: InsertInventoryChangeLog :exec
INSERT INTO inventory_change_log (
    id, item_id, quantity_id, action_type_code,
    scanning_station_id, account_id, inventory_log_id,
    responsible_user_id, created_at, updated_at
) VALUES (
    sqlc.arg('id'), sqlc.arg('item_id'), sqlc.arg('quantity_id'), sqlc.arg('action_type_code'),
    sqlc.narg('scanning_station_id'), sqlc.arg('account_id'), sqlc.narg('inventory_log_id'),
    sqlc.narg('responsible_user_id'), NOW(3), NOW(3)
);

-- FetchPhysicalInventoryForItem is what the shelf holds net of the demand nothing has covered — the
-- figure a reconcile measures its target against, and the level the inventory log records.
--
-- It has to agree with what the item's inventory endpoint reports, or a correction is computed
-- against a number the operator never saw. Two things make it disagree if they are left out:
--
--   • Allocations. An open issue drawn from a receipt appears on both sides, so subtracting whole
--     receipts from whole issues counts every allocated unit twice. An item with 120 received and a
--     220 issue that has already drawn 180 of it read as -220 while the page showed 0, and a
--     reconcile to 0 then wrote a receipt for 220 instead of removing anything.
--   • Units. Receipts, issues and allocations are each recorded in whatever unit their source used;
--     adding 120 ea to 60 pr as though they were the same number is arithmetic on labels.
--
-- Everything is normalised through its unit's ratio and netted per row, then the total is expressed
-- in `unit_id`. An unknown or empty unit leaves it in base units. Nothing is clamped: a row drawn on
-- for more than it holds nets negative and carries that into the total, which is what makes the
-- level the sum of the movements recorded against the item.
-- name: FetchPhysicalInventoryForItem :one
SELECT CAST((
    COALESCE(
        (SELECT SUM(CAST(q.value AS DECIMAL(65,30)) * (u.ratio_numerator / u.ratio_denominator) - COALESCE((
            SELECT SUM(CAST(aq.value AS DECIMAL(65,30)) * (au.ratio_numerator / au.ratio_denominator))
            FROM inventory_allocation ia
            JOIN quantity aq ON aq.id = ia.quantity_id
            JOIN unit au ON au.id = aq.unit_id
            WHERE ia.inventory_receipt_id = ir.id
         ), 0))
         FROM inventory_receipt ir
         JOIN quantity q ON ir.quantity_id = q.id
         JOIN unit u ON u.id = q.unit_id
         WHERE ir.item_id = sqlc.arg('item_id')
         AND (ir.owner_account_id = sqlc.arg('owner_account_id') OR ir.holder_account_id = sqlc.arg('owner_account_id'))
         AND ir.status_code = 'available'), 0
    ) - COALESCE(
        (SELECT SUM(CAST(q.value AS DECIMAL(65,30)) * (u.ratio_numerator / u.ratio_denominator) - COALESCE((
            SELECT SUM(CAST(aq.value AS DECIMAL(65,30)) * (au.ratio_numerator / au.ratio_denominator))
            FROM inventory_allocation ia
            JOIN quantity aq ON aq.id = ia.quantity_id
            JOIN unit au ON au.id = aq.unit_id
            WHERE ia.inventory_issue_id = ii.id
         ), 0))
         FROM inventory_issue ii
         JOIN quantity q ON ii.quantity_id = q.id
         JOIN unit u ON u.id = q.unit_id
         WHERE ii.item_id = sqlc.arg('item_id')
         AND ii.account_id = sqlc.arg('owner_account_id')
         AND ii.status_code = 'open'), 0
    )
) / COALESCE(NULLIF((
    SELECT u.ratio_numerator / u.ratio_denominator FROM unit u WHERE u.id = sqlc.arg('unit_id')
), 0), 1) AS DECIMAL(65,30)) AS physical_inventory;

-- FetchPhysicalInventoryBaseForItems is the batched, base-unit form of FetchPhysicalInventoryForItem.
--
-- It returns the same shelf figure — available receipts net of what open issues have drawn, each row
-- normalised through its own unit's ratio — but for a set of items in one pass and left in base
-- units, so a caller applies the per-item target-unit divide itself. The correlated per-row
-- subqueries the single-item form runs once per item are replaced by four sums grouped by item and
-- joined onto the item list, which is what lets one query stand in for the N the audit trail used to
-- run per scan. Nothing is clamped: a row drawn on for more than it holds nets negative and carries.
-- name: FetchPhysicalInventoryBaseForItems :many
SELECT
    i.id AS item_id,
    CAST(
        (COALESCE(r.qty, 0) - COALESCE(ra.qty, 0)) - (COALESCE(iss.qty, 0) - COALESCE(ia.qty, 0))
    AS DECIMAL(65,30)) AS physical_base
FROM item i
LEFT JOIN (
    SELECT ir.item_id AS item_id,
           SUM(CAST(q.value AS DECIMAL(65,30)) * (u.ratio_numerator / u.ratio_denominator)) AS qty
    FROM inventory_receipt ir
    JOIN quantity q ON q.id = ir.quantity_id
    JOIN unit u ON u.id = q.unit_id
    WHERE (ir.owner_account_id = sqlc.arg('account_id') OR ir.holder_account_id = sqlc.arg('account_id'))
      AND ir.status_code = 'available'
    GROUP BY ir.item_id
) r ON r.item_id = i.id
LEFT JOIN (
    SELECT ir.item_id AS item_id,
           SUM(CAST(aq.value AS DECIMAL(65,30)) * (au.ratio_numerator / au.ratio_denominator)) AS qty
    FROM inventory_allocation ia
    JOIN inventory_receipt ir ON ir.id = ia.inventory_receipt_id
    JOIN quantity aq ON aq.id = ia.quantity_id
    JOIN unit au ON au.id = aq.unit_id
    WHERE (ir.owner_account_id = sqlc.arg('account_id') OR ir.holder_account_id = sqlc.arg('account_id'))
      AND ir.status_code = 'available'
    GROUP BY ir.item_id
) ra ON ra.item_id = i.id
LEFT JOIN (
    SELECT ii.item_id AS item_id,
           SUM(CAST(q.value AS DECIMAL(65,30)) * (u.ratio_numerator / u.ratio_denominator)) AS qty
    FROM inventory_issue ii
    JOIN quantity q ON q.id = ii.quantity_id
    JOIN unit u ON u.id = q.unit_id
    WHERE ii.account_id = sqlc.arg('account_id')
      AND ii.status_code = 'open'
    GROUP BY ii.item_id
) iss ON iss.item_id = i.id
LEFT JOIN (
    SELECT ii.item_id AS item_id,
           SUM(CAST(aq.value AS DECIMAL(65,30)) * (au.ratio_numerator / au.ratio_denominator)) AS qty
    FROM inventory_allocation ia
    JOIN inventory_issue ii ON ii.id = ia.inventory_issue_id
    JOIN quantity aq ON aq.id = ia.quantity_id
    JOIN unit au ON au.id = aq.unit_id
    WHERE ii.account_id = sqlc.arg('account_id')
      AND ii.status_code = 'open'
    GROUP BY ii.item_id
) ia ON ia.item_id = i.id
WHERE i.id IN (sqlc.slice('item_ids'))
  AND i.account_id = sqlc.arg('account_id');

-- name: GetBatchSecondsAndWaste :one
SELECT
    sq.value AS seconds_value,
    sq.unit_id AS seconds_unit_id,
    wq.value AS waste_value,
    wq.unit_id AS waste_unit_id
FROM batch b
LEFT JOIN quantity sq ON sq.id = b.seconds_quantity_id
LEFT JOIN quantity wq ON wq.id = b.waste_quantity_id
WHERE b.id = sqlc.arg('batch_id');
