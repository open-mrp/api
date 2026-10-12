-- +goose NO TRANSACTION
-- +goose Up
-- A report filtered to customers reads a buyer's lines off sales_line_fact_buyer_idx, then looks each
-- one up in the clustered key, which orders lines by date: one buyer's lines sit on a different page
-- each, so a cold report reads a page per line. sales_line_fact_buyer_cover_idx carries every column
-- the totals and breakdowns read, so a buyer's lines are one contiguous range with no lookups.
-- sales_line_fact_buyer_idx stays for the line page, which reads whole rows, and for a customer filter
-- beside another filter, where the planner must weigh the two ranges' lengths alone.
--
-- At varchar(191) those columns overrun InnoDB's 3072-byte key limit. Ids are at most
-- id.MaxLength characters, so the id columns narrow to it first. The index keeps
-- (invoiced_at, invoice_line_id) right after the buyer, matching the narrow key's order.
ALTER TABLE `sales_line_fact`
  MODIFY `account_id` varchar(64) NOT NULL,
  MODIFY `invoice_line_id` varchar(64) NOT NULL,
  MODIFY `invoice_id` varchar(64) NOT NULL,
  MODIFY `buyer_account_id` varchar(64) NOT NULL,
  MODIFY `sales_rep_id` varchar(64) NULL,
  MODIFY `order_discount_id` varchar(64) NULL,
  MODIFY `item_id` varchar(64) NOT NULL,
  MODIFY `product_line_id` varchar(64) NOT NULL,
  ADD KEY `sales_line_fact_buyer_cover_idx` (`account_id`, `buyer_account_id`, `invoiced_at`, `invoice_line_id`,
    `sales_order_type_code`, `invoice_id`, `item_id`, `product_line_id`, `sales_rep_id`, `order_discount_id`,
    `total_invoiced`, `total_cost`, `quantity_base`);

-- +goose Down
ALTER TABLE `sales_line_fact`
  DROP KEY `sales_line_fact_buyer_cover_idx`,
  MODIFY `account_id` varchar(191) NOT NULL,
  MODIFY `invoice_line_id` varchar(191) NOT NULL,
  MODIFY `invoice_id` varchar(191) NOT NULL,
  MODIFY `buyer_account_id` varchar(191) NOT NULL,
  MODIFY `sales_rep_id` varchar(191) NULL,
  MODIFY `order_discount_id` varchar(191) NULL,
  MODIFY `item_id` varchar(191) NOT NULL,
  MODIFY `product_line_id` varchar(191) NOT NULL;
