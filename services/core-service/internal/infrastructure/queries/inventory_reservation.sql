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

-- ClaimReservedInventoryIssueAsOpen consumes a whole reservation in place.
--
-- Guarded on `reserved` and checked for rows affected, unlike the unguarded UPDATE it replaces: the
-- reservation this transaction read may have been deleted in between by an order edit (ReduceReservedForOrderItem, DeleteReservedInventoryIssuesBySalesOrder,
-- DeleteReservedInventoryIssuesByOrderID). The unguarded UPDATE then matched nothing and the caller
-- carried on to write allocations against an issue that no longer exists, retiring the receipts they
-- drew to cover demand that is not there. There is no foreign key on
-- inventory_allocation.inventory_issue_id to catch it afterwards.
-- name: ClaimReservedInventoryIssueAsOpen :execresult
UPDATE inventory_issue
SET status_code = 'open',
    issued_at = NOW(3),
    batch_id = COALESCE(sqlc.narg('batch_id'), batch_id),
    updated_at = NOW(3)
WHERE id = sqlc.arg('id') AND status_code = 'reserved';

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

-- Through each row's own ratio: allocations against one receipt can be stamped in different units,
-- so adding the raw column values produces a number in no unit at all.
-- name: GetAllocationSumForReceipt :one
SELECT COALESCE(SUM(CAST(q.value AS DECIMAL(65,30)) * (u.ratio_numerator / u.ratio_denominator)), 0) AS total_allocated
FROM inventory_allocation ia
JOIN quantity q ON q.id = ia.quantity_id
JOIN unit u ON u.id = q.unit_id
WHERE ia.inventory_receipt_id = sqlc.arg('receipt_id');

-- ReadReceiptAllocationsForUpdate is a CURRENT read of what a receipt has actually been drawn: a
-- locking read sees the latest committed row versions regardless of this transaction's snapshot.
--
-- It exists for one reason and has a defined end. Every writer in this service holds the receipt's
-- own X lock before it writes an allocation against it, so for those writers the plain sum in
-- GetAllocationSumsForReceipts is already correct. dashboard/apps/api's Prisma allocator holds
-- neither that lock nor any other and writes these rows on live invoice-delete and order-release
-- paths. When that allocator is gone, drop the locking clause and this becomes a plain read kept as
-- an arithmetic regression check.
--
-- A bare FOR UPDATE, and BOTH joined tables must be locked by it.
--
-- A locking read is current only for the tables it locks; anything else in the join is still read
-- from the transaction's snapshot. Lock `ia` alone and an allocation committed after this
-- transaction's view opened is found in `ia`, joined against a `quantity` row the snapshot cannot
-- see, and dropped by the INNER JOIN — so the read reports a receipt as undrawn while looking
-- straight at the row that drew it. That is silent, and it defeats the one query whose whole job is
-- to see writers this transaction never serialized against.
--
-- Locking both is bounded: an allocation owns its quantity row outright
-- (inventory_allocation_quantity_id_key is unique), so this is not the shared-row problem that keeps
-- `unit` out of the join below. `ia` and `q` are the only tables here, so a bare FOR UPDATE locks
-- exactly them and nothing more.
--
-- It was `FOR UPDATE OF ia, q`, which says the same thing and which vtgate rejects outright:
-- "Error 1105 (HY000): syntax error at position 191 near 'OF'", on every allocate_open_issues
-- message in production. MySQL 8 accepts the OF clause and every test here runs against plain
-- MySQL 8, so nothing local could have caught it. Never reintroduce it — see
-- TestVitessCompat_NoForUpdateOfClause.
--
-- Raw rows rather than a SUM, and no `unit` join: the sum has to go through each allocation's own
-- ratio, and a locking read must not take locks on rows every account in the database shares.
-- name: ReadReceiptAllocationsForUpdate :many
SELECT ia.id, q.unit_id, q.value
FROM inventory_allocation ia
JOIN quantity q ON q.id = ia.quantity_id
WHERE ia.inventory_receipt_id = sqlc.arg('receipt_id')
FOR UPDATE;

-- ReadIssueCoverageForUpdate is ReadReceiptAllocationsForUpdate on the other side of the ledger: what
-- an issue has actually been covered by, read currently rather than from the transaction's snapshot.
--
-- It decides whether the issue closes, which is the decision GetAllocationSumForIssue used to make
-- from a view frozen before the receipt locks were held. See that query's note; the same reasoning
-- and the same end condition apply.
-- name: ReadIssueCoverageForUpdate :many
SELECT ia.id, q.unit_id, q.value
FROM inventory_allocation ia
JOIN quantity q ON q.id = ia.quantity_id
WHERE ia.inventory_issue_id = sqlc.arg('issue_id')
FOR UPDATE;

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

-- ListOpenIssueIDsForItemPaged names the demand worth trying. It is deliberately not a locking read
-- and projects nothing but the keyset, so it is answered from inventory_issue_open_paging_idx alone:
-- no record locks, no gap locks, no trailing next-key lock, and no locks at all on `quantity`.
--
-- Nothing is decided from what it returns. Every row is re-read by ClaimOpenIssueForAllocation, by
-- primary key, under FOR UPDATE, in its own transaction — a row that closed in between returns
-- nothing there and is skipped. That is the difference from 3e99b962, which made a read non-locking
-- and then fed its quantity and its allocated sum straight into the arithmetic.
--
-- What the FOR UPDATE this replaces actually cost: the item's whole
-- (account_id, item_id, 'open', created_at) range plus every gap between and after it, and X locks on
-- up to 200 shared `quantity` rows, held for the length of a walk over every issue in the page. New
-- demand for the item lands in that trailing gap, so for the life of the transaction no batch scan,
-- shipment or reservation for that item could be recorded. What it bought — deferring the read view
-- past the receipt locks — it never delivered; see the note on ClaimOpenIssueForAllocation.
-- name: ListOpenIssueIDsForItemPaged :many
SELECT ii.id, ii.created_at
FROM inventory_issue ii
WHERE ii.account_id = sqlc.arg('account_id')
AND ii.item_id = sqlc.arg('item_id')
AND ii.status_code = 'open'
AND (
    ii.created_at > sqlc.arg('cursor_created_at')
    OR (ii.created_at = sqlc.arg('cursor_created_at') AND ii.id > sqlc.arg('cursor_id'))
)
ORDER BY ii.created_at ASC, ii.id ASC
LIMIT ?;

-- ClaimOpenIssueForAllocation re-reads one issue by primary key, and only if it is still open.
--
-- Reached by primary key, so it takes the clustered lock first and the secondary index entry only
-- when the close maintains it — the same direction as every other status writer. The secondary-index
-- range scan this replaces went the other way round, which is one row's worth of cycle against any
-- UPDATE ... WHERE id = ?.
--
-- `unit` is still not joined: a locking read locks every row it touches and `unit` rows are shared by
-- every account. The ratio comes from GetUnitRatios, after this lock and after the receipt lock.
-- name: ClaimOpenIssueForAllocation :one
SELECT ii.id, q.id AS quantity_id, q.value AS quantity_value, q.unit_id,
       ii.storage_location_id, ii.lot_id
FROM inventory_issue ii
JOIN quantity q ON q.id = ii.quantity_id
WHERE ii.id = sqlc.arg('id')
AND ii.account_id = sqlc.arg('account_id')
AND ii.status_code = 'open'
FOR UPDATE;

-- CountAvailableReceiptsForItem answers "is there anything to draw on at all" before any transaction
-- is opened, so an item whose whole open backlog is uncoverable costs one read rather than one
-- transaction per issue. The busiest items in this database have zero available receipts against
-- hundreds of open issues.
--
-- It deliberately ignores the storage_location/lot pinning FindReceiptsForAllocation applies, so a
-- non-zero count does not mean a pinned issue has a candidate. It is a cost hint, never a decision.
-- name: CountAvailableReceiptsForItem :one
SELECT COUNT(*) FROM inventory_receipt ir
WHERE (ir.owner_account_id = sqlc.arg('account_id') OR ir.holder_account_id = sqlc.arg('account_id'))
AND ir.item_id = sqlc.arg('item_id')
AND ir.status_code = 'available';

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

-- ListReservedIssuesForOrder names an order's reservations, with everything releasing one needs: the
-- item whose ordering root has to be held, and the quantity row that goes with the issue.
--
-- Answered from inventory_issue_order_id_idx, the same access path the delete it replaces used.
-- name: ListReservedIssuesForOrder :many
SELECT ii.id, ii.item_id, ii.quantity_id
FROM inventory_issue ii
WHERE ii.order_id = sqlc.arg('order_id')
AND ii.account_id = sqlc.arg('account_id')
AND ii.status_code = 'reserved';

-- ListReservedItemIDsForOrders names the items a release will write, so the caller can take their
-- ordering root as the first statement of its transaction rather than discovering the set halfway
-- through it. Read on the pool, before the transaction opens — see ledgerlock, Corollary A.
-- name: ListReservedItemIDsForOrders :many
SELECT DISTINCT ii.item_id
FROM inventory_issue ii
WHERE ii.order_id IN (sqlc.slice('order_ids'))
AND ii.account_id = sqlc.arg('account_id')
AND ii.status_code = 'reserved';
