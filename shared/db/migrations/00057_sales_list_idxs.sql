-- +goose NO TRANSACTION
-- +goose Up
-- The sales order list's per-filter keys were descending. InnoDB scans an ascending key in both
-- directions and a descending one only forward, so the previous page (read oldest first) sorted the
-- filter's whole range. Rebuilt ascending under the same names, which the list's FORCE INDEX names.
-- The ship-by key ranges a ship-by window when no status narrows it.
ALTER TABLE `sales_order`
  DROP KEY `sales_order_owner_created_idx`,
  ADD KEY `sales_order_owner_created_idx` (`owner_account_id`, `created_at`, `id`),
  DROP KEY `sales_order_owner_status_created_idx`,
  ADD KEY `sales_order_owner_status_created_idx` (`owner_account_id`, `sales_order_status_code`, `created_at`, `id`),
  DROP KEY `sales_order_owner_buyer_created_idx`,
  ADD KEY `sales_order_owner_buyer_created_idx` (`owner_account_id`, `buyer_account_id`, `created_at`, `id`),
  DROP KEY `sales_order_owner_sales_rep_created_idx`,
  ADD KEY `sales_order_owner_sales_rep_created_idx` (`owner_account_id`, `sales_rep_id`, `created_at`, `id`),
  ADD KEY `sales_order_owner_ship_by_idx` (`owner_account_id`, `ship_by_date`, `id`);

-- The sales line list reads an account's facts newest first, so each filter's key ends in that order and
-- a list on one filter reads a page and stops. The buyer key already held these columns (InnoDB appends
-- the clustered key), but the planner only sorts by columns a key names.
ALTER TABLE `sales_line_fact`
  DROP KEY `sales_line_fact_buyer_idx`,
  ADD KEY `sales_line_fact_buyer_idx` (`account_id`, `buyer_account_id`, `invoiced_at`, `invoice_line_id`),
  ADD KEY `sales_line_fact_item_idx` (`account_id`, `item_id`, `invoiced_at`, `invoice_line_id`),
  ADD KEY `sales_line_fact_sales_rep_idx` (`account_id`, `sales_rep_id`, `invoiced_at`, `invoice_line_id`),
  ADD KEY `sales_line_fact_account_product_line_idx` (`account_id`, `product_line_id`, `invoiced_at`, `invoice_line_id`);

-- The invoice list's keys, ascending for the same reason as the sales order ones, with id so the order
-- is total. The unpaid key keeps sales_order_id, which the dashboard's unpaid list reads from it. The
-- order key carries is_paid_in_full, so a customer's unpaid invoices are found without reading its paid ones.
ALTER TABLE `invoice`
  DROP KEY `invoice_account_sales_order_idx`,
  ADD KEY `invoice_account_sales_order_idx` (`account_id`, `sales_order_id`, `is_paid_in_full`),
  DROP KEY `invoice_account_created_idx`,
  ADD KEY `invoice_account_created_idx` (`account_id`, `created_at`, `id`),
  DROP KEY `invoice_account_unpaid_created_idx`,
  ADD KEY `invoice_account_unpaid_created_idx` (`account_id`, `is_paid_in_full`, `created_at`, `id`, `sales_order_id`),
  ADD KEY `invoice_account_over_paid_created_idx` (`account_id`, `is_over_paid`, `created_at`, `id`);

-- +goose Down
ALTER TABLE `invoice`
  DROP KEY `invoice_account_over_paid_created_idx`,
  DROP KEY `invoice_account_unpaid_created_idx`,
  ADD KEY `invoice_account_unpaid_created_idx` (`account_id`, `is_paid_in_full`, `created_at` DESC, `sales_order_id`),
  DROP KEY `invoice_account_created_idx`,
  ADD KEY `invoice_account_created_idx` (`account_id`, `created_at` DESC, `id` DESC),
  DROP KEY `invoice_account_sales_order_idx`,
  ADD KEY `invoice_account_sales_order_idx` (`account_id`, `sales_order_id`);

ALTER TABLE `sales_line_fact`
  DROP KEY `sales_line_fact_account_product_line_idx`,
  DROP KEY `sales_line_fact_sales_rep_idx`,
  DROP KEY `sales_line_fact_item_idx`,
  DROP KEY `sales_line_fact_buyer_idx`,
  ADD KEY `sales_line_fact_buyer_idx` (`account_id`, `buyer_account_id`);

ALTER TABLE `sales_order`
  DROP KEY `sales_order_owner_ship_by_idx`,
  DROP KEY `sales_order_owner_sales_rep_created_idx`,
  ADD KEY `sales_order_owner_sales_rep_created_idx` (`owner_account_id`, `sales_rep_id`, `created_at` DESC, `id` DESC),
  DROP KEY `sales_order_owner_buyer_created_idx`,
  ADD KEY `sales_order_owner_buyer_created_idx` (`owner_account_id`, `buyer_account_id`, `created_at` DESC, `id` DESC),
  DROP KEY `sales_order_owner_status_created_idx`,
  ADD KEY `sales_order_owner_status_created_idx` (`owner_account_id`, `sales_order_status_code`, `created_at` DESC, `id` DESC),
  DROP KEY `sales_order_owner_created_idx`,
  ADD KEY `sales_order_owner_created_idx` (`owner_account_id`, `created_at` DESC, `id` DESC);
