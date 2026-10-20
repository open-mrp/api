-- +goose NO TRANSACTION
-- +goose Up

-- Reports that read an invoice's lines with their quantities (the demand forecast's monthly revenue) reached
-- each line through its invoice's key and then looked up the line, and its quantity row, by random id: a page
-- apiece when cold. This key holds an invoice's lines together with the order line they invoice and the
-- quantity copy they carry (00081), so those reports read an invoice's lines as one range.
--
-- It extends invoice_line_invoice_id_sales_order_line_id_idx, which no query names, so that key is dropped.
ALTER TABLE `invoice_line`
  ADD KEY `invoice_line_invoice_copies_idx` (`invoice_id`, `sales_order_line_id`, `quantity_value`, `quantity_unit_id`),
  DROP KEY `invoice_line_invoice_id_sales_order_line_id_idx`;

-- +goose Down
ALTER TABLE `invoice_line`
  ADD KEY `invoice_line_invoice_id_sales_order_line_id_idx` (`invoice_id`, `sales_order_line_id`),
  DROP KEY `invoice_line_invoice_copies_idx`;
