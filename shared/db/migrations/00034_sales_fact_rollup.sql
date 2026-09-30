-- +goose NO TRANSACTION
-- +goose Up
-- Line amounts shrink from DECIMAL(65,30) (30 bytes) to DECIMAL(28,10) (13 bytes). The extra digits were
-- the tail of MySQL's 30-place division, not money: rounding each line at 10 places moves a year's total
-- by well under a millionth of a cent. Narrower values make every scan and sum over the table cheaper.
ALTER TABLE `sales_line_fact`
  MODIFY `quantity_base` decimal(28,10) NULL,
  MODIFY `total_invoiced` decimal(28,10) NULL,
  MODIFY `total_cost` decimal(28,10) NULL;

-- Sales totals pre-summed per UTC hour, day and month, so a report over a long window reads a few
-- thousand bucket rows instead of every line. A report reads whole buckets here and the partial
-- buckets at its window's edges from sales_line_fact.
--
-- Each dimension (buyer, item, product line, sales rep, discount, or 'total' for the whole account)
-- has its own rows, because an invoice's distinct count only adds up across a dimension it cannot
-- span: an invoice has one buyer, sales rep and discount, and falls in one bucket, but its lines can
-- span items and product lines. product_line_key '' holds a dimension's totals over every product
-- line; a product line's id holds its share alone, so a report filtered to one product line stays
-- exact. sales_rep_key is the order's sales rep ('' for none), so a sales rep filter is a key range.
--
-- Hour rows exist only for 'total', which the daily chart regroups into the caller's local days.
--
-- The full natural key (account, type, dimension, product line, grain, bucket, dimension id, sales rep)
-- exceeds InnoDB's 3072-byte key limit at varchar(191). Its last two columns, which reads only filter
-- and never range over, stand in the key as row_hash, MD5(dimension_id NUL sales_rep_key); the columns
-- themselves are stored in full.
CREATE TABLE `sales_fact_rollup` (
  `account_id` varchar(191) NOT NULL,
  `sales_order_type_code` varchar(191) NOT NULL,
  `dimension` varchar(32) NOT NULL,
  `product_line_key` varchar(191) NOT NULL,
  `grain` varchar(8) NOT NULL,
  `bucket_start` datetime NOT NULL,
  `row_hash` binary(16) NOT NULL,
  `dimension_id` varchar(191) NOT NULL,
  `sales_rep_key` varchar(191) NOT NULL,
  `quantity_base` decimal(28,10) NULL,
  `total_invoiced` decimal(28,10) NULL,
  `total_cost` decimal(28,10) NULL,
  `invoice_count` int NOT NULL,
  `line_count` int NOT NULL,
  PRIMARY KEY (`account_id`, `sales_order_type_code`, `dimension`, `product_line_key`, `grain`, `bucket_start`, `row_hash`),
  -- Rebuilding one account's day or month finds its rows here, across every dimension.
  KEY `sales_fact_rollup_bucket_idx` (`account_id`, `bucket_start`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Progress of the rollup sweep, which rebuilds every (account, day) the facts or rollups hold; its
-- first pass is the backfill, and reads use the rollups only once one pass has completed.
CREATE TABLE `sales_rollup_sync` (
  `name` varchar(64) NOT NULL,
  `cursor_account_id` varchar(191) NULL,
  `cursor_day` date NULL,
  `pass_started_at` datetime(3) NULL,
  `last_completed_at` datetime(3) NULL,
  `updated_at` datetime(3) NOT NULL,
  PRIMARY KEY (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- +goose Down
DROP TABLE `sales_rollup_sync`;
DROP TABLE `sales_fact_rollup`;
ALTER TABLE `sales_line_fact`
  MODIFY `quantity_base` decimal(65,30) NULL,
  MODIFY `total_invoiced` decimal(65,30) NULL,
  MODIFY `total_cost` decimal(65,30) NULL;
