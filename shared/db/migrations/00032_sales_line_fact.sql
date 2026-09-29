-- +goose NO TRANSACTION
-- +goose Up
-- Sales analytics priced every invoice line on read: ~18 joins through quantity, rate and unit, repeated
-- for every line in the window. sales_line_fact holds each invoice line already priced, so a report is
-- one range read of a single table.
--
-- The amounts are the legacy analytics expressions evaluated once (unrounded, DECIMAL(65,30)), so sums
-- over this table equal the old per-read sums exactly. Display names, customer groups and base-unit
-- labels are joined at read time, after aggregation, so renames never leave a row stale.
--
-- The clustered key leads with (account_id, invoiced_at): every report is an account-scoped date range,
-- and clustering on it makes that range a sequential read of whole rows with no back-lookups.
CREATE TABLE `sales_line_fact` (
  `account_id` varchar(191) NOT NULL,
  `invoiced_at` datetime(3) NOT NULL,
  `invoice_line_id` varchar(191) NOT NULL,
  `invoice_id` varchar(191) NOT NULL,
  `sales_order_id` varchar(191) NOT NULL,
  `sales_order_type_code` varchar(191) NOT NULL,
  `buyer_account_id` varchar(191) NOT NULL,
  `sales_rep_id` varchar(191) NULL,
  `order_discount_id` varchar(191) NULL,
  `product_id` varchar(191) NOT NULL,
  `item_id` varchar(191) NOT NULL,
  `product_line_id` varchar(191) NOT NULL,
  `quantity_base` decimal(65,30) NULL,
  `total_invoiced` decimal(65,30) NULL,
  `total_cost` decimal(65,30) NULL,
  `refreshed_at` datetime(3) NOT NULL,
  PRIMARY KEY (`account_id`, `invoiced_at`, `invoice_line_id`),
  UNIQUE KEY `sales_line_fact_invoice_line_key` (`invoice_line_id`),
  KEY `sales_line_fact_invoice_idx` (`invoice_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Scopes whose facts must be recomputed. Writers mark a scope (an invoice, sales order, order line or
-- product); the refresher resolves it to invoices, recomputes them, then deletes the mark only if
-- marked_at is unchanged, so a re-mark that lands mid-refresh survives for the next pass.
CREATE TABLE `sales_fact_dirty` (
  `scope_type` varchar(32) NOT NULL,
  `scope_id` varchar(191) NOT NULL,
  `account_id` varchar(191) NOT NULL,
  `marked_at` datetime(3) NOT NULL,
  PRIMARY KEY (`scope_type`, `scope_id`),
  KEY `sales_fact_dirty_marked_idx` (`marked_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Progress of the reconcile sweep, which walks every invoice in (created_at, id) order and corrects any
-- fact that drifted from its source; its first pass is the backfill. A pass in progress has a cursor;
-- last_completed_at is set when a pass reaches the end. Reads refuse to serve from sales_line_fact until
-- one pass has completed.
CREATE TABLE `sales_fact_sync` (
  `name` varchar(64) NOT NULL,
  `cursor_created_at` datetime(3) NULL,
  `cursor_invoice_id` varchar(191) NULL,
  `pass_started_at` datetime(3) NULL,
  `last_completed_at` datetime(3) NULL,
  `updated_at` datetime(3) NOT NULL,
  PRIMARY KEY (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- +goose Down
DROP TABLE `sales_fact_sync`;
DROP TABLE `sales_fact_dirty`;
DROP TABLE `sales_line_fact`;
