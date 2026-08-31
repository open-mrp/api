-- +goose NO TRANSACTION
-- +goose Up

-- The pick list's `q` search is a substring match on a pick's number and its order's customer PO number.
-- Substring search cannot use a b-tree or a default-parser FULLTEXT index (which tokenizes on word
-- boundaries), so on a large account it degrades to a full table scan (~4.5s on a 120k-pick account,
-- >10s combined with a product-line filter). An ngram FULLTEXT index tokenizes into overlapping 2-grams,
-- so a boolean-mode phrase search ('"23839"') matches "23839" inside "L-123839" straight from the index.

-- pick_number_idx was a default-parser FULLTEXT index that no query used (search ran through LIKE).
-- Replace it with an ngram index so MATCH(number) can serve substring search.
ALTER TABLE `pick`
  DROP KEY `pick_number_idx`;

ALTER TABLE `pick`
  ADD FULLTEXT KEY `pick_number_ngram_idx` (`number`) WITH PARSER ngram;

-- MATCH(customer_po_number) cannot name which index to use, so with two single-column FULLTEXT indexes
-- on the column MySQL silently picks one — and it picks the pre-existing default-parser index, which
-- tokenizes on word boundaries and so never matches a substring. Drop it so the ngram index is the only
-- candidate. No query MATCHes this column today; the default index is dead weight from the initial schema.
ALTER TABLE `sales_order`
  DROP KEY `sales_order_customer_po_number_idx`;

ALTER TABLE `sales_order`
  ADD FULLTEXT KEY `sales_order_customer_po_number_ngram_idx` (`customer_po_number`) WITH PARSER ngram;

-- +goose Down

ALTER TABLE `sales_order`
  DROP KEY `sales_order_customer_po_number_ngram_idx`;

ALTER TABLE `sales_order`
  ADD FULLTEXT KEY `sales_order_customer_po_number_idx` (`customer_po_number`);

ALTER TABLE `pick`
  DROP KEY `pick_number_ngram_idx`;

ALTER TABLE `pick`
  ADD FULLTEXT KEY `pick_number_idx` (`number`);
