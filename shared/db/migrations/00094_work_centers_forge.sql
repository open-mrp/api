-- +goose NO TRANSACTION
-- +goose Up

-- Machines can be archived: they keep their history but can't be newly assigned.
ALTER TABLE `machine` ADD COLUMN `is_active` tinyint(1) NOT NULL DEFAULT '1';

-- Work center, machine and scanning station lists page newest first within an account; name search is a
-- tenant-scoped LIKE, so the word-parser FULLTEXT indexes are unused.
ALTER TABLE `department`
  ADD KEY `department_account_id_created_at_id_idx` (`account_id`, `created_at`, `id`),
  DROP KEY `department_name_idx`;
ALTER TABLE `machine`
  ADD KEY `machine_account_id_created_at_id_idx` (`account_id`, `created_at`, `id`),
  DROP KEY `machine_name_idx`;
ALTER TABLE `scanning_station`
  ADD KEY `scanning_station_account_id_created_at_id_idx` (`account_id`, `created_at`, `id`),
  DROP KEY `scanning_station_name_idx`;

-- Deleting a machine probes its downtime events.
ALTER TABLE `machine_downtime_event` ADD KEY `machine_downtime_event_machine_id_idx` (`machine_id`);

-- +goose Down

ALTER TABLE `machine_downtime_event` DROP KEY `machine_downtime_event_machine_id_idx`;
ALTER TABLE `scanning_station`
  ADD FULLTEXT KEY `scanning_station_name_idx` (`name`),
  DROP KEY `scanning_station_account_id_created_at_id_idx`;
ALTER TABLE `machine`
  ADD FULLTEXT KEY `machine_name_idx` (`name`),
  DROP KEY `machine_account_id_created_at_id_idx`;
ALTER TABLE `department`
  ADD FULLTEXT KEY `department_name_idx` (`name`),
  DROP KEY `department_account_id_created_at_id_idx`;
ALTER TABLE `machine` DROP COLUMN `is_active`;
