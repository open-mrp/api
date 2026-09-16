-- +goose NO TRANSACTION
-- +goose Up
-- The pick list's default view is `status=open` sorted by ship-by date. pick_account_ship_by_idx
-- orders but cannot pin that filter, so MySQL walked the account's picks from the earliest ship-by
-- date — the oldest, long-finished ones — probing finished_at per row and discarding almost all of
-- them before the page filled (10s+ on the largest account, past the RPC deadline). Open picks are a
-- tiny slice of a mostly-closed table, so the filter has to lead. `finished_at IS NULL` is an index
-- equality, which leaves ship_by_sort_date and id serving the ORDER BY in order.
-- `status=closed` keeps using pick_account_ship_by_idx: there the residual filter matches nearly
-- every row, so an in-order scan fills a page immediately.
ALTER TABLE `pick`
  ADD KEY `pick_account_finished_ship_by_idx` (`account_id`, `finished_at`, `ship_by_sort_date`, `id`);

-- +goose Down
ALTER TABLE `pick`
  DROP KEY `pick_account_finished_ship_by_idx`;
