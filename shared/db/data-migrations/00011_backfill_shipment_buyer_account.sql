-- +goose Up

-- Copy each shipment's customer from its order. New shipments get it at insert (CreateShipment) and
-- customer merges move it (MergeCustomerShipments), so this is history only. Idempotent: it touches
-- only rows that differ from their order, so it can be re-run to catch shipments written by an older
-- image or by the dashboard's pack route. updated_at is left alone: this is a derived value.
UPDATE `shipment` s
JOIN `sales_order` so ON so.id = s.sales_order_id
SET s.buyer_account_id = so.buyer_account_id
WHERE s.buyer_account_id IS NULL OR s.buyer_account_id <> so.buyer_account_id;

-- +goose Down

-- Not reversible: the column is dropped by its schema migration's Down.
SELECT 1;
