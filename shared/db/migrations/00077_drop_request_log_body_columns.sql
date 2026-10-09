-- +goose NO TRANSACTION
-- +goose Up

-- Request-log bodies live in the payloads bucket under request_log.payload_key. Ship only once the
-- request_log_payloads backfill has completed (backfill_progress.completed_at set): any body still
-- in these columns is lost with them. With the bodies already moved, the rebuild this drop triggers
-- copies only the slim rows.
ALTER TABLE `request_log`
  DROP COLUMN `query_json`,
  DROP COLUMN `request_body_json`,
  DROP COLUMN `response_body_json`,
  DROP COLUMN `stack_trace`;

-- +goose Down

ALTER TABLE `request_log`
  ADD COLUMN `query_json` json DEFAULT NULL,
  ADD COLUMN `request_body_json` json DEFAULT NULL,
  ADD COLUMN `response_body_json` json DEFAULT NULL,
  ADD COLUMN `stack_trace` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
