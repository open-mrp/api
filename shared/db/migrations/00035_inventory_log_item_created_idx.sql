-- +goose NO TRANSACTION
-- +goose Up
-- The inventory snapshot as of a date reads each item's latest log at or before it. With only
-- single-column keys that meant reading every log row for the items and grouping them; this key
-- answers MAX(created_at) per item with one index dive each, and then finds that row by the same key.
-- It leads with account_id, so it replaces the account_id key.
ALTER TABLE `inventory_log`
  ADD KEY `inventory_log_account_id_item_id_created_at_idx` (`account_id`, `item_id`, `created_at`),
  DROP KEY `inventory_log_account_id_idx`;

-- +goose Down
ALTER TABLE `inventory_log`
  ADD KEY `inventory_log_account_id_idx` (`account_id`),
  DROP KEY `inventory_log_account_id_item_id_created_at_idx`;
