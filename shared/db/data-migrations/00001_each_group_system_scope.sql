-- +goose Up

-- each_group is shared — the system product lines (credit, service, shipping, tax) point at it —
-- but it was scoped to the first tenant, unlike time_group and currency_group. Unit group reads
-- are scoped `account_id = ? OR account_id IS NULL`, so for every other tenant the group resolved
-- to no rows and the not-found 404'd the whole product line list.
UPDATE `unit_group` SET `account_id` = NULL WHERE `id` = 'each_group';

-- +goose Down

-- Not reversible: the account it was wrongly scoped to differs per environment, and restoring it
-- restores the 404.
SELECT 1;
