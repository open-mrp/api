-- +goose NO TRANSACTION
-- +goose Up

-- A machine's items in list order, so a page of them walks the key and stops at its limit.
ALTER TABLE `machine_item`
  ADD KEY `machine_item_account_machine_created_idx` (`account_id`, `machine_id`, `created_at`, `item_id`);

-- +goose Down

ALTER TABLE `machine_item`
  DROP KEY `machine_item_account_machine_created_idx`;
