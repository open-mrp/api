-- +goose NO TRANSACTION
-- +goose Up

-- A log's query parameters are part of the log resource, so lists read them from the row rather than from the
-- payload object. Nullable and last, so the ADD COLUMN is an instant metadata change. Older logs keep theirs only
-- in the payload object.
ALTER TABLE `request_log`
  ADD COLUMN `query_params` json DEFAULT NULL;

-- +goose Down

ALTER TABLE `request_log`
  DROP COLUMN `query_params`;
