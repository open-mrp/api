-- +goose NO TRANSACTION
-- +goose Up
-- The purchase order list with no status filter pages by (created_at, id) within one order type.
-- Without this key it reads every purchase order of the account and sorts them; purchase orders are a
-- small share of an account's orders, so the (owner, created_at) key would walk the sales orders too.
ALTER TABLE `sales_order`
  ADD KEY `sales_order_owner_type_created_idx` (`owner_account_id`, `sales_order_type_code`, `created_at` DESC, `id` DESC);

-- +goose Down
ALTER TABLE `sales_order`
  DROP KEY `sales_order_owner_type_created_idx`;
