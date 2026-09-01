-- +goose Up

-- One lock row per existing item.
--
-- Not required for correctness: LockItemForLedger is INSERT ... ON DUPLICATE KEY UPDATE, so it creates
-- the row on first acquisition and the cold branch is as safe as the warm one — see the note on
-- 00016_inventory_item_lock. This is a warm-up, so the first allocation of every existing item takes an
-- update rather than an insert, and so the table's size is a known quantity from the day it ships
-- rather than something that grows as traffic discovers it.
--
-- INSERT IGNORE and a SELECT rather than a fixed list: it must be re-runnable, and it must cover items
-- created between the DDL deploy and this running. Items created after it are covered by the cold
-- branch, which is the normal path for anything newly received.
INSERT IGNORE INTO `inventory_item_lock` (`item_id`, `created_at`)
SELECT `id`, NOW(3) FROM `item`;

-- +goose Down

-- Not reversible, and deliberately so. Deleting from this table takes gap locks on the primary key,
-- which is the one access pattern 00016 rules out — an acquisition running concurrently with the
-- delete would meet exactly the insert-intention conflict the ON DUPLICATE KEY shape exists to avoid.
-- Rolling the seed back also buys nothing: the rows are inert, and dropping the table (00016 Down)
-- removes them.
SELECT 1;
