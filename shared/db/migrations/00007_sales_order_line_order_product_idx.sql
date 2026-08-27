-- +goose NO TRANSACTION
-- +goose Up

-- GetFrequentlyOrderedProducts reads exactly two columns from this table, sales_order_id to join and product_id to group, so widening the existing index by product_id makes that access index-only and drops one random clustered read per order line. The customer with the longest history scans ~1,700 lines per call.
-- sales_order_line_sales_order_id_idx is the exact left prefix of the new index, so it is replaced rather than supplemented: no query loses a lookup, and the table's index count is unchanged.
ALTER TABLE `sales_order_line`
  ADD KEY `sales_order_line_order_product_idx` (`sales_order_id`, `product_id`);

ALTER TABLE `sales_order_line`
  DROP KEY `sales_order_line_sales_order_id_idx`;

-- +goose Down

ALTER TABLE `sales_order_line`
  ADD KEY `sales_order_line_sales_order_id_idx` (`sales_order_id`);

ALTER TABLE `sales_order_line`
  DROP KEY `sales_order_line_order_product_idx`;
