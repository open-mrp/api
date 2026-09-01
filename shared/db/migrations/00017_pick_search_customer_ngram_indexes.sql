-- +goose NO TRANSACTION
-- +goose Up

-- The pick list's `q` box searches four things a picker might have in hand: the pick's own number, the
-- order it fulfills, the customer's PO number, and the customer. The first three are served by ngram
-- FULLTEXT indexes (00013); the customer was not searchable at all, so a picker holding a customer's
-- name or account number could not find their picks.
--
-- Both columns need the ngram parser for the same reason 00013 gives: a default-parser index tokenizes
-- on word boundaries, so it can never match a substring, and a b-tree cannot serve a leading wildcard.
-- An ngram index tokenizes into overlapping 2-grams, so a boolean-mode phrase search finds "acme"
-- inside "Acme Fabrication Ltd".

-- account_name_idx is a default-parser FULLTEXT index that no query uses — same dead weight from the
-- initial schema that 00013 found on pick.number and sales_order.customer_po_number. It has to be
-- dropped rather than left alongside: MATCH(name) cannot name which index to use, and with two
-- single-column FULLTEXT indexes on one column MySQL silently picks the default-parser one, which
-- would never match a substring. Verified unused across both this repo's queries and the dashboard's.
ALTER TABLE `account`
  DROP KEY `account_name_idx`;

ALTER TABLE `account`
  ADD FULLTEXT KEY `account_name_ngram_idx` (`name`) WITH PARSER ngram;

-- external_number is the customer's account number as the merchant knows it — the number printed on
-- their paperwork, and the one a picker is most likely to be handed. It had no FULLTEXT index at all.
ALTER TABLE `account_relation`
  ADD FULLTEXT KEY `account_relation_external_number_ngram_idx` (`external_number`) WITH PARSER ngram;

-- The rebuild is required, not hygiene. account_relation already carried three FULLTEXT indexes
-- (alias, notes, and the pair), so adding a fourth is an in-place ALTER that indexes the rows present
-- when it runs and then does not maintain the new index for subsequent INSERTs. The failure is
-- particularly nasty because it is invisible: every row that existed at migration time keeps matching,
-- so the search looks like it works, and only customers created afterwards are unfindable.
--
-- Measured on MySQL 8.4: after the plain ADD, a freshly inserted external_number did not match even
-- after OPTIMIZE TABLE with innodb_optimize_fulltext_only; after this rebuild it matched immediately.
-- The account index above does not need one because dropping and re-adding its index already rebuilt
-- that table.
ALTER TABLE `account_relation` ENGINE=InnoDB;

-- +goose Down

ALTER TABLE `account_relation`
  DROP KEY `account_relation_external_number_ngram_idx`;

ALTER TABLE `account`
  DROP KEY `account_name_ngram_idx`;

ALTER TABLE `account`
  ADD FULLTEXT KEY `account_name_idx` (`name`);
