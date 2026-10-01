-- +goose Up
DELETE ii, q
FROM inventory_issue ii
JOIN sales_order so ON so.id = ii.order_id
JOIN quantity q ON q.id = ii.quantity_id
WHERE ii.status_code = 'reserved'
AND so.sales_order_status_code = 'fulfilled'
AND NOT EXISTS (SELECT 1 FROM inventory_allocation ia WHERE ia.inventory_issue_id = ii.id);

DELETE ii, q
FROM inventory_issue ii
JOIN sales_order so ON so.id = ii.order_id
JOIN quantity q ON q.id = ii.quantity_id
WHERE ii.status_code = 'reserved'
AND so.sales_order_status_code = 'issued'
AND NOT EXISTS (SELECT 1 FROM inventory_allocation ia WHERE ia.inventory_issue_id = ii.id)
AND NOT EXISTS (
    SELECT 1 FROM sales_order_line sol
    JOIN product p ON p.id = sol.product_id
    WHERE sol.sales_order_id = so.id AND sol.item_id = ii.item_id
    AND p.product_type_code = 'sale'
);

-- +goose Down
SELECT 1;