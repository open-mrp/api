-- +goose NO TRANSACTION
-- +goose Up
-- The settlement that recorded a transaction inline (settlement create's new_transactions), so
-- deleting that settlement removes the transactions whose amounts only it applied.
ALTER TABLE `transaction`
  ADD COLUMN `created_by_settlement_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  ADD KEY `transaction_created_by_settlement_id_idx` (`created_by_settlement_id`);

-- +goose Down
ALTER TABLE `transaction`
  DROP KEY `transaction_created_by_settlement_id_idx`,
  DROP COLUMN `created_by_settlement_id`;
