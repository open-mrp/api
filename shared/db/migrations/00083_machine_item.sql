-- +goose NO TRANSACTION
-- +goose Up

-- Items a machine is configured to run. Production schedule discovery and eligibility
-- read this set instead of inferring affinity from scan history.
CREATE TABLE `machine_item` (
  `account_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `machine_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `item_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`machine_id`, `item_id`),
  KEY `machine_item_account_machine_item_idx` (`account_id`, `machine_id`, `item_id`),
  KEY `machine_item_account_item_machine_idx` (`account_id`, `item_id`, `machine_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- +goose Down

DROP TABLE `machine_item`;
