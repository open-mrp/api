-- +goose NO TRANSACTION
-- +goose Up

-- The legacy EDI pipeline's storage. OpenMRP no longer exchanges EDI documents itself: an external EDI
-- runner does, keeping its partner line numbers and sent markers in metadata (backfilled in data
-- migration 00014). Nothing reads these since the release that removed the EDI code.
DROP TABLE `edi_transmission`;
DROP TABLE `edi_run`;
DROP TABLE `dc_location`;

ALTER TABLE `sales_order_line`
  DROP COLUMN `edi_line_item_id`;

ALTER TABLE `invoice`
  DROP COLUMN `is_edi_sent`;

ALTER TABLE `account_relation`
  DROP COLUMN `is_edi_enabled`;

-- +goose Down

-- Restores the shapes, not the data.
ALTER TABLE `account_relation`
  ADD COLUMN `is_edi_enabled` tinyint(1) NOT NULL DEFAULT '0';

ALTER TABLE `invoice`
  ADD COLUMN `is_edi_sent` tinyint(1) NOT NULL DEFAULT '0';

ALTER TABLE `sales_order_line`
  ADD COLUMN `edi_line_item_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL;

CREATE TABLE `dc_location` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `location` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `account_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `owner_account_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  PRIMARY KEY (`id`),
  KEY `dc_location_account_id_idx` (`account_id`),
  KEY `dc_location_owner_account_id_idx` (`owner_account_id`),
  FULLTEXT KEY `dc_location_location_idx` (`location`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE `edi_run` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `completed_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `account_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `has_succeeded` tinyint(1) NOT NULL DEFAULT '0',
  `failures` json DEFAULT NULL,
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  KEY `edi_run_account_id_completed_at_id_idx` (`account_id`,`completed_at`,`id`),
  KEY `edi_run_account_id_has_succeeded_completed_at_id_idx` (`account_id`,`has_succeeded`,`completed_at`,`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE `edi_transmission` (
  `id` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL,
  `account_id` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL,
  `direction` varchar(16) COLLATE utf8mb4_unicode_ci NOT NULL,
  `document_type` varchar(8) COLLATE utf8mb4_unicode_ci NOT NULL,
  `subject_type` varchar(32) COLLATE utf8mb4_unicode_ci NOT NULL,
  `subject_id` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL,
  `counterparty_account_id` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL,
  `status` varchar(16) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'pending',
  `attempts` int NOT NULL DEFAULT '0',
  `available_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `claim_owner` varchar(64) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `claim_token` varchar(64) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `claim_expires_at` datetime(3) DEFAULT NULL,
  `last_error` text COLLATE utf8mb4_unicode_ci,
  `transmitted_at` datetime(3) DEFAULT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `edi_transmission_subject_key` (`account_id`,`document_type`,`subject_id`),
  KEY `edi_transmission_claim_idx` (`account_id`,`status`,`available_at`),
  KEY `edi_transmission_counterparty_idx` (`counterparty_account_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
