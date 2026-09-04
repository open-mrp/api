-- name: ListDeliveriesForward :many
SELECT
    d.id,
    d.number,
    d.delivery_status_code,
    d.accepted_at,
    d.rejected_at,
    d.created_at,
    d.updated_at,
    so.id AS purchase_order_id,
    so.number AS purchase_order_number,
    so.sales_order_status_code AS purchase_order_status,
    COUNT(dl.id) AS line_count
FROM delivery d
JOIN sales_order so ON d.sales_order_id = so.id
LEFT JOIN delivery_line dl ON dl.delivery_id = d.id
WHERE d.account_id = sqlc.arg('account_id')
AND (
    sqlc.narg('search_query') IS NULL
    OR d.number LIKE sqlc.narg('search_query')
    OR so.number LIKE sqlc.narg('search_query')
)
AND (
    sqlc.narg('status') IS NULL
    OR d.delivery_status_code = sqlc.narg('status')
)
AND (
    sqlc.arg('include_item_filter') = false
    OR EXISTS (
        SELECT 1 FROM delivery_line dl2
        JOIN receiving_order_line rol ON dl2.receiving_order_line_id = rol.id
        JOIN sales_order_line sol ON rol.sales_order_line_id = sol.id
        WHERE dl2.delivery_id = d.id
        AND sol.item_id IN (sqlc.slice('item_ids'))
    )
)
AND (
    sqlc.arg('include_supplier_filter') = false
    OR so.seller_account_id IN (sqlc.slice('supplier_ids'))
)
AND (
    sqlc.narg('start_date') IS NULL
    OR d.created_at >= sqlc.narg('start_date')
)
AND (
    sqlc.narg('end_date') IS NULL
    OR d.created_at <= sqlc.narg('end_date')
)
AND (
    sqlc.narg('cursor_created_at') IS NULL
    OR d.created_at < sqlc.narg('cursor_created_at')
    OR (d.created_at = sqlc.narg('cursor_created_at') AND d.id < sqlc.narg('cursor_id'))
)
GROUP BY d.id, d.number, d.delivery_status_code, d.accepted_at, d.rejected_at, d.created_at, d.updated_at, so.id, so.number, so.sales_order_status_code
ORDER BY d.created_at DESC, d.id DESC
LIMIT ?;

-- name: ListDeliveriesBackward :many
SELECT
    d.id,
    d.number,
    d.delivery_status_code,
    d.accepted_at,
    d.rejected_at,
    d.created_at,
    d.updated_at,
    so.id AS purchase_order_id,
    so.number AS purchase_order_number,
    so.sales_order_status_code AS purchase_order_status,
    COUNT(dl.id) AS line_count
FROM delivery d
JOIN sales_order so ON d.sales_order_id = so.id
LEFT JOIN delivery_line dl ON dl.delivery_id = d.id
WHERE d.account_id = sqlc.arg('account_id')
AND (
    sqlc.narg('search_query') IS NULL
    OR d.number LIKE sqlc.narg('search_query')
    OR so.number LIKE sqlc.narg('search_query')
)
AND (
    sqlc.narg('status') IS NULL
    OR d.delivery_status_code = sqlc.narg('status')
)
AND (
    sqlc.arg('include_item_filter') = false
    OR EXISTS (
        SELECT 1 FROM delivery_line dl2
        JOIN receiving_order_line rol ON dl2.receiving_order_line_id = rol.id
        JOIN sales_order_line sol ON rol.sales_order_line_id = sol.id
        WHERE dl2.delivery_id = d.id
        AND sol.item_id IN (sqlc.slice('item_ids'))
    )
)
AND (
    sqlc.arg('include_supplier_filter') = false
    OR so.seller_account_id IN (sqlc.slice('supplier_ids'))
)
AND (
    sqlc.narg('start_date') IS NULL
    OR d.created_at >= sqlc.narg('start_date')
)
AND (
    sqlc.narg('end_date') IS NULL
    OR d.created_at <= sqlc.narg('end_date')
)
AND (
    d.created_at > sqlc.arg('cursor_created_at')
    OR (d.created_at = sqlc.arg('cursor_created_at') AND d.id > sqlc.arg('cursor_id'))
)
GROUP BY d.id, d.number, d.delivery_status_code, d.accepted_at, d.rejected_at, d.created_at, d.updated_at, so.id, so.number, so.sales_order_status_code
ORDER BY d.created_at ASC, d.id ASC
LIMIT ?;

-- name: GetDelivery :one
SELECT
    d.id,
    d.number,
    d.delivery_status_code,
    d.accepted_at,
    d.rejected_at,
    d.created_at,
    d.updated_at,
    so.id AS purchase_order_id,
    so.number AS purchase_order_number,
    so.sales_order_status_code AS purchase_order_status
FROM delivery d
JOIN sales_order so ON d.sales_order_id = so.id
WHERE d.id = sqlc.arg('id')
AND d.account_id = sqlc.arg('account_id');

-- name: ListDeliveryLines :many
SELECT
    dl.id,
    dl.accepted_at,
    dl.rejected_at,
    dl.created_at,
    dl.updated_at,
    q.id AS quantity_id,
    q.value AS quantity_value,
    qu.id AS quantity_unit_id,
    qu.abbreviation AS quantity_unit_abbreviation,
    -- The purchase order line the goods were ordered on, reached through the receiving line this
    -- delivery line was stocked against.
    sol.id AS order_line_id,
    r.id AS unit_cost_id,
    r.value AS unit_cost_value,
    r.numerator_unit_id AS unit_cost_numerator_unit_id,
    r.denominator_unit_id AS unit_cost_denominator_unit_id,
    rnu.abbreviation AS unit_cost_numerator_unit_abbreviation,
    rdu.abbreviation AS unit_cost_denominator_unit_abbreviation,
    r.created_at AS unit_cost_created_at,
    r.updated_at AS unit_cost_updated_at,
    sol.item_id,
    i.sku AS item_sku,
    i.description AS item_description,
    dl.storage_location_id,
    sl.name AS storage_location_name,
    dl.lot_id,
    l.lot_number AS lot_number
FROM delivery_line dl
JOIN quantity q ON dl.quantity_id = q.id
JOIN unit qu ON q.unit_id = qu.id
JOIN rate r ON dl.unit_cost_id = r.id
JOIN unit rnu ON r.numerator_unit_id = rnu.id
JOIN unit rdu ON r.denominator_unit_id = rdu.id
JOIN receiving_order_line rol ON dl.receiving_order_line_id = rol.id
JOIN sales_order_line sol ON rol.sales_order_line_id = sol.id
LEFT JOIN item i ON sol.item_id = i.id
LEFT JOIN storage_location sl ON dl.storage_location_id = sl.id
LEFT JOIN lot l ON dl.lot_id = l.id
WHERE dl.delivery_id = sqlc.arg('delivery_id')
ORDER BY dl.created_at ASC, dl.id ASC;

-- ListReceivingOrderRefsForOrders names the receiving order created for each of the given purchase orders.
--
-- One receiving order exists per issued purchase order, so this is the link a delivery follows to reach the order it was received against.
-- name: ListReceivingOrderRefsForOrders :many
SELECT ro.order_id, ro.id, ro.number,
    CASE WHEN ro.completed_at IS NULL THEN 'open' ELSE 'completed' END AS status
FROM receiving_order ro
WHERE ro.order_id IN (sqlc.slice('order_ids'));
