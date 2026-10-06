-- +goose NO TRANSACTION
-- +goose Up
-- SKU search matches item.sku with LIKE on the (account_id, sku) key, so no query reads this index.
ALTER TABLE `item`
  DROP KEY `item_sku_ngram_idx`;

-- +goose Down
ALTER TABLE `item`
  ADD FULLTEXT KEY `item_sku_ngram_idx` (`sku`) WITH PARSER ngram;
