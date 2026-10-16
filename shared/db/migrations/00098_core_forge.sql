-- +goose NO TRANSACTION
-- +goose Up

-- The record an email carried (an invoice, an order acknowledgment's order, a statement's customer), shown in the
-- email log's `related`. Emails logged before this have none. The key serves the `related_ids[]` filter in list order.
ALTER TABLE `email_log`
  ADD COLUMN `related_type` varchar(32) DEFAULT NULL,
  ADD COLUMN `related_id` varchar(191) DEFAULT NULL,
  ADD KEY `email_log_account_related_created_idx` (`account_id`, `related_id`, `created_at` DESC, `id` DESC);

-- The async job list reads an account's jobs newest first; `job` has only (account_id, created_by), so every page
-- would sort the account's jobs.
ALTER TABLE `job`
  ADD KEY `job_account_created_idx` (`account_id`, `created_at` DESC, `job_id` DESC);

-- +goose Down

ALTER TABLE `job`
  DROP KEY `job_account_created_idx`;

ALTER TABLE `email_log`
  DROP KEY `email_log_account_related_created_idx`,
  DROP COLUMN `related_id`,
  DROP COLUMN `related_type`;
