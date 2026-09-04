-- name: ListPurchaseOrdersForward :many
SELECT
    so.id,
    so.number,
    so.sales_order_status_code AS status_code,
    sos.name AS status_name,
    so.sales_order_type_code AS type_code,
    sot.name AS type_name,
    so.seller_account_id AS supplier_id,
    sa.name AS supplier_name,
    ar.external_number AS supplier_number,
    so.is_acknowledgment_sent,
    so.priority_code,
    pr.name AS priority_name,
    pr.id AS priority_id,
    so.issued_at,
    so.completed_at,
    so.created_at,
    so.updated_at,
    (SELECT COUNT(*) FROM sales_order_line sol_count WHERE sol_count.sales_order_id = so.id) AS line_count
FROM sales_order so
JOIN account_relation ar ON ar.owner_account_id = so.owner_account_id
    AND ar.counterparty_account_id = so.seller_account_id
JOIN account sa ON sa.id = so.seller_account_id
JOIN sales_order_status sos ON sos.code = so.sales_order_status_code
JOIN sales_order_type sot ON sot.code = so.sales_order_type_code
JOIN priority pr ON pr.code = so.priority_code
WHERE so.owner_account_id = sqlc.arg('account_id')
AND so.sales_order_type_code = 'purchase_order'
AND (
    sqlc.narg('search_query') IS NULL
    OR so.number LIKE sqlc.narg('search_query')
    OR sa.name LIKE sqlc.narg('search_query')
    OR so.customer_po_number LIKE sqlc.narg('search_query')
)
AND (
    sqlc.arg('include_status_filter') = false
    OR so.sales_order_status_code IN (sqlc.slice('status_codes'))
)
AND (
    sqlc.arg('include_item_filter') = false
    OR EXISTS (
        SELECT 1 FROM sales_order_line sol2
        WHERE sol2.sales_order_id = so.id
        AND sol2.item_id IN (sqlc.slice('item_ids'))
    )
)
AND (
    sqlc.arg('include_supplier_filter') = false
    OR so.seller_account_id IN (sqlc.slice('supplier_ids'))
)
AND (
    sqlc.narg('start_date') IS NULL
    OR so.created_at >= sqlc.narg('start_date')
)
AND (
    sqlc.narg('end_date') IS NULL
    OR so.created_at <= sqlc.narg('end_date')
)
AND (
    sqlc.narg('cursor_created_at') IS NULL
    OR so.created_at < sqlc.narg('cursor_created_at')
    OR (so.created_at = sqlc.narg('cursor_created_at') AND so.id < sqlc.narg('cursor_id'))
)
ORDER BY so.created_at DESC, so.id DESC
LIMIT ?;

-- name: ListPurchaseOrdersBackward :many
SELECT
    so.id,
    so.number,
    so.sales_order_status_code AS status_code,
    sos.name AS status_name,
    so.sales_order_type_code AS type_code,
    sot.name AS type_name,
    so.seller_account_id AS supplier_id,
    sa.name AS supplier_name,
    ar.external_number AS supplier_number,
    so.is_acknowledgment_sent,
    so.priority_code,
    pr.name AS priority_name,
    pr.id AS priority_id,
    so.issued_at,
    so.completed_at,
    so.created_at,
    so.updated_at,
    (SELECT COUNT(*) FROM sales_order_line sol_count WHERE sol_count.sales_order_id = so.id) AS line_count
FROM sales_order so
JOIN account_relation ar ON ar.owner_account_id = so.owner_account_id
    AND ar.counterparty_account_id = so.seller_account_id
JOIN account sa ON sa.id = so.seller_account_id
JOIN sales_order_status sos ON sos.code = so.sales_order_status_code
JOIN sales_order_type sot ON sot.code = so.sales_order_type_code
JOIN priority pr ON pr.code = so.priority_code
WHERE so.owner_account_id = sqlc.arg('account_id')
AND so.sales_order_type_code = 'purchase_order'
AND (
    sqlc.narg('search_query') IS NULL
    OR so.number LIKE sqlc.narg('search_query')
    OR sa.name LIKE sqlc.narg('search_query')
    OR so.customer_po_number LIKE sqlc.narg('search_query')
)
AND (
    sqlc.arg('include_status_filter') = false
    OR so.sales_order_status_code IN (sqlc.slice('status_codes'))
)
AND (
    sqlc.arg('include_item_filter') = false
    OR EXISTS (
        SELECT 1 FROM sales_order_line sol2
        WHERE sol2.sales_order_id = so.id
        AND sol2.item_id IN (sqlc.slice('item_ids'))
    )
)
AND (
    sqlc.arg('include_supplier_filter') = false
    OR so.seller_account_id IN (sqlc.slice('supplier_ids'))
)
AND (
    sqlc.narg('start_date') IS NULL
    OR so.created_at >= sqlc.narg('start_date')
)
AND (
    sqlc.narg('end_date') IS NULL
    OR so.created_at <= sqlc.narg('end_date')
)
AND (
    so.created_at > sqlc.arg('cursor_created_at')
    OR (so.created_at = sqlc.arg('cursor_created_at') AND so.id > sqlc.arg('cursor_id'))
)
ORDER BY so.created_at ASC, so.id ASC
LIMIT ?;

-- name: GetPurchaseOrder :one
SELECT
    so.id,
    so.number,
    so.note,
    so.is_acknowledgment_sent,
    so.billing_address_id,
    so.shipping_address_id,
    so.carrier_id,
    so.carrier_option_id,
    so.carrier_billing_type,
    so.carrier_billing_account,
    so.priority_code,
    so.shipping_term_id,
    so.sales_order_status_code,
    so.sales_order_type_code,
    so.payment_term_id,
    so.buyer_account_id,
    so.seller_account_id,
    so.owner_account_id,
    so.issued_at,
    so.completed_at,
    so.promised_at,
    so.created_at,
    so.updated_at,
    -- Supplier
    sa.name AS supplier_name,
    ar.external_number AS supplier_number,
    -- Status
    sos.name AS status_name,
    -- Type
    sot.name AS type_name,
    -- Priority
    pr.name AS priority_name,
    pr.id AS priority_id,
    -- Bill-to address
    bill_addr.name AS bill_to_name,
    bill_addr.is_drop_ship AS bill_to_is_drop_ship,
    bill_addr.created_at AS bill_to_created_at,
    bill_addr.updated_at AS bill_to_updated_at,
    bill_geo.street_line_1 AS bill_to_street_line_1,
    bill_geo.street_line_2 AS bill_to_street_line_2,
    bill_geo.locality AS bill_to_locality,
    bill_geo.state AS bill_to_state,
    bill_geo.postal_code AS bill_to_postal_code,
    bill_geo.country AS bill_to_country,
    bill_addr.phone AS bill_to_phone,
    bill_addr.email AS bill_to_email,
    -- Ship-to address
    ship_addr.name AS ship_to_name,
    ship_addr.is_drop_ship AS ship_to_is_drop_ship,
    ship_addr.created_at AS ship_to_created_at,
    ship_addr.updated_at AS ship_to_updated_at,
    ship_geo.street_line_1 AS ship_to_street_line_1,
    ship_geo.street_line_2 AS ship_to_street_line_2,
    ship_geo.locality AS ship_to_locality,
    ship_geo.state AS ship_to_state,
    ship_geo.postal_code AS ship_to_postal_code,
    ship_geo.country AS ship_to_country,
    ship_addr.phone AS ship_to_phone,
    ship_addr.email AS ship_to_email,
    -- Carrier
    cr.name AS carrier_name,
    cr.is_portal_enabled AS carrier_is_portal_enabled,
    cr.created_at AS carrier_created_at,
    cr.updated_at AS carrier_updated_at,
    co.name AS carrier_option_name,
    co.is_portal_enabled AS service_level_is_portal_enabled,
    co.service_level_token AS service_level_token,
    co.created_at AS service_level_created_at,
    co.updated_at AS service_level_updated_at,
    -- Payment term
    pt.name AS payment_term_name,
    pt.is_active AS payment_term_is_active,
    pt.created_at AS payment_term_created_at,
    pt.updated_at AS payment_term_updated_at,
    -- Shipping term
    st.name AS shipping_term_name,
    st.is_freight_exempt AS shipping_term_is_freight_exempt,
    st.is_carrier_rate AS shipping_term_is_carrier_rate,
    st.created_at AS shipping_term_created_at,
    st.updated_at AS shipping_term_updated_at,
    -- Receiving order
    ro.id AS receiving_order_id,
    ro.number AS receiving_order_number,
    -- Empty rather than NULL for an order with no receiving order, so the column is a plain string:
    -- the gateway turns an empty status into an absent one.
    CASE WHEN ro.id IS NULL THEN ''
         WHEN ro.completed_at IS NULL THEN 'open'
         ELSE 'completed' END AS receiving_order_status
FROM sales_order so
JOIN account_relation ar ON ar.owner_account_id = so.owner_account_id
    AND ar.counterparty_account_id = so.seller_account_id
JOIN account sa ON sa.id = so.seller_account_id
JOIN sales_order_status sos ON sos.code = so.sales_order_status_code
JOIN sales_order_type sot ON sot.code = so.sales_order_type_code
JOIN priority pr ON pr.code = so.priority_code
LEFT JOIN address bill_addr ON bill_addr.id = so.billing_address_id
LEFT JOIN geolocation bill_geo ON bill_geo.id = bill_addr.geolocation_id
LEFT JOIN address ship_addr ON ship_addr.id = so.shipping_address_id
LEFT JOIN geolocation ship_geo ON ship_geo.id = ship_addr.geolocation_id
LEFT JOIN carrier cr ON cr.id = so.carrier_id
LEFT JOIN carrier_option co ON co.id = so.carrier_option_id
LEFT JOIN payment_term pt ON pt.id = so.payment_term_id
LEFT JOIN shipping_term st ON st.id = so.shipping_term_id
LEFT JOIN receiving_order ro ON ro.order_id = so.id
WHERE so.id = sqlc.arg('sales_order_id')
AND so.owner_account_id = sqlc.arg('account_id')
AND so.sales_order_type_code = 'purchase_order';

-- name: GetPurchaseOrderLines :many
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
    -- Quantity received
    (SELECT COALESCE(SUM(rolq.value), 0) FROM receiving_order_line rol
        JOIN quantity rolq ON rolq.id = rol.quantity_id
        WHERE rol.sales_order_line_id = sol.id) AS quantity_received_value,
    -- Unit price
    up.id AS unit_price_id,
    up.value AS unit_price_value,
    up_nu.id AS unit_price_numerator_unit_id,
    up_nu.abbreviation AS unit_price_numerator_unit_abbreviation,
    up_du.id AS unit_price_denominator_unit_id,
    up_du.abbreviation AS unit_price_denominator_unit_abbreviation,
    up.created_at AS unit_price_created_at,
    up.updated_at AS unit_price_updated_at,
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
WHERE sol.sales_order_id = sqlc.arg('sales_order_id')
ORDER BY sol.line_item_number ASC;

-- name: GetPurchaseOrderLinesByIDs :many
-- Fetches purchase order lines by their own ids, for a receiving or delivery line that names the
-- line it was raised from. Scoped through the order it belongs to: a line is only visible to the
-- account that owns its purchase order.
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
    -- Quantity received
    (SELECT COALESCE(SUM(rolq.value), 0) FROM receiving_order_line rol
        JOIN quantity rolq ON rolq.id = rol.quantity_id
        WHERE rol.sales_order_line_id = sol.id) AS quantity_received_value,
    -- Unit price
    up.id AS unit_price_id,
    up.value AS unit_price_value,
    up_nu.id AS unit_price_numerator_unit_id,
    up_nu.abbreviation AS unit_price_numerator_unit_abbreviation,
    up_du.id AS unit_price_denominator_unit_id,
    up_du.abbreviation AS unit_price_denominator_unit_abbreviation,
    up.created_at AS unit_price_created_at,
    up.updated_at AS unit_price_updated_at,
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
JOIN sales_order so ON so.id = sol.sales_order_id
JOIN quantity q ON q.id = sol.quantity_id
JOIN unit qu ON qu.id = q.unit_id
JOIN rate up ON up.id = sol.unit_price_id
JOIN unit up_nu ON up_nu.id = up.numerator_unit_id
JOIN unit up_du ON up_du.id = up.denominator_unit_id
LEFT JOIN rate uc ON uc.id = sol.unit_cost_id
LEFT JOIN unit uc_nu ON uc_nu.id = uc.numerator_unit_id
LEFT JOIN unit uc_du ON uc_du.id = uc.denominator_unit_id
LEFT JOIN item i ON i.id = sol.item_id
WHERE sol.id IN (sqlc.slice('ids'))
  AND so.owner_account_id = sqlc.arg('account_id')
  AND so.sales_order_type_code = 'purchase_order'
ORDER BY sol.line_item_number ASC;


-- name: CreatePurchaseOrder :exec
INSERT INTO sales_order (
    id, number, note, is_acknowledgment_sent,
    billing_address_id, shipping_address_id,
    carrier_id, carrier_option_id, carrier_billing_type, carrier_billing_account,
    priority_code, shipping_term_id,
    sales_order_status_code, sales_order_type_code,
    payment_term_id,
    buyer_account_id, seller_account_id, owner_account_id,
    created_at, updated_at
) VALUES (
    sqlc.arg('id'), sqlc.arg('number'), sqlc.narg('note'), false,
    sqlc.arg('billing_address_id'), sqlc.arg('shipping_address_id'),
    sqlc.narg('carrier_id'), sqlc.narg('carrier_option_id'), sqlc.narg('carrier_billing_type'), sqlc.narg('carrier_billing_account'),
    sqlc.arg('priority_code'), sqlc.narg('shipping_term_id'),
    sqlc.arg('sales_order_status_code'), 'purchase_order',
    sqlc.narg('payment_term_id'),
    sqlc.arg('buyer_account_id'), sqlc.arg('seller_account_id'), sqlc.arg('owner_account_id'),
    NOW(3), NOW(3)
);

-- name: UpdatePurchaseOrder :exec
UPDATE sales_order SET
    note = COALESCE(sqlc.narg('note'), note),
    number = COALESCE(sqlc.narg('number'), number),
    priority_code = COALESCE(sqlc.narg('priority_code'), priority_code),
    billing_address_id = sqlc.narg('billing_address_id'),
    shipping_address_id = sqlc.narg('shipping_address_id'),
    promised_at = COALESCE(sqlc.narg('promised_at'), promised_at),
    updated_at = NOW(3)
WHERE id = sqlc.arg('id')
AND owner_account_id = sqlc.arg('account_id')
AND sales_order_type_code = 'purchase_order'
AND buyer_account_id = sqlc.arg('account_id');

-- name: UpdatePurchaseOrderStatus :exec
UPDATE sales_order SET
    sales_order_status_code = sqlc.arg('status_code'),
    issued_at = sqlc.narg('issued_at'),
    completed_at = sqlc.narg('completed_at'),
    updated_at = NOW(3)
WHERE id = sqlc.arg('id')
AND owner_account_id = sqlc.arg('account_id');

-- name: DeletePurchaseOrder :exec
DELETE FROM sales_order
WHERE id = sqlc.arg('id')
AND owner_account_id = sqlc.arg('account_id');

-- name: IsDuplicatePurchaseOrderNumber :one
SELECT COUNT(*) AS cnt FROM sales_order
WHERE owner_account_id = sqlc.arg('account_id')
AND number = sqlc.arg('number')
AND sales_order_type_code = 'purchase_order'
AND (sqlc.narg('exclude_id') IS NULL OR id != sqlc.narg('exclude_id'));

-- name: AllocateNextPurchaseOrderNumber :execresult
-- Atomically reserves the next purchase-order number for the account and returns it via LAST_INSERT_ID.
-- The single upsert holds a row lock on the per-account counter, so concurrent creates serialize and
-- never collide (the old read-MAX-then-write pattern raced). Read it back with LastInsertId().
INSERT INTO sys_property (id, account_id, sys_property_type_code, value, created_at, updated_at)
VALUES (sqlc.arg('id'), sqlc.arg('account_id'), 'purchase_order_number', LAST_INSERT_ID(1), NOW(3), NOW(3))
ON DUPLICATE KEY UPDATE value = LAST_INSERT_ID(value + 1), updated_at = NOW(3);

-- name: DeletePurchaseOrderLinesBySalesOrder :exec
DELETE FROM sales_order_line WHERE sales_order_id = sqlc.arg('sales_order_id');

-- name: DeleteOrderEmailContactsByOrder :exec
DELETE FROM order_email_contact WHERE sales_order_id = sqlc.arg('sales_order_id');

-- name: CreateOrderEmailContact :exec
INSERT INTO order_email_contact (id, sales_order_id, account_user_id, notification_type_code, created_at, updated_at)
VALUES (sqlc.arg('id'), sqlc.arg('sales_order_id'), sqlc.arg('account_user_id'), sqlc.arg('notification_type_code'), NOW(3), NOW(3));

-- name: GetOrderEmailContacts :many
SELECT oec.id, oec.account_user_id
FROM order_email_contact oec
WHERE oec.sales_order_id = sqlc.arg('sales_order_id')
AND oec.notification_type_code = 'purchaseOrderSubmission';

-- name: GetPurchaseOrderSupplierID :one
SELECT so.seller_account_id
FROM sales_order so
WHERE so.id = sqlc.arg('sales_order_id')
AND so.owner_account_id = sqlc.arg('account_id');

-- name: UpdatePurchaseOrderAcknowledgmentSent :exec
UPDATE sales_order SET
    is_acknowledgment_sent = true,
    updated_at = NOW(3)
WHERE id = sqlc.arg('id')
AND owner_account_id = sqlc.arg('account_id');

-- name: GetPurchaseOrderSubmissionRecipients :many
SELECT u.email FROM order_email_contact oec
JOIN account_user au ON au.id = oec.account_user_id
JOIN user u ON u.id = au.user_id
WHERE oec.sales_order_id = sqlc.arg('purchase_order_id')
AND oec.notification_type_code = 'purchase_order_submission'
AND u.email IS NOT NULL;

-- name: MarkPurchaseOrderSubmissionSent :exec
UPDATE sales_order SET
    is_acknowledgment_sent = true,
    updated_at = NOW(3)
WHERE id = sqlc.arg('id')
AND owner_account_id = sqlc.arg('account_id');
