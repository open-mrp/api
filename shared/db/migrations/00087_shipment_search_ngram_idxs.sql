-- +goose NO TRANSACTION
-- +goose Up

-- The shipment list's `q` search matches a term anywhere in the shipment's number, note, bill of lading
-- and master tracking number, and in its order's number. LIKE '%q%' cannot use a b-tree, so the search
-- tested every shipment of the account in list order: a rare term read the whole tenant (p50 4.7s, about
-- 520k rows). An ngram FULLTEXT index finds the rows holding every 2-gram of the term, so each column is
-- read for its few matches first, as the pick search reads its own.
--
-- MATCH(col) cannot name its index, and with a default-parser FULLTEXT index on the same columns MySQL
-- may pick that one, which never matches a substring. No query MATCHes these columns, so the default
-- ones (and the unused multi-column ones) are dropped first.
ALTER TABLE `shipment`
  DROP KEY `shipment_number_idx`,
  DROP KEY `shipment_note_idx`,
  DROP KEY `shipment_bill_of_lading_idx`,
  DROP KEY `shipment_master_tracking_number_idx`,
  DROP KEY `shipment_number_note_idx`,
  DROP KEY `shipment_number_bill_of_lading_idx`,
  DROP KEY `shipment_number_master_tracking_number_idx`;

-- InnoDB builds one FULLTEXT index per statement.
ALTER TABLE `shipment`
  ADD FULLTEXT KEY `shipment_number_ngram_idx` (`number`) WITH PARSER ngram;

ALTER TABLE `shipment`
  ADD FULLTEXT KEY `shipment_note_ngram_idx` (`note`) WITH PARSER ngram;

ALTER TABLE `shipment`
  ADD FULLTEXT KEY `shipment_bill_of_lading_ngram_idx` (`bill_of_lading`) WITH PARSER ngram;

ALTER TABLE `shipment`
  ADD FULLTEXT KEY `shipment_master_tracking_number_ngram_idx` (`master_tracking_number`) WITH PARSER ngram;

ALTER TABLE `sales_order`
  DROP KEY `sales_order_number_idx`;

ALTER TABLE `sales_order`
  ADD FULLTEXT KEY `sales_order_number_ngram_idx` (`number`) WITH PARSER ngram;

-- +goose Down

ALTER TABLE `sales_order`
  DROP KEY `sales_order_number_ngram_idx`;

ALTER TABLE `sales_order`
  ADD FULLTEXT KEY `sales_order_number_idx` (`number`);

ALTER TABLE `shipment`
  DROP KEY `shipment_master_tracking_number_ngram_idx`,
  DROP KEY `shipment_bill_of_lading_ngram_idx`,
  DROP KEY `shipment_note_ngram_idx`,
  DROP KEY `shipment_number_ngram_idx`;

ALTER TABLE `shipment`
  ADD FULLTEXT KEY `shipment_number_idx` (`number`);

ALTER TABLE `shipment`
  ADD FULLTEXT KEY `shipment_note_idx` (`note`);

ALTER TABLE `shipment`
  ADD FULLTEXT KEY `shipment_bill_of_lading_idx` (`bill_of_lading`);

ALTER TABLE `shipment`
  ADD FULLTEXT KEY `shipment_master_tracking_number_idx` (`master_tracking_number`);

ALTER TABLE `shipment`
  ADD FULLTEXT KEY `shipment_number_note_idx` (`number`, `note`);

ALTER TABLE `shipment`
  ADD FULLTEXT KEY `shipment_number_bill_of_lading_idx` (`number`, `bill_of_lading`);

ALTER TABLE `shipment`
  ADD FULLTEXT KEY `shipment_number_master_tracking_number_idx` (`number`, `master_tracking_number`);
