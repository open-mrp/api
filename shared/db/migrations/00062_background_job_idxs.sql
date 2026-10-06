-- +goose NO TRANSACTION
-- +goose Up
-- A buyer summary rebuild reads every qualifying fact of its buyers from this index alone. The type code
-- is narrowed so the key, account_id included, fits InnoDB's 3072-byte limit; codes are a few words.
ALTER TABLE `sales_line_fact`
  MODIFY COLUMN `sales_order_type_code` varchar(32) NOT NULL AFTER `sales_order_id`,
  ADD KEY `sales_line_fact_buyer_summary_idx` (`account_id`, `buyer_account_id`, `sales_order_type_code`, `is_priced`, `ordered_at`, `product_line_id`, `total_invoiced`);

-- A day rebuild reads its hour and day rows; grain in the key skips the first of the month's month rows without a row lookup.
ALTER TABLE `sales_fact_rollup`
  DROP KEY `sales_fact_rollup_bucket_idx`,
  ADD KEY `sales_fact_rollup_bucket_idx` (`account_id`, `bucket_start`, `grain`);

-- +goose Down
ALTER TABLE `sales_fact_rollup`
  DROP KEY `sales_fact_rollup_bucket_idx`,
  ADD KEY `sales_fact_rollup_bucket_idx` (`account_id`, `bucket_start`);

ALTER TABLE `sales_line_fact`
  DROP KEY `sales_line_fact_buyer_summary_idx`,
  MODIFY COLUMN `sales_order_type_code` varchar(191) NOT NULL AFTER `sales_order_id`;
