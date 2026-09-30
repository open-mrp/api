-- +goose NO TRANSACTION
-- +goose Up
-- Sales reports filtered to several product lines read their money from the per-product-line rollup
-- rows, which add up across lines. Invoice counts do not (an invoice can span two of the lines), so they
-- are counted from the facts: this index reads only the chosen lines' entries in the window, and covers
-- the count outright for a summary. (Four varchar(191) columns are 3056 of InnoDB's 3072 key bytes, so
-- a breakdown's group column is read from the row.)
ALTER TABLE `sales_line_fact`
  ADD KEY `sales_line_fact_product_line_idx` (`account_id`, `sales_order_type_code`, `product_line_id`, `invoiced_at`, `invoice_id`);

-- +goose Down
ALTER TABLE `sales_line_fact` DROP KEY `sales_line_fact_product_line_idx`;
