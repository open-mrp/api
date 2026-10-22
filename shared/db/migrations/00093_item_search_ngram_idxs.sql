-- +goose NO TRANSACTION
-- +goose Up

-- The item and product lists' `q` search matches a term anywhere in the item's SKU or description.
-- LIKE '%q%' cannot use a b-tree, so the search tested every item of the account and sorted the
-- matches by relevance: every search read the whole catalog (p50 59ms, about 8.5k rows). An ngram
-- FULLTEXT index finds the rows holding every 2-gram of the term, so each column is read for its few
-- matches first, as the shipment search reads its own.
--
-- MATCH(col) cannot name its index, and with a default-parser FULLTEXT index on the same column MySQL
-- may pick that one, which never matches a substring. No query MATCHes item, so the default ones are
-- dropped first.
ALTER TABLE `item`
  DROP KEY `item_description_idx`,
  DROP KEY `item_sku_description_idx`;

-- InnoDB builds one FULLTEXT index per statement.
ALTER TABLE `item`
  ADD FULLTEXT KEY `item_sku_ngram_idx` (`sku`) WITH PARSER ngram;

ALTER TABLE `item`
  ADD FULLTEXT KEY `item_description_ngram_idx` (`description`) WITH PARSER ngram;

-- +goose Down

ALTER TABLE `item`
  DROP KEY `item_description_ngram_idx`,
  DROP KEY `item_sku_ngram_idx`;

ALTER TABLE `item`
  ADD FULLTEXT KEY `item_description_idx` (`description`);

ALTER TABLE `item`
  ADD FULLTEXT KEY `item_sku_description_idx` (`sku`, `description`);
