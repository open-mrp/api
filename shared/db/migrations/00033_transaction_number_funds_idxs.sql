-- +goose NO TRANSACTION
-- +goose Up
-- A transaction number is unique within its account; nothing enforced it, and the number check
-- before every create could not use an index, so it filtered the account's transactions row by row.
-- The unique key serves that lookup and stops a race from recording two payments under one number.
-- No account holds duplicate numbers today.
--
-- Open credits are an account's transactions not yet fully allocated, newest funds first; this key
-- yields them in that order without reading the account's settled history.
ALTER TABLE `transaction`
  ADD UNIQUE KEY `transaction_account_id_number_key` (`account_id`, `number`),
  ADD KEY `transaction_open_credits_idx` (`account_id`, `is_fully_allocated`, `funds_received_at`, `id`);

-- +goose Down
ALTER TABLE `transaction`
  DROP KEY `transaction_open_credits_idx`,
  DROP KEY `transaction_account_id_number_key`;
