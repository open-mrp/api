-- +goose NO TRANSACTION
-- +goose Up
-- Operations money moves out of quantity and rate rows in a currency unit onto the row that owns it: an
-- amount, its ISO 4217 currency, and for a price the unit it is per. Amounts keep the rate and quantity
-- rows' decimal(65,30), so a value reads back exactly as it did from them.
--
-- Writers fill these alongside the old rows, which stay authoritative for older code until CLEAN drops
-- their ids. Rows written before this migration are filled by the paced MoneyColumnBackfill; until a row
-- has its columns, readers fall back to its old row.
ALTER TABLE `inventory_receipt`
  ADD COLUMN `unit_cost_amount` decimal(65,30) NULL,
  ADD COLUMN `unit_cost_currency` char(3) CHARACTER SET ascii COLLATE ascii_bin NULL,
  ADD COLUMN `unit_cost_per_unit_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL;

ALTER TABLE `inventory_allocation`
  ADD COLUMN `unit_cost_amount` decimal(65,30) NULL,
  ADD COLUMN `unit_cost_currency` char(3) CHARACTER SET ascii COLLATE ascii_bin NULL,
  ADD COLUMN `unit_cost_per_unit_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL,
  ADD COLUMN `total_cost_amount` decimal(65,30) NULL,
  ADD COLUMN `total_cost_currency` char(3) CHARACTER SET ascii COLLATE ascii_bin NULL;

ALTER TABLE `delivery_line`
  ADD COLUMN `unit_cost_amount` decimal(65,30) NULL,
  ADD COLUMN `unit_cost_currency` char(3) CHARACTER SET ascii COLLATE ascii_bin NULL,
  ADD COLUMN `unit_cost_per_unit_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL;

ALTER TABLE `department`
  ADD COLUMN `labor_rate_amount` decimal(65,30) NULL,
  ADD COLUMN `labor_rate_currency` char(3) CHARACTER SET ascii COLLATE ascii_bin NULL,
  ADD COLUMN `labor_rate_per_unit_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL;

ALTER TABLE `production_step`
  ADD COLUMN `labor_rate_amount` decimal(65,30) NULL,
  ADD COLUMN `labor_rate_currency` char(3) CHARACTER SET ascii COLLATE ascii_bin NULL,
  ADD COLUMN `labor_rate_per_unit_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL,
  ADD COLUMN `overhead_rate_amount` decimal(65,30) NULL,
  ADD COLUMN `overhead_rate_currency` char(3) CHARACTER SET ascii COLLATE ascii_bin NULL,
  ADD COLUMN `overhead_rate_per_unit_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL;

-- A posting's money is its amount; amount_amount would only repeat it.
ALTER TABLE `journal_posting`
  ADD COLUMN `amount` decimal(65,30) NULL,
  ADD COLUMN `currency` char(3) CHARACTER SET ascii COLLATE ascii_bin NULL;

-- +goose Down
ALTER TABLE `journal_posting`
  DROP COLUMN `currency`,
  DROP COLUMN `amount`;

ALTER TABLE `production_step`
  DROP COLUMN `overhead_rate_per_unit_id`,
  DROP COLUMN `overhead_rate_currency`,
  DROP COLUMN `overhead_rate_amount`,
  DROP COLUMN `labor_rate_per_unit_id`,
  DROP COLUMN `labor_rate_currency`,
  DROP COLUMN `labor_rate_amount`;

ALTER TABLE `department`
  DROP COLUMN `labor_rate_per_unit_id`,
  DROP COLUMN `labor_rate_currency`,
  DROP COLUMN `labor_rate_amount`;

ALTER TABLE `delivery_line`
  DROP COLUMN `unit_cost_per_unit_id`,
  DROP COLUMN `unit_cost_currency`,
  DROP COLUMN `unit_cost_amount`;

ALTER TABLE `inventory_allocation`
  DROP COLUMN `total_cost_currency`,
  DROP COLUMN `total_cost_amount`,
  DROP COLUMN `unit_cost_per_unit_id`,
  DROP COLUMN `unit_cost_currency`,
  DROP COLUMN `unit_cost_amount`;

ALTER TABLE `inventory_receipt`
  DROP COLUMN `unit_cost_per_unit_id`,
  DROP COLUMN `unit_cost_currency`,
  DROP COLUMN `unit_cost_amount`;
