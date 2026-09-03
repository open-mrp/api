-- +goose Up

-- Fill the denormalized ship-by sort key for picks that predate the column (all rows start at the
-- 9999-12-31 default). New picks are populated at insert by CreatePick, so this is history only.
-- Idempotent: it touches only rows whose key still differs from their order's current value, so a
-- prior batched backfill (run out of band on a large production account) leaves this a no-op.
-- updated_at is deliberately left alone — this is a derived value, not a change to the pick.
UPDATE `pick` p
JOIN `sales_order` so ON so.id = p.sales_order_id
SET p.ship_by_sort_date = COALESCE(so.ship_by_date, '9999-12-31')
WHERE p.ship_by_sort_date <> COALESCE(so.ship_by_date, '9999-12-31');

-- +goose Down

-- Not reversible: the pre-backfill values were the column default, not meaningful data.
SELECT 1;
