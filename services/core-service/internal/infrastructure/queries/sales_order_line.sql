-- name: GetSalesOrderLine :one
SELECT
    sol.id,
    sol.line_item_number,
    sol.product_sku,
    sol.product_description,
    sol.product_id,
    sol.item_id,
    i.sku AS item_sku,
    sol.edi_line_item_id,
    -- Quantity ordered
    q.id AS quantity_id,
    q.value AS quantity_value,
    qu.id AS quantity_unit_id,
    qu.name AS quantity_unit_name,
    qu.abbreviation AS quantity_unit_abbreviation,
    qu.unit_dimension_code AS quantity_unit_type,
    -- Quantity picked
    (SELECT COALESCE(SUM(plq.value), 0) FROM pick_line pl
        JOIN quantity plq ON plq.id = pl.quantity_id
        WHERE pl.sales_order_line_id = sol.id) AS quantity_picked_value,
    -- Quantity packed
    (SELECT COALESCE(SUM(plq2.value), 0) FROM pick_line pl2
        JOIN quantity plq2 ON plq2.id = pl2.quantity_id
        WHERE pl2.sales_order_line_id = sol.id AND pl2.packed_at IS NOT NULL) AS quantity_packed_value,
    -- Quantity invoiced
    (SELECT COALESCE(SUM(ilq.value), 0) FROM invoice_line il
        JOIN quantity ilq ON ilq.id = il.quantity_id
        WHERE il.sales_order_line_id = sol.id) AS quantity_invoiced_value,
    -- Unit price
    up.id AS unit_price_id,
    up.value AS unit_price_value,
    up_nu.id AS unit_price_numerator_unit_id,
    up_nu.abbreviation AS unit_price_numerator_unit_abbreviation,
    up_du.id AS unit_price_denominator_unit_id,
    up_du.abbreviation AS unit_price_denominator_unit_abbreviation,
    -- Unit cost
    uc.id AS unit_cost_id,
    uc.value AS unit_cost_value,
    uc_nu.id AS unit_cost_numerator_unit_id,
    uc_nu.abbreviation AS unit_cost_numerator_unit_abbreviation,
    uc_du.id AS unit_cost_denominator_unit_id,
    uc_du.abbreviation AS unit_cost_denominator_unit_abbreviation,
    -- Timestamps
    sol.completed_at,
    sol.created_at,
    sol.updated_at
FROM sales_order_line sol
JOIN quantity q ON q.id = sol.quantity_id
JOIN unit qu ON qu.id = q.unit_id
JOIN rate up ON up.id = sol.unit_price_id
JOIN unit up_nu ON up_nu.id = up.numerator_unit_id
JOIN unit up_du ON up_du.id = up.denominator_unit_id
LEFT JOIN rate uc ON uc.id = sol.unit_cost_id
LEFT JOIN unit uc_nu ON uc_nu.id = uc.numerator_unit_id
LEFT JOIN unit uc_du ON uc_du.id = uc.denominator_unit_id
LEFT JOIN item i ON i.id = sol.item_id
WHERE sol.id = sqlc.arg('sales_order_line_id');

-- name: CreateSalesOrderLine :exec
INSERT INTO sales_order_line (
    id, product_sku, product_description, edi_line_item_id,
    line_item_number, product_id, item_id, sales_order_id,
    quantity_id, unit_price_id, unit_cost_id,
    created_at, updated_at
) VALUES (
    sqlc.arg('id'), sqlc.arg('product_sku'), sqlc.narg('product_description'),
    sqlc.narg('edi_line_item_id'), sqlc.arg('line_item_number'),
    sqlc.narg('product_id'), sqlc.narg('item_id'), sqlc.arg('sales_order_id'),
    sqlc.arg('quantity_id'), sqlc.arg('unit_price_id'), sqlc.narg('unit_cost_id'),
    NOW(3), NOW(3)
);

-- name: UpdateSalesOrderLine :exec
UPDATE sales_order_line SET
    product_sku = COALESCE(sqlc.narg('product_sku'), product_sku),
    product_description = COALESCE(sqlc.narg('product_description'), product_description),
    product_id = COALESCE(sqlc.narg('product_id'), product_id),
    item_id = COALESCE(sqlc.narg('item_id'), item_id),
    edi_line_item_id = COALESCE(sqlc.narg('edi_line_item_id'), edi_line_item_id),
    updated_at = NOW(3)
WHERE id = sqlc.arg('id');

-- name: DeleteSalesOrderLine :exec
DELETE FROM sales_order_line WHERE id = sqlc.arg('id');

-- name: IsLineInOrder :one
SELECT EXISTS(
    SELECT 1 FROM sales_order_line sol
    JOIN sales_order so ON so.id = sol.sales_order_id
    WHERE sol.id = sqlc.arg('sales_order_line_id')
    AND sol.sales_order_id = sqlc.arg('sales_order_id')
    AND so.owner_account_id = sqlc.arg('account_id')
) AS `exists`;

-- name: GetNextLineItemNumber :one
SELECT COALESCE(MAX(line_item_number), 0) + 1 AS next_number
FROM sales_order_line
WHERE sales_order_id = sqlc.arg('sales_order_id');

-- name: DeletePickLinesBySalesOrderLine :exec
DELETE FROM pick_line WHERE sales_order_line_id = sqlc.arg('sales_order_line_id');

-- name: DeleteShipmentLinesBySalesOrderLine :exec
DELETE FROM shipment_line WHERE sales_order_line_id = sqlc.arg('sales_order_line_id');

-- name: DeleteInvoiceLinesBySalesOrderLine :exec
DELETE FROM invoice_line WHERE sales_order_line_id = sqlc.arg('sales_order_line_id');

-- name: CreateOrderLineQuantity :exec
INSERT INTO quantity (id, value, unit_id, created_at, updated_at)
VALUES (sqlc.arg('id'), sqlc.arg('value'), sqlc.arg('unit_id'), NOW(3), NOW(3));

-- name: UpdateOrderLineQuantityValue :exec
UPDATE quantity SET value = sqlc.arg('value'), unit_id = COALESCE(sqlc.narg('unit_id'), unit_id), updated_at = NOW(3)
WHERE id = sqlc.arg('id');

-- name: CreateOrderLineRate :exec
INSERT INTO rate (id, value, numerator_unit_id, denominator_unit_id, created_at, updated_at)
VALUES (sqlc.arg('id'), sqlc.arg('value'), sqlc.arg('numerator_unit_id'), sqlc.arg('denominator_unit_id'), NOW(3), NOW(3));

-- name: SetSalesOrderLineUnitCost :exec
UPDATE sales_order_line SET unit_cost_id = sqlc.arg('unit_cost_id'), updated_at = NOW(3)
WHERE id = sqlc.arg('id');

-- name: UpdateOrderLineRateValue :exec
UPDATE rate SET
    value = COALESCE(sqlc.narg('value'), value),
    numerator_unit_id = COALESCE(sqlc.narg('numerator_unit_id'), numerator_unit_id),
    denominator_unit_id = COALESCE(sqlc.narg('denominator_unit_id'), denominator_unit_id),
    updated_at = NOW(3)
WHERE id = sqlc.arg('id');
