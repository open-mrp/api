-- +goose Up

-- Copy each pick's customer from its order. New picks get it at insert (CreatePick) and customer
-- merges move it (MergeCustomerPicks), so this is history only. Idempotent: it touches only rows
-- that differ from their order, so it can be re-run to catch picks written by an older image.
-- updated_at is left alone: this is a derived value, not a change to the pick.
UPDATE `pick` p
JOIN `sales_order` so ON so.id = p.sales_order_id
SET p.buyer_account_id = so.buyer_account_id
WHERE p.buyer_account_id IS NULL OR p.buyer_account_id <> so.buyer_account_id;

-- +goose Down

-- Not reversible: the column is dropped by its schema migration's Down.
SELECT 1;
