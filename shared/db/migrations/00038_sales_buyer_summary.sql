-- +goose NO TRANSACTION
-- +goose Up
-- Each line's order date and whether its unit price is positive, so a customer's first order and
-- lifetime sales can be derived from the facts alone (the new-customers report). The refresher compares
-- every column, so an edited order date or price reaches the facts like any other change. Existing rows
-- are filled by the next reconcile pass, which the buyer summary sweep starts and waits for.
ALTER TABLE `sales_line_fact`
  ADD COLUMN `ordered_at` datetime(3) NULL,
  ADD COLUMN `is_priced` tinyint(1) NOT NULL DEFAULT 0,
  ADD KEY `sales_line_fact_buyer_idx` (`account_id`, `buyer_account_id`);

-- One row per customer with at least one qualifying sale: a sales order line priced above zero, outside
-- the shipping and misc product lines, on an order with an issue date. first_ordered_at is the earliest
-- such order's issue date and total_invoiced the lines' lifetime sum, as the legacy report computed them.
CREATE TABLE `sales_buyer_summary` (
  `account_id` varchar(191) NOT NULL,
  `buyer_account_id` varchar(191) NOT NULL,
  `first_ordered_at` datetime(3) NOT NULL,
  `total_invoiced` decimal(65,30) NOT NULL,
  `refreshed_at` datetime(3) NOT NULL,
  PRIMARY KEY (`account_id`, `buyer_account_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- sales_fact_dirty becomes the one queue of everything derived from the facts: besides the fact scopes it
-- holds 'buyer_summary' and 'rollup_day' marks, each kind drained on its own through this index.
ALTER TABLE `sales_fact_dirty`
  ADD KEY `sales_fact_dirty_scope_marked_idx` (`scope_type`, `marked_at`);

-- +goose Down
ALTER TABLE `sales_fact_dirty` DROP KEY `sales_fact_dirty_scope_marked_idx`;
DROP TABLE IF EXISTS `sales_buyer_summary`;
ALTER TABLE `sales_line_fact`
  DROP KEY `sales_line_fact_buyer_idx`,
  DROP COLUMN `is_priced`,
  DROP COLUMN `ordered_at`;
