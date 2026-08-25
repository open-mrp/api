-- +goose Up

-- alerted_at is the dedup marker for the message failure monitor (see shared/messaging/failure_monitor.go).
-- It is stamped once a failed/stuck row has been included in an alert email so later scans skip it.
-- The message_inbox index keeps the every-few-minutes scan a cheap (status, alerted_at) range instead
-- of a full table scan; message_outbox already has a (status, next_run_at) index that covers the
-- status = 'failed' lookup, so it only needs the column.
ALTER TABLE `message_inbox`
  ADD COLUMN `alerted_at` datetime(3) DEFAULT NULL,
  ADD KEY `message_inbox_alert_scan_idx` (`status`, `alerted_at`);

ALTER TABLE `message_outbox`
  ADD COLUMN `alerted_at` datetime(3) DEFAULT NULL;

-- +goose Down

ALTER TABLE `message_inbox`
  DROP KEY `message_inbox_alert_scan_idx`,
  DROP COLUMN `alerted_at`;

ALTER TABLE `message_outbox`
  DROP COLUMN `alerted_at`;
