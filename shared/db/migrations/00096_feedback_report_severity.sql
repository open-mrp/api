-- +goose NO TRANSACTION
-- +goose Up

ALTER TABLE `feedback_report`
  ADD COLUMN `severity` varchar(16) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'medium';

-- +goose Down

ALTER TABLE `feedback_report`
  DROP COLUMN `severity`;
