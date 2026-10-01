-- +goose NO TRANSACTION
-- +goose Up
-- A report filtered to its own groups, or to a sales rep, pins them ahead of the buckets.
-- sales_order_type_code is left out to fit InnoDB's 3072-byte key limit; every fact is a 'sales_order'.
ALTER TABLE `sales_fact_rollup`
  ADD KEY `sales_fact_rollup_dimension_id_idx` (`account_id`, `dimension`, `product_line_key`, `dimension_id`, `grain`, `bucket_start`),
  ADD KEY `sales_fact_rollup_sales_rep_idx` (`account_id`, `dimension`, `product_line_key`, `sales_rep_key`, `grain`, `bucket_start`);

-- Delivery performance filtered to a customer or sales rep pins them ahead of the due date; the last key
-- finds the few orders issued without a due date without reading the window's committed ones.
ALTER TABLE `sales_order`
  ADD KEY `sales_order_owner_buyer_ship_by_idx` (`owner_account_id`, `buyer_account_id`, `ship_by_date`),
  ADD KEY `sales_order_owner_rep_ship_by_idx` (`owner_account_id`, `sales_rep_id`, `ship_by_date`),
  ADD KEY `sales_order_owner_ship_by_issued_idx` (`owner_account_id`, `ship_by_date`, `issued_at`);

-- +goose Down
ALTER TABLE `sales_order`
  DROP KEY `sales_order_owner_ship_by_issued_idx`,
  DROP KEY `sales_order_owner_rep_ship_by_idx`,
  DROP KEY `sales_order_owner_buyer_ship_by_idx`;

ALTER TABLE `sales_fact_rollup`
  DROP KEY `sales_fact_rollup_sales_rep_idx`,
  DROP KEY `sales_fact_rollup_dimension_id_idx`;
