-- +goose NO TRANSACTION
-- +goose Up
-- The (account, UTC day) rollup buckets that must be rebuilt from sales_line_fact. A refresh marks a
-- day before it writes the day's facts and clears the mark only after the day's buckets are rebuilt,
-- so a failed or interrupted rebuild is retried on the next tick instead of leaving the buckets behind
-- their facts until the daily rollup sweep. A mark re-marked while its rebuild ran survives the clear.
CREATE TABLE `sales_rollup_dirty` (
  `account_id` varchar(191) NOT NULL,
  `day` date NOT NULL,
  `marked_at` datetime(3) NOT NULL,
  PRIMARY KEY (`account_id`, `day`),
  KEY `sales_rollup_dirty_marked_idx` (`marked_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- +goose Down
DROP TABLE `sales_rollup_dirty`;
