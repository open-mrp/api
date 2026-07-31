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
    -- Product type
    p.product_type_code AS product_type_code,
    -- Timestamps
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
LEFT JOIN product p ON p.id = sol.product_id
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

-- name: GetFirstSystemLineNumber :one
-- Lowest line_item_number occupied by a credit or freight (shipping) line on the
-- order, or 0 when the order has none. These "system" lines are kept at the
-- bottom of the line list, so a newly added product line slots in just above them.
SELECT CAST(COALESCE(MIN(sol.line_item_number), 0) AS SIGNED) AS first_system_number
FROM sales_order_line sol
JOIN product p ON p.id = sol.product_id
WHERE sol.sales_order_id = sqlc.arg('sales_order_id')
AND p.product_type_code IN ('shipping', 'credit');

-- name: ShiftSalesOrderLineNumbersAtOrAbove :exec
-- Pushes every line at or below the given position down by one to open a slot for
-- a line being inserted above the credit/freight block.
UPDATE sales_order_line
SET line_item_number = line_item_number + 1, updated_at = NOW(3)
WHERE sales_order_id = sqlc.arg('sales_order_id')
AND line_item_number >= sqlc.arg('from_line_item_number');

-- name: IsSystemLineProduct :one
-- Whether the product is a credit or freight (shipping) system product, which are
-- always kept at the bottom of the line list rather than slotted above.
SELECT EXISTS(
    SELECT 1 FROM product
    WHERE id = sqlc.arg('product_id')
    AND product_type_code IN ('shipping', 'credit')
) AS is_system;

-- name: SetSalesOrderLineItemNumber :exec
UPDATE sales_order_line
SET line_item_number = sqlc.arg('line_item_number'), updated_at = NOW(3)
WHERE id = sqlc.arg('id');

-- name: GetSalesOrderLineOrder :many
-- Lists the order's lines in current display order along with whether each is a
-- credit/freight system line, for reorder validation and re-sequencing. Lines
-- with no product (custom lines) are treated as regular product lines.
SELECT
    sol.id,
    sol.line_item_number,
    p.product_type_code
FROM sales_order_line sol
LEFT JOIN product p ON p.id = sol.product_id
WHERE sol.sales_order_id = sqlc.arg('sales_order_id')
ORDER BY sol.line_item_number ASC;

-- name: HasShipmentAgainstOrderLine :one
-- Whether the order line is part of any shipment (packed or shipped). A line packed
-- into a shipment must not be deletable, so this does NOT filter on shipped_at.
SELECT EXISTS(
    SELECT 1 FROM shipment_line sl
    WHERE sl.sales_order_line_id = sqlc.arg('sales_order_line_id')
) AS has_shipment;

-- name: DeletePickLinesBySalesOrderLine :exec
DELETE FROM pick_line WHERE sales_order_line_id = sqlc.arg('sales_order_line_id');

-- name: DeleteShipmentLinesBySalesOrderLine :exec
DELETE FROM shipment_line WHERE sales_order_line_id = sqlc.arg('sales_order_line_id');

-- name: DeleteInvoiceLinesBySalesOrderLine :exec
DELETE FROM invoice_line WHERE sales_order_line_id = sqlc.arg('sales_order_line_id');

-- name: SyncInvoiceLineQuantitiesBySalesOrderLine :execrows
-- Each invoice line snapshots its own quantity row at creation time; when the order
-- line's quantity (value or unit) changes, push the new values into the invoice lines
-- that were mirroring the order line so order and invoice never drift apart.
-- Legacy semantics (dashboard invoice.repo.ts): a line billed either the full ordered
-- quantity (order-created invoices, non-shipped items) or a partial shipped snapshot.
-- Only the former follow the order line, so the sync is gated on the invoice line still
-- holding the order line's pre-update value; partial snapshots are left untouched.
UPDATE quantity SET value = sqlc.arg('value'), unit_id = sqlc.arg('unit_id'), updated_at = NOW(3)
WHERE id IN (SELECT quantity_id FROM invoice_line WHERE sales_order_line_id = sqlc.arg('sales_order_line_id'))
  AND value = sqlc.arg('previous_value');

-- name: TouchInvoiceLinesBySalesOrderLine :exec
-- Companion to SyncInvoiceLineQuantitiesBySalesOrderLine: bump updated_at on the invoice
-- lines whose quantity row is about to be synced. Must run BEFORE the quantity update in
-- the same transaction, while the rows still hold the pre-update value.
UPDATE invoice_line SET updated_at = NOW(3)
WHERE sales_order_line_id = sqlc.arg('sales_order_line_id')
  AND EXISTS (
    SELECT 1 FROM quantity q
    WHERE q.id = invoice_line.quantity_id AND q.value = sqlc.arg('previous_value')
  );

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
