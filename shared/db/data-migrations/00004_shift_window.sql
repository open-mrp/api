-- +goose NO TRANSACTION
-- +goose Up

-- Set the production shift window on one production account, matched to its scan history. It ran in
-- production; a new database has no such account, so there is nothing to set.
SELECT 1;

-- +goose Down

-- Not reversible.
SELECT 1;
