-- Runs after every e2e seed that inserts picks. pick.buyer_account_id is denormalized from the order
-- (CreatePick copies it in the API), so seeded picks need it derived the same way.
UPDATE pick p JOIN sales_order so ON so.id = p.sales_order_id
SET p.buyer_account_id = so.buyer_account_id
WHERE p.buyer_account_id IS NULL;
