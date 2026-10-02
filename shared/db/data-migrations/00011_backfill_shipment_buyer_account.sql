-- +goose NO TRANSACTION
-- +goose Up

-- Copy each shipment's customer from its order. New shipments get it at insert (CreateShipment) and
-- customer merges move it (MergeCustomerShipmentBuyers), so this is history only. Idempotent: it
-- touches only rows that differ from their order, so it can be re-run to catch shipments written by an
-- older image. updated_at is left alone: this is a derived value, not a change to the shipment.
--
-- Batched, each batch its own transaction: one UPDATE over every shipment (125k in prod) outran
-- Vitess's transaction limit and was killed. Each statement fills up to 5,000 shipments that still
-- differ; repeated past the table's size, the later ones find nothing to do.

UPDATE `shipment` s
JOIN (
    SELECT s2.id, so.buyer_account_id
    FROM `shipment` s2
    JOIN `sales_order` so ON so.id = s2.sales_order_id
    WHERE s2.buyer_account_id IS NULL OR s2.buyer_account_id <> so.buyer_account_id
    ORDER BY s2.id
    LIMIT 5000
) b ON b.id = s.id
SET s.buyer_account_id = b.buyer_account_id;

UPDATE `shipment` s
JOIN (
    SELECT s2.id, so.buyer_account_id
    FROM `shipment` s2
    JOIN `sales_order` so ON so.id = s2.sales_order_id
    WHERE s2.buyer_account_id IS NULL OR s2.buyer_account_id <> so.buyer_account_id
    ORDER BY s2.id
    LIMIT 5000
) b ON b.id = s.id
SET s.buyer_account_id = b.buyer_account_id;

UPDATE `shipment` s
JOIN (
    SELECT s2.id, so.buyer_account_id
    FROM `shipment` s2
    JOIN `sales_order` so ON so.id = s2.sales_order_id
    WHERE s2.buyer_account_id IS NULL OR s2.buyer_account_id <> so.buyer_account_id
    ORDER BY s2.id
    LIMIT 5000
) b ON b.id = s.id
SET s.buyer_account_id = b.buyer_account_id;

UPDATE `shipment` s
JOIN (
    SELECT s2.id, so.buyer_account_id
    FROM `shipment` s2
    JOIN `sales_order` so ON so.id = s2.sales_order_id
    WHERE s2.buyer_account_id IS NULL OR s2.buyer_account_id <> so.buyer_account_id
    ORDER BY s2.id
    LIMIT 5000
) b ON b.id = s.id
SET s.buyer_account_id = b.buyer_account_id;

UPDATE `shipment` s
JOIN (
    SELECT s2.id, so.buyer_account_id
    FROM `shipment` s2
    JOIN `sales_order` so ON so.id = s2.sales_order_id
    WHERE s2.buyer_account_id IS NULL OR s2.buyer_account_id <> so.buyer_account_id
    ORDER BY s2.id
    LIMIT 5000
) b ON b.id = s.id
SET s.buyer_account_id = b.buyer_account_id;

UPDATE `shipment` s
JOIN (
    SELECT s2.id, so.buyer_account_id
    FROM `shipment` s2
    JOIN `sales_order` so ON so.id = s2.sales_order_id
    WHERE s2.buyer_account_id IS NULL OR s2.buyer_account_id <> so.buyer_account_id
    ORDER BY s2.id
    LIMIT 5000
) b ON b.id = s.id
SET s.buyer_account_id = b.buyer_account_id;

UPDATE `shipment` s
JOIN (
    SELECT s2.id, so.buyer_account_id
    FROM `shipment` s2
    JOIN `sales_order` so ON so.id = s2.sales_order_id
    WHERE s2.buyer_account_id IS NULL OR s2.buyer_account_id <> so.buyer_account_id
    ORDER BY s2.id
    LIMIT 5000
) b ON b.id = s.id
SET s.buyer_account_id = b.buyer_account_id;

UPDATE `shipment` s
JOIN (
    SELECT s2.id, so.buyer_account_id
    FROM `shipment` s2
    JOIN `sales_order` so ON so.id = s2.sales_order_id
    WHERE s2.buyer_account_id IS NULL OR s2.buyer_account_id <> so.buyer_account_id
    ORDER BY s2.id
    LIMIT 5000
) b ON b.id = s.id
SET s.buyer_account_id = b.buyer_account_id;

UPDATE `shipment` s
JOIN (
    SELECT s2.id, so.buyer_account_id
    FROM `shipment` s2
    JOIN `sales_order` so ON so.id = s2.sales_order_id
    WHERE s2.buyer_account_id IS NULL OR s2.buyer_account_id <> so.buyer_account_id
    ORDER BY s2.id
    LIMIT 5000
) b ON b.id = s.id
SET s.buyer_account_id = b.buyer_account_id;

UPDATE `shipment` s
JOIN (
    SELECT s2.id, so.buyer_account_id
    FROM `shipment` s2
    JOIN `sales_order` so ON so.id = s2.sales_order_id
    WHERE s2.buyer_account_id IS NULL OR s2.buyer_account_id <> so.buyer_account_id
    ORDER BY s2.id
    LIMIT 5000
) b ON b.id = s.id
SET s.buyer_account_id = b.buyer_account_id;

UPDATE `shipment` s
JOIN (
    SELECT s2.id, so.buyer_account_id
    FROM `shipment` s2
    JOIN `sales_order` so ON so.id = s2.sales_order_id
    WHERE s2.buyer_account_id IS NULL OR s2.buyer_account_id <> so.buyer_account_id
    ORDER BY s2.id
    LIMIT 5000
) b ON b.id = s.id
SET s.buyer_account_id = b.buyer_account_id;

UPDATE `shipment` s
JOIN (
    SELECT s2.id, so.buyer_account_id
    FROM `shipment` s2
    JOIN `sales_order` so ON so.id = s2.sales_order_id
    WHERE s2.buyer_account_id IS NULL OR s2.buyer_account_id <> so.buyer_account_id
    ORDER BY s2.id
    LIMIT 5000
) b ON b.id = s.id
SET s.buyer_account_id = b.buyer_account_id;

UPDATE `shipment` s
JOIN (
    SELECT s2.id, so.buyer_account_id
    FROM `shipment` s2
    JOIN `sales_order` so ON so.id = s2.sales_order_id
    WHERE s2.buyer_account_id IS NULL OR s2.buyer_account_id <> so.buyer_account_id
    ORDER BY s2.id
    LIMIT 5000
) b ON b.id = s.id
SET s.buyer_account_id = b.buyer_account_id;

UPDATE `shipment` s
JOIN (
    SELECT s2.id, so.buyer_account_id
    FROM `shipment` s2
    JOIN `sales_order` so ON so.id = s2.sales_order_id
    WHERE s2.buyer_account_id IS NULL OR s2.buyer_account_id <> so.buyer_account_id
    ORDER BY s2.id
    LIMIT 5000
) b ON b.id = s.id
SET s.buyer_account_id = b.buyer_account_id;

UPDATE `shipment` s
JOIN (
    SELECT s2.id, so.buyer_account_id
    FROM `shipment` s2
    JOIN `sales_order` so ON so.id = s2.sales_order_id
    WHERE s2.buyer_account_id IS NULL OR s2.buyer_account_id <> so.buyer_account_id
    ORDER BY s2.id
    LIMIT 5000
) b ON b.id = s.id
SET s.buyer_account_id = b.buyer_account_id;

UPDATE `shipment` s
JOIN (
    SELECT s2.id, so.buyer_account_id
    FROM `shipment` s2
    JOIN `sales_order` so ON so.id = s2.sales_order_id
    WHERE s2.buyer_account_id IS NULL OR s2.buyer_account_id <> so.buyer_account_id
    ORDER BY s2.id
    LIMIT 5000
) b ON b.id = s.id
SET s.buyer_account_id = b.buyer_account_id;

UPDATE `shipment` s
JOIN (
    SELECT s2.id, so.buyer_account_id
    FROM `shipment` s2
    JOIN `sales_order` so ON so.id = s2.sales_order_id
    WHERE s2.buyer_account_id IS NULL OR s2.buyer_account_id <> so.buyer_account_id
    ORDER BY s2.id
    LIMIT 5000
) b ON b.id = s.id
SET s.buyer_account_id = b.buyer_account_id;

UPDATE `shipment` s
JOIN (
    SELECT s2.id, so.buyer_account_id
    FROM `shipment` s2
    JOIN `sales_order` so ON so.id = s2.sales_order_id
    WHERE s2.buyer_account_id IS NULL OR s2.buyer_account_id <> so.buyer_account_id
    ORDER BY s2.id
    LIMIT 5000
) b ON b.id = s.id
SET s.buyer_account_id = b.buyer_account_id;

UPDATE `shipment` s
JOIN (
    SELECT s2.id, so.buyer_account_id
    FROM `shipment` s2
    JOIN `sales_order` so ON so.id = s2.sales_order_id
    WHERE s2.buyer_account_id IS NULL OR s2.buyer_account_id <> so.buyer_account_id
    ORDER BY s2.id
    LIMIT 5000
) b ON b.id = s.id
SET s.buyer_account_id = b.buyer_account_id;

UPDATE `shipment` s
JOIN (
    SELECT s2.id, so.buyer_account_id
    FROM `shipment` s2
    JOIN `sales_order` so ON so.id = s2.sales_order_id
    WHERE s2.buyer_account_id IS NULL OR s2.buyer_account_id <> so.buyer_account_id
    ORDER BY s2.id
    LIMIT 5000
) b ON b.id = s.id
SET s.buyer_account_id = b.buyer_account_id;

UPDATE `shipment` s
JOIN (
    SELECT s2.id, so.buyer_account_id
    FROM `shipment` s2
    JOIN `sales_order` so ON so.id = s2.sales_order_id
    WHERE s2.buyer_account_id IS NULL OR s2.buyer_account_id <> so.buyer_account_id
    ORDER BY s2.id
    LIMIT 5000
) b ON b.id = s.id
SET s.buyer_account_id = b.buyer_account_id;

UPDATE `shipment` s
JOIN (
    SELECT s2.id, so.buyer_account_id
    FROM `shipment` s2
    JOIN `sales_order` so ON so.id = s2.sales_order_id
    WHERE s2.buyer_account_id IS NULL OR s2.buyer_account_id <> so.buyer_account_id
    ORDER BY s2.id
    LIMIT 5000
) b ON b.id = s.id
SET s.buyer_account_id = b.buyer_account_id;

UPDATE `shipment` s
JOIN (
    SELECT s2.id, so.buyer_account_id
    FROM `shipment` s2
    JOIN `sales_order` so ON so.id = s2.sales_order_id
    WHERE s2.buyer_account_id IS NULL OR s2.buyer_account_id <> so.buyer_account_id
    ORDER BY s2.id
    LIMIT 5000
) b ON b.id = s.id
SET s.buyer_account_id = b.buyer_account_id;

UPDATE `shipment` s
JOIN (
    SELECT s2.id, so.buyer_account_id
    FROM `shipment` s2
    JOIN `sales_order` so ON so.id = s2.sales_order_id
    WHERE s2.buyer_account_id IS NULL OR s2.buyer_account_id <> so.buyer_account_id
    ORDER BY s2.id
    LIMIT 5000
) b ON b.id = s.id
SET s.buyer_account_id = b.buyer_account_id;

UPDATE `shipment` s
JOIN (
    SELECT s2.id, so.buyer_account_id
    FROM `shipment` s2
    JOIN `sales_order` so ON so.id = s2.sales_order_id
    WHERE s2.buyer_account_id IS NULL OR s2.buyer_account_id <> so.buyer_account_id
    ORDER BY s2.id
    LIMIT 5000
) b ON b.id = s.id
SET s.buyer_account_id = b.buyer_account_id;

UPDATE `shipment` s
JOIN (
    SELECT s2.id, so.buyer_account_id
    FROM `shipment` s2
    JOIN `sales_order` so ON so.id = s2.sales_order_id
    WHERE s2.buyer_account_id IS NULL OR s2.buyer_account_id <> so.buyer_account_id
    ORDER BY s2.id
    LIMIT 5000
) b ON b.id = s.id
SET s.buyer_account_id = b.buyer_account_id;

UPDATE `shipment` s
JOIN (
    SELECT s2.id, so.buyer_account_id
    FROM `shipment` s2
    JOIN `sales_order` so ON so.id = s2.sales_order_id
    WHERE s2.buyer_account_id IS NULL OR s2.buyer_account_id <> so.buyer_account_id
    ORDER BY s2.id
    LIMIT 5000
) b ON b.id = s.id
SET s.buyer_account_id = b.buyer_account_id;

UPDATE `shipment` s
JOIN (
    SELECT s2.id, so.buyer_account_id
    FROM `shipment` s2
    JOIN `sales_order` so ON so.id = s2.sales_order_id
    WHERE s2.buyer_account_id IS NULL OR s2.buyer_account_id <> so.buyer_account_id
    ORDER BY s2.id
    LIMIT 5000
) b ON b.id = s.id
SET s.buyer_account_id = b.buyer_account_id;

UPDATE `shipment` s
JOIN (
    SELECT s2.id, so.buyer_account_id
    FROM `shipment` s2
    JOIN `sales_order` so ON so.id = s2.sales_order_id
    WHERE s2.buyer_account_id IS NULL OR s2.buyer_account_id <> so.buyer_account_id
    ORDER BY s2.id
    LIMIT 5000
) b ON b.id = s.id
SET s.buyer_account_id = b.buyer_account_id;

UPDATE `shipment` s
JOIN (
    SELECT s2.id, so.buyer_account_id
    FROM `shipment` s2
    JOIN `sales_order` so ON so.id = s2.sales_order_id
    WHERE s2.buyer_account_id IS NULL OR s2.buyer_account_id <> so.buyer_account_id
    ORDER BY s2.id
    LIMIT 5000
) b ON b.id = s.id
SET s.buyer_account_id = b.buyer_account_id;

UPDATE `shipment` s
JOIN (
    SELECT s2.id, so.buyer_account_id
    FROM `shipment` s2
    JOIN `sales_order` so ON so.id = s2.sales_order_id
    WHERE s2.buyer_account_id IS NULL OR s2.buyer_account_id <> so.buyer_account_id
    ORDER BY s2.id
    LIMIT 5000
) b ON b.id = s.id
SET s.buyer_account_id = b.buyer_account_id;

UPDATE `shipment` s
JOIN (
    SELECT s2.id, so.buyer_account_id
    FROM `shipment` s2
    JOIN `sales_order` so ON so.id = s2.sales_order_id
    WHERE s2.buyer_account_id IS NULL OR s2.buyer_account_id <> so.buyer_account_id
    ORDER BY s2.id
    LIMIT 5000
) b ON b.id = s.id
SET s.buyer_account_id = b.buyer_account_id;

UPDATE `shipment` s
JOIN (
    SELECT s2.id, so.buyer_account_id
    FROM `shipment` s2
    JOIN `sales_order` so ON so.id = s2.sales_order_id
    WHERE s2.buyer_account_id IS NULL OR s2.buyer_account_id <> so.buyer_account_id
    ORDER BY s2.id
    LIMIT 5000
) b ON b.id = s.id
SET s.buyer_account_id = b.buyer_account_id;

UPDATE `shipment` s
JOIN (
    SELECT s2.id, so.buyer_account_id
    FROM `shipment` s2
    JOIN `sales_order` so ON so.id = s2.sales_order_id
    WHERE s2.buyer_account_id IS NULL OR s2.buyer_account_id <> so.buyer_account_id
    ORDER BY s2.id
    LIMIT 5000
) b ON b.id = s.id
SET s.buyer_account_id = b.buyer_account_id;

UPDATE `shipment` s
JOIN (
    SELECT s2.id, so.buyer_account_id
    FROM `shipment` s2
    JOIN `sales_order` so ON so.id = s2.sales_order_id
    WHERE s2.buyer_account_id IS NULL OR s2.buyer_account_id <> so.buyer_account_id
    ORDER BY s2.id
    LIMIT 5000
) b ON b.id = s.id
SET s.buyer_account_id = b.buyer_account_id;

UPDATE `shipment` s
JOIN (
    SELECT s2.id, so.buyer_account_id
    FROM `shipment` s2
    JOIN `sales_order` so ON so.id = s2.sales_order_id
    WHERE s2.buyer_account_id IS NULL OR s2.buyer_account_id <> so.buyer_account_id
    ORDER BY s2.id
    LIMIT 5000
) b ON b.id = s.id
SET s.buyer_account_id = b.buyer_account_id;

UPDATE `shipment` s
JOIN (
    SELECT s2.id, so.buyer_account_id
    FROM `shipment` s2
    JOIN `sales_order` so ON so.id = s2.sales_order_id
    WHERE s2.buyer_account_id IS NULL OR s2.buyer_account_id <> so.buyer_account_id
    ORDER BY s2.id
    LIMIT 5000
) b ON b.id = s.id
SET s.buyer_account_id = b.buyer_account_id;

UPDATE `shipment` s
JOIN (
    SELECT s2.id, so.buyer_account_id
    FROM `shipment` s2
    JOIN `sales_order` so ON so.id = s2.sales_order_id
    WHERE s2.buyer_account_id IS NULL OR s2.buyer_account_id <> so.buyer_account_id
    ORDER BY s2.id
    LIMIT 5000
) b ON b.id = s.id
SET s.buyer_account_id = b.buyer_account_id;

UPDATE `shipment` s
JOIN (
    SELECT s2.id, so.buyer_account_id
    FROM `shipment` s2
    JOIN `sales_order` so ON so.id = s2.sales_order_id
    WHERE s2.buyer_account_id IS NULL OR s2.buyer_account_id <> so.buyer_account_id
    ORDER BY s2.id
    LIMIT 5000
) b ON b.id = s.id
SET s.buyer_account_id = b.buyer_account_id;

UPDATE `shipment` s
JOIN (
    SELECT s2.id, so.buyer_account_id
    FROM `shipment` s2
    JOIN `sales_order` so ON so.id = s2.sales_order_id
    WHERE s2.buyer_account_id IS NULL OR s2.buyer_account_id <> so.buyer_account_id
    ORDER BY s2.id
    LIMIT 5000
) b ON b.id = s.id
SET s.buyer_account_id = b.buyer_account_id;

UPDATE `shipment` s
JOIN (
    SELECT s2.id, so.buyer_account_id
    FROM `shipment` s2
    JOIN `sales_order` so ON so.id = s2.sales_order_id
    WHERE s2.buyer_account_id IS NULL OR s2.buyer_account_id <> so.buyer_account_id
    ORDER BY s2.id
    LIMIT 5000
) b ON b.id = s.id
SET s.buyer_account_id = b.buyer_account_id;

UPDATE `shipment` s
JOIN (
    SELECT s2.id, so.buyer_account_id
    FROM `shipment` s2
    JOIN `sales_order` so ON so.id = s2.sales_order_id
    WHERE s2.buyer_account_id IS NULL OR s2.buyer_account_id <> so.buyer_account_id
    ORDER BY s2.id
    LIMIT 5000
) b ON b.id = s.id
SET s.buyer_account_id = b.buyer_account_id;

UPDATE `shipment` s
JOIN (
    SELECT s2.id, so.buyer_account_id
    FROM `shipment` s2
    JOIN `sales_order` so ON so.id = s2.sales_order_id
    WHERE s2.buyer_account_id IS NULL OR s2.buyer_account_id <> so.buyer_account_id
    ORDER BY s2.id
    LIMIT 5000
) b ON b.id = s.id
SET s.buyer_account_id = b.buyer_account_id;

UPDATE `shipment` s
JOIN (
    SELECT s2.id, so.buyer_account_id
    FROM `shipment` s2
    JOIN `sales_order` so ON so.id = s2.sales_order_id
    WHERE s2.buyer_account_id IS NULL OR s2.buyer_account_id <> so.buyer_account_id
    ORDER BY s2.id
    LIMIT 5000
) b ON b.id = s.id
SET s.buyer_account_id = b.buyer_account_id;

UPDATE `shipment` s
JOIN (
    SELECT s2.id, so.buyer_account_id
    FROM `shipment` s2
    JOIN `sales_order` so ON so.id = s2.sales_order_id
    WHERE s2.buyer_account_id IS NULL OR s2.buyer_account_id <> so.buyer_account_id
    ORDER BY s2.id
    LIMIT 5000
) b ON b.id = s.id
SET s.buyer_account_id = b.buyer_account_id;

UPDATE `shipment` s
JOIN (
    SELECT s2.id, so.buyer_account_id
    FROM `shipment` s2
    JOIN `sales_order` so ON so.id = s2.sales_order_id
    WHERE s2.buyer_account_id IS NULL OR s2.buyer_account_id <> so.buyer_account_id
    ORDER BY s2.id
    LIMIT 5000
) b ON b.id = s.id
SET s.buyer_account_id = b.buyer_account_id;

UPDATE `shipment` s
JOIN (
    SELECT s2.id, so.buyer_account_id
    FROM `shipment` s2
    JOIN `sales_order` so ON so.id = s2.sales_order_id
    WHERE s2.buyer_account_id IS NULL OR s2.buyer_account_id <> so.buyer_account_id
    ORDER BY s2.id
    LIMIT 5000
) b ON b.id = s.id
SET s.buyer_account_id = b.buyer_account_id;

UPDATE `shipment` s
JOIN (
    SELECT s2.id, so.buyer_account_id
    FROM `shipment` s2
    JOIN `sales_order` so ON so.id = s2.sales_order_id
    WHERE s2.buyer_account_id IS NULL OR s2.buyer_account_id <> so.buyer_account_id
    ORDER BY s2.id
    LIMIT 5000
) b ON b.id = s.id
SET s.buyer_account_id = b.buyer_account_id;

UPDATE `shipment` s
JOIN (
    SELECT s2.id, so.buyer_account_id
    FROM `shipment` s2
    JOIN `sales_order` so ON so.id = s2.sales_order_id
    WHERE s2.buyer_account_id IS NULL OR s2.buyer_account_id <> so.buyer_account_id
    ORDER BY s2.id
    LIMIT 5000
) b ON b.id = s.id
SET s.buyer_account_id = b.buyer_account_id;

UPDATE `shipment` s
JOIN (
    SELECT s2.id, so.buyer_account_id
    FROM `shipment` s2
    JOIN `sales_order` so ON so.id = s2.sales_order_id
    WHERE s2.buyer_account_id IS NULL OR s2.buyer_account_id <> so.buyer_account_id
    ORDER BY s2.id
    LIMIT 5000
) b ON b.id = s.id
SET s.buyer_account_id = b.buyer_account_id;

UPDATE `shipment` s
JOIN (
    SELECT s2.id, so.buyer_account_id
    FROM `shipment` s2
    JOIN `sales_order` so ON so.id = s2.sales_order_id
    WHERE s2.buyer_account_id IS NULL OR s2.buyer_account_id <> so.buyer_account_id
    ORDER BY s2.id
    LIMIT 5000
) b ON b.id = s.id
SET s.buyer_account_id = b.buyer_account_id;

UPDATE `shipment` s
JOIN (
    SELECT s2.id, so.buyer_account_id
    FROM `shipment` s2
    JOIN `sales_order` so ON so.id = s2.sales_order_id
    WHERE s2.buyer_account_id IS NULL OR s2.buyer_account_id <> so.buyer_account_id
    ORDER BY s2.id
    LIMIT 5000
) b ON b.id = s.id
SET s.buyer_account_id = b.buyer_account_id;

UPDATE `shipment` s
JOIN (
    SELECT s2.id, so.buyer_account_id
    FROM `shipment` s2
    JOIN `sales_order` so ON so.id = s2.sales_order_id
    WHERE s2.buyer_account_id IS NULL OR s2.buyer_account_id <> so.buyer_account_id
    ORDER BY s2.id
    LIMIT 5000
) b ON b.id = s.id
SET s.buyer_account_id = b.buyer_account_id;

UPDATE `shipment` s
JOIN (
    SELECT s2.id, so.buyer_account_id
    FROM `shipment` s2
    JOIN `sales_order` so ON so.id = s2.sales_order_id
    WHERE s2.buyer_account_id IS NULL OR s2.buyer_account_id <> so.buyer_account_id
    ORDER BY s2.id
    LIMIT 5000
) b ON b.id = s.id
SET s.buyer_account_id = b.buyer_account_id;

UPDATE `shipment` s
JOIN (
    SELECT s2.id, so.buyer_account_id
    FROM `shipment` s2
    JOIN `sales_order` so ON so.id = s2.sales_order_id
    WHERE s2.buyer_account_id IS NULL OR s2.buyer_account_id <> so.buyer_account_id
    ORDER BY s2.id
    LIMIT 5000
) b ON b.id = s.id
SET s.buyer_account_id = b.buyer_account_id;

UPDATE `shipment` s
JOIN (
    SELECT s2.id, so.buyer_account_id
    FROM `shipment` s2
    JOIN `sales_order` so ON so.id = s2.sales_order_id
    WHERE s2.buyer_account_id IS NULL OR s2.buyer_account_id <> so.buyer_account_id
    ORDER BY s2.id
    LIMIT 5000
) b ON b.id = s.id
SET s.buyer_account_id = b.buyer_account_id;

UPDATE `shipment` s
JOIN (
    SELECT s2.id, so.buyer_account_id
    FROM `shipment` s2
    JOIN `sales_order` so ON so.id = s2.sales_order_id
    WHERE s2.buyer_account_id IS NULL OR s2.buyer_account_id <> so.buyer_account_id
    ORDER BY s2.id
    LIMIT 5000
) b ON b.id = s.id
SET s.buyer_account_id = b.buyer_account_id;

UPDATE `shipment` s
JOIN (
    SELECT s2.id, so.buyer_account_id
    FROM `shipment` s2
    JOIN `sales_order` so ON so.id = s2.sales_order_id
    WHERE s2.buyer_account_id IS NULL OR s2.buyer_account_id <> so.buyer_account_id
    ORDER BY s2.id
    LIMIT 5000
) b ON b.id = s.id
SET s.buyer_account_id = b.buyer_account_id;

UPDATE `shipment` s
JOIN (
    SELECT s2.id, so.buyer_account_id
    FROM `shipment` s2
    JOIN `sales_order` so ON so.id = s2.sales_order_id
    WHERE s2.buyer_account_id IS NULL OR s2.buyer_account_id <> so.buyer_account_id
    ORDER BY s2.id
    LIMIT 5000
) b ON b.id = s.id
SET s.buyer_account_id = b.buyer_account_id;

UPDATE `shipment` s
JOIN (
    SELECT s2.id, so.buyer_account_id
    FROM `shipment` s2
    JOIN `sales_order` so ON so.id = s2.sales_order_id
    WHERE s2.buyer_account_id IS NULL OR s2.buyer_account_id <> so.buyer_account_id
    ORDER BY s2.id
    LIMIT 5000
) b ON b.id = s.id
SET s.buyer_account_id = b.buyer_account_id;

-- +goose Down

-- Not reversible: the column is dropped by its schema migration's Down.
SELECT 1;
