-- +goose NO TRANSACTION
-- +goose Up

-- invoice had no b-tree index on number at all, only the two FULLTEXT ones, so InvoiceRepo.isDuplicate's (account_id, number) equality lookup fell back to invoice_account_created_idx and filtered the account's entire range: 124,953 rows read to return one, 546ms median and 1.3s at p99. account_id alone is not selective and will not become so, so the index leads with it and gets its selectivity from number.
ALTER TABLE `invoice`
  ADD KEY `invoice_account_number_idx` (`account_id`, `number`);

-- InvoiceRepo.resolveSearchIDs selects only id under (account_id, sales_order_id IN (...)), which invoice_sales_order_id_idx serves with one clustered read per candidate just to check account_id. InnoDB appends the primary key, so this index covers the whole branch and the reads go away. invoice_sales_order_id_idx is kept: it is not a left prefix of this index and still serves lookups that have no account_id.
ALTER TABLE `invoice`
  ADD KEY `invoice_account_sales_order_idx` (`account_id`, `sales_order_id`);

-- +goose Down

ALTER TABLE `invoice`
  DROP KEY `invoice_account_number_idx`;

ALTER TABLE `invoice`
  DROP KEY `invoice_account_sales_order_idx`;
