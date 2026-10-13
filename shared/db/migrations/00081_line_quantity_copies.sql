-- +goose NO TRANSACTION
-- +goose Up
-- Order lines, invoice lines and pick lines each own their quantity row, and an order line its price
-- and cost rates (the ids are unique per line). Reports that price or count many lines looked each of
-- those rows up by its random id, a page apiece when cold. Each line now carries a copy of its rows'
-- values and units, written in the same transaction as the row, and the new keys hold an order's lines
-- (or an order line's invoice and pick lines) together with those copies, so a report reads them
-- without visiting the line or its quantity and rate rows.
--
-- The existing lines are filled by a paced backfill (LineCopyBackfill); reports read the rows themselves
-- until it has finished. The existing keys stay, since queries force them by name.
--
-- Unit ids are generated ASCII, so their copies are stored as ASCII: a key covers a query only while it
-- and the primary key it carries fit InnoDB's 3072 bytes, which utf8mb4 copies would overrun.
ALTER TABLE `sales_order_line`
  ADD COLUMN `quantity_value` decimal(65,30) NULL,
  ADD COLUMN `quantity_unit_id` varchar(64) CHARACTER SET ascii COLLATE ascii_bin NULL,
  ADD COLUMN `unit_price_value` decimal(65,30) NULL,
  ADD COLUMN `unit_price_numerator_unit_id` varchar(64) CHARACTER SET ascii COLLATE ascii_bin NULL,
  ADD COLUMN `unit_price_denominator_unit_id` varchar(64) CHARACTER SET ascii COLLATE ascii_bin NULL,
  ADD COLUMN `unit_cost_value` decimal(65,30) NULL,
  ADD COLUMN `unit_cost_numerator_unit_id` varchar(64) CHARACTER SET ascii COLLATE ascii_bin NULL,
  ADD COLUMN `unit_cost_denominator_unit_id` varchar(64) CHARACTER SET ascii COLLATE ascii_bin NULL,
  ADD KEY `sales_order_line_order_copies_idx` (`sales_order_id`, `product_id`, `quantity_value`, `quantity_unit_id`,
    `unit_price_value`, `unit_price_numerator_unit_id`, `unit_price_denominator_unit_id`,
    `unit_cost_value`, `unit_cost_numerator_unit_id`, `unit_cost_denominator_unit_id`);

ALTER TABLE `invoice_line`
  ADD COLUMN `quantity_value` decimal(65,30) NULL,
  ADD COLUMN `quantity_unit_id` varchar(64) CHARACTER SET ascii COLLATE ascii_bin NULL,
  ADD KEY `invoice_line_sol_copies_idx` (`sales_order_line_id`, `quantity_value`, `quantity_unit_id`);

ALTER TABLE `pick_line`
  ADD COLUMN `quantity_value` decimal(65,30) NULL,
  ADD COLUMN `quantity_unit_id` varchar(64) CHARACTER SET ascii COLLATE ascii_bin NULL,
  ADD KEY `pick_line_sol_packed_copies_idx` (`sales_order_line_id`, `packed_at`, `quantity_value`, `quantity_unit_id`);

-- +goose Down
ALTER TABLE `pick_line`
  DROP KEY `pick_line_sol_packed_copies_idx`,
  DROP COLUMN `quantity_unit_id`,
  DROP COLUMN `quantity_value`;

ALTER TABLE `invoice_line`
  DROP KEY `invoice_line_sol_copies_idx`,
  DROP COLUMN `quantity_unit_id`,
  DROP COLUMN `quantity_value`;

ALTER TABLE `sales_order_line`
  DROP KEY `sales_order_line_order_copies_idx`,
  DROP COLUMN `unit_cost_denominator_unit_id`,
  DROP COLUMN `unit_cost_numerator_unit_id`,
  DROP COLUMN `unit_cost_value`,
  DROP COLUMN `unit_price_denominator_unit_id`,
  DROP COLUMN `unit_price_numerator_unit_id`,
  DROP COLUMN `unit_price_value`,
  DROP COLUMN `quantity_unit_id`,
  DROP COLUMN `quantity_value`;
