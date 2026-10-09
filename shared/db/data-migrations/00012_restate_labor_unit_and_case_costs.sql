-- +goose NO TRANSACTION
-- +goose Up

-- Restated costs on one production account that two cost-rollup defects had left wrong; both defects
-- were fixed in the code this shipped with. It ran in production, and a new database has no rows the
-- defects produced, so there is nothing to restate.
SELECT 1;

-- +goose Down

-- Not reversible.
SELECT 1;
