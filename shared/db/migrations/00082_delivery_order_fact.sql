-- +goose NO TRANSACTION
-- +goose Up
-- Delivery performance read every committed order in its window through its lines, their quantities and
-- the packed quantities of their pick lines: several lookups per line into large tables, repeated on every
-- read. delivery_order_fact holds each committed order's sale lines already summed per product line, so a
-- window is one range read of a single table.
--
-- Quantities are the raw stored values summed exactly as the live query sums them (no unit conversion), so
-- re-summing an order's rows over its product lines equals the live per-order sum. Names and customer
-- groups are joined at read time, after aggregation, so renames never leave a row stale.
--
-- Only orders the report measures have rows: sales orders, issued or fulfilled, with a ship-by date, and
-- only their sale-type lines. The clustered key leads with (owner_account_id, ship_by_date), the report's
-- scope and range. product_line_key is the line's product line, or '' for lines with none.
CREATE TABLE `delivery_order_fact` (
  `owner_account_id` varchar(191) NOT NULL,
  `ship_by_date` date NOT NULL,
  `sales_order_id` varchar(191) NOT NULL,
  `product_line_key` varchar(191) NOT NULL,
  `sales_order_number` varchar(255) NOT NULL,
  `buyer_account_id` varchar(191) NOT NULL,
  `sales_rep_id` varchar(191) NULL,
  `sales_order_status_code` varchar(32) NOT NULL,
  `issued_at` datetime(3) NULL,
  `first_ship_at` datetime(3) NULL,
  `completed_at` datetime(3) NULL,
  `lead_time_days` int NULL,
  `lead_time_source_code` varchar(32) NULL,
  `quantity_ordered` decimal(65,30) NOT NULL,
  `quantity_packed` decimal(65,30) NOT NULL,
  `refreshed_at` datetime(3) NOT NULL,
  PRIMARY KEY (`owner_account_id`, `ship_by_date`, `sales_order_id`, `product_line_key`),
  KEY `delivery_order_fact_order_idx` (`sales_order_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Scopes whose orders' facts must be recomputed: an order, or a line, pick, pick line, quantity or product
-- the refresher resolves to orders. A mark is deleted only if marked_at is unchanged, so a re-mark that
-- lands mid-refresh survives for the next drain.
CREATE TABLE `delivery_fact_dirty` (
  `scope_type` varchar(32) NOT NULL,
  `scope_id` varchar(191) NOT NULL,
  `account_id` varchar(191) NOT NULL,
  `marked_at` datetime(3) NOT NULL,
  PRIMARY KEY (`scope_type`, `scope_id`),
  KEY `delivery_fact_dirty_marked_idx` (`marked_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- The delivery reconcile walks sales orders by id; its row in sales_sync keeps the last one it reached.
ALTER TABLE `sales_sync` ADD COLUMN `cursor_sales_order_id` varchar(191) NULL AFTER `cursor_buyer_account_id`;

-- +goose Down
ALTER TABLE `sales_sync` DROP COLUMN `cursor_sales_order_id`;
DROP TABLE IF EXISTS `delivery_fact_dirty`;
DROP TABLE IF EXISTS `delivery_order_fact`;
