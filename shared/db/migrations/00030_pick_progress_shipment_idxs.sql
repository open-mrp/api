-- +goose NO TRANSACTION
-- +goose Up

-- GetPickProgress rolls up pick_line per pick, filtering on pick_id and joining on sales_order_line_id and quantity_id
-- with a packed_at CASE. The only usable index is the single-column pick_id, so the aggregate reads each matched
-- pick_line row for the join keys. This composite covers the query end to end.
ALTER TABLE `pick_line`
  ADD KEY `pick_line_pick_sol_qty_packed_idx` (`pick_id`, `sales_order_line_id`, `quantity_id`, `packed_at`);

-- The pick header subquery MAX(shipped_at) per sales order can only use the single-column sales_order_id index today,
-- then reads shipped_at off each matched row. Trailing shipped_at lets it satisfy the MAX from the index alone.
ALTER TABLE `shipment`
  ADD KEY `shipment_sales_order_shipped_idx` (`sales_order_id`, `shipped_at`);

-- +goose Down

ALTER TABLE `shipment`
  DROP KEY `shipment_sales_order_shipped_idx`;

ALTER TABLE `pick_line`
  DROP KEY `pick_line_pick_sol_qty_packed_idx`;
