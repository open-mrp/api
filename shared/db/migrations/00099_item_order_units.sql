-- +goose NO TRANSACTION
-- +goose Up

-- An item names its own units instead of taking them from its category's or product line's unit group: the unit its
-- inventory, costs and usage are kept in (null for a service, which is never stocked), the unit orders select by
-- default, and the other units it can be ordered in. default_order_unit_id is set on every item whose units are
-- stored; until the paced backfill (ItemUnitsBackfill) reaches an item, readers derive its units from the groups.
ALTER TABLE `item`
  ADD COLUMN `stocking_unit_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL,
  ADD COLUMN `default_order_unit_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL,
  ADD KEY `item_stocking_unit_id_idx` (`stocking_unit_id`),
  ADD KEY `item_default_order_unit_id_idx` (`default_order_unit_id`);

-- A unit an item can be ordered in besides its stocking unit, with its portal visibility, the discount taken off a
-- line's default price in it, and optionally how many stocking units one of it holds for this item, overriding the
-- unit's own ratio.
CREATE TABLE `item_order_unit` (
  `item_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `unit_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `account_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `position` int NOT NULL DEFAULT '0',
  `portal_visibility` varchar(32) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'visible',
  `discount_type` varchar(32) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL,
  `discount_percent_off` decimal(65,30) NULL,
  `discount_amount_off_amount` decimal(65,30) NULL,
  `discount_amount_off_currency` char(3) CHARACTER SET ascii COLLATE ascii_bin NULL,
  `stocking_quantity` decimal(65,30) NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`item_id`, `unit_id`),
  KEY `item_order_unit_unit_id_idx` (`unit_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- +goose Down

DROP TABLE IF EXISTS `item_order_unit`;
ALTER TABLE `item`
  DROP KEY `item_default_order_unit_id_idx`,
  DROP KEY `item_stocking_unit_id_idx`,
  DROP COLUMN `default_order_unit_id`,
  DROP COLUMN `stocking_unit_id`;
