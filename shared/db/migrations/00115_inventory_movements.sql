-- +goose NO TRANSACTION
-- +goose Up
-- Inventory change logs become inventory movements: each row says what happened (type), who caused it (actor), where
-- and in which lot, why a correction was made, the document behind it, and the item's cost when it moved.
--
-- Rows written before this migration have none of these. Readers derive type and actor from action_type_code,
-- scanning_station_id and responsible_user_id until the paced inventory_movement_types backfill stores them.
ALTER TABLE `inventory_change_log`
  ADD COLUMN `type_code` varchar(32) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL,
  ADD COLUMN `actor_type` varchar(32) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL,
  ADD COLUMN `actor_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL,
  ADD COLUMN `location_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL,
  ADD COLUMN `lot_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL,
  ADD COLUMN `reason_code` varchar(32) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL,
  ADD COLUMN `related_type` varchar(32) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL,
  ADD COLUMN `related_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL,
  ADD COLUMN `unit_cost_amount` decimal(65,30) NULL,
  ADD COLUMN `unit_cost_currency` char(3) CHARACTER SET ascii COLLATE ascii_bin NULL,
  ADD COLUMN `unit_cost_per_unit_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL,
  ADD KEY `inventory_change_log_acct_type_created_idx` (`account_id`, `type_code`, `created_at` DESC, `id` DESC),
  ADD KEY `inventory_change_log_acct_actor_type_created_idx` (`account_id`, `actor_type`, `created_at` DESC, `id` DESC),
  ADD KEY `inventory_change_log_actor_created_idx` (`actor_id`, `created_at` DESC, `id` DESC),
  ADD KEY `inventory_change_log_location_created_idx` (`location_id`, `created_at` DESC, `id` DESC),
  ADD KEY `inventory_change_log_lot_created_idx` (`lot_id`, `created_at` DESC, `id` DESC);

-- +goose Down
ALTER TABLE `inventory_change_log`
  DROP KEY `inventory_change_log_lot_created_idx`,
  DROP KEY `inventory_change_log_location_created_idx`,
  DROP KEY `inventory_change_log_actor_created_idx`,
  DROP KEY `inventory_change_log_acct_actor_type_created_idx`,
  DROP KEY `inventory_change_log_acct_type_created_idx`,
  DROP COLUMN `unit_cost_per_unit_id`,
  DROP COLUMN `unit_cost_currency`,
  DROP COLUMN `unit_cost_amount`,
  DROP COLUMN `related_id`,
  DROP COLUMN `related_type`,
  DROP COLUMN `reason_code`,
  DROP COLUMN `lot_id`,
  DROP COLUMN `location_id`,
  DROP COLUMN `actor_id`,
  DROP COLUMN `actor_type`,
  DROP COLUMN `type_code`;
