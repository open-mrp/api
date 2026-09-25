-- +goose Up

-- Re-runs 00005 before the pick list reads pick.buyer_account_id: images older than CreatePick's
-- buyer copy kept creating picks until that release finished rolling out, and those rows are still
-- NULL. Idempotent, so it is a no-op where 00005 already covered everything.
UPDATE `pick` p
JOIN `sales_order` so ON so.id = p.sales_order_id
SET p.buyer_account_id = so.buyer_account_id
WHERE p.buyer_account_id IS NULL OR p.buyer_account_id <> so.buyer_account_id;

-- +goose Down

SELECT 1;
