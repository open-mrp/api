-- +goose NO TRANSACTION
-- +goose Up
-- invoiced_at ahead of the summed columns lets one buyer's history be read a window at a time.
-- The nightly buyer sweep reads each buyer's latest fact refresh from refreshed_idx, one entry per buyer.
ALTER TABLE `sales_line_fact`
  DROP KEY `sales_line_fact_buyer_summary_idx`,
  ADD KEY `sales_line_fact_buyer_summary_idx` (`account_id`, `buyer_account_id`, `sales_order_type_code`, `is_priced`, `invoiced_at`, `ordered_at`, `product_line_id`, `total_invoiced`),
  ADD KEY `sales_line_fact_buyer_refreshed_idx` (`account_id`, `buyer_account_id`, `refreshed_at`);

-- +goose Down
ALTER TABLE `sales_line_fact`
  DROP KEY `sales_line_fact_buyer_refreshed_idx`,
  DROP KEY `sales_line_fact_buyer_summary_idx`,
  ADD KEY `sales_line_fact_buyer_summary_idx` (`account_id`, `buyer_account_id`, `sales_order_type_code`, `is_priced`, `ordered_at`, `product_line_id`, `total_invoiced`);
