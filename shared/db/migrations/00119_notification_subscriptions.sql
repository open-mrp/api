-- +goose NO TRANSACTION
-- +goose Up

-- Who is notified about a record. One row per person and record (record IDs are unique across types); an
-- unsubscribed row keeps automatic subscription from adding the person back.
CREATE TABLE `subscription` (
  `id` varchar(191) NOT NULL,
  `account_id` varchar(191) NOT NULL,
  `account_user_id` varchar(191) NOT NULL,
  `record_type` varchar(64) NOT NULL,
  `record_id` varchar(191) NOT NULL,
  `reason` varchar(20) NOT NULL,
  `status` varchar(20) NOT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `subscription_user_record_key` (`account_user_id`, `record_id`),
  KEY `subscription_record_status_idx` (`account_id`, `record_type`, `record_id`, `status`),
  KEY `subscription_user_created_idx` (`account_user_id`, `created_at` DESC, `id` DESC),
  KEY `subscription_user_status_created_idx` (`account_user_id`, `status`, `created_at` DESC, `id` DESC)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Notification settings that aren't per type: a person's auto-subscribe list and pause (account_user_id set), or the
-- account's defaults (account_user_id empty). Per-type personal settings stay in notification_preference.
CREATE TABLE `notification_setting` (
  `id` varchar(191) NOT NULL,
  `account_id` varchar(191) NOT NULL,
  `account_user_id` varchar(191) NOT NULL DEFAULT '',
  `defaults` json DEFAULT NULL,
  `auto_subscribe` json DEFAULT NULL,
  `paused_until` datetime(3) DEFAULT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `notification_setting_scope_key` (`account_id`, `account_user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- A role's notification defaults ({default, types, auto_subscribe}); NULL sets nothing.
ALTER TABLE `role`
  ADD COLUMN `notification_defaults` json DEFAULT NULL;

-- +goose Down

ALTER TABLE `role`
  DROP COLUMN `notification_defaults`;

DROP TABLE IF EXISTS `notification_setting`;

DROP TABLE IF EXISTS `subscription`;
