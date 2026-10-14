-- +goose NO TRANSACTION
-- +goose Up

-- Whether the operator who scanned a batch may sign it off in this department.
ALTER TABLE `department`
  ADD COLUMN `release_operator_signing_code` varchar(32) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'allowed';

-- The finished-goods lot a batch was given when quality released it.
ALTER TABLE `batch`
  ADD COLUMN `release_lot_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL;

-- The lot a final release assigned, kept on the signature that assigned it with its number as it read.
ALTER TABLE `batch_signature`
  ADD COLUMN `lot_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  ADD COLUMN `lot_number` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL;

-- +goose Down

ALTER TABLE `batch_signature` DROP COLUMN `lot_id`, DROP COLUMN `lot_number`;
ALTER TABLE `batch` DROP COLUMN `release_lot_id`;
ALTER TABLE `department` DROP COLUMN `release_operator_signing_code`;
