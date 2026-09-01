-- LockItemForLedger takes an item's ledger lock. See docs/patterns/architecture-patterns.md,
-- "Inventory ledger lock order" — this statement is the rule's whole mechanism.
--
-- INSERT ... ON DUPLICATE KEY UPDATE rather than SELECT ... FOR UPDATE, because the row may not exist
-- yet and both obvious shapes deadlock on that path. A locking read that matches no row takes a GAP
-- lock, and the INSERT that would follow takes an insert-intention lock, which conflicts with another
-- transaction's gap lock over the same gap: two transactions creating two different new items deadlock
-- on the mechanism built to stop deadlocks. A plain INSERT that hits a duplicate key takes a SHARED
-- lock, so INSERT-then-FOR-UPDATE is an S-to-X upgrade and two transactions on the same item deadlock
-- immediately. ON DUPLICATE KEY takes an exclusive lock on the duplicate row, which is the acquisition.
--
-- What makes the cold branch safe is NOT the backfill — items are created continuously, including by
-- the dashboard, so the insert branch stays warm for exactly the newly received items. It is that this
-- is the only statement ever issued against this table: no access path takes a gap lock here, so there
-- is nothing for an insert-intention lock to conflict with. Never add a SELECT, a range scan or a
-- DELETE. TestLedgerLock_RejectedShapeIsUnsafe measures what happens if you do.
--
-- `ON DUPLICATE KEY UPDATE item_id = item_id` is the house idiom for a no-op upsert (see
-- AddItemAttribute in item.sql).
-- name: LockItemForLedger :exec
INSERT INTO inventory_item_lock (item_id, created_at)
VALUES (sqlc.arg('item_id'), NOW(3))
ON DUPLICATE KEY UPDATE item_id = item_id;
