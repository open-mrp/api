-- +goose NO TRANSACTION
-- +goose Up

-- Who must sign a batch off before it leaves this department, whether an unsigned batch is stopped or
-- only flagged, the risk assessment behind that choice, and the yes/no questions the signer answers. A
-- batch is in the department of the scanning station it was last scanned at.
ALTER TABLE `department`
  ADD COLUMN `release_requirement_code` varchar(32) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'none',
  ADD COLUMN `release_enforcement_code` varchar(32) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'block',
  ADD COLUMN `release_rationale` text CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci,
  ADD COLUMN `release_checklist` json DEFAULT NULL;

-- Whether the station scans only while an operator is checked in.
ALTER TABLE `scanning_station`
  ADD COLUMN `operator_check_in_code` varchar(32) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'optional';

-- The operator checked in at the station when the batch was scanned.
ALTER TABLE `batch`
  ADD COLUMN `operator_account_user_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL;

-- An electronic signature on a batch at a production step. Rows are never updated or deleted, and
-- outlive the batch when an undo removes it; the signer's name and role are copied so the record reads
-- as it did when it was signed.
CREATE TABLE `batch_signature` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `account_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `batch_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `production_step_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `production_run_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `type_code` varchar(32) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `decision_code` varchar(32) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `capacity_code` varchar(32) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `signer_account_user_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `signer_name` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `signer_role_name` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `checklist` json DEFAULT NULL,
  `note` text CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci,
  `actor_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `actor_type` varchar(32) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `signed_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  KEY `batch_signature_account_batch_signed_idx` (`account_id`, `batch_id`, `signed_at` DESC, `id` DESC),
  KEY `batch_signature_account_run_step_idx` (`account_id`, `production_run_id`, `production_step_id`, `signed_at` DESC)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- An operator checked in at a scanning station, identified by their signing PIN. Active until checked
-- out, replaced by the next operator, or past expires_at.
CREATE TABLE `operator_check_in` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `account_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `scanning_station_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `account_user_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `operator_name` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `checked_in_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `expires_at` datetime(3) NOT NULL,
  `checked_out_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `operator_check_in_account_station_checked_in_idx` (`account_id`, `scanning_station_id`, `checked_in_at` DESC, `id` DESC)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- +goose Down

DROP TABLE `operator_check_in`;
DROP TABLE `batch_signature`;
ALTER TABLE `batch` DROP COLUMN `operator_account_user_id`;
ALTER TABLE `scanning_station` DROP COLUMN `operator_check_in_code`;
ALTER TABLE `department`
  DROP COLUMN `release_requirement_code`,
  DROP COLUMN `release_enforcement_code`,
  DROP COLUMN `release_rationale`,
  DROP COLUMN `release_checklist`;
