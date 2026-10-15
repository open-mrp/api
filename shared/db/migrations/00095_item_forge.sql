-- +goose NO TRANSACTION
-- +goose Up

-- Items can be archived: they stay on the records using them but can't be added to new orders, production or
-- purchasing. Items soft-deleted before this read as archived until the deleted_at column is dropped.
-- An item names where it comes from (stock | make | buy); null inherits its product line's, then the account's.
ALTER TABLE `item`
  ADD COLUMN `is_active` tinyint(1) NOT NULL DEFAULT '1',
  ADD COLUMN `supply_code` varchar(32) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL;

-- Search is a tenant-scoped LIKE on sku and description; the word-parser FULLTEXT indexes are unused.
ALTER TABLE `item`
  DROP KEY `item_description_idx`,
  DROP KEY `item_sku_description_idx`;

-- The sales lead time of a sold item: how long production or purchase takes before it can ship.
ALTER TABLE `product` ADD COLUMN `lead_time_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL;

-- Deleting an item probes the planned orders and downtime events naming it.
ALTER TABLE `production_schedule_line` ADD KEY `production_schedule_line_item_id_idx` (`item_id`);
ALTER TABLE `machine_downtime_event` ADD KEY `machine_downtime_event_item_id_idx` (`item_id`);

-- +goose Down

ALTER TABLE `machine_downtime_event` DROP KEY `machine_downtime_event_item_id_idx`;
ALTER TABLE `production_schedule_line` DROP KEY `production_schedule_line_item_id_idx`;
ALTER TABLE `product` DROP COLUMN `lead_time_id`;
ALTER TABLE `item`
  ADD FULLTEXT KEY `item_description_idx` (`description`),
  ADD FULLTEXT KEY `item_sku_description_idx` (`sku`, `description`);
ALTER TABLE `item`
  DROP COLUMN `supply_code`,
  DROP COLUMN `is_active`;
