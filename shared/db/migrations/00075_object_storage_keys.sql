-- +goose NO TRANSACTION
-- +goose Up

-- Object keys for documents kept in the payloads bucket instead of in the row. A NULL key means the
-- document, if any, is still in the row's own JSON columns. Nullable and last, so each ADD COLUMN is
-- an instant metadata change rather than a rebuild of request_log.
ALTER TABLE `request_log`
  ADD COLUMN `payload_key` varchar(255) DEFAULT NULL;

-- Exactly one of job_items / job_items_key is set: a bulk payload over the inline limit is stored
-- in the bucket, so job_items goes NULL for it.
ALTER TABLE `job`
  MODIFY COLUMN `job_items` json DEFAULT NULL,
  ADD COLUMN `job_items_key` varchar(255) DEFAULT NULL;

-- A cached response over the inline limit is moved to the bucket once its transaction commits:
-- one update clears response_body and sets the key.
ALTER TABLE `idempotency_key`
  ADD COLUMN `response_body_key` varchar(255) DEFAULT NULL;

ALTER TABLE `service_idempotency_key`
  ADD COLUMN `response_body_key` varchar(255) DEFAULT NULL;

-- +goose Down

ALTER TABLE `service_idempotency_key`
  DROP COLUMN `response_body_key`;

ALTER TABLE `idempotency_key`
  DROP COLUMN `response_body_key`;

ALTER TABLE `job`
  DROP COLUMN `job_items_key`,
  MODIFY COLUMN `job_items` json NOT NULL;

ALTER TABLE `request_log`
  DROP COLUMN `payload_key`;
