-- +goose NO TRANSACTION
-- +goose Up
-- The pick list filters by customer, which lives on sales_order. Filtering through the join left the
-- (account_id, <sort>) indexes walking every pick in the account to find a customer's few (1.3-2.2s
-- for a 20-pick customer on the largest account). buyer_account_id is denormalized from the order so
-- a customer filter pins an index that still serves each sort's ORDER BY.
-- pick_account_number_idx serves short (prefix) pick-number searches, which the ngram index cannot.
ALTER TABLE `pick`
  ADD COLUMN `buyer_account_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL,
  ADD KEY `pick_account_buyer_created_idx` (`account_id`, `buyer_account_id`, `created_at` DESC, `id` DESC),
  ADD KEY `pick_account_buyer_ship_by_idx` (`account_id`, `buyer_account_id`, `ship_by_sort_date`, `id`),
  ADD KEY `pick_account_number_idx` (`account_id`, `number`);

-- +goose Down
ALTER TABLE `pick`
  DROP KEY `pick_account_number_idx`,
  DROP KEY `pick_account_buyer_ship_by_idx`,
  DROP KEY `pick_account_buyer_created_idx`,
  DROP COLUMN `buyer_account_id`;
