-- +goose NO TRANSACTION
-- +goose Up

-- Pick search matches the start of a pick number, a range on the number key, and of a PO number, whose
-- orders' picks are each looked up by order. Both keys carry every column a search's matches are paged
-- on, so neither visits the pick's row. The number key supersedes pick_account_number_idx, which
-- deployed pods still force, so that one is dropped in a later release.
ALTER TABLE `pick`
  ADD KEY `pick_account_number_page_idx` (`account_id`, `number`, `created_at`, `ship_by_sort_date`, `finished_at`, `buyer_account_id`),
  ADD KEY `pick_sales_order_page_idx` (`sales_order_id`, `account_id`, `created_at`, `ship_by_sort_date`, `finished_at`, `buyer_account_id`);

-- +goose Down

ALTER TABLE `pick`
  DROP KEY `pick_sales_order_page_idx`,
  DROP KEY `pick_account_number_page_idx`;
