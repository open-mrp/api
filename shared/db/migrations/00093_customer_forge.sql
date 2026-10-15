-- +goose NO TRANSACTION
-- +goose Up

-- Staff instructions for a customer's or supplier's orders replace the free-form note.
ALTER TABLE `account_relation` ADD COLUMN `instructions` text CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL;

-- A customer's default supply (stock | make | buy) replaces its fulfillment policy.
ALTER TABLE `account_relation` ADD COLUMN `supply_code` varchar(32) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL;

-- Customers can be archived: they stay on their records but take no new orders.
ALTER TABLE `account_relation` ADD COLUMN `is_active` tinyint(1) NOT NULL DEFAULT '1';

-- +goose Down

ALTER TABLE `account_relation` DROP COLUMN `is_active`;
ALTER TABLE `account_relation` DROP COLUMN `supply_code`;
ALTER TABLE `account_relation` DROP COLUMN `instructions`;
