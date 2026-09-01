-- +goose Up

-- One row per item, existing only to be locked, and the single ordering root for the inventory ledger.
--
-- Six flows write inventory_issue, inventory_receipt, inventory_allocation and their quantity/rate rows
-- in whatever order their own work dictates — a receiving order writes the receipt before it can offer
-- it, a shipment flips the reservation before it can draw it, a reversal frees the receipts before it
-- can restore the demand it was covering. There is no order all of them can agree on: the allocator
-- must hold the demand before it can choose which receipts to draw, and the reversal must free the
-- receipts before it can put the demand back. So they agree on this instead — two transactions are
-- never both inside the section for one item, and the order they use once inside it stops mattering.
--
-- Keyed on item_id alone, not (account_id, item_id): FindReceiptsForAllocation matches
-- `owner_account_id = ? OR holder_account_id = ?`, so consigned stock under one account is drawn down
-- by another account's demand. An account-scoped key would hand two contending flows two different
-- rows over one contended receipt, which is a lock that does not lock.
--
-- It carries no data, and that is deliberate. A lock on a row that means something is a row somebody
-- eventually wants to update for an unrelated reason, and then the mutex has a second population of
-- writers nobody accounted for.
--
-- INSERT ... ON DUPLICATE KEY UPDATE item_id = item_id must remain the ONLY statement ever issued
-- against this table. Never add a SELECT, a range scan or a DELETE. Both obvious alternatives deadlock
-- on the very path this table exists to make safe:
--
--   * A locking read that matches no row takes a GAP lock, and the INSERT that would follow takes an
--     insert-intention lock, which conflicts with another transaction's gap lock over the same gap.
--     Two transactions creating two DIFFERENT new items then deadlock on the mechanism built to stop
--     deadlocks.
--   * A plain INSERT that hits a duplicate key takes a SHARED lock, so INSERT-then-FOR-UPDATE is an
--     S-to-X upgrade and two transactions on the same item deadlock immediately.
--
-- ON DUPLICATE KEY takes an exclusive lock on the duplicate row, which is the acquisition. What makes
-- the cold branch safe is not the backfill — items are created continuously, including by the
-- dashboard, so the insert branch stays warm for exactly the newly received items. It is that no other
-- access path exists here, so no gap lock is ever taken and there is nothing for an insert-intention
-- lock to conflict with.
--
-- Rows are never deleted. An item that stops being stocked keeps its row; reclaiming them would mean a
-- DELETE, and a DELETE takes the gap locks the paragraph above rules out.
CREATE TABLE `inventory_item_lock` (
  `item_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`item_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- +goose Down

DROP TABLE `inventory_item_lock`;
