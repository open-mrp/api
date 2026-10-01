-- Runs after every e2e seed that inserts picks or shipments. Their buyer_account_id is denormalized
-- from the order (CreatePick and CreateShipment copy it in the API), so seeded rows need it derived
-- the same way.
UPDATE pick p JOIN sales_order so ON so.id = p.sales_order_id
SET p.buyer_account_id = so.buyer_account_id
WHERE p.buyer_account_id IS NULL;

UPDATE shipment s JOIN sales_order so ON so.id = s.sales_order_id
SET s.buyer_account_id = so.buyer_account_id
WHERE s.buyer_account_id IS NULL;
