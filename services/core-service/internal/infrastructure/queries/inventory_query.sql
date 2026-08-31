-- FetchCurrentInventoryForItem is what the item can still promise: stock on the shelf, less the
-- demand already spoken for. Reserved issues hold stock for an order that has not shipped; open ones
-- are demand nothing has covered yet. Both come off, which is what separates this from on-hand.
--
-- It filtered issues on `status_code = 'committed'`, a status the ledger has never written — the
-- statuses in use are `open`, `reserved` and `closed` — so the subtraction matched nothing and every
-- item promised its whole shelf however much of it was already sold.
--
-- Allocations come off each side for the same reason FetchPhysicalInventoryForItem takes them off:
-- an issue drawn from a receipt appears on both, so whole receipts minus whole issues counts every
-- allocated unit twice. Every row is normalised through its own unit's ratio and the total expressed
-- in the item's base unit, the unit returned beside it.
-- name: FetchCurrentInventoryForItem :one
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
         JOIN quantity q ON q.id = ir.quantity_id
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
         JOIN quantity q ON q.id = ii.quantity_id
         JOIN unit u ON u.id = q.unit_id
         WHERE ii.item_id = sqlc.arg('item_id')
         AND ii.account_id = sqlc.arg('owner_account_id')
         AND ii.status_code IN ('open', 'reserved')), 0
    )
) / COALESCE(NULLIF((
    SELECT bu.ratio_numerator / bu.ratio_denominator
    FROM item i
    JOIN item_category ic ON ic.id = i.item_category_id
    JOIN unit_group ug ON ug.id = ic.unit_group_id
    JOIN unit bu ON bu.id = ug.base_unit_id
    WHERE i.id = sqlc.arg('item_id')
), 0), 1) AS DECIMAL(65,30)) AS available_to_promise,
    COALESCE(
        (SELECT qu.abbreviation FROM item i
         JOIN item_category ic ON i.item_category_id = ic.id
         JOIN unit_group ug ON ic.unit_group_id = ug.id
         JOIN unit qu ON ug.base_unit_id = qu.id
         WHERE i.id = sqlc.arg('item_id') LIMIT 1), ''
    ) AS unit_abbreviation;

-- FetchOnHandInventoryBulk is the on-hand column of the item list: available receipts net of what has
-- been drawn against them, for a set of items in one pass, each in its own base unit.
--
-- It is the same shelf figure the item detail page shows, and it has to stay that way — a list and a
-- detail page disagreeing about one SKU is what sends an operator to count a bin. Two defects had it
-- reading several times the true level:
--
--   • `LEFT JOIN inventory_allocation ia ON ia.inventory_receipt_id = ir.id` inside the receipt sum,
--     with `ia` never referenced anywhere in the query. The join did nothing but fan the receipt row
--     out once per allocation against it, so a receipt drawn on four times was added four times. An
--     item reading 3,840 held 480.
--   • No unit ratios. Receipt values in pairs minus allocation values in each, reported in pairs.
--
-- Structured as four grouped joins rather than correlated subqueries, and each repeats the item list
-- in its own WHERE clause, for the reason FetchPhysicalInventoryBaseForItems gives: without it every
-- one aggregates the account's whole ledger and the outer join throws all but a handful away.
--
-- DECIMAL, not SIGNED. Half a pair is a real reading — it is what an odd number of `each` allocated
-- against a pair-stocked receipt leaves — and rounding it away makes the list disagree with the
-- reconcile it is checked against. Nothing is clamped either: a row drawn on for more than it holds
-- nets negative and carries, which is what keeps the level the sum of its movements.
-- name: FetchOnHandInventoryBulk :many
SELECT
    i.id AS item_id,
    CAST(
        (COALESCE(r.qty, 0) - COALESCE(ra.qty, 0)) / (bu.ratio_numerator / bu.ratio_denominator)
    AS DECIMAL(65,30)) AS on_hand_quantity,
    bu.id AS unit_id,
    bu.abbreviation AS unit_abbreviation,
    ug.unit_type_code AS unit_type
FROM item i
JOIN item_category ic ON i.item_category_id = ic.id
JOIN unit_group ug ON ic.unit_group_id = ug.id
JOIN unit bu ON ug.base_unit_id = bu.id
LEFT JOIN (
    SELECT ir.item_id AS item_id,
           SUM(CAST(q.value AS DECIMAL(65,30)) * (u.ratio_numerator / u.ratio_denominator)) AS qty
    FROM inventory_receipt ir
    JOIN quantity q ON q.id = ir.quantity_id
    JOIN unit u ON u.id = q.unit_id
    WHERE ir.item_id IN (sqlc.slice('item_ids'))
      AND (ir.owner_account_id = sqlc.arg('owner_account_id') OR ir.holder_account_id = sqlc.arg('owner_account_id'))
      AND ir.status_code = 'available'
    GROUP BY ir.item_id
) r ON r.item_id = i.id
LEFT JOIN (
    SELECT ir2.item_id AS item_id,
           SUM(CAST(aq.value AS DECIMAL(65,30)) * (au.ratio_numerator / au.ratio_denominator)) AS qty
    FROM inventory_allocation ia2
    JOIN inventory_receipt ir2 ON ir2.id = ia2.inventory_receipt_id
    JOIN quantity aq ON aq.id = ia2.quantity_id
    JOIN unit au ON au.id = aq.unit_id
    WHERE ir2.item_id IN (sqlc.slice('item_ids'))
      AND (ir2.owner_account_id = sqlc.arg('owner_account_id') OR ir2.holder_account_id = sqlc.arg('owner_account_id'))
      AND ir2.status_code = 'available'
    GROUP BY ir2.item_id
) ra ON ra.item_id = i.id
WHERE i.id IN (sqlc.slice('item_ids'))
  AND i.account_id = sqlc.arg('owner_account_id')
  AND i.deleted_at IS NULL;
