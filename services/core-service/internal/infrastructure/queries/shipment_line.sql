-- name: ListShipmentLinesForward :many
SELECT
    sl.id,
    sl.shipment_id,
    sl.sales_order_line_id,
    sol.product_sku,
    sol.product_description,
    sol.item_id AS order_line_item_id,
    -- Quantity
    q.id AS quantity_id,
    q.value AS quantity_value,
    u.id AS quantity_unit_id,
    u.name AS quantity_unit_name,
    u.abbreviation AS quantity_unit_abbreviation,
    u.unit_dimension_code AS quantity_unit_type,
    -- Timestamps
    sl.created_at,
    sl.updated_at
FROM shipment_line sl
JOIN sales_order_line sol ON sol.id = sl.sales_order_line_id
JOIN quantity q ON q.id = sl.quantity_id
JOIN unit u ON u.id = q.unit_id
WHERE sl.shipment_id = sqlc.arg('shipment_id')
AND (
    sqlc.narg('search') IS NULL
    OR sl.id LIKE CONCAT('%', sqlc.narg('search'), '%')
    OR sol.product_sku LIKE CONCAT('%', sqlc.narg('search'), '%')
    OR COALESCE(sol.product_description, '') LIKE CONCAT('%', sqlc.narg('search'), '%')
)
AND (
    sqlc.narg('cursor_created_at') IS NULL
    OR sl.created_at < sqlc.narg('cursor_created_at')
    OR (sl.created_at = sqlc.narg('cursor_created_at') AND sl.id < sqlc.narg('cursor_id'))
)
ORDER BY sl.created_at DESC, sl.id DESC
LIMIT ?;

-- name: ListShipmentLinesBackward :many
SELECT
    sl.id,
    sl.shipment_id,
    sl.sales_order_line_id,
    sol.product_sku,
    sol.product_description,
    sol.item_id AS order_line_item_id,
    -- Quantity
    q.id AS quantity_id,
    q.value AS quantity_value,
    u.id AS quantity_unit_id,
    u.name AS quantity_unit_name,
    u.abbreviation AS quantity_unit_abbreviation,
    u.unit_dimension_code AS quantity_unit_type,
    -- Timestamps
    sl.created_at,
    sl.updated_at
FROM shipment_line sl
JOIN sales_order_line sol ON sol.id = sl.sales_order_line_id
JOIN quantity q ON q.id = sl.quantity_id
JOIN unit u ON u.id = q.unit_id
WHERE sl.shipment_id = sqlc.arg('shipment_id')
AND (
    sqlc.narg('search') IS NULL
    OR sl.id LIKE CONCAT('%', sqlc.narg('search'), '%')
    OR sol.product_sku LIKE CONCAT('%', sqlc.narg('search'), '%')
    OR COALESCE(sol.product_description, '') LIKE CONCAT('%', sqlc.narg('search'), '%')
)
AND (
    sl.created_at > sqlc.arg('cursor_created_at')
    OR (sl.created_at = sqlc.arg('cursor_created_at') AND sl.id > sqlc.arg('cursor_id'))
)
ORDER BY sl.created_at ASC, sl.id ASC
LIMIT ?;

-- name: GetShipmentLine :one
SELECT
    sl.id,
    sl.shipment_id,
    sl.sales_order_line_id,
    sol.product_sku,
    sol.product_description,
    sol.item_id AS order_line_item_id,
    -- Quantity
    q.id AS quantity_id,
    q.value AS quantity_value,
    u.id AS quantity_unit_id,
    u.name AS quantity_unit_name,
    u.abbreviation AS quantity_unit_abbreviation,
    u.unit_dimension_code AS quantity_unit_type,
    -- Timestamps
    sl.created_at,
    sl.updated_at
FROM shipment_line sl
JOIN sales_order_line sol ON sol.id = sl.sales_order_line_id
JOIN quantity q ON q.id = sl.quantity_id
JOIN unit u ON u.id = q.unit_id
WHERE sl.id = sqlc.arg('shipment_line_id');

-- name: CreateShipmentLine :exec
INSERT INTO shipment_line (
    id, shipment_id, sales_order_line_id, quantity_id,
    created_at, updated_at
) VALUES (
    sqlc.arg('id'), sqlc.arg('shipment_id'), sqlc.arg('sales_order_line_id'),
    sqlc.arg('quantity_id'), NOW(3), NOW(3)
);

-- name: CreateShipmentLineQuantity :exec
INSERT INTO quantity (id, value, unit_id, created_at, updated_at)
VALUES (sqlc.arg('id'), sqlc.arg('value'), sqlc.arg('unit_id'), NOW(3), NOW(3));

-- name: UpdateShipmentLine :execresult
UPDATE quantity SET
    value = COALESCE(sqlc.narg('value'), value),
    unit_id = sqlc.narg('unit_id'),
    updated_at = NOW(3)
WHERE quantity.id = (
    SELECT sl.quantity_id FROM shipment_line sl
    WHERE sl.id = sqlc.arg('shipment_line_id')
);

-- name: DeleteShipmentLine :exec
DELETE FROM shipment_line WHERE id = sqlc.arg('id');

-- name: DeleteShipmentLineQuantity :exec
DELETE FROM quantity WHERE quantity.id = (
    SELECT sl.quantity_id FROM shipment_line sl
    WHERE sl.id = sqlc.arg('shipment_line_id')
);

-- name: IsShipmentLineInShipment :one
SELECT EXISTS(
    SELECT 1 FROM shipment_line
    WHERE id = sqlc.arg('shipment_line_id')
    AND shipment_id = sqlc.arg('shipment_id')
) AS `exists`;

-- name: ListShipmentLinesByShipment :many
SELECT
    sl.id,
    sl.shipment_id,
    sl.sales_order_line_id,
    sol.product_sku,
    sol.product_description,
    sol.item_id AS order_line_item_id,
    -- Quantity
    q.id AS quantity_id,
    q.value AS quantity_value,
    u.id AS quantity_unit_id,
    u.name AS quantity_unit_name,
    u.abbreviation AS quantity_unit_abbreviation,
    u.unit_dimension_code AS quantity_unit_type,
    -- Timestamps
    sl.created_at,
    sl.updated_at
FROM shipment_line sl
JOIN sales_order_line sol ON sol.id = sl.sales_order_line_id
JOIN quantity q ON q.id = sl.quantity_id
JOIN unit u ON u.id = q.unit_id
WHERE sl.shipment_id = sqlc.arg('shipment_id')
ORDER BY sl.created_at DESC, sl.id DESC;

-- name: DeleteShipmentLinesByShipment :exec
DELETE FROM shipment_line WHERE shipment_id = sqlc.arg('shipment_id');

-- name: DeleteShipmentLineQuantitiesByShipment :exec
DELETE FROM quantity WHERE id IN (
    SELECT quantity_id FROM shipment_line
    WHERE shipment_id = sqlc.arg('shipment_id')
);
