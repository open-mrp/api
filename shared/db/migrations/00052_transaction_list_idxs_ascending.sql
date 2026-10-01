-- +goose NO TRANSACTION
-- +goose Up
-- The transaction list's per-filter keys were descending. InnoDB scans an ascending key in both
-- directions and a descending one only forward, so the previous page (read oldest first) sorted the
-- filter's whole range. Rebuilt ascending under the same names, which the list's FORCE INDEX names.
ALTER TABLE `transaction`
  DROP KEY `transaction_account_id_transaction_type_code_created_at_id_idx`,
  ADD KEY `transaction_account_id_transaction_type_code_created_at_id_idx` (`account_id`, `transaction_type_code`, `created_at`, `id`),
  DROP KEY `transaction_account_id_customer_account_id_created_at_id_idx`,
  ADD KEY `transaction_account_id_customer_account_id_created_at_id_idx` (`account_id`, `customer_account_id`, `created_at`, `id`),
  DROP KEY `transaction_account_id_transaction_method_code_created_at_id_idx`,
  ADD KEY `transaction_account_id_transaction_method_code_created_at_id_idx` (`account_id`, `transaction_method_code`, `created_at`, `id`),
  DROP KEY `transaction_account_id_adjustment_type_code_created_at_id_idx`,
  ADD KEY `transaction_account_id_adjustment_type_code_created_at_id_idx` (`account_id`, `adjustment_type_code`, `created_at`, `id`);

-- +goose Down
ALTER TABLE `transaction`
  DROP KEY `transaction_account_id_transaction_type_code_created_at_id_idx`,
  ADD KEY `transaction_account_id_transaction_type_code_created_at_id_idx` (`account_id`, `transaction_type_code`, `created_at` DESC, `id` DESC),
  DROP KEY `transaction_account_id_customer_account_id_created_at_id_idx`,
  ADD KEY `transaction_account_id_customer_account_id_created_at_id_idx` (`account_id`, `customer_account_id`, `created_at` DESC, `id` DESC),
  DROP KEY `transaction_account_id_transaction_method_code_created_at_id_idx`,
  ADD KEY `transaction_account_id_transaction_method_code_created_at_id_idx` (`account_id`, `transaction_method_code`, `created_at` DESC, `id` DESC),
  DROP KEY `transaction_account_id_adjustment_type_code_created_at_id_idx`,
  ADD KEY `transaction_account_id_adjustment_type_code_created_at_id_idx` (`account_id`, `adjustment_type_code`, `created_at` DESC, `id` DESC);
