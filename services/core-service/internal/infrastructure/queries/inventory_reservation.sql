-- name: FindReservedIssuesByOrderItem :many
SELECT
    ii.id,
    q.id AS quantity_id,
    q.value AS quantity_value,
    q.unit_id,
    ii.storage_location_id,
    ii.lot_id,
    ii.batch_id
FROM inventory_issue ii
JOIN quantity q ON q.id = ii.quantity_id
WHERE ii.order_id = sqlc.arg('order_id')
AND ii.account_id = sqlc.arg('account_id')
AND ii.item_id = sqlc.arg('item_id')
AND ii.status_code = 'reserved'
ORDER BY ii.created_at ASC;

-- name: DeleteInventoryIssueByID :exec
DELETE FROM inventory_issue WHERE id = sqlc.arg('id');

-- name: DeleteQuantityByID :exec
DELETE FROM quantity WHERE id = sqlc.arg('id');

-- name: UpdateQuantityValue :exec
UPDATE quantity SET value = sqlc.arg('value') WHERE id = sqlc.arg('id');

-- Allocation rows are recorded in whatever unit the code that wrote them chose, so the sum is taken
-- through each row's own ratio; dividing by `unit_ratio` puts it in the issue's unit.
-- name: FindReservedIssuesWithAllocationSums :many
SELECT
    ii.id,
    q.id AS quantity_id,
    q.value AS quantity_value,
    q.unit_id,
    CAST(u.ratio_numerator / u.ratio_denominator AS DECIMAL(65,30)) AS unit_ratio,
    ii.storage_location_id,
    ii.lot_id,
    ii.batch_id,
    COALESCE(SUM(CAST(aq.value AS DECIMAL(65,30)) * (au.ratio_numerator / au.ratio_denominator)), 0) AS allocated_sum
FROM inventory_issue ii
JOIN quantity q ON q.id = ii.quantity_id
JOIN unit u ON u.id = q.unit_id
LEFT JOIN inventory_allocation ia ON ia.inventory_issue_id = ii.id
LEFT JOIN quantity aq ON aq.id = ia.quantity_id
LEFT JOIN unit au ON au.id = aq.unit_id
WHERE ii.order_id = sqlc.arg('order_id')
AND ii.account_id = sqlc.arg('account_id')
AND ii.item_id = sqlc.arg('item_id')
AND ii.status_code = 'reserved'
GROUP BY ii.id, q.id, q.value, q.unit_id, u.ratio_numerator, u.ratio_denominator, ii.storage_location_id, ii.lot_id, ii.batch_id
ORDER BY ii.created_at ASC;

-- UpdateInventoryIssueStatusToOpen consumes a whole reservation in place. The batch that consumed it
-- is stamped on the row so deleting that batch can find the reservation and hand it back; COALESCE
-- keeps whatever tag the row already carried when no batch is supplied.
-- name: UpdateInventoryIssueStatusToOpen :exec
UPDATE inventory_issue
SET status_code = 'open',
    issued_at = NOW(3),
    batch_id = COALESCE(sqlc.narg('batch_id'), batch_id),
    updated_at = NOW(3)
WHERE id = sqlc.arg('id');

-- name: InsertInventoryIssueForReservation :exec
INSERT INTO inventory_issue (
    id, account_id, item_id, quantity_id,
    status_code, order_id, batch_id,
    storage_location_id, lot_id,
    issued_at, created_at, updated_at
) VALUES (
    sqlc.arg('id'),
    sqlc.arg('account_id'),
    sqlc.arg('item_id'),
    sqlc.arg('quantity_id'),
    sqlc.arg('status_code'),
    sqlc.narg('order_id'),
    sqlc.narg('batch_id'),
    sqlc.narg('storage_location_id'),
    sqlc.narg('lot_id'),
    NOW(3),
    NOW(3),
    NOW(3)
);

-- FindReceiptsForAllocation lists the receipts an issue may draw from, oldest first.
--
-- FOR UPDATE holds the candidates until the allocating transaction commits. The caller decides how
-- much a receipt has left by reading its allocations and subtracting, so without the lock two
-- consumptions of the same item both saw the same receipt as free and each allocated the whole of
-- it — consuming stock that was never used. Only `available` receipts are candidates, so the lock
-- covers the few rows actually in play rather than the item's whole receipt history.
-- name: FindReceiptsForAllocation :many
--
-- The account may hold stock it does not own, so both sides are candidates: consigned material sits
-- on the shelf under the owner's account id and is drawn down by the holder's demand.
--
-- The receipt's own unit cost comes back with it. What an allocation cost is the price the stock was
-- received at, not the item's price today, which is the whole reason the cost is copied onto the
-- receipt when it lands.
SELECT
    ir.id,
    q.id AS quantity_id,
    q.value AS quantity_value,
    q.unit_id,
    uc.value AS unit_cost_value,
    uc.numerator_unit_id AS unit_cost_numerator_unit_id,
    uc.denominator_unit_id AS unit_cost_denominator_unit_id
FROM inventory_receipt ir
JOIN quantity q ON q.id = ir.quantity_id
JOIN rate uc ON uc.id = ir.unit_cost_id
-- Held stock counts as allocatable: consigned goods sit under a holder while another account owns
-- them, and excluding them makes the item look short when it is physically on the shelf.
WHERE (ir.owner_account_id = sqlc.arg('account_id') OR ir.holder_account_id = sqlc.arg('account_id'))
AND ir.item_id = sqlc.arg('item_id')
AND ir.status_code = 'available'
-- An issue pinned to a location or lot may only draw from stock sitting there; an unpinned issue
-- draws from anywhere.
AND (sqlc.narg('storage_location_id') IS NULL OR ir.storage_location_id = sqlc.narg('storage_location_id'))
AND (sqlc.narg('lot_id') IS NULL OR ir.lot_id = sqlc.narg('lot_id'))
ORDER BY ir.received_at ASC
FOR UPDATE;

-- Retires receipts whose quantity is entirely spoken for so later runs stop reconsidering them.
--
-- Allocation is derived from the rows either way — a receipt whose allocations cover it has nothing
-- left to give whatever its status says — but leaving it `available` means every later pass re-reads
-- and re-locks it, and it still reads as free stock to anything that asks by status.
-- name: MarkInventoryReceiptsAllocated :exec
UPDATE inventory_receipt
SET status_code = 'allocated', updated_at = NOW(3)
WHERE id IN (sqlc.slice('ids')) AND status_code <> 'allocated';

-- Retires an issue whose demand is fully covered, which is what stops a later receipt allocating
-- against it a second time.
-- name: CloseFullyAllocatedInventoryIssue :exec
UPDATE inventory_issue
SET status_code = 'closed', issued_at = NOW(3), updated_at = NOW(3)
WHERE id = sqlc.arg('id') AND status_code <> 'closed';

-- Through each row's own ratio. See GetAllocationSumsForReceipts.
-- name: GetAllocationSumForReceipt :one
SELECT COALESCE(SUM(CAST(q.value AS DECIMAL(65,30)) * (u.ratio_numerator / u.ratio_denominator)), 0) AS total_allocated
FROM inventory_allocation ia
JOIN quantity q ON q.id = ia.quantity_id
JOIN unit u ON u.id = q.unit_id
WHERE ia.inventory_receipt_id = sqlc.arg('receipt_id');

-- GetAllocationSumsForReceipts answers for a whole candidate set at once. Allocation walks receipts
-- oldest first and needs each one's drawn-down total; asking per receipt put a round trip inside that
-- loop, so an item with a long tail of open receipts cost a query apiece to find most of them full.
-- Receipts with no allocations are absent rather than zero — the caller treats a missing row as zero.
--
-- Each row is taken through its own unit's ratio before it is added. Allocations against one receipt
-- can be recorded in different units — the row carries whatever unit the code that wrote it chose —
-- so adding the raw column values produces a number in no unit at all. Divide the total by a unit's
-- ratio to read it in that unit.
-- name: GetAllocationSumsForReceipts :many
SELECT ia.inventory_receipt_id, COALESCE(SUM(CAST(q.value AS DECIMAL(65,30)) * (u.ratio_numerator / u.ratio_denominator)), 0) AS total_allocated
FROM inventory_allocation ia
JOIN quantity q ON q.id = ia.quantity_id
JOIN unit u ON u.id = q.unit_id
WHERE ia.inventory_receipt_id IN (sqlc.slice('receipt_ids'))
GROUP BY ia.inventory_receipt_id;

-- GetUnitRatios gives each unit its ratio, which every unit carries against the same reference for
-- its dimension. Any two units convert directly through them: `value * ratio_from / ratio_to`.
--
-- Read on its own rather than joined into FindReceiptsForAllocation because that query is
-- FOR UPDATE, and a join would take locks on `unit` rows every account in the database shares.
-- name: GetUnitRatios :many
SELECT
    u.id,
    CAST(u.ratio_numerator / u.ratio_denominator AS DECIMAL(65,30)) AS ratio
FROM unit u
WHERE u.id IN (sqlc.slice('unit_ids'));

-- name: InsertInventoryAllocation :exec
INSERT INTO inventory_allocation (
    id, inventory_receipt_id, inventory_issue_id,
    quantity_id, unit_cost_id, total_cost_id,
    created_at, updated_at
) VALUES (
    sqlc.arg('id'),
    sqlc.arg('inventory_receipt_id'),
    sqlc.arg('inventory_issue_id'),
    sqlc.arg('quantity_id'),
    sqlc.arg('unit_cost_id'),
    sqlc.arg('total_cost_id'),
    NOW(3),
    NOW(3)
);
