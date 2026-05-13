-- name: ListShipmentsForward :many
SELECT
    s.id,
    s.number,
    s.bill_of_lading,
    s.note,
    s.master_tracking_number,
    s.shipped_at,
    s.shipment_status_code AS status_code,
    ss.name AS status_name,
    s.sales_order_id,
    so.number AS sales_order_number,
    s.carrier_id,
    cr.name AS carrier_name,
    cr.is_portal_enabled AS carrier_is_portal_enabled,
    s.carrier_option_id,
    co.name AS carrier_option_name,
    co.is_portal_enabled AS service_level_is_portal_enabled,
    co.service_level_token AS service_level_token,
    ar.counterparty_account_id AS customer_id,
    ba.name AS customer_name,
    ar.external_number AS customer_number,
    ar.account_status_code AS customer_status_code,
    ar.commission_status_code AS customer_commission_policy,
    ar.created_at AS customer_created_at,
    ar.updated_at AS customer_updated_at,
    so.created_at AS sales_order_created_at,
    so.updated_at AS sales_order_updated_at,
    s.created_at,
    s.updated_at
FROM shipment s
JOIN shipment_status ss ON ss.code = s.shipment_status_code
JOIN sales_order so ON so.id = s.sales_order_id
JOIN account_relation ar ON ar.owner_account_id = so.owner_account_id
    AND ar.counterparty_account_id = so.buyer_account_id
JOIN account ba ON ba.id = so.buyer_account_id
JOIN carrier cr ON cr.id = s.carrier_id
LEFT JOIN carrier_option co ON co.id = s.carrier_option_id
WHERE s.account_id = sqlc.arg('account_id')
AND (
    sqlc.narg('status_code') IS NULL
    OR s.shipment_status_code = sqlc.narg('status_code')
)
AND (
    sqlc.narg('search_query') IS NULL
    OR s.number LIKE sqlc.narg('search_query')
    OR s.note LIKE sqlc.narg('search_query')
    OR s.bill_of_lading LIKE sqlc.narg('search_query')
    OR s.master_tracking_number LIKE sqlc.narg('search_query')
    OR so.number LIKE sqlc.narg('search_query')
    OR ba.name LIKE sqlc.narg('search_query')
    OR so.customer_po_number LIKE sqlc.narg('search_query')
)
AND (
    sqlc.arg('include_item_filter') = false
    OR EXISTS (
        SELECT 1 FROM shipment_line sl
        JOIN sales_order_line sol ON sol.id = sl.sales_order_line_id
        WHERE sl.shipment_id = s.id
        AND sol.item_id IN (sqlc.slice('item_ids'))
    )
)
AND (
    sqlc.arg('include_customer_filter') = false
    OR so.buyer_account_id IN (sqlc.slice('customer_ids'))
)
AND (
    sqlc.arg('include_product_line_filter') = false
    OR EXISTS (
        SELECT 1 FROM shipment_line sl2
        JOIN sales_order_line sol2 ON sol2.id = sl2.sales_order_line_id
        JOIN product p ON p.id = sol2.product_id
        WHERE sl2.shipment_id = s.id
        AND p.product_line_id IN (sqlc.slice('product_line_ids'))
    )
)
AND (
    sqlc.arg('include_customer_group_filter') = false
    OR ar.account_group_id IN (sqlc.slice('customer_group_ids'))
)
AND (
    sqlc.arg('include_sales_rep_filter') = false
    OR ar.default_sales_rep_id IN (sqlc.slice('sales_rep_ids'))
)
AND (
    sqlc.narg('start_date') IS NULL
    OR s.created_at >= sqlc.narg('start_date')
)
AND (
    sqlc.narg('end_date') IS NULL
    OR s.created_at <= sqlc.narg('end_date')
)
AND (
    sqlc.narg('cursor_created_at') IS NULL
    OR s.created_at < sqlc.narg('cursor_created_at')
    OR (s.created_at = sqlc.narg('cursor_created_at') AND s.id < sqlc.narg('cursor_id'))
)
ORDER BY s.created_at DESC, s.id DESC
LIMIT ?;

-- name: ListShipmentsBackward :many
SELECT
    s.id,
    s.number,
    s.bill_of_lading,
    s.note,
    s.master_tracking_number,
    s.shipped_at,
    s.shipment_status_code AS status_code,
    ss.name AS status_name,
    s.sales_order_id,
    so.number AS sales_order_number,
    s.carrier_id,
    cr.name AS carrier_name,
    cr.is_portal_enabled AS carrier_is_portal_enabled,
    s.carrier_option_id,
    co.name AS carrier_option_name,
    co.is_portal_enabled AS service_level_is_portal_enabled,
    co.service_level_token AS service_level_token,
    ar.counterparty_account_id AS customer_id,
    ba.name AS customer_name,
    ar.external_number AS customer_number,
    ar.account_status_code AS customer_status_code,
    ar.commission_status_code AS customer_commission_policy,
    ar.created_at AS customer_created_at,
    ar.updated_at AS customer_updated_at,
    so.created_at AS sales_order_created_at,
    so.updated_at AS sales_order_updated_at,
    s.created_at,
    s.updated_at
FROM shipment s
JOIN shipment_status ss ON ss.code = s.shipment_status_code
JOIN sales_order so ON so.id = s.sales_order_id
JOIN account_relation ar ON ar.owner_account_id = so.owner_account_id
    AND ar.counterparty_account_id = so.buyer_account_id
JOIN account ba ON ba.id = so.buyer_account_id
JOIN carrier cr ON cr.id = s.carrier_id
LEFT JOIN carrier_option co ON co.id = s.carrier_option_id
WHERE s.account_id = sqlc.arg('account_id')
AND (
    sqlc.narg('status_code') IS NULL
    OR s.shipment_status_code = sqlc.narg('status_code')
)
AND (
    sqlc.narg('search_query') IS NULL
    OR s.number LIKE sqlc.narg('search_query')
    OR s.note LIKE sqlc.narg('search_query')
    OR s.bill_of_lading LIKE sqlc.narg('search_query')
    OR s.master_tracking_number LIKE sqlc.narg('search_query')
    OR so.number LIKE sqlc.narg('search_query')
    OR ba.name LIKE sqlc.narg('search_query')
    OR so.customer_po_number LIKE sqlc.narg('search_query')
)
AND (
    sqlc.arg('include_item_filter') = false
    OR EXISTS (
        SELECT 1 FROM shipment_line sl
        JOIN sales_order_line sol ON sol.id = sl.sales_order_line_id
        WHERE sl.shipment_id = s.id
        AND sol.item_id IN (sqlc.slice('item_ids'))
    )
)
AND (
    sqlc.arg('include_customer_filter') = false
    OR so.buyer_account_id IN (sqlc.slice('customer_ids'))
)
AND (
    sqlc.arg('include_product_line_filter') = false
    OR EXISTS (
        SELECT 1 FROM shipment_line sl2
        JOIN sales_order_line sol2 ON sol2.id = sl2.sales_order_line_id
        JOIN product p ON p.id = sol2.product_id
        WHERE sl2.shipment_id = s.id
        AND p.product_line_id IN (sqlc.slice('product_line_ids'))
    )
)
AND (
    sqlc.arg('include_customer_group_filter') = false
    OR ar.account_group_id IN (sqlc.slice('customer_group_ids'))
)
AND (
    sqlc.arg('include_sales_rep_filter') = false
    OR ar.default_sales_rep_id IN (sqlc.slice('sales_rep_ids'))
)
AND (
    sqlc.narg('start_date') IS NULL
    OR s.created_at >= sqlc.narg('start_date')
)
AND (
    sqlc.narg('end_date') IS NULL
    OR s.created_at <= sqlc.narg('end_date')
)
AND (
    s.created_at > sqlc.arg('cursor_created_at')
    OR (s.created_at = sqlc.arg('cursor_created_at') AND s.id > sqlc.arg('cursor_id'))
)
ORDER BY s.created_at ASC, s.id ASC
LIMIT ?;

-- name: GetShipment :one
SELECT
    s.id,
    s.number,
    s.bill_of_lading,
    s.note,
    s.master_tracking_number,
    s.shipped_at,
    s.shipment_status_code AS status_code,
    ss.name AS status_name,
    s.sales_order_id,
    so.number AS sales_order_number,
    so.customer_po_number,
    COALESCE(so.carrier_billing_type, ar.carrier_billing_type) AS carrier_billing_type,
    COALESCE(so.carrier_billing_account, ar.carrier_billing_account) AS carrier_billing_account,
    s.carrier_id,
    cr.name AS carrier_name,
    cr.is_portal_enabled AS carrier_is_portal_enabled,
    cr.created_at AS carrier_created_at,
    cr.updated_at AS carrier_updated_at,
    s.carrier_option_id,
    co.name AS carrier_option_name,
    co.is_portal_enabled AS service_level_is_portal_enabled,
    co.service_level_token AS service_level_token,
    co.created_at AS service_level_created_at,
    co.updated_at AS service_level_updated_at,
    s.shipping_address_id,
    addr.name AS shipping_address_name,
    s.shipped_by_id,
    shipped_by_user.name AS shipped_by_name,
    s.invoice_id,
    inv.number AS invoice_number,
    ar.counterparty_account_id AS customer_id,
    ba.name AS customer_name,
    ar.external_number AS customer_number,
    ar.account_status_code AS customer_status_code,
    ar.commission_status_code AS customer_commission_policy,
    ar.created_at AS customer_created_at,
    ar.updated_at AS customer_updated_at,
    p.id AS pick_id,
    p.number AS pick_number,
    p.created_at AS pick_created_at,
    p.updated_at AS pick_updated_at,
    billing_geo.country AS billing_address_country,
    billing_geo.postal_code AS billing_address_zip,
    s.account_id,
    s.created_at,
    s.updated_at,
    so.created_at AS sales_order_created_at,
    so.updated_at AS sales_order_updated_at,
    addr.created_at AS shipping_address_created_at,
    addr.updated_at AS shipping_address_updated_at,
    shipped_by_au.status_code AS shipped_by_status_code,
    shipped_by_au.created_at AS shipped_by_created_at,
    shipped_by_au.updated_at AS shipped_by_updated_at,
    inv.created_at AS invoice_created_at,
    inv.updated_at AS invoice_updated_at
FROM shipment s
JOIN shipment_status ss ON ss.code = s.shipment_status_code
JOIN sales_order so ON so.id = s.sales_order_id
JOIN account_relation ar ON ar.owner_account_id = so.owner_account_id
    AND ar.counterparty_account_id = so.buyer_account_id
JOIN account ba ON ba.id = so.buyer_account_id
JOIN carrier cr ON cr.id = s.carrier_id
LEFT JOIN carrier_option co ON co.id = s.carrier_option_id
LEFT JOIN address addr ON addr.id = s.shipping_address_id
LEFT JOIN account_user shipped_by_au ON shipped_by_au.id = s.shipped_by_id
LEFT JOIN user shipped_by_user ON shipped_by_user.id = shipped_by_au.user_id
LEFT JOIN invoice inv ON inv.id = s.invoice_id
LEFT JOIN pick p ON p.sales_order_id = so.id
LEFT JOIN address billing_addr ON billing_addr.id = so.billing_address_id
LEFT JOIN geolocation billing_geo ON billing_geo.id = billing_addr.geolocation_id
WHERE s.id = sqlc.arg('id')
AND s.account_id = sqlc.arg('account_id');

-- name: UpdateShipment :execresult
UPDATE shipment SET
    note = COALESCE(sqlc.narg('note'), note),
    number = COALESCE(sqlc.narg('number'), number),
    master_tracking_number = COALESCE(sqlc.narg('master_tracking_number'), master_tracking_number),
    carrier_id = COALESCE(sqlc.narg('carrier_id'), carrier_id),
    carrier_option_id = sqlc.narg('carrier_option_id'),
    updated_at = NOW(3)
WHERE id = sqlc.arg('id')
AND account_id = sqlc.arg('account_id');

-- name: DeleteShipment :exec
DELETE FROM shipment
WHERE id = sqlc.arg('id')
AND account_id = sqlc.arg('account_id');

-- name: MarkShipmentShipped :exec
UPDATE shipment SET
    shipment_status_code = 'shipped',
    shipped_at = NOW(3),
    shipped_by_id = sqlc.arg('shipped_by_id'),
    updated_at = NOW(3)
WHERE id = sqlc.arg('id')
AND account_id = sqlc.arg('account_id');

-- name: MarkShipmentVoided :exec
UPDATE shipment SET
    shipment_status_code = 'packed',
    shipped_at = NULL,
    shipped_by_id = NULL,
    invoice_id = NULL,
    master_tracking_number = NULL,
    updated_at = NOW(3)
WHERE id = sqlc.arg('id')
AND account_id = sqlc.arg('account_id');

-- name: FindInvoiceIDByShipment :one
SELECT inv.id
FROM invoice inv
WHERE inv.id = (
    SELECT s.invoice_id FROM shipment s
    WHERE s.id = sqlc.arg('shipment_id')
    AND s.account_id = sqlc.arg('account_id')
)
AND inv.account_id = sqlc.arg('account_id');

-- name: CheckShipmentInAccount :one
SELECT EXISTS(
    SELECT 1 FROM shipment
    WHERE id = sqlc.arg('id')
    AND account_id = sqlc.arg('account_id')
) AS `exists`;
