-- name: ListInvoicesForward :many
SELECT
    inv.id,
    inv.number,
    inv.note,
    inv.is_paid_in_full,
    inv.is_edi_sent,
    inv.has_been_sent,
    inv.created_at,
    inv.updated_at,
    so.id AS order_id,
    so.number AS order_number,
    so.priority_code,
    buyer.id AS customer_id,
    buyer.name AS customer_name,
    ar.external_number AS customer_number,
    ar.account_status_code AS customer_status_code,
    ar.commission_status_code AS customer_commission_policy,
    ar.is_edi_enabled AS customer_is_edi_enabled,
    sh.id AS shipment_id,
    addr.id AS billing_address_id,
    addr.name AS billing_address_name,
    geo.street_line_1 AS billing_address_line1,
    geo.street_line_2 AS billing_address_line2,
    geo.locality AS billing_address_city,
    geo.state AS billing_address_state,
    geo.postal_code AS billing_address_zip,
    geo.country AS billing_address_country,
    pt.id AS payment_term_id,
    pt.name AS payment_term_name,
    pt.is_active AS payment_term_is_active,
    COUNT(il.id) AS line_count,
    COALESCE(totals.total_invoiced, 0) AS total_invoiced,
    CASE WHEN EXISTS (
        SELECT 1 FROM order_email_contact oec
        WHERE oec.sales_order_id = so.id
        AND oec.notification_type_code = 'invoice'
    ) THEN true ELSE false END AS accepts_invoice_emails
FROM invoice inv
JOIN sales_order so ON inv.sales_order_id = so.id
JOIN account_relation ar ON ar.owner_account_id = inv.account_id
    AND ar.counterparty_account_id = so.buyer_account_id
    AND ar.account_relation_role_code = 'customer'
JOIN account buyer ON buyer.id = so.buyer_account_id
LEFT JOIN shipment sh ON sh.invoice_id = inv.id
JOIN address addr ON addr.id = inv.billing_address_id
JOIN geolocation geo ON geo.id = addr.geolocation_id
LEFT JOIN payment_term pt ON pt.id = so.payment_term_id
LEFT JOIN invoice_line il ON il.invoice_id = inv.id
LEFT JOIN (
    SELECT
        il2.invoice_id,
        SUM(q.value * r.value) AS total_invoiced
    FROM invoice_line il2
    JOIN quantity q ON q.id = il2.quantity_id
    JOIN sales_order_line sol ON sol.id = il2.sales_order_line_id
    JOIN rate r ON r.id = sol.unit_price_id
    GROUP BY il2.invoice_id
) totals ON totals.invoice_id = inv.id
WHERE inv.account_id = sqlc.arg('account_id')
AND (
    sqlc.narg('search_query') IS NULL
    OR inv.number LIKE sqlc.narg('search_query')
    OR inv.note LIKE sqlc.narg('search_query')
    OR buyer.name LIKE sqlc.narg('search_query')
)
AND (
    sqlc.narg('status') IS NULL
    OR (sqlc.narg('status') = 'paid' AND inv.is_paid_in_full = true)
    OR (sqlc.narg('status') = 'unpaid' AND inv.is_paid_in_full = false AND inv.is_over_paid = false)
    OR (sqlc.narg('status') = 'overpaid' AND inv.is_over_paid = true)
)
AND (
    sqlc.arg('include_item_filter') = false
    OR EXISTS (
        SELECT 1 FROM invoice_line il3
        JOIN sales_order_line sol3 ON il3.sales_order_line_id = sol3.id
        WHERE il3.invoice_id = inv.id
        AND sol3.item_id IN (sqlc.slice('item_ids'))
    )
)
AND (
    sqlc.arg('include_customer_filter') = false
    OR so.buyer_account_id IN (sqlc.slice('customer_ids'))
)
AND (
    sqlc.arg('include_product_line_filter') = false
    OR EXISTS (
        SELECT 1 FROM invoice_line il4
        JOIN sales_order_line sol4 ON il4.sales_order_line_id = sol4.id
        JOIN product p4 ON p4.item_id = sol4.item_id
        WHERE il4.invoice_id = inv.id
        AND p4.product_line_id IN (sqlc.slice('product_line_ids'))
    )
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
    OR inv.created_at >= sqlc.narg('start_date')
)
AND (
    sqlc.narg('end_date') IS NULL
    OR inv.created_at <= sqlc.narg('end_date')
)
AND (
    sqlc.narg('cursor_created_at') IS NULL
    OR inv.created_at < sqlc.narg('cursor_created_at')
    OR (inv.created_at = sqlc.narg('cursor_created_at') AND inv.id < sqlc.narg('cursor_id'))
)
GROUP BY inv.id, inv.number, inv.note, inv.is_paid_in_full, inv.is_edi_sent, inv.has_been_sent,
    inv.created_at, inv.updated_at, so.id, so.number, so.priority_code,
    buyer.id, buyer.name, ar.external_number, ar.account_status_code, ar.commission_status_code, ar.is_edi_enabled,
    sh.id, addr.id, addr.name, geo.street_line_1, geo.street_line_2, geo.locality, geo.state,
    geo.postal_code, geo.country, pt.id, pt.name, totals.total_invoiced
ORDER BY inv.created_at DESC, inv.id DESC
LIMIT ?;

-- name: ListInvoicesBackward :many
SELECT
    inv.id,
    inv.number,
    inv.note,
    inv.is_paid_in_full,
    inv.is_edi_sent,
    inv.has_been_sent,
    inv.created_at,
    inv.updated_at,
    so.id AS order_id,
    so.number AS order_number,
    so.priority_code,
    buyer.id AS customer_id,
    buyer.name AS customer_name,
    ar.external_number AS customer_number,
    ar.account_status_code AS customer_status_code,
    ar.commission_status_code AS customer_commission_policy,
    ar.is_edi_enabled AS customer_is_edi_enabled,
    sh.id AS shipment_id,
    addr.id AS billing_address_id,
    addr.name AS billing_address_name,
    geo.street_line_1 AS billing_address_line1,
    geo.street_line_2 AS billing_address_line2,
    geo.locality AS billing_address_city,
    geo.state AS billing_address_state,
    geo.postal_code AS billing_address_zip,
    geo.country AS billing_address_country,
    pt.id AS payment_term_id,
    pt.name AS payment_term_name,
    pt.is_active AS payment_term_is_active,
    COUNT(il.id) AS line_count,
    COALESCE(totals.total_invoiced, 0) AS total_invoiced,
    CASE WHEN EXISTS (
        SELECT 1 FROM order_email_contact oec
        WHERE oec.sales_order_id = so.id
        AND oec.notification_type_code = 'invoice'
    ) THEN true ELSE false END AS accepts_invoice_emails
FROM invoice inv
JOIN sales_order so ON inv.sales_order_id = so.id
JOIN account_relation ar ON ar.owner_account_id = inv.account_id
    AND ar.counterparty_account_id = so.buyer_account_id
    AND ar.account_relation_role_code = 'customer'
JOIN account buyer ON buyer.id = so.buyer_account_id
LEFT JOIN shipment sh ON sh.invoice_id = inv.id
JOIN address addr ON addr.id = inv.billing_address_id
JOIN geolocation geo ON geo.id = addr.geolocation_id
LEFT JOIN payment_term pt ON pt.id = so.payment_term_id
LEFT JOIN invoice_line il ON il.invoice_id = inv.id
LEFT JOIN (
    SELECT
        il2.invoice_id,
        SUM(q.value * r.value) AS total_invoiced
    FROM invoice_line il2
    JOIN quantity q ON q.id = il2.quantity_id
    JOIN sales_order_line sol ON sol.id = il2.sales_order_line_id
    JOIN rate r ON r.id = sol.unit_price_id
    GROUP BY il2.invoice_id
) totals ON totals.invoice_id = inv.id
WHERE inv.account_id = sqlc.arg('account_id')
AND (
    sqlc.narg('search_query') IS NULL
    OR inv.number LIKE sqlc.narg('search_query')
    OR inv.note LIKE sqlc.narg('search_query')
    OR buyer.name LIKE sqlc.narg('search_query')
)
AND (
    sqlc.narg('status') IS NULL
    OR (sqlc.narg('status') = 'paid' AND inv.is_paid_in_full = true)
    OR (sqlc.narg('status') = 'unpaid' AND inv.is_paid_in_full = false AND inv.is_over_paid = false)
    OR (sqlc.narg('status') = 'overpaid' AND inv.is_over_paid = true)
)
AND (
    sqlc.arg('include_item_filter') = false
    OR EXISTS (
        SELECT 1 FROM invoice_line il3
        JOIN sales_order_line sol3 ON il3.sales_order_line_id = sol3.id
        WHERE il3.invoice_id = inv.id
        AND sol3.item_id IN (sqlc.slice('item_ids'))
    )
)
AND (
    sqlc.arg('include_customer_filter') = false
    OR so.buyer_account_id IN (sqlc.slice('customer_ids'))
)
AND (
    sqlc.arg('include_product_line_filter') = false
    OR EXISTS (
        SELECT 1 FROM invoice_line il4
        JOIN sales_order_line sol4 ON il4.sales_order_line_id = sol4.id
        JOIN product p4 ON p4.item_id = sol4.item_id
        WHERE il4.invoice_id = inv.id
        AND p4.product_line_id IN (sqlc.slice('product_line_ids'))
    )
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
    OR inv.created_at >= sqlc.narg('start_date')
)
AND (
    sqlc.narg('end_date') IS NULL
    OR inv.created_at <= sqlc.narg('end_date')
)
AND (
    inv.created_at > sqlc.arg('cursor_created_at')
    OR (inv.created_at = sqlc.arg('cursor_created_at') AND inv.id > sqlc.arg('cursor_id'))
)
GROUP BY inv.id, inv.number, inv.note, inv.is_paid_in_full, inv.is_edi_sent, inv.has_been_sent,
    inv.created_at, inv.updated_at, so.id, so.number, so.priority_code,
    buyer.id, buyer.name, ar.external_number, ar.account_status_code, ar.commission_status_code, ar.is_edi_enabled,
    sh.id, addr.id, addr.name, geo.street_line_1, geo.street_line_2, geo.locality, geo.state,
    geo.postal_code, geo.country, pt.id, pt.name, totals.total_invoiced
ORDER BY inv.created_at ASC, inv.id ASC
LIMIT ?;

-- name: GetInvoice :one
SELECT
    inv.id,
    inv.number,
    inv.note,
    inv.is_paid_in_full,
    inv.is_over_paid,
    inv.is_edi_sent,
    inv.has_been_sent,
    inv.created_at,
    inv.updated_at,
    so.id AS order_id,
    so.number AS order_number,
    so.buyer_account_id AS customer_id,
    so.payment_term_id AS payment_term_id,
    addr.id AS billing_address_id,
    addr.name AS billing_address_name,
    geo.street_line_1 AS billing_address_line1,
    geo.street_line_2 AS billing_address_line2,
    geo.locality AS billing_address_city,
    geo.state AS billing_address_state,
    geo.postal_code AS billing_address_zip,
    geo.country AS billing_address_country,
    sh.id AS shipment_id,
    sh.number AS shipment_number,
    CASE WHEN EXISTS (
        SELECT 1 FROM order_email_contact oec
        WHERE oec.sales_order_id = so.id
        AND oec.notification_type_code = 'invoice'
    ) THEN true ELSE false END AS accepts_invoice_emails
FROM invoice inv
JOIN sales_order so ON inv.sales_order_id = so.id
JOIN address addr ON addr.id = inv.billing_address_id
JOIN geolocation geo ON geo.id = addr.geolocation_id
LEFT JOIN shipment sh ON sh.invoice_id = inv.id
WHERE inv.id = sqlc.arg('id')
AND inv.account_id = sqlc.arg('account_id');

-- name: GetInvoiceLines :many
SELECT
    il.id,
    il.created_at,
    il.updated_at,
    q.id AS quantity_id,
    q.value AS quantity_value,
    qu.id AS quantity_unit_id,
    qu.abbreviation AS quantity_unit_abbreviation,
    r.id AS unit_price_id,
    r.value AS unit_price_value,
    r.numerator_unit_id AS unit_price_numerator_unit_id,
    r.denominator_unit_id AS unit_price_denominator_unit_id,
    sol.id AS order_line_id,
    sol.item_id AS order_line_item_id,
    i.sku AS order_line_item_sku
FROM invoice_line il
JOIN quantity q ON q.id = il.quantity_id
JOIN unit qu ON qu.id = q.unit_id
JOIN sales_order_line sol ON sol.id = il.sales_order_line_id
JOIN rate r ON r.id = sol.unit_price_id
LEFT JOIN item i ON i.id = sol.item_id
WHERE il.invoice_id = sqlc.arg('invoice_id')
ORDER BY il.created_at ASC, il.id ASC;

-- name: GetInvoiceAllocations :many
SELECT
    ta.id,
    ta.transaction_id,
    ta.note,
    ta.created_at,
    ta.updated_at,
    q.id AS amount_id,
    q.value AS amount_value,
    u.id AS amount_unit_id,
    u.abbreviation AS amount_unit_abbreviation
FROM transaction_allocation ta
JOIN quantity q ON q.id = ta.amount_id
JOIN unit u ON u.id = q.unit_id
WHERE ta.invoice_id = sqlc.arg('invoice_id')
ORDER BY ta.created_at ASC, ta.id ASC;

-- name: UpdateInvoice :exec
UPDATE invoice
SET
    note = COALESCE(sqlc.narg('note'), note),
    has_been_sent = COALESCE(sqlc.narg('has_been_sent'), has_been_sent),
    is_edi_sent = COALESCE(sqlc.narg('is_edi_sent'), is_edi_sent),
    is_paid_in_full = COALESCE(sqlc.narg('is_paid_in_full'), is_paid_in_full),
    updated_at = CURRENT_TIMESTAMP(3)
WHERE id = sqlc.arg('id')
AND account_id = sqlc.arg('account_id');

-- name: GetInvoiceSummaryByID :one
SELECT
    inv.id,
    inv.number,
    inv.note,
    inv.is_paid_in_full,
    inv.is_edi_sent,
    inv.has_been_sent,
    inv.created_at,
    inv.updated_at,
    so.id AS order_id,
    so.number AS order_number,
    so.priority_code,
    buyer.id AS customer_id,
    buyer.name AS customer_name,
    ar.external_number AS customer_number,
    ar.account_status_code AS customer_status_code,
    ar.commission_status_code AS customer_commission_policy,
    ar.is_edi_enabled AS customer_is_edi_enabled,
    sh.id AS shipment_id,
    addr.id AS billing_address_id,
    addr.name AS billing_address_name,
    geo.street_line_1 AS billing_address_line1,
    geo.street_line_2 AS billing_address_line2,
    geo.locality AS billing_address_city,
    geo.state AS billing_address_state,
    geo.postal_code AS billing_address_zip,
    geo.country AS billing_address_country,
    pt.id AS payment_term_id,
    pt.name AS payment_term_name,
    pt.is_active AS payment_term_is_active,
    (SELECT COUNT(*) FROM invoice_line il WHERE il.invoice_id = inv.id) AS line_count,
    COALESCE((
        SELECT SUM(q.value * r.value)
        FROM invoice_line il2
        JOIN quantity q ON q.id = il2.quantity_id
        JOIN sales_order_line sol ON sol.id = il2.sales_order_line_id
        JOIN rate r ON r.id = sol.unit_price_id
        WHERE il2.invoice_id = inv.id
    ), 0) AS total_invoiced,
    CASE WHEN EXISTS (
        SELECT 1 FROM order_email_contact oec
        WHERE oec.sales_order_id = so.id
        AND oec.notification_type_code = 'invoice'
    ) THEN true ELSE false END AS accepts_invoice_emails
FROM invoice inv
JOIN sales_order so ON inv.sales_order_id = so.id
JOIN account_relation ar ON ar.owner_account_id = inv.account_id
    AND ar.counterparty_account_id = so.buyer_account_id
    AND ar.account_relation_role_code = 'customer'
JOIN account buyer ON buyer.id = so.buyer_account_id
LEFT JOIN shipment sh ON sh.invoice_id = inv.id
JOIN address addr ON addr.id = inv.billing_address_id
JOIN geolocation geo ON geo.id = addr.geolocation_id
LEFT JOIN payment_term pt ON pt.id = so.payment_term_id
WHERE inv.id = sqlc.arg('id')
AND inv.account_id = sqlc.arg('account_id');

-- name: ListCustomerInvoicesForward :many
SELECT
    inv.id,
    inv.number,
    inv.is_paid_in_full,
    inv.created_at,
    inv.updated_at,
    so.customer_po_number,
    buyer.id AS customer_id,
    buyer.name AS customer_name,
    ar.external_number AS customer_number,
    ar.account_status_code AS customer_status_code,
    ar.commission_status_code AS customer_commission_policy,
    ar.parent_account_relation_id,
    par.counterparty_account_id AS parent_account_id,
    ar.payment_term_id AS customer_payment_term_id,
    addr.id AS billing_address_id,
    addr.name AS billing_address_name,
    COALESCE(totals.total_invoiced, 0) AS total_invoiced
FROM invoice inv
JOIN sales_order so ON inv.sales_order_id = so.id
JOIN account_relation ar ON ar.owner_account_id = inv.account_id
    AND ar.counterparty_account_id = so.buyer_account_id
    AND ar.account_relation_role_code = 'customer'
JOIN account buyer ON buyer.id = so.buyer_account_id
LEFT JOIN account_relation par ON par.id = ar.parent_account_relation_id
LEFT JOIN address addr ON addr.id = so.billing_address_id
LEFT JOIN (
    SELECT
        il2.invoice_id,
        SUM(q.value * r.value) AS total_invoiced
    FROM invoice_line il2
    JOIN quantity q ON q.id = il2.quantity_id
    JOIN sales_order_line sol ON sol.id = il2.sales_order_line_id
    JOIN rate r ON r.id = sol.unit_price_id
    GROUP BY il2.invoice_id
) totals ON totals.invoice_id = inv.id
WHERE inv.account_id = sqlc.arg('account_id')
AND inv.is_paid_in_full = false
AND inv.is_over_paid = false
AND (
    (sqlc.arg('include_child_accounts') = false AND so.buyer_account_id = sqlc.arg('customer_account_id'))
    OR (sqlc.arg('include_child_accounts') = true AND (
        so.buyer_account_id = sqlc.arg('customer_account_id')
        OR EXISTS (
            SELECT 1 FROM account_relation child_ar
            WHERE child_ar.owner_account_id = inv.account_id
            AND child_ar.counterparty_account_id = so.buyer_account_id
            AND child_ar.account_relation_role_code = 'customer'
            AND child_ar.parent_account_relation_id = (
                SELECT par_ar.id FROM account_relation par_ar
                WHERE par_ar.owner_account_id = inv.account_id
                AND par_ar.counterparty_account_id = sqlc.arg('customer_account_id')
                AND par_ar.account_relation_role_code = 'customer'
            )
        )
    ))
)
AND (
    sqlc.narg('search_query') IS NULL
    OR inv.number LIKE sqlc.narg('search_query')
)
AND (
    sqlc.narg('cursor_created_at') IS NULL
    OR inv.created_at < sqlc.narg('cursor_created_at')
    OR (inv.created_at = sqlc.narg('cursor_created_at') AND inv.id < sqlc.narg('cursor_id'))
)
ORDER BY inv.created_at DESC, inv.id DESC
LIMIT ?;

-- name: ListCustomerInvoicesBackward :many
SELECT
    inv.id,
    inv.number,
    inv.is_paid_in_full,
    inv.created_at,
    inv.updated_at,
    so.customer_po_number,
    buyer.id AS customer_id,
    buyer.name AS customer_name,
    ar.external_number AS customer_number,
    ar.account_status_code AS customer_status_code,
    ar.commission_status_code AS customer_commission_policy,
    ar.parent_account_relation_id,
    par.counterparty_account_id AS parent_account_id,
    ar.payment_term_id AS customer_payment_term_id,
    addr.id AS billing_address_id,
    addr.name AS billing_address_name,
    COALESCE(totals.total_invoiced, 0) AS total_invoiced
FROM invoice inv
JOIN sales_order so ON inv.sales_order_id = so.id
JOIN account_relation ar ON ar.owner_account_id = inv.account_id
    AND ar.counterparty_account_id = so.buyer_account_id
    AND ar.account_relation_role_code = 'customer'
JOIN account buyer ON buyer.id = so.buyer_account_id
LEFT JOIN account_relation par ON par.id = ar.parent_account_relation_id
LEFT JOIN address addr ON addr.id = so.billing_address_id
LEFT JOIN (
    SELECT
        il2.invoice_id,
        SUM(q.value * r.value) AS total_invoiced
    FROM invoice_line il2
    JOIN quantity q ON q.id = il2.quantity_id
    JOIN sales_order_line sol ON sol.id = il2.sales_order_line_id
    JOIN rate r ON r.id = sol.unit_price_id
    GROUP BY il2.invoice_id
) totals ON totals.invoice_id = inv.id
WHERE inv.account_id = sqlc.arg('account_id')
AND inv.is_paid_in_full = false
AND inv.is_over_paid = false
AND (
    (sqlc.arg('include_child_accounts') = false AND so.buyer_account_id = sqlc.arg('customer_account_id'))
    OR (sqlc.arg('include_child_accounts') = true AND (
        so.buyer_account_id = sqlc.arg('customer_account_id')
        OR EXISTS (
            SELECT 1 FROM account_relation child_ar
            WHERE child_ar.owner_account_id = inv.account_id
            AND child_ar.counterparty_account_id = so.buyer_account_id
            AND child_ar.account_relation_role_code = 'customer'
            AND child_ar.parent_account_relation_id = (
                SELECT par_ar.id FROM account_relation par_ar
                WHERE par_ar.owner_account_id = inv.account_id
                AND par_ar.counterparty_account_id = sqlc.arg('customer_account_id')
                AND par_ar.account_relation_role_code = 'customer'
            )
        )
    ))
)
AND (
    sqlc.narg('search_query') IS NULL
    OR inv.number LIKE sqlc.narg('search_query')
)
AND (
    inv.created_at > sqlc.arg('cursor_created_at')
    OR (inv.created_at = sqlc.arg('cursor_created_at') AND inv.id > sqlc.arg('cursor_id'))
)
ORDER BY inv.created_at ASC, inv.id ASC
LIMIT ?;

-- name: IsDuplicateInvoiceNumber :one
SELECT COUNT(*) AS cnt FROM invoice
WHERE account_id = sqlc.arg('account_id')
AND number = sqlc.arg('number');

-- name: GetInvoiceEmailRecipients :many
SELECT u.email FROM order_email_contact oec
JOIN invoice inv ON inv.sales_order_id = oec.sales_order_id
JOIN account_user au ON au.id = oec.account_user_id
JOIN user u ON u.id = au.user_id
WHERE inv.id = sqlc.arg('invoice_id')
AND oec.notification_type_code = 'invoice'
AND u.email IS NOT NULL;

-- name: MarkInvoiceEmailSent :exec
UPDATE invoice
SET has_been_sent = 1
WHERE id = sqlc.arg('invoice_id')
AND account_id = sqlc.arg('account_id');

-- name: DeleteInvoiceLinesByInvoice :exec
DELETE FROM invoice_line
WHERE invoice_id = sqlc.arg('invoice_id');

-- name: DeleteInvoice :exec
DELETE FROM invoice
WHERE id = sqlc.arg('id')
AND account_id = sqlc.arg('account_id');

-- name: CountInvoicesByAccountSince :one
-- Returns the count of invoices created for this account at or after the given
-- timestamp. Used to enforce the per-billing-period invoice plan limit.
SELECT COUNT(*) AS cnt FROM invoice
WHERE account_id = sqlc.arg('account_id')
AND created_at >= sqlc.arg('since');
