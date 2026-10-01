-- +goose NO TRANSACTION
-- +goose Up
-- The receiving order list pages by (created_at, id) within an account. With only the account key it
-- reads every receiving order of the account and sorts them before it can return the first page.
-- Ascending, because InnoDB scans an ascending key both ways: newest-first pages read it in reverse,
-- and previous-page requests (created_at ASC) read it forward, with no sort either way.
ALTER TABLE `receiving_order`
  ADD KEY `receiving_order_account_created_idx` (`account_id`, `created_at`, `id`);

-- +goose Down
ALTER TABLE `receiving_order`
  DROP KEY `receiving_order_account_created_idx`;
