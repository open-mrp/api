-- +goose NO TRANSACTION
-- +goose Up

-- One row per backfill run by shared/db/backfill: where it stopped, so a restart or redeploy resumes
-- there, and whether it has finished, so a finished one never runs again.
CREATE TABLE `backfill_progress` (
  `name` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `cursor_value` varchar(512) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `rows_done` bigint NOT NULL DEFAULT 0,
  `completed_at` datetime(3) DEFAULT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- +goose Down

DROP TABLE `backfill_progress`;
