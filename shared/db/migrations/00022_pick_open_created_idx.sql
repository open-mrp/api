-- +goose NO TRANSACTION
-- +goose Up
-- The created-at pick list with `status=open` needs the same shape 00020 gave the ship-by list:
-- pick_account_created_idx orders but leaves the filter residual, so the scan reads the account's
-- finished history before a page of open picks fills. `finished_at IS NULL` as an index equality
-- leaves created_at and id serving the ORDER BY in order. DESC matches pick_account_created_idx,
-- since the list's first page reads newest first.
ALTER TABLE `pick`
  ADD KEY `pick_account_finished_created_idx` (`account_id`, `finished_at`, `created_at` DESC, `id` DESC);

-- +goose Down
ALTER TABLE `pick`
  DROP KEY `pick_account_finished_created_idx`;
