-- +goose NO TRANSACTION
-- +goose Up
-- The email log list reads newest first within an account, but its only account index is
-- (account_id), so every page filesorts the account's whole history. This gives the list its order.
ALTER TABLE `email_log`
  ADD KEY `email_log_account_created_idx` (`account_id`, `created_at` DESC, `id` DESC);

-- +goose Down
ALTER TABLE `email_log`
  DROP KEY `email_log_account_created_idx`;
