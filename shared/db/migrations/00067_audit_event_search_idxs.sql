-- +goose NO TRANSACTION
-- +goose Up

-- The audit-event list reads an account's events twice, once as the acting account and once as the
-- account acted upon, each through a key that yields a page in list order (newest first). Search and the
-- resource and root filters pin a column within that scope; without a key leading with the scope and that
-- column, a search that matches little walks every event the account has.
--
-- The acting account already has keys for resource id, resource type and action; it gains request id.
-- The account acted upon gains all four, and both scopes gain an ordered root key. The root key keeps its
-- name, which the list query forces, and gains the order and tiebreak the page is read in.
ALTER TABLE `audit_event`
  ADD KEY `audit_event_account_id_request_id_occurred_at_type_id_idx` (`account_id`, `request_id`, `occurred_at` DESC, `type_id` DESC),
  ADD KEY `audit_event_target_resource_id_idx` (`target_account_id`, `resource_id`, `occurred_at` DESC, `type_id` DESC),
  ADD KEY `audit_event_target_request_id_idx` (`target_account_id`, `request_id`, `occurred_at` DESC, `type_id` DESC),
  ADD KEY `audit_event_target_resource_type_idx` (`target_account_id`, `resource_type`, `occurred_at` DESC, `type_id` DESC),
  ADD KEY `audit_event_target_action_idx` (`target_account_id`, `action`, `occurred_at` DESC, `type_id` DESC),
  ADD KEY `audit_event_target_root_idx` (`target_account_id`, `root_resource_type`, `root_resource_id`, `occurred_at` DESC, `type_id` DESC),
  DROP KEY `audit_event_root_idx`,
  ADD KEY `audit_event_root_idx` (`account_id`, `root_resource_type`, `root_resource_id`, `occurred_at` DESC, `type_id` DESC);

-- +goose Down

ALTER TABLE `audit_event`
  DROP KEY `audit_event_root_idx`,
  ADD KEY `audit_event_root_idx` (`account_id`, `root_resource_type`, `root_resource_id`, `occurred_at`),
  DROP KEY `audit_event_target_root_idx`,
  DROP KEY `audit_event_target_action_idx`,
  DROP KEY `audit_event_target_resource_type_idx`,
  DROP KEY `audit_event_target_request_id_idx`,
  DROP KEY `audit_event_target_resource_id_idx`,
  DROP KEY `audit_event_account_id_request_id_occurred_at_type_id_idx`;
