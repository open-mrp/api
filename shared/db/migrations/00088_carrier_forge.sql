-- +goose NO TRANSACTION
-- +goose Up

-- Carriers and service levels can be archived: they stay on the records using them but can't be newly assigned.
-- A carrier names its default service level instead of the service level flagging itself, and carries the SCAC
-- that EDI documents and bills of lading identify it by.
ALTER TABLE `carrier`
  ADD COLUMN `is_active` tinyint(1) NOT NULL DEFAULT '1',
  ADD COLUMN `scac` varchar(4) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  ADD COLUMN `default_carrier_option_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  ADD KEY `carrier_default_carrier_option_id_idx` (`default_carrier_option_id`);

ALTER TABLE `carrier_option` ADD COLUMN `is_active` tinyint(1) NOT NULL DEFAULT '1';

-- Name search is a tenant-scoped substring LIKE; the word-parser FULLTEXT indexes are unused.
ALTER TABLE `carrier`
  DROP KEY `carrier_name_idx`,
  DROP KEY `carrier_description_idx`,
  DROP KEY `carrier_name_description_idx`;
ALTER TABLE `carrier_option`
  DROP KEY `carrier_option_name_idx`,
  DROP KEY `carrier_option_code_idx`,
  DROP KEY `carrier_option_name_code_idx`;

-- +goose Down

ALTER TABLE `carrier_option`
  ADD FULLTEXT KEY `carrier_option_name_idx` (`name`),
  ADD FULLTEXT KEY `carrier_option_code_idx` (`code`),
  ADD FULLTEXT KEY `carrier_option_name_code_idx` (`name`, `code`);
ALTER TABLE `carrier`
  ADD FULLTEXT KEY `carrier_name_idx` (`name`),
  ADD FULLTEXT KEY `carrier_description_idx` (`description`),
  ADD FULLTEXT KEY `carrier_name_description_idx` (`name`, `description`);
ALTER TABLE `carrier_option` DROP COLUMN `is_active`;
ALTER TABLE `carrier`
  DROP KEY `carrier_default_carrier_option_id_idx`,
  DROP COLUMN `default_carrier_option_id`,
  DROP COLUMN `scac`,
  DROP COLUMN `is_active`;
