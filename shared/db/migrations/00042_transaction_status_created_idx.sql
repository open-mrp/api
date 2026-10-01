-- +goose NO TRANSACTION
-- +goose Up
-- The transaction list filtered by status reads in created order. Without a key in that order, a status
-- matching most of an account (allocated) joined and sorted the whole account to return one page.
-- Ascending, because InnoDB scans an ascending key in both directions and a descending one only
-- forward: on a descending key, the previous page (read oldest first) sorts the whole range.
ALTER TABLE `transaction`
  ADD KEY `transaction_account_id_is_fully_allocated_created_at_id_idx` (`account_id`, `is_fully_allocated`, `created_at`, `id`);

-- +goose Down
ALTER TABLE `transaction`
  DROP KEY `transaction_account_id_is_fully_allocated_created_at_id_idx`;
