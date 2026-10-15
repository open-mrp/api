-- +goose NO TRANSACTION
-- +goose Up

-- Attributes can be archived: they stay on items but can't be newly assigned.
ALTER TABLE `attribute` ADD COLUMN `is_active` tinyint(1) NOT NULL DEFAULT '1';

-- An attribute's value is unique within its property (case-insensitive by collation), not across the account.
-- The unique key leads with property_id, so it also serves every lookup the old property index did.
ALTER TABLE `attribute`
  ADD UNIQUE KEY `attribute_property_id_text_key` (`property_id`, `text`),
  DROP KEY `attribute_property_id_idx`,
  DROP KEY `attribute_text_idx`;

-- Name search is a tenant-scoped substring LIKE; the word-parser FULLTEXT indexes are unused.
ALTER TABLE `property` DROP KEY `property_name_idx`;
ALTER TABLE `item_category` DROP KEY `item_category_name_idx`;

-- +goose Down

ALTER TABLE `item_category` ADD FULLTEXT KEY `item_category_name_idx` (`name`);
ALTER TABLE `property` ADD FULLTEXT KEY `property_name_idx` (`name`);
ALTER TABLE `attribute`
  ADD KEY `attribute_property_id_idx` (`property_id`),
  ADD FULLTEXT KEY `attribute_text_idx` (`text`),
  DROP KEY `attribute_property_id_text_key`;
ALTER TABLE `attribute` DROP COLUMN `is_active`;
