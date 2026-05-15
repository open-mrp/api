-- name: ListSalesOrdersForward :many
SELECT
    so.id,
    so.number,
    so.customer_po_number,
    so.sales_order_status_code AS status_code,
    sos.name AS status_name,
    so.sales_order_type_code AS type_code,
    sot.name AS type_name,
    ar.counterparty_account_id AS customer_id,
    ba.name AS customer_name,
    ar.external_number AS customer_number,
    ar.account_status_code AS customer_status_code,
    ar.commission_status_code AS customer_commission_policy,
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
    AND ar.counterparty_account_id = so.buyer_account_id
JOIN account ba ON ba.id = so.buyer_account_id
JOIN sales_order_status sos ON sos.code = so.sales_order_status_code
JOIN sales_order_type sot ON sot.code = so.sales_order_type_code
JOIN priority pr ON pr.code = so.priority_code
WHERE so.owner_account_id = sqlc.arg('account_id')
AND so.seller_account_id = so.owner_account_id
AND (
    sqlc.narg('buyer_account_id') IS NULL
    OR so.buyer_account_id = sqlc.narg('buyer_account_id')
)
AND (
    sqlc.narg('search_query') IS NULL
    OR so.number LIKE sqlc.narg('search_query')
    OR so.customer_po_number LIKE sqlc.narg('search_query')
    OR ba.name LIKE sqlc.narg('search_query')
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
    sqlc.arg('include_product_line_filter') = false
    OR EXISTS (
        SELECT 1 FROM sales_order_line sol3
        JOIN product p ON p.id = sol3.product_id
        WHERE sol3.sales_order_id = so.id
        AND p.product_line_id IN (sqlc.slice('product_line_ids'))
    )
)
AND (
    sqlc.arg('include_customer_filter') = false
    OR so.buyer_account_id IN (sqlc.slice('customer_ids'))
)
AND (
    sqlc.arg('include_customer_group_filter') = false
    OR ar.account_group_id IN (sqlc.slice('customer_group_ids'))
)
AND (
    sqlc.arg('include_sales_rep_filter') = false
    OR so.sales_rep_id IN (sqlc.slice('sales_rep_ids'))
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
    sqlc.arg('exclude_internal_orders') = false
    OR so.buyer_account_id != so.owner_account_id
)
AND (
    sqlc.narg('cursor_created_at') IS NULL
    OR so.created_at < sqlc.narg('cursor_created_at')
    OR (so.created_at = sqlc.narg('cursor_created_at') AND so.id < sqlc.narg('cursor_id'))
)
ORDER BY so.created_at DESC, so.id DESC
LIMIT ?;

-- name: ListSalesOrdersBackward :many
SELECT
    so.id,
    so.number,
    so.customer_po_number,
    so.sales_order_status_code AS status_code,
    sos.name AS status_name,
    so.sales_order_type_code AS type_code,
    sot.name AS type_name,
    ar.counterparty_account_id AS customer_id,
    ba.name AS customer_name,
    ar.external_number AS customer_number,
    ar.account_status_code AS customer_status_code,
    ar.commission_status_code AS customer_commission_policy,
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
    AND ar.counterparty_account_id = so.buyer_account_id
JOIN account ba ON ba.id = so.buyer_account_id
JOIN sales_order_status sos ON sos.code = so.sales_order_status_code
JOIN sales_order_type sot ON sot.code = so.sales_order_type_code
JOIN priority pr ON pr.code = so.priority_code
WHERE so.owner_account_id = sqlc.arg('account_id')
AND so.seller_account_id = so.owner_account_id
AND (
    sqlc.narg('buyer_account_id') IS NULL
    OR so.buyer_account_id = sqlc.narg('buyer_account_id')
)
AND (
    sqlc.narg('search_query') IS NULL
    OR so.number LIKE sqlc.narg('search_query')
    OR so.customer_po_number LIKE sqlc.narg('search_query')
    OR ba.name LIKE sqlc.narg('search_query')
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
    sqlc.arg('include_product_line_filter') = false
    OR EXISTS (
        SELECT 1 FROM sales_order_line sol3
        JOIN product p ON p.id = sol3.product_id
        WHERE sol3.sales_order_id = so.id
        AND p.product_line_id IN (sqlc.slice('product_line_ids'))
    )
)
AND (
    sqlc.arg('include_customer_filter') = false
    OR so.buyer_account_id IN (sqlc.slice('customer_ids'))
)
AND (
    sqlc.arg('include_customer_group_filter') = false
    OR ar.account_group_id IN (sqlc.slice('customer_group_ids'))
)
AND (
    sqlc.arg('include_sales_rep_filter') = false
    OR so.sales_rep_id IN (sqlc.slice('sales_rep_ids'))
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
    sqlc.arg('exclude_internal_orders') = false
    OR so.buyer_account_id != so.owner_account_id
)
AND (
    so.created_at > sqlc.arg('cursor_created_at')
    OR (so.created_at = sqlc.arg('cursor_created_at') AND so.id > sqlc.arg('cursor_id'))
)
ORDER BY so.created_at ASC, so.id ASC
LIMIT ?;

-- name: GetSalesOrder :one
SELECT
    so.id,
    so.number,
    so.customer_po_number,
    so.note,
    so.is_acknowledgment_sent,
    so.billing_address_id,
    so.shipping_address_id,
    so.carrier_id,
    so.carrier_option_id,
    so.carrier_billing_type,
    so.carrier_billing_account,
    so.priority_code,
    so.sales_rep_id,
    so.shipping_term_id,
    so.sales_order_status_code,
    so.sales_order_type_code,
    so.payment_term_id,
    so.production_run_id,
    so.order_discount_id,
    so.buyer_account_id,
    so.seller_account_id,
    so.owner_account_id,
    so.issued_at,
    so.completed_at,
    so.first_ship_at,
    so.expired_at,
    so.promised_at,
    so.created_at,
    so.updated_at,
    -- Customer
    ba.name AS customer_name,
    ar.external_number AS customer_number,
    ar.account_status_code AS customer_status_code,
    ar.commission_status_code AS customer_commission_policy,
    ar.created_at AS customer_created_at,
    ar.updated_at AS customer_updated_at,
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
    bill_geo.id AS bill_to_geolocation_id,
    bill_geo.street_line_1 AS bill_to_street_line_1,
    bill_geo.street_line_2 AS bill_to_street_line_2,
    bill_geo.locality AS bill_to_locality,
    bill_geo.state AS bill_to_state,
    bill_geo.postal_code AS bill_to_postal_code,
    bill_geo.country AS bill_to_country,
    bill_addr.phone AS bill_to_phone,
    bill_addr.email AS bill_to_email,
    bill_addr.created_at AS bill_to_created_at,
    bill_addr.updated_at AS bill_to_updated_at,
    -- Ship-to address
    ship_addr.name AS ship_to_name,
    ship_addr.is_drop_ship AS ship_to_is_drop_ship,
    ship_geo.id AS ship_to_geolocation_id,
    ship_geo.street_line_1 AS ship_to_street_line_1,
    ship_geo.street_line_2 AS ship_to_street_line_2,
    ship_geo.locality AS ship_to_locality,
    ship_geo.state AS ship_to_state,
    ship_geo.postal_code AS ship_to_postal_code,
    ship_geo.country AS ship_to_country,
    ship_addr.phone AS ship_to_phone,
    ship_addr.email AS ship_to_email,
    ship_addr.created_at AS ship_to_created_at,
    ship_addr.updated_at AS ship_to_updated_at,
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
    -- Sales rep
    sr_user.name AS sales_rep_name,
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
    -- Order discount
    od.name AS order_discount_name,
    od.code AS order_discount_code,
    od.percentage AS order_discount_percentage,
    od.value AS order_discount_amount,
    od.discount_type_code AS order_discount_discount_type,
    (SELECT COUNT(*) FROM sales_order so2 WHERE so2.order_discount_id = od.id) AS order_discount_order_count,
    od.created_at AS order_discount_created_at,
    od.updated_at AS order_discount_updated_at,
    -- Pick
    pk.id AS pick_id
FROM sales_order so
JOIN account_relation ar ON ar.owner_account_id = so.owner_account_id
    AND ar.counterparty_account_id = so.buyer_account_id
JOIN account ba ON ba.id = so.buyer_account_id
JOIN sales_order_status sos ON sos.code = so.sales_order_status_code
JOIN sales_order_type sot ON sot.code = so.sales_order_type_code
JOIN priority pr ON pr.code = so.priority_code
LEFT JOIN address bill_addr ON bill_addr.id = so.billing_address_id
LEFT JOIN geolocation bill_geo ON bill_geo.id = bill_addr.geolocation_id
LEFT JOIN address ship_addr ON ship_addr.id = so.shipping_address_id
LEFT JOIN geolocation ship_geo ON ship_geo.id = ship_addr.geolocation_id
LEFT JOIN carrier cr ON cr.id = so.carrier_id
LEFT JOIN carrier_option co ON co.id = so.carrier_option_id
LEFT JOIN account_user sr_au ON sr_au.id = so.sales_rep_id
LEFT JOIN user sr_user ON sr_user.id = sr_au.user_id
LEFT JOIN payment_term pt ON pt.id = so.payment_term_id
LEFT JOIN shipping_term st ON st.id = so.shipping_term_id
LEFT JOIN order_discount od ON od.id = so.order_discount_id
LEFT JOIN pick pk ON pk.sales_order_id = so.id
WHERE so.id = sqlc.arg('sales_order_id')
AND so.owner_account_id = sqlc.arg('account_id')
AND so.seller_account_id = so.owner_account_id;

-- name: GetSalesOrderForCustomer :one
SELECT
    so.id,
    so.number,
    so.customer_po_number,
    so.note,
    so.is_acknowledgment_sent,
    so.billing_address_id,
    so.shipping_address_id,
    so.carrier_id,
    so.carrier_option_id,
    so.carrier_billing_type,
    so.carrier_billing_account,
    so.priority_code,
    so.sales_rep_id,
    so.shipping_term_id,
    so.sales_order_status_code,
    so.sales_order_type_code,
    so.payment_term_id,
    so.production_run_id,
    so.order_discount_id,
    so.buyer_account_id,
    so.seller_account_id,
    so.owner_account_id,
    so.issued_at,
    so.completed_at,
    so.first_ship_at,
    so.expired_at,
    so.promised_at,
    so.created_at,
    so.updated_at,
    -- Customer
    ba.name AS customer_name,
    ar.external_number AS customer_number,
    ar.account_status_code AS customer_status_code,
    ar.commission_status_code AS customer_commission_policy,
    ar.created_at AS customer_created_at,
    ar.updated_at AS customer_updated_at,
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
    bill_geo.id AS bill_to_geolocation_id,
    bill_geo.street_line_1 AS bill_to_street_line_1,
    bill_geo.street_line_2 AS bill_to_street_line_2,
    bill_geo.locality AS bill_to_locality,
    bill_geo.state AS bill_to_state,
    bill_geo.postal_code AS bill_to_postal_code,
    bill_geo.country AS bill_to_country,
    bill_addr.phone AS bill_to_phone,
    bill_addr.email AS bill_to_email,
    bill_addr.created_at AS bill_to_created_at,
    bill_addr.updated_at AS bill_to_updated_at,
    -- Ship-to address
    ship_addr.name AS ship_to_name,
    ship_addr.is_drop_ship AS ship_to_is_drop_ship,
    ship_geo.id AS ship_to_geolocation_id,
    ship_geo.street_line_1 AS ship_to_street_line_1,
    ship_geo.street_line_2 AS ship_to_street_line_2,
    ship_geo.locality AS ship_to_locality,
    ship_geo.state AS ship_to_state,
    ship_geo.postal_code AS ship_to_postal_code,
    ship_geo.country AS ship_to_country,
    ship_addr.phone AS ship_to_phone,
    ship_addr.email AS ship_to_email,
    ship_addr.created_at AS ship_to_created_at,
    ship_addr.updated_at AS ship_to_updated_at,
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
    -- Sales rep
    sr_user.name AS sales_rep_name,
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
    -- Order discount
    od.name AS order_discount_name,
    od.code AS order_discount_code,
    od.percentage AS order_discount_percentage,
    od.value AS order_discount_amount,
    od.discount_type_code AS order_discount_discount_type,
    (SELECT COUNT(*) FROM sales_order so2 WHERE so2.order_discount_id = od.id) AS order_discount_order_count,
    od.created_at AS order_discount_created_at,
    od.updated_at AS order_discount_updated_at,
    -- Pick
    pk.id AS pick_id
FROM sales_order so
JOIN account_relation ar ON ar.owner_account_id = so.owner_account_id
    AND ar.counterparty_account_id = so.buyer_account_id
JOIN account ba ON ba.id = so.buyer_account_id
JOIN sales_order_status sos ON sos.code = so.sales_order_status_code
JOIN sales_order_type sot ON sot.code = so.sales_order_type_code
JOIN priority pr ON pr.code = so.priority_code
LEFT JOIN address bill_addr ON bill_addr.id = so.billing_address_id
LEFT JOIN geolocation bill_geo ON bill_geo.id = bill_addr.geolocation_id
LEFT JOIN address ship_addr ON ship_addr.id = so.shipping_address_id
LEFT JOIN geolocation ship_geo ON ship_geo.id = ship_addr.geolocation_id
LEFT JOIN carrier cr ON cr.id = so.carrier_id
LEFT JOIN carrier_option co ON co.id = so.carrier_option_id
LEFT JOIN account_user sr_au ON sr_au.id = so.sales_rep_id
LEFT JOIN user sr_user ON sr_user.id = sr_au.user_id
LEFT JOIN payment_term pt ON pt.id = so.payment_term_id
LEFT JOIN shipping_term st ON st.id = so.shipping_term_id
LEFT JOIN order_discount od ON od.id = so.order_discount_id
LEFT JOIN pick pk ON pk.sales_order_id = so.id
WHERE so.id = sqlc.arg('sales_order_id')
AND so.owner_account_id = sqlc.arg('account_id')
AND so.seller_account_id = so.owner_account_id
AND so.buyer_account_id = sqlc.arg('buyer_account_id');

-- name: GetSalesOrderLines :many
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
WHERE sol.sales_order_id = sqlc.arg('sales_order_id')
ORDER BY sol.line_item_number ASC;

-- name: CreateSalesOrder :exec
INSERT INTO sales_order (
    id, number, customer_po_number, note, is_acknowledgment_sent,
    billing_address_id, shipping_address_id,
    carrier_id, carrier_option_id, carrier_billing_type, carrier_billing_account,
    priority_code, sales_rep_id, shipping_term_id,
    sales_order_status_code, sales_order_type_code,
    payment_term_id, order_discount_id,
    buyer_account_id, seller_account_id, owner_account_id,
    created_at, updated_at
) VALUES (
    sqlc.arg('id'), sqlc.arg('number'), sqlc.narg('customer_po_number'), sqlc.narg('note'), false,
    sqlc.arg('billing_address_id'), sqlc.arg('shipping_address_id'),
    sqlc.narg('carrier_id'), sqlc.narg('carrier_option_id'), sqlc.narg('carrier_billing_type'), sqlc.narg('carrier_billing_account'),
    sqlc.arg('priority_code'), sqlc.narg('sales_rep_id'), sqlc.narg('shipping_term_id'),
    sqlc.arg('sales_order_status_code'), sqlc.arg('sales_order_type_code'),
    sqlc.narg('payment_term_id'), sqlc.narg('order_discount_id'),
    sqlc.arg('buyer_account_id'), sqlc.arg('seller_account_id'), sqlc.arg('owner_account_id'),
    NOW(3), NOW(3)
);

-- name: UpdateSalesOrder :exec
UPDATE sales_order SET
    number = COALESCE(sqlc.narg('number'), number),
    customer_po_number = COALESCE(sqlc.narg('customer_po_number'), customer_po_number),
    note = COALESCE(sqlc.narg('note'), note),
    carrier_id = sqlc.narg('carrier_id'),
    carrier_option_id = sqlc.narg('carrier_option_id'),
    carrier_billing_type = COALESCE(sqlc.narg('carrier_billing_type'), carrier_billing_type),
    carrier_billing_account = COALESCE(sqlc.narg('carrier_billing_account'), carrier_billing_account),
    priority_code = COALESCE(sqlc.narg('priority_code'), priority_code),
    sales_rep_id = sqlc.narg('sales_rep_id'),
    shipping_term_id = sqlc.narg('shipping_term_id'),
    payment_term_id = sqlc.narg('payment_term_id'),
    order_discount_id = sqlc.narg('order_discount_id'),
    is_acknowledgment_sent = COALESCE(sqlc.narg('is_acknowledgment_sent'), is_acknowledgment_sent),
    promised_at = COALESCE(sqlc.narg('promised_at'), promised_at),
    buyer_account_id = sqlc.narg('buyer_account_id'),
    billing_address_id = COALESCE(sqlc.narg('billing_address_id'), billing_address_id),
    shipping_address_id = COALESCE(sqlc.narg('shipping_address_id'), shipping_address_id),
    updated_at = NOW(3)
WHERE id = sqlc.arg('id')
AND owner_account_id = sqlc.arg('account_id');

-- name: UpdateSalesOrderStatus :exec
UPDATE sales_order SET
    sales_order_status_code = sqlc.arg('status_code'),
    issued_at = sqlc.narg('issued_at'),
    completed_at = sqlc.narg('completed_at'),
    updated_at = NOW(3)
WHERE id = sqlc.arg('id')
AND owner_account_id = sqlc.arg('account_id');

-- name: MarkSalesOrderUnfulfilled :exec
UPDATE sales_order SET
    sales_order_status_code = 'issued',
    completed_at = NULL,
    first_ship_at = NULL,
    updated_at = NOW(3)
WHERE id = sqlc.arg('id')
AND owner_account_id = sqlc.arg('account_id');

-- name: NoteSalesOrderFirstShipAt :exec
UPDATE sales_order SET
    first_ship_at = COALESCE(first_ship_at, NOW(3)),
    updated_at = NOW(3)
WHERE id = sqlc.arg('id')
AND owner_account_id = sqlc.arg('account_id');

-- name: DeleteSalesOrder :exec
DELETE FROM sales_order
WHERE id = sqlc.arg('id')
AND owner_account_id = sqlc.arg('account_id');

-- name: IsOrderForCustomer :one
SELECT EXISTS(
    SELECT 1 FROM sales_order
    WHERE id = sqlc.arg('sales_order_id')
    AND buyer_account_id = sqlc.arg('buyer_account_id')
) AS `exists`;

-- name: IsDuplicateOrderNumber :one
SELECT COUNT(*) AS cnt FROM sales_order
WHERE owner_account_id = sqlc.arg('account_id')
AND seller_account_id = sqlc.arg('account_id')
AND number = sqlc.arg('number')
AND (sqlc.narg('exclude_id') IS NULL OR id != sqlc.narg('exclude_id'));

-- name: IsDuplicateCustomerPO :one
SELECT COUNT(*) AS cnt FROM sales_order
WHERE owner_account_id = sqlc.arg('account_id')
AND seller_account_id = sqlc.arg('account_id')
AND buyer_account_id = sqlc.arg('buyer_account_id')
AND customer_po_number = sqlc.arg('customer_po_number')
AND (sqlc.narg('exclude_id') IS NULL OR id != sqlc.narg('exclude_id'));

-- name: GetNextOrderNumber :one
SELECT COALESCE(
    (SELECT MAX(CAST(sp.value AS UNSIGNED)) + 1
     FROM sys_property sp
     WHERE sp.account_id = sqlc.arg('account_id')
     AND sp.sys_property_type_code = 'sales_order_number'),
    1
) AS next_number;

-- name: UpdateNextOrderNumber :exec
INSERT INTO sys_property (id, account_id, sys_property_type_code, value, created_at, updated_at)
VALUES (sqlc.arg('id'), sqlc.arg('account_id'), 'sales_order_number', sqlc.arg('value'), NOW(3), NOW(3))
ON DUPLICATE KEY UPDATE value = sqlc.arg('value'), updated_at = NOW(3);

-- name: GetSalesOrderPickID :one
SELECT pk.id
FROM pick pk
WHERE pk.sales_order_id = sqlc.arg('sales_order_id');

-- name: DeletePickBySalesOrder :exec
DELETE FROM pick WHERE sales_order_id = sqlc.arg('sales_order_id');

-- name: DeletePickLinesBySalesOrder :exec
DELETE pl FROM pick_line pl
JOIN pick pk ON pk.id = pl.pick_id
WHERE pk.sales_order_id = sqlc.arg('sales_order_id');

-- name: DeleteShipmentLineQuantitiesBySalesOrder :exec
DELETE q FROM quantity q
JOIN shipment_line sl ON sl.quantity_id = q.id
JOIN sales_order_line sol ON sol.id = sl.sales_order_line_id
WHERE sol.sales_order_id = sqlc.arg('sales_order_id');

-- name: DeleteShipmentLinesBySalesOrder :exec
DELETE sl FROM shipment_line sl
JOIN sales_order_line sol ON sol.id = sl.sales_order_line_id
WHERE sol.sales_order_id = sqlc.arg('sales_order_id');

-- name: DeleteInvoiceLineQuantitiesBySalesOrder :exec
DELETE q FROM quantity q
JOIN invoice_line il ON il.quantity_id = q.id
JOIN sales_order_line sol ON sol.id = il.sales_order_line_id
WHERE sol.sales_order_id = sqlc.arg('sales_order_id');

-- name: DeleteInvoiceLinesBySalesOrder :exec
DELETE il FROM invoice_line il
JOIN sales_order_line sol ON sol.id = il.sales_order_line_id
WHERE sol.sales_order_id = sqlc.arg('sales_order_id');

-- name: DeleteSalesOrderLinesBySalesOrder :exec
DELETE FROM sales_order_line WHERE sales_order_id = sqlc.arg('sales_order_id');

-- name: DeleteOrderPaymentIntents :exec
DELETE FROM order_payment_intent WHERE sales_order_id = sqlc.arg('sales_order_id');

-- name: DeleteOrderEmailContacts :exec
DELETE FROM order_email_contact WHERE sales_order_id = sqlc.arg('sales_order_id');

-- name: CreateSalesOrderEmailContact :exec
INSERT INTO order_email_contact (id, sales_order_id, account_user_id, notification_type_code, created_at, updated_at)
VALUES (sqlc.arg('id'), sqlc.arg('sales_order_id'), sqlc.arg('account_user_id'), sqlc.arg('notification_type_code'), NOW(3), NOW(3));

-- name: DeleteSalesOrderEmailContactsByOrderAndType :exec
DELETE FROM order_email_contact
WHERE sales_order_id = sqlc.arg('sales_order_id')
AND notification_type_code = sqlc.arg('notification_type_code');

-- name: DeleteReservedInventoryIssuesBySalesOrder :exec
DELETE FROM inventory_issue
WHERE order_id = sqlc.arg('sales_order_id')
AND account_id = sqlc.arg('account_id')
AND status_code = 'reserved';

-- name: CreatePick :exec
INSERT INTO pick (id, number, sales_order_id, account_id, created_at, updated_at)
VALUES (sqlc.arg('id'), sqlc.arg('number'), sqlc.arg('sales_order_id'), sqlc.arg('account_id'), NOW(3), NOW(3));

-- name: CreatePickLineForOrderLine :exec
INSERT INTO pick_line (id, pick_id, quantity_id, sales_order_line_id, created_at, updated_at)
VALUES (sqlc.arg('id'), sqlc.arg('pick_id'), sqlc.arg('quantity_id'), sqlc.arg('sales_order_line_id'), NOW(3), NOW(3));

-- name: DeleteQuantitiesByPickLines :exec
DELETE q FROM quantity q
JOIN pick_line pl ON pl.quantity_id = q.id
JOIN pick pk ON pk.id = pl.pick_id
WHERE pk.sales_order_id = sqlc.arg('sales_order_id');

-- name: DeleteQuantitiesByPickLinesForLine :exec
DELETE q FROM quantity q
JOIN pick_line pl ON pl.quantity_id = q.id
WHERE pl.sales_order_line_id = sqlc.arg('sales_order_line_id');

-- name: CheckSalesOrderPaymentStatus :one
SELECT (
  EXISTS(
    SELECT 1 FROM order_payment_intent opi WHERE opi.sales_order_id = sqlc.arg('sales_order_id')
  )
  OR (
    so.sales_order_status_code = 'fulfilled'
    AND NOT EXISTS(
      SELECT 1 FROM invoice_line il
      JOIN sales_order_line sol ON sol.id = il.sales_order_line_id
      JOIN invoice i ON i.id = il.invoice_id
      WHERE sol.sales_order_id = sqlc.arg('sales_order_id')
      AND i.is_paid_in_full = false
    )
    AND EXISTS(
      SELECT 1 FROM invoice_line il2
      JOIN sales_order_line sol2 ON sol2.id = il2.sales_order_line_id
      WHERE sol2.sales_order_id = sqlc.arg('sales_order_id')
    )
  )
) AS has_payment_intent
FROM sales_order so
WHERE so.id = sqlc.arg('sales_order_id');

-- name: GetSalesOrderLinesForBOM :many
SELECT
    sol.id,
    sol.item_id,
    q.value AS quantity_value,
    q.unit_id AS quantity_unit_id
FROM sales_order_line sol
JOIN quantity q ON q.id = sol.quantity_id
WHERE sol.sales_order_id = sqlc.arg('sales_order_id')
AND sol.item_id IS NOT NULL;

-- name: SetSalesOrderProductionRunID :exec
UPDATE sales_order
SET production_run_id = sqlc.arg('production_run_id'), updated_at = NOW(3)
WHERE id = sqlc.arg('id') AND owner_account_id = sqlc.arg('account_id');

-- name: GetOrderAcknowledgementRecipients :many
SELECT u.email FROM order_email_contact oec
JOIN account_user au ON au.id = oec.account_user_id
JOIN user u ON u.id = au.user_id
WHERE oec.sales_order_id = sqlc.arg('sales_order_id')
AND oec.notification_type_code = 'order_acknowledgement'
AND u.email IS NOT NULL;

-- name: MarkAcknowledgementSent :exec
UPDATE sales_order SET
    is_acknowledgment_sent = true,
    updated_at = NOW(3)
WHERE id = sqlc.arg('id')
AND owner_account_id = sqlc.arg('account_id');

-- name: GetSalesOrderSaleLinesForIssue :many
SELECT sol.id, sol.item_id, q.value AS quantity_value, q.unit_id AS quantity_unit_id
FROM sales_order_line sol
JOIN quantity q ON q.id = sol.quantity_id
JOIN product p ON p.id = sol.product_id
WHERE sol.sales_order_id = sqlc.arg('sales_order_id')
AND p.product_type_code = 'sale';

-- name: CreateReservedInventoryIssueForSalesOrder :exec
INSERT INTO inventory_issue (id, account_id, item_id, quantity_id, status_code, order_id, created_at, updated_at)
VALUES (sqlc.arg('id'), sqlc.arg('account_id'), sqlc.arg('item_id'), sqlc.arg('quantity_id'), 'reserved', sqlc.arg('order_id'), NOW(3), NOW(3));

-- name: DeleteInventoryAllocationsByReservedSalesOrderIssues :exec
DELETE ia FROM inventory_allocation ia
JOIN inventory_issue ii ON ii.id = ia.inventory_issue_id
WHERE ii.order_id = sqlc.arg('sales_order_id')
AND ii.account_id = sqlc.arg('account_id')
AND ii.status_code = 'reserved';

-- name: HasShippedShipmentForSalesOrder :one
SELECT EXISTS(
    SELECT 1 FROM shipment s
    WHERE s.sales_order_id = sqlc.arg('sales_order_id')
    AND s.shipment_status_code = 'shipped'
) AS has_shipped_shipment;
