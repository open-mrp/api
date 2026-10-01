-- +goose NO TRANSACTION
-- +goose Up
-- A report filtered to its own groups, or to a sales rep, pins them ahead of the buckets.
-- sales_order_type_code is left out to fit InnoDB's 3072-byte key limit; every fact is a 'sales_order'.
ALTER TABLE `sales_fact_rollup`
  ADD KEY `sales_fact_rollup_dimension_id_idx` (`account_id`, `dimension`, `product_line_key`, `dimension_id`, `grain`, `bucket_start`),
  ADD KEY `sales_fact_rollup_sales_rep_idx` (`account_id`, `dimension`, `product_line_key`, `sales_rep_key`, `grain`, `bucket_start`);

-- +goose Down
ALTER TABLE `sales_fact_rollup`
  DROP KEY `sales_fact_rollup_sales_rep_idx`,
  DROP KEY `sales_fact_rollup_dimension_id_idx`;
