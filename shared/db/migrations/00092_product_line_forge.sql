-- +goose NO TRANSACTION
-- +goose Up

-- Product lines can be archived: they stay on products but can't be newly assigned or granted.
ALTER TABLE `product_line` ADD COLUMN `is_active` tinyint(1) NOT NULL DEFAULT '1';

-- A line's default supply (stock | make | buy) replaces its fulfillment policy.
ALTER TABLE `product_line` ADD COLUMN `supply_code` varchar(32) NULL;

-- The unit group is optional: products of a line without one are measured in their category's group.
ALTER TABLE `product_line` MODIFY `unit_group_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL;

-- Name search is a tenant-scoped substring LIKE; the word-parser FULLTEXT indexes are unused.
ALTER TABLE `product_line`
  DROP KEY `product_line_name_idx`,
  DROP KEY `product_line_description_idx`,
  DROP KEY `product_line_name_description_idx`;

-- The lot a category's items are made in moves here from product lines.
ALTER TABLE `item_category` ADD COLUMN `default_lot_size_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL;

-- +goose Down

ALTER TABLE `item_category` DROP COLUMN `default_lot_size_id`;
ALTER TABLE `product_line`
  ADD FULLTEXT KEY `product_line_name_idx` (`name`),
  ADD FULLTEXT KEY `product_line_description_idx` (`description`),
  ADD FULLTEXT KEY `product_line_name_description_idx` (`name`, `description`);
ALTER TABLE `product_line` MODIFY `unit_group_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL;
ALTER TABLE `product_line` DROP COLUMN `supply_code`;
ALTER TABLE `product_line` DROP COLUMN `is_active`;
