-- +goose NO TRANSACTION
-- +goose Up

-- Snapped a handful of production scan quantities, and the allocations and reconciles drawn from them,
-- off a 4e-16 rounding residue the scan consumer once left. The consumer no longer rounds that way, the
-- rows were fixed in production, and a new database has none of them, so there is nothing to snap.
SELECT 1;

-- +goose Down

-- Not reversible.
SELECT 1;
