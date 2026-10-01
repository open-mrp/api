-- +goose NO TRANSACTION
-- +goose Up
-- The settlement list's key was descending. InnoDB scans an ascending key in both directions and a
-- descending one only forward, so the previous page (read oldest first) sorted the account's whole
-- range. Rebuilt ascending under the same name.
ALTER TABLE `settlement`
  DROP KEY `settlement_account_created_idx`,
  ADD KEY `settlement_account_created_idx` (`account_id`, `created_at`, `id`);

-- The allocation entry list scopes by the transaction's account and filters by its type, so no key
-- could lead with either. Both are copied from the transaction, which never changes them.
ALTER TABLE `transaction_allocation`
  ADD COLUMN `account_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL,
  ADD COLUMN `transaction_type_code` varchar(32) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL,
  ADD KEY `transaction_allocation_account_created_idx` (`account_id`, `created_at`, `id`),
  ADD KEY `transaction_allocation_account_type_created_idx` (`account_id`, `transaction_type_code`, `created_at`, `id`);

-- +goose Down
ALTER TABLE `transaction_allocation`
  DROP KEY `transaction_allocation_account_type_created_idx`,
  DROP KEY `transaction_allocation_account_created_idx`,
  DROP COLUMN `transaction_type_code`,
  DROP COLUMN `account_id`;

ALTER TABLE `settlement`
  DROP KEY `settlement_account_created_idx`,
  ADD KEY `settlement_account_created_idx` (`account_id`, `created_at` DESC, `id` DESC);
