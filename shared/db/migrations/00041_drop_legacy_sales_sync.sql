-- +goose NO TRANSACTION
-- +goose Up
-- The sales fact refresher's bookkeeping now lives in sales_sync (one row per pass) and sales_fact_dirty
-- (one queue, with 'rollup_day' marks). Nothing has read these since the release that stopped carrying
-- their state over, so they go.
DROP TABLE IF EXISTS `sales_rollup_dirty`;
DROP TABLE IF EXISTS `sales_rollup_sync`;
DROP TABLE IF EXISTS `sales_fact_sync`;

-- +goose Down
CREATE TABLE `sales_fact_sync` (
  `name` varchar(64) NOT NULL,
  `cursor_created_at` datetime(3) NULL,
  `cursor_invoice_id` varchar(191) NULL,
  `pass_started_at` datetime(3) NULL,
  `last_completed_at` datetime(3) NULL,
  `updated_at` datetime(3) NOT NULL,
  PRIMARY KEY (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE `sales_rollup_sync` (
  `name` varchar(64) NOT NULL,
  `cursor_account_id` varchar(191) NULL,
  `cursor_day` date NULL,
  `pass_started_at` datetime(3) NULL,
  `last_completed_at` datetime(3) NULL,
  `updated_at` datetime(3) NOT NULL,
  PRIMARY KEY (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE `sales_rollup_dirty` (
  `account_id` varchar(191) NOT NULL,
  `day` date NOT NULL,
  `marked_at` datetime(3) NOT NULL,
  PRIMARY KEY (`account_id`, `day`),
  KEY `sales_rollup_dirty_marked_idx` (`marked_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
