-- +goose NO TRANSACTION
-- +goose Up

-- SKU search on the inventory change-log list is a substring match on item.sku. Substring search cannot
-- use the (account_id, sku) b-tree unique key, nor item_sku_idx — a default-parser FULLTEXT index that
-- tokenizes on word boundaries, so it never matches inside a SKU like "L-123839". Both degrade to a full
-- item scan. An ngram FULLTEXT index tokenizes into overlapping 2-grams, so a boolean-mode phrase search
-- ('"3839"') matches "3839" as a substring straight from the index.
--
-- No query MATCHes item.sku today (item search runs through LIKE), so item_sku_idx is dead weight and
-- nothing depends on the default parser. Replace it rather than adding a second single-column FULLTEXT
-- index on sku: with two, MySQL silently picks one for MATCH(sku) — and would pick the default-parser one.
-- item_sku_description_idx (sku, description) is left alone; MATCH(sku) requires an index on exactly (sku)
-- so the two-column index is not a candidate and cannot conflict.
ALTER TABLE `item`
  DROP KEY `item_sku_idx`;

ALTER TABLE `item`
  ADD FULLTEXT KEY `item_sku_ngram_idx` (`sku`) WITH PARSER ngram;

-- +goose Down

ALTER TABLE `item`
  DROP KEY `item_sku_ngram_idx`;

ALTER TABLE `item`
  ADD FULLTEXT KEY `item_sku_idx` (`sku`);
