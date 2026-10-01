-- +goose NO TRANSACTION
-- +goose Up
-- These list keys were descending. InnoDB scans an ascending key in both directions and a descending
-- one only forward, so a previous page (read oldest first) sorted the filter's whole range. Rebuilt
-- ascending under the same names, which the lists' FORCE INDEX names.
ALTER TABLE `inventory_change_log`
  DROP KEY `inventory_change_log_account_created_idx`,
  ADD KEY `inventory_change_log_account_created_idx` (`account_id`, `created_at`, `id`),
  DROP KEY `inventory_change_log_account_id_item_id_created_at_id_idx`,
  ADD KEY `inventory_change_log_account_id_item_id_created_at_id_idx` (`account_id`, `item_id`, `created_at`, `id`),
  DROP KEY `inventory_change_log_acct_action_type_code_created_idx`,
  ADD KEY `inventory_change_log_acct_action_type_code_created_idx` (`account_id`, `action_type_code`, `created_at`, `id`),
  DROP KEY `inventory_change_log_acct_resp_user_created_idx`,
  ADD KEY `inventory_change_log_acct_resp_user_created_idx` (`account_id`, `responsible_user_id`, `created_at`, `id`);

ALTER TABLE `batch`
  DROP KEY `batch_account_id_scanning_station_id_scanned_at_id_idx`,
  ADD KEY `batch_account_id_scanning_station_id_scanned_at_id_idx` (`account_id`, `scanning_station_id`, `scanned_at`, `id`);

ALTER TABLE `machine_downtime_event`
  DROP KEY `machine_downtime_account_started_idx`,
  ADD KEY `machine_downtime_account_started_idx` (`account_id`, `started_at`, `id`),
  DROP KEY `machine_downtime_account_machine_started_idx`,
  ADD KEY `machine_downtime_account_machine_started_idx` (`account_id`, `machine_id`, `started_at`, `id`),
  DROP KEY `machine_downtime_account_dept_started_idx`,
  ADD KEY `machine_downtime_account_dept_started_idx` (`account_id`, `department_id`, `started_at`, `id`),
  DROP KEY `machine_downtime_account_reason_started_idx`,
  ADD KEY `machine_downtime_account_reason_started_idx` (`account_id`, `reason_code`, `started_at`, `id`),
  DROP KEY `machine_downtime_account_ended_started_idx`,
  ADD KEY `machine_downtime_account_ended_started_idx` (`account_id`, `ended_at`, `started_at`, `id`);

-- The run list's order: the account's runs newest first, read a page at a time instead of all sorted.
ALTER TABLE `production_run`
  ADD KEY `production_run_account_created_idx` (`account_id`, `created_at`, `id`);

-- +goose Down
ALTER TABLE `production_run`
  DROP KEY `production_run_account_created_idx`;

ALTER TABLE `machine_downtime_event`
  DROP KEY `machine_downtime_account_started_idx`,
  ADD KEY `machine_downtime_account_started_idx` (`account_id`, `started_at` DESC, `id` DESC),
  DROP KEY `machine_downtime_account_machine_started_idx`,
  ADD KEY `machine_downtime_account_machine_started_idx` (`account_id`, `machine_id`, `started_at` DESC, `id` DESC),
  DROP KEY `machine_downtime_account_dept_started_idx`,
  ADD KEY `machine_downtime_account_dept_started_idx` (`account_id`, `department_id`, `started_at` DESC, `id` DESC),
  DROP KEY `machine_downtime_account_reason_started_idx`,
  ADD KEY `machine_downtime_account_reason_started_idx` (`account_id`, `reason_code`, `started_at` DESC, `id` DESC),
  DROP KEY `machine_downtime_account_ended_started_idx`,
  ADD KEY `machine_downtime_account_ended_started_idx` (`account_id`, `ended_at`, `started_at` DESC, `id` DESC);

ALTER TABLE `batch`
  DROP KEY `batch_account_id_scanning_station_id_scanned_at_id_idx`,
  ADD KEY `batch_account_id_scanning_station_id_scanned_at_id_idx` (`account_id`, `scanning_station_id`, `scanned_at` DESC, `id` DESC);

ALTER TABLE `inventory_change_log`
  DROP KEY `inventory_change_log_account_created_idx`,
  ADD KEY `inventory_change_log_account_created_idx` (`account_id`, `created_at` DESC, `id` DESC),
  DROP KEY `inventory_change_log_account_id_item_id_created_at_id_idx`,
  ADD KEY `inventory_change_log_account_id_item_id_created_at_id_idx` (`account_id`, `item_id`, `created_at` DESC, `id` DESC),
  DROP KEY `inventory_change_log_acct_action_type_code_created_idx`,
  ADD KEY `inventory_change_log_acct_action_type_code_created_idx` (`account_id`, `action_type_code`, `created_at` DESC, `id` DESC),
  DROP KEY `inventory_change_log_acct_resp_user_created_idx`,
  ADD KEY `inventory_change_log_acct_resp_user_created_idx` (`account_id`, `responsible_user_id`, `created_at` DESC, `id` DESC);
