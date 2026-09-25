-- +goose NO TRANSACTION
-- +goose Up
-- A customer's open picks are a few of its history: the largest customer has 4,102 picks and 3 open.
-- On pick_account_buyer_{ship_by,created}_idx `status=open` is a residual filter that reads the
-- whole history (41ms for the two largest customers), so, as for the unfiltered list, finished_at
-- leads the sort columns and the open filter becomes an index equality.
ALTER TABLE `pick`
  ADD KEY `pick_account_buyer_finished_ship_by_idx` (`account_id`, `buyer_account_id`, `finished_at`, `ship_by_sort_date`, `id`),
  ADD KEY `pick_account_buyer_finished_created_idx` (`account_id`, `buyer_account_id`, `finished_at`, `created_at` DESC, `id` DESC);

-- +goose Down
ALTER TABLE `pick`
  DROP KEY `pick_account_buyer_finished_created_idx`,
  DROP KEY `pick_account_buyer_finished_ship_by_idx`;
