-- +goose NO TRANSACTION
-- +goose Up
-- The sales-order list sums each line's picked/packed/invoiced quantity from pick_line and invoice_line.
-- Aggregating each child once per page still does a clustered back-lookup per child row for quantity_id
-- (and packed_at) off the single-column sales_order_line_id index. These composites cover that side of
-- the aggregation: sales_order_line_id leads the equality join, quantity_id lets the sum read value's
-- owning quantity by PK without touching the child row, and on pick_line packed_at sits second so
-- `packed_at IS NOT NULL` (the packed subtotal and the ship-by report) is an index range, not a residual
-- filter. Each new composite left-prefixes the single-column index it replaces, so that index is dropped.
ALTER TABLE `pick_line`
  ADD KEY `pick_line_sol_packed_qty_idx` (`sales_order_line_id`, `packed_at`, `quantity_id`),
  DROP KEY `pick_line_sales_order_line_id_idx`;

ALTER TABLE `invoice_line`
  ADD KEY `invoice_line_sol_qty_idx` (`sales_order_line_id`, `quantity_id`),
  DROP KEY `invoice_line_sales_order_line_id_idx`;

-- +goose Down
ALTER TABLE `invoice_line`
  ADD KEY `invoice_line_sales_order_line_id_idx` (`sales_order_line_id`),
  DROP KEY `invoice_line_sol_qty_idx`;

ALTER TABLE `pick_line`
  ADD KEY `pick_line_sales_order_line_id_idx` (`sales_order_line_id`),
  DROP KEY `pick_line_sol_packed_qty_idx`;
