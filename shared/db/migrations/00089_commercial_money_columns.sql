-- +goose NO TRANSACTION
-- +goose Up
-- Money moves out of quantity and rate rows in a currency unit onto the row that owns it: an amount, a
-- currency code and, for a price, the unit it is per. These columns are a copy of the old rows,
-- written in the same transaction as them, and a paced backfill (MoneyColumnBackfill) fills the rows
-- written before they existed. The old rows stay the record until every writer has moved and the
-- backfills have run; their id columns are dropped after that.
--
-- An order line's price and cost already have copies of their value and per-unit (00081), so it only
-- gains the currency of each.
ALTER TABLE `item`
  ADD COLUMN `sales_unit_price_amount` decimal(65,30) NULL,
  ADD COLUMN `sales_unit_price_currency` char(3) CHARACTER SET ascii COLLATE ascii_bin NULL,
  ADD COLUMN `sales_unit_price_per_unit_id` varchar(191) NULL,
  ADD COLUMN `unit_cost_amount` decimal(65,30) NULL,
  ADD COLUMN `unit_cost_currency` char(3) CHARACTER SET ascii COLLATE ascii_bin NULL,
  ADD COLUMN `unit_cost_per_unit_id` varchar(191) NULL;

ALTER TABLE `sales_order_line`
  ADD COLUMN `unit_price_currency` char(3) CHARACTER SET ascii COLLATE ascii_bin NULL,
  ADD COLUMN `unit_cost_currency` char(3) CHARACTER SET ascii COLLATE ascii_bin NULL;

ALTER TABLE `account_relation`
  ADD COLUMN `credit_limit_amount` decimal(65,30) NULL,
  ADD COLUMN `credit_limit_currency` char(3) CHARACTER SET ascii COLLATE ascii_bin NULL;

ALTER TABLE `shipping_term`
  ADD COLUMN `flat_rate_amount` decimal(65,30) NULL,
  ADD COLUMN `flat_rate_currency` char(3) CHARACTER SET ascii COLLATE ascii_bin NULL,
  ADD COLUMN `minimum_order_value_amount` decimal(65,30) NULL,
  ADD COLUMN `minimum_order_value_currency` char(3) CHARACTER SET ascii COLLATE ascii_bin NULL;

ALTER TABLE `shipping_case`
  ADD COLUMN `freight_amount` decimal(65,30) NULL,
  ADD COLUMN `freight_currency` char(3) CHARACTER SET ascii COLLATE ascii_bin NULL;

ALTER TABLE `target`
  ADD COLUMN `amount` decimal(65,30) NULL,
  ADD COLUMN `currency` char(3) CHARACTER SET ascii COLLATE ascii_bin NULL;

ALTER TABLE `product_line_target`
  ADD COLUMN `amount` decimal(65,30) NULL,
  ADD COLUMN `currency` char(3) CHARACTER SET ascii COLLATE ascii_bin NULL;

-- +goose Down
ALTER TABLE `product_line_target`
  DROP COLUMN `currency`,
  DROP COLUMN `amount`;

ALTER TABLE `target`
  DROP COLUMN `currency`,
  DROP COLUMN `amount`;

ALTER TABLE `shipping_case`
  DROP COLUMN `freight_currency`,
  DROP COLUMN `freight_amount`;

ALTER TABLE `shipping_term`
  DROP COLUMN `minimum_order_value_currency`,
  DROP COLUMN `minimum_order_value_amount`,
  DROP COLUMN `flat_rate_currency`,
  DROP COLUMN `flat_rate_amount`;

ALTER TABLE `account_relation`
  DROP COLUMN `credit_limit_currency`,
  DROP COLUMN `credit_limit_amount`;

ALTER TABLE `sales_order_line`
  DROP COLUMN `unit_cost_currency`,
  DROP COLUMN `unit_price_currency`;

ALTER TABLE `item`
  DROP COLUMN `unit_cost_per_unit_id`,
  DROP COLUMN `unit_cost_currency`,
  DROP COLUMN `unit_cost_amount`,
  DROP COLUMN `sales_unit_price_per_unit_id`,
  DROP COLUMN `sales_unit_price_currency`,
  DROP COLUMN `sales_unit_price_amount`;
