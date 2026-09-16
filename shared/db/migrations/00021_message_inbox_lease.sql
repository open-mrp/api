-- +goose Up

-- The inbox recorded only 'received' and 'processed', so 'received' meant three different things at once: in flight, crashed before doing the work, and did the work but died before the marker committed. InboxConsumer resolved that ambiguity by re-invoking the handler, which re-applied any message in the third state.
-- lock_owner / lock_expires_at give 'received' a lease so a live attempt is distinguishable from an abandoned one, matching message_outbox and service_idempotency_key. failed_at makes failure explicit rather than inferred from last_error. 'discarded' is the terminal state for deterministic failures that can never succeed on retry, and 'ignored' for messages that were never this handler's work — terminal in the same way, but not something the failure monitor alerts on.
-- No index on lock_expires_at: nothing scans by it. Claim and the completion writes address a row by
-- primary key, and the failure monitor's scan leads on message_inbox_alert_scan_idx (status, alerted_at)
-- with the lease only a residual filter.
ALTER TABLE `message_inbox`
  ADD COLUMN `failed_at` datetime(3) DEFAULT NULL,
  ADD COLUMN `lock_owner` varchar(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  ADD COLUMN `lock_expires_at` datetime(3) DEFAULT NULL;

-- +goose Down

ALTER TABLE `message_inbox`
  DROP COLUMN `lock_expires_at`,
  DROP COLUMN `lock_owner`,
  DROP COLUMN `failed_at`;
