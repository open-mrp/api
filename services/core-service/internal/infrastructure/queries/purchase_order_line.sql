-- name: GetPurchaseOrderLine :one
SELECT
    sol.id,
    sol.line_item_number,
    sol.product_sku,
    sol.product_description,
    sol.product_id,
    sol.item_id,
    i.sku AS item_sku,
    -- Quantity ordered
    q.id AS quantity_id,
    q.value AS quantity_value,
    qu.id AS quantity_unit_id,
    qu.name AS quantity_unit_name,
    qu.abbreviation AS quantity_unit_abbreviation,
    qu.unit_dimension_code AS quantity_unit_type,
    -- Quantity received
    -- Each receiving line is converted into the ordered unit before it is added: a line can be
    -- received in any unit of the item's group, and the response labels the sum with the ordered unit.
    (SELECT COALESCE(SUM(CASE WHEN rolq.unit_id = qu.id THEN rolq.value
            ELSE rolq.value * rolu.ratio_numerator * qu.ratio_denominator / (rolu.ratio_denominator * qu.ratio_numerator) END), 0)
        FROM receiving_order_line rol
        JOIN quantity rolq ON rolq.id = rol.quantity_id
        JOIN unit rolu ON rolu.id = rolq.unit_id
        WHERE rol.sales_order_line_id = sol.id) AS quantity_received_value,
    -- Unit price
    up.id AS unit_price_id,
    up.value AS unit_price_value,
    up_nu.id AS unit_price_numerator_unit_id,
    up_nu.abbreviation AS unit_price_numerator_unit_abbreviation,
    up_du.id AS unit_price_denominator_unit_id,
    up_du.abbreviation AS unit_price_denominator_unit_abbreviation,
    -- The base ratios of the quantity's unit and of the unit the price is quoted per, which
    -- shared/pricing converts between the way the dashboard's multiplyRate does. Null where the
    -- dashboard would price the line differently: units of different dimensions, a unit with an
    -- offset, or a price in a non-base currency unit.
    CASE WHEN qu.id = up_du.id AND qu.id <> up_nu.id THEN '1'
        WHEN qu.unit_dimension_code = up_du.unit_dimension_code AND qu.unit_dimension_code <> up_nu.unit_dimension_code AND qu.offset_numerator = 0 AND up_du.offset_numerator = 0 AND (qu.is_base_unit = 0 OR qu.ratio_numerator = qu.ratio_denominator) AND (up_du.is_base_unit = 0 OR up_du.ratio_numerator = up_du.ratio_denominator) AND (up_nu.is_base_unit = 1 OR (up_nu.ratio_numerator = up_nu.ratio_denominator AND up_nu.offset_numerator = 0))
        THEN CAST(qu.ratio_numerator AS CHAR) END AS pricing_quantity_ratio_numerator,
    CASE WHEN qu.id = up_du.id AND qu.id <> up_nu.id THEN '1'
        WHEN qu.unit_dimension_code = up_du.unit_dimension_code AND qu.unit_dimension_code <> up_nu.unit_dimension_code AND qu.offset_numerator = 0 AND up_du.offset_numerator = 0 AND (qu.is_base_unit = 0 OR qu.ratio_numerator = qu.ratio_denominator) AND (up_du.is_base_unit = 0 OR up_du.ratio_numerator = up_du.ratio_denominator) AND (up_nu.is_base_unit = 1 OR (up_nu.ratio_numerator = up_nu.ratio_denominator AND up_nu.offset_numerator = 0))
        THEN CAST(qu.ratio_denominator AS CHAR) END AS pricing_quantity_ratio_denominator,
    CASE WHEN qu.id = up_du.id AND qu.id <> up_nu.id THEN '1'
        WHEN qu.unit_dimension_code = up_du.unit_dimension_code AND qu.unit_dimension_code <> up_nu.unit_dimension_code AND qu.offset_numerator = 0 AND up_du.offset_numerator = 0 AND (qu.is_base_unit = 0 OR qu.ratio_numerator = qu.ratio_denominator) AND (up_du.is_base_unit = 0 OR up_du.ratio_numerator = up_du.ratio_denominator) AND (up_nu.is_base_unit = 1 OR (up_nu.ratio_numerator = up_nu.ratio_denominator AND up_nu.offset_numerator = 0))
        THEN CAST(up_du.ratio_numerator AS CHAR) END AS pricing_price_ratio_numerator,
    CASE WHEN qu.id = up_du.id AND qu.id <> up_nu.id THEN '1'
        WHEN qu.unit_dimension_code = up_du.unit_dimension_code AND qu.unit_dimension_code <> up_nu.unit_dimension_code AND qu.offset_numerator = 0 AND up_du.offset_numerator = 0 AND (qu.is_base_unit = 0 OR qu.ratio_numerator = qu.ratio_denominator) AND (up_du.is_base_unit = 0 OR up_du.ratio_numerator = up_du.ratio_denominator) AND (up_nu.is_base_unit = 1 OR (up_nu.ratio_numerator = up_nu.ratio_denominator AND up_nu.offset_numerator = 0))
        THEN CAST(up_du.ratio_denominator AS CHAR) END AS pricing_price_ratio_denominator,
    -- Unit cost
    uc.id AS unit_cost_id,
    uc.value AS unit_cost_value,
    uc_nu.id AS unit_cost_numerator_unit_id,
    uc_nu.abbreviation AS unit_cost_numerator_unit_abbreviation,
    uc_du.id AS unit_cost_denominator_unit_id,
    uc_du.abbreviation AS unit_cost_denominator_unit_abbreviation,
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
WHERE sol.id = sqlc.arg('sales_order_line_id')
AND sol.sales_order_id = sqlc.arg('sales_order_id');

-- name: CreatePurchaseOrderLine :exec
INSERT INTO sales_order_line (
    id, product_sku, product_description,
    line_item_number, product_id, item_id, sales_order_id,
    quantity_id, unit_price_id, unit_cost_id,
    created_at, updated_at
) VALUES (
    sqlc.arg('id'), sqlc.arg('product_sku'), sqlc.narg('product_description'),
    sqlc.arg('line_item_number'),
    -- A line bought as a product is stocked as that product's item.
    sqlc.narg('product_id'),
    COALESCE(sqlc.narg('item_id'), (SELECT p.item_id FROM product p WHERE p.id = sqlc.narg('product_id'))),
    sqlc.arg('sales_order_id'),
    sqlc.arg('quantity_id'), sqlc.arg('unit_price_id'), sqlc.narg('unit_cost_id'),
    NOW(3), NOW(3)
);

-- name: UpdatePurchaseOrderLine :exec
UPDATE sales_order_line sol SET
    product_sku = COALESCE(sqlc.narg('product_sku'), product_sku),
    product_description = COALESCE(sqlc.narg('product_description'), product_description),
    product_id = sqlc.narg('product_id'),
    item_id = COALESCE(sqlc.narg('item_id'), (SELECT p.item_id FROM product p WHERE p.id = sqlc.narg('product_id'))),
    updated_at = NOW(3)
WHERE sol.id = sqlc.arg('id')
AND sol.sales_order_id = sqlc.arg('sales_order_id');

-- name: DeletePurchaseOrderLine :exec
DELETE FROM sales_order_line
WHERE id = sqlc.arg('id')
AND sales_order_id = sqlc.arg('sales_order_id');

-- name: IsLineInPurchaseOrder :one
SELECT EXISTS(
    SELECT 1 FROM sales_order_line
    WHERE id = sqlc.arg('sales_order_line_id')
    AND sales_order_id = sqlc.arg('sales_order_id')
) AS `exists`;

-- name: GetNextPurchaseOrderLineItemNumber :one
SELECT COALESCE(MAX(line_item_number), 0) + 1 AS next_number
FROM sales_order_line
WHERE sales_order_id = sqlc.arg('sales_order_id');

-- name: DeleteReceivingOrderLinesByPurchaseOrderLine :exec
DELETE FROM receiving_order_line WHERE sales_order_line_id = sqlc.arg('sales_order_line_id');

-- name: DeleteQuantitiesByReceivingOrderLinesForLine :exec
DELETE q FROM quantity q
JOIN receiving_order_line rol ON rol.quantity_id = q.id
WHERE rol.sales_order_line_id = sqlc.arg('sales_order_line_id');

-- name: GetPurchaseOrderLineItemUnitGroups :many
-- The unit group each item's category measures it in, for checking the unit a line's quantity is in.
SELECT i.id AS item_id, ic.unit_group_id
FROM item i
JOIN item_category ic ON ic.id = i.item_category_id
WHERE i.id IN (sqlc.slice('item_ids'))
AND i.account_id = sqlc.arg('account_id');
