-- +goose NO TRANSACTION
-- +goose Up
-- The change log's list keys were descending. InnoDB scans an ascending key in both directions and a
-- descending one only forward, so a previous page (read oldest first) sorted the filter's whole range.
-- Rebuilt ascending under the same names, which the list's FORCE INDEX names.
ALTER TABLE `inventory_change_log`
  DROP KEY `inventory_change_log_account_created_idx`,
  ADD KEY `inventory_change_log_account_created_idx` (`account_id`, `created_at`, `id`),
  DROP KEY `inventory_change_log_account_id_item_id_created_at_id_idx`,
  ADD KEY `inventory_change_log_account_id_item_id_created_at_id_idx` (`account_id`, `item_id`, `created_at`, `id`),
  DROP KEY `inventory_change_log_acct_action_type_code_created_idx`,
  ADD KEY `inventory_change_log_acct_action_type_code_created_idx` (`account_id`, `action_type_code`, `created_at`, `id`),
  DROP KEY `inventory_change_log_acct_resp_user_created_idx`,
  ADD KEY `inventory_change_log_acct_resp_user_created_idx` (`account_id`, `responsible_user_id`, `created_at`, `id`);

-- +goose Down
ALTER TABLE `inventory_change_log`
  DROP KEY `inventory_change_log_account_created_idx`,
  ADD KEY `inventory_change_log_account_created_idx` (`account_id`, `created_at` DESC, `id` DESC),
  DROP KEY `inventory_change_log_account_id_item_id_created_at_id_idx`,
  ADD KEY `inventory_change_log_account_id_item_id_created_at_id_idx` (`account_id`, `item_id`, `created_at` DESC, `id` DESC),
  DROP KEY `inventory_change_log_acct_action_type_code_created_idx`,
  ADD KEY `inventory_change_log_acct_action_type_code_created_idx` (`account_id`, `action_type_code`, `created_at` DESC, `id` DESC),
  DROP KEY `inventory_change_log_acct_resp_user_created_idx`,
  ADD KEY `inventory_change_log_acct_resp_user_created_idx` (`account_id`, `responsible_user_id`, `created_at` DESC, `id` DESC);
