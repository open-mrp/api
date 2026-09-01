-- +goose NO TRANSACTION
-- +goose Up

-- The pick list's `q` box is documented as searching four things — the pick's number, its order, the
-- customer's PO number, and the customer — but only the first three were ever implemented. Searching
-- a customer's name or number returned nothing, which is what
-- TestPicksList_SearchMatchesTheOrderAndCustomerItCameFrom has been failing on.
--
-- Serving the customer half needs the same treatment 00013 gave the other columns: substring search
-- cannot use a b-tree or a default-parser FULLTEXT index (which tokenizes on word boundaries), so
-- each column matched by MATCH() needs an ngram FULLTEXT index of its own.

-- account_name_idx is a default-parser FULLTEXT index that no query MATCHes today. Two FULLTEXT
-- indexes on one column leave MySQL to pick between them silently, and it picks the default-parser
-- one — which never matches a substring — so this is replaced rather than added alongside. Same
-- reasoning as 00013's treatment of sales_order.customer_po_number.
ALTER TABLE `account`
  DROP KEY `account_name_idx`;

ALTER TABLE `account`
  ADD FULLTEXT KEY `account_name_ngram_idx` (`name`) WITH PARSER ngram;

-- The customer's number lives on the relation, not the account: it is the number *this* seller filed
-- their customer under.
ALTER TABLE `account_relation`
  ADD FULLTEXT KEY `account_relation_external_number_ngram_idx` (`external_number`) WITH PARSER ngram;

-- +goose Down

ALTER TABLE `account_relation`
  DROP KEY `account_relation_external_number_ngram_idx`;

ALTER TABLE `account`
  DROP KEY `account_name_ngram_idx`;

ALTER TABLE `account`
  ADD FULLTEXT KEY `account_name_idx` (`name`);
