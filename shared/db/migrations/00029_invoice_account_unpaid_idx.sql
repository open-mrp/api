-- +goose NO TRANSACTION
-- +goose Up
-- The dashboard's unpaid-invoices-for-a-customer list (account_id, is_paid_in_full, sales_order.buyer_account_id,
-- ORDER BY created_at DESC) had no invoice index to drive from, so it walked every sales order the buyer ever had
-- and probed each one's invoice: 4,103 rows each way to return 31 for Carolon's largest customer, 81ms warm and
-- 831ms cold. An account's unpaid invoices are few (680 of Carolon's 125k), so this index starts there instead:
-- the index returns them already sorted, and sales_order_id is in the index so reaching each
-- invoice's order needs no read of the invoice row.
ALTER TABLE `invoice`
  ADD KEY `invoice_account_unpaid_created_idx` (`account_id`, `is_paid_in_full`, `created_at` DESC, `sales_order_id`);

-- +goose Down
ALTER TABLE `invoice`
  DROP KEY `invoice_account_unpaid_created_idx`;
