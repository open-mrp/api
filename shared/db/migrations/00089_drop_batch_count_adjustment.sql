-- +goose NO TRANSACTION
-- +goose Up

-- Splits no longer recount the batch they close, so nothing writes or reads these records.
DROP TABLE `batch_count_adjustment`;

-- +goose Down

-- Restores the shape, not the data.
CREATE TABLE `batch_count_adjustment` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `account_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `batch_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `closing_batch_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `previous_value` decimal(65,30) NOT NULL,
  `counted_value` decimal(65,30) NOT NULL,
  `unit_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  KEY `batch_count_adjustment_account_closing_idx` (`account_id`, `closing_batch_id`),
  KEY `batch_count_adjustment_account_batch_idx` (`account_id`, `batch_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
