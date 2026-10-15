-- +goose NO TRANSACTION
-- +goose Up

-- Locations can be archived: they keep their stock and history but can't be newly assigned.
ALTER TABLE `storage_location` ADD COLUMN `is_active` tinyint(1) NOT NULL DEFAULT '1';

-- The location list pages newest first within an account; name search is a tenant-scoped substring LIKE, so the
-- word-parser FULLTEXT index is unused.
ALTER TABLE `storage_location`
  ADD KEY `storage_location_account_id_created_at_id_idx` (`account_id`, `created_at`, `id`),
  DROP KEY `storage_location_name_idx`;

-- Every account has one default site: the location documents ship from, receive at and make in unless they name
-- another site.
ALTER TABLE `account`
  ADD COLUMN `default_site_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL,
  ADD KEY `account_default_site_id_idx` (`default_site_id`);

-- +goose Down

ALTER TABLE `account` DROP KEY `account_default_site_id_idx`, DROP COLUMN `default_site_id`;
ALTER TABLE `storage_location`
  ADD FULLTEXT KEY `storage_location_name_idx` (`name`),
  DROP KEY `storage_location_account_id_created_at_id_idx`;
ALTER TABLE `storage_location` DROP COLUMN `is_active`;
