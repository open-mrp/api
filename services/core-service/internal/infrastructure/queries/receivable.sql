-- name: ListReceivablesForward :many
SELECT
    inv.id,
    inv.number AS invoice_number,
    inv.is_paid_in_full,
    inv.created_at,
    so.customer_po_number AS po_number,
    buyer.id AS customer_id,
    buyer.name AS customer_name,
    ar.external_number AS customer_number,
    ROUND(
        COALESCE((
            SELECT SUM(q.value * r.value)
            FROM invoice_line il
            JOIN quantity q ON q.id = il.quantity_id
            JOIN sales_order_line sol ON sol.id = il.sales_order_line_id
            JOIN rate r ON r.id = sol.unit_price_id
            WHERE il.invoice_id = inv.id
        ), 0)
        -
        COALESCE((
            SELECT SUM(aq.value)
            FROM transaction_allocation ta
            JOIN quantity aq ON aq.id = ta.amount_id
            WHERE ta.invoice_id = inv.id
            AND ta.created_at < sqlc.narg('allocation_cutoff_date')
        ), 0),
    2) AS remaining_balance
FROM invoice inv
JOIN sales_order so ON inv.sales_order_id = so.id
JOIN account_relation ar ON ar.owner_account_id = inv.account_id
    AND ar.counterparty_account_id = so.buyer_account_id
    AND ar.account_relation_role_code = 'customer'
JOIN account buyer ON buyer.id = so.buyer_account_id
WHERE inv.account_id = sqlc.arg('account_id')
AND inv.is_paid_in_full = false
AND (sqlc.narg('cutoff_date') IS NULL OR inv.created_at < sqlc.narg('cutoff_date'))
AND (
    sqlc.narg('search_query') IS NULL
    OR inv.number LIKE sqlc.narg('search_query')
    OR buyer.name LIKE sqlc.narg('search_query')
)
AND (
    sqlc.narg('cursor_created_at') IS NULL
    OR inv.created_at < sqlc.narg('cursor_created_at')
    OR (inv.created_at = sqlc.narg('cursor_created_at') AND inv.id < sqlc.narg('cursor_id'))
)
ORDER BY inv.created_at DESC, inv.id DESC
LIMIT ?;

-- name: ListReceivablesBackward :many
SELECT
    inv.id,
    inv.number AS invoice_number,
    inv.is_paid_in_full,
    inv.created_at,
    so.customer_po_number AS po_number,
    buyer.id AS customer_id,
    buyer.name AS customer_name,
    ar.external_number AS customer_number,
    ROUND(
        COALESCE((
            SELECT SUM(q.value * r.value)
            FROM invoice_line il
            JOIN quantity q ON q.id = il.quantity_id
            JOIN sales_order_line sol ON sol.id = il.sales_order_line_id
            JOIN rate r ON r.id = sol.unit_price_id
            WHERE il.invoice_id = inv.id
        ), 0)
        -
        COALESCE((
            SELECT SUM(aq.value)
            FROM transaction_allocation ta
            JOIN quantity aq ON aq.id = ta.amount_id
            WHERE ta.invoice_id = inv.id
            AND ta.created_at < sqlc.narg('allocation_cutoff_date')
        ), 0),
    2) AS remaining_balance
FROM invoice inv
JOIN sales_order so ON inv.sales_order_id = so.id
JOIN account_relation ar ON ar.owner_account_id = inv.account_id
    AND ar.counterparty_account_id = so.buyer_account_id
    AND ar.account_relation_role_code = 'customer'
JOIN account buyer ON buyer.id = so.buyer_account_id
WHERE inv.account_id = sqlc.arg('account_id')
AND inv.is_paid_in_full = false
AND (sqlc.narg('cutoff_date') IS NULL OR inv.created_at < sqlc.narg('cutoff_date'))
AND (
    sqlc.narg('search_query') IS NULL
    OR inv.number LIKE sqlc.narg('search_query')
    OR buyer.name LIKE sqlc.narg('search_query')
)
AND (
    inv.created_at > sqlc.arg('cursor_created_at')
    OR (inv.created_at = sqlc.arg('cursor_created_at') AND inv.id > sqlc.arg('cursor_id'))
)
ORDER BY inv.created_at ASC, inv.id ASC
LIMIT ?;

-- name: ListReceivablesByCustomerForward :many
SELECT
    inv.id,
    inv.number AS invoice_number,
    inv.is_paid_in_full,
    inv.created_at,
    so.customer_po_number AS po_number,
    buyer.id AS customer_id,
    buyer.name AS customer_name,
    ar.external_number AS customer_number,
    ROUND(
        COALESCE((
            SELECT SUM(q.value * r.value)
            FROM invoice_line il
            JOIN quantity q ON q.id = il.quantity_id
            JOIN sales_order_line sol ON sol.id = il.sales_order_line_id
            JOIN rate r ON r.id = sol.unit_price_id
            WHERE il.invoice_id = inv.id
        ), 0)
        -
        COALESCE((
            SELECT SUM(aq.value)
            FROM transaction_allocation ta
            JOIN quantity aq ON aq.id = ta.amount_id
            WHERE ta.invoice_id = inv.id
            AND ta.created_at < sqlc.narg('allocation_cutoff_date')
        ), 0),
    2) AS remaining_balance
FROM invoice inv
JOIN sales_order so ON inv.sales_order_id = so.id
JOIN account_relation ar ON ar.owner_account_id = inv.account_id
    AND ar.counterparty_account_id = so.buyer_account_id
    AND ar.account_relation_role_code = 'customer'
JOIN account buyer ON buyer.id = so.buyer_account_id
WHERE inv.account_id = sqlc.arg('account_id')
AND so.buyer_account_id = sqlc.arg('customer_account_id')
AND inv.is_paid_in_full = false
AND (sqlc.narg('cutoff_date') IS NULL OR inv.created_at < sqlc.narg('cutoff_date'))
AND (
    sqlc.narg('search_query') IS NULL
    OR inv.number LIKE sqlc.narg('search_query')
    OR buyer.name LIKE sqlc.narg('search_query')
)
AND (
    sqlc.narg('cursor_created_at') IS NULL
    OR inv.created_at < sqlc.narg('cursor_created_at')
    OR (inv.created_at = sqlc.narg('cursor_created_at') AND inv.id < sqlc.narg('cursor_id'))
)
ORDER BY inv.created_at DESC, inv.id DESC
LIMIT ?;

-- name: ListReceivablesByCustomerBackward :many
SELECT
    inv.id,
    inv.number AS invoice_number,
    inv.is_paid_in_full,
    inv.created_at,
    so.customer_po_number AS po_number,
    buyer.id AS customer_id,
    buyer.name AS customer_name,
    ar.external_number AS customer_number,
    ROUND(
        COALESCE((
            SELECT SUM(q.value * r.value)
            FROM invoice_line il
            JOIN quantity q ON q.id = il.quantity_id
            JOIN sales_order_line sol ON sol.id = il.sales_order_line_id
            JOIN rate r ON r.id = sol.unit_price_id
            WHERE il.invoice_id = inv.id
        ), 0)
        -
        COALESCE((
            SELECT SUM(aq.value)
            FROM transaction_allocation ta
            JOIN quantity aq ON aq.id = ta.amount_id
            WHERE ta.invoice_id = inv.id
            AND ta.created_at < sqlc.narg('allocation_cutoff_date')
        ), 0),
    2) AS remaining_balance
FROM invoice inv
JOIN sales_order so ON inv.sales_order_id = so.id
JOIN account_relation ar ON ar.owner_account_id = inv.account_id
    AND ar.counterparty_account_id = so.buyer_account_id
    AND ar.account_relation_role_code = 'customer'
JOIN account buyer ON buyer.id = so.buyer_account_id
WHERE inv.account_id = sqlc.arg('account_id')
AND so.buyer_account_id = sqlc.arg('customer_account_id')
AND inv.is_paid_in_full = false
AND (sqlc.narg('cutoff_date') IS NULL OR inv.created_at < sqlc.narg('cutoff_date'))
AND (
    sqlc.narg('search_query') IS NULL
    OR inv.number LIKE sqlc.narg('search_query')
    OR buyer.name LIKE sqlc.narg('search_query')
)
AND (
    inv.created_at > sqlc.arg('cursor_created_at')
    OR (inv.created_at = sqlc.arg('cursor_created_at') AND inv.id > sqlc.arg('cursor_id'))
)
ORDER BY inv.created_at ASC, inv.id ASC
LIMIT ?;

-- name: GetOpenCreditsByCustomer :many
SELECT
    t.id,
    t.number,
    t.created_at,
    tq.value AS original_amount,
    ROUND(
        tq.value - COALESCE((
            SELECT SUM(aq.value)
            FROM transaction_allocation ta
            JOIN quantity aq ON aq.id = ta.amount_id
            WHERE ta.transaction_id = t.id
        ), 0),
    2) AS leftover_amount
FROM transaction t
JOIN quantity tq ON tq.id = t.amount_id
WHERE t.account_id = sqlc.arg('account_id')
AND t.customer_account_id = sqlc.arg('customer_account_id')
AND t.is_fully_allocated = false
ORDER BY t.created_at DESC;
