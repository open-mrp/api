-- name: SelectSalesFactSource :many
-- The legacy sales-analytics row for each invoice line of the given invoices, priced with the exact
-- expressions the dashboard's analytics used on read, then rounded to the column's 10 places. The CAST
-- must match the column type: the refresher compares these values to the stored ones, and any rounding
-- difference would rewrite every line on every pass. The inner joins decide which lines count as sales
-- at all (a line whose product has no product line or category is not a sale).
SELECT
    i.account_id,
    i.created_at AS invoiced_at,
    il.id AS invoice_line_id,
    i.id AS invoice_id,
    so.id AS sales_order_id,
    so.sales_order_type_code,
    so.buyer_account_id,
    so.sales_rep_id,
    so.order_discount_id,
    fg.id AS product_id,
    pb.id AS item_id,
    fg.product_line_id,
    NULLIF(CAST(
        (
            (
                (q_in.value * (u_in.ratio_numerator / u_in.ratio_denominator))
                + (u_in.offset_numerator / u_in.offset_denominator)
            )
            - (bu_unit.offset_numerator / bu_unit.offset_denominator)
        )
        / NULLIF((bu_unit.ratio_numerator / bu_unit.ratio_denominator), 0)
        AS DECIMAL(28,10)
    ), NULL) AS quantity_base,
    NULLIF(CAST(
        (
            (q_in.value * (u_in.ratio_numerator / u_in.ratio_denominator))
            + (u_in.offset_numerator / u_in.offset_denominator)
        )
        *
        (
            (
                (r_price.value * (u_price_num.ratio_numerator / u_price_num.ratio_denominator))
                + (u_price_num.offset_numerator / u_price_num.offset_denominator)
            )
            / NULLIF(((u_price_den.ratio_numerator / u_price_den.ratio_denominator) + (u_price_den.offset_numerator / u_price_den.offset_denominator)), 0)
        )
        AS DECIMAL(28,10)
    ), NULL) AS total_invoiced,
    NULLIF(CAST(
        (
            (q_in.value * (u_in.ratio_numerator / u_in.ratio_denominator))
            + (u_in.offset_numerator / u_in.offset_denominator)
        )
        *
        (
            (
                (COALESCE(r_cost.value, 0) * (COALESCE(u_cost_num.ratio_numerator, 1) / COALESCE(u_cost_num.ratio_denominator, 1)))
                + (COALESCE(u_cost_num.offset_numerator, 0) / COALESCE(u_cost_num.offset_denominator, 1))
            )
            / NULLIF(((COALESCE(u_cost_den.ratio_numerator, 1) / COALESCE(u_cost_den.ratio_denominator, 1)) + (COALESCE(u_cost_den.offset_numerator, 0) / COALESCE(u_cost_den.offset_denominator, 1))), 0)
        )
        AS DECIMAL(28,10)
    ), NULL) AS total_cost,
    so.issued_at AS ordered_at,
    CAST(COALESCE(r_price.value > 0, FALSE) AS SIGNED) AS is_priced
FROM invoice_line il
JOIN invoice i ON i.id = il.invoice_id
JOIN sales_order_line sol ON sol.id = il.sales_order_line_id
JOIN sales_order so ON so.id = i.sales_order_id
JOIN product fg ON fg.id = sol.product_id
JOIN item pb ON pb.id = fg.item_id
JOIN item_category ic ON ic.id = pb.item_category_id
JOIN product_line pl ON pl.id = fg.product_line_id
JOIN quantity q_in ON q_in.id = il.quantity_id
JOIN unit u_in ON u_in.id = q_in.unit_id
LEFT JOIN unit_group ug ON ug.id = ic.unit_group_id
LEFT JOIN unit bu_unit ON bu_unit.id = ug.base_unit_id
LEFT JOIN rate r_price ON r_price.id = sol.unit_price_id
LEFT JOIN unit u_price_num ON u_price_num.id = r_price.numerator_unit_id
LEFT JOIN unit u_price_den ON u_price_den.id = r_price.denominator_unit_id
LEFT JOIN rate r_cost ON r_cost.id = sol.unit_cost_id
LEFT JOIN unit u_cost_num ON u_cost_num.id = r_cost.numerator_unit_id
LEFT JOIN unit u_cost_den ON u_cost_den.id = r_cost.denominator_unit_id
WHERE il.invoice_id IN (sqlc.slice('invoice_ids'));

-- name: SelectSalesFactsByInvoiceIDs :many
SELECT account_id, invoiced_at, invoice_line_id, invoice_id, sales_order_id, sales_order_type_code,
       buyer_account_id, sales_rep_id, order_discount_id, product_id, item_id, product_line_id,
       quantity_base, total_invoiced, total_cost, ordered_at, is_priced
FROM sales_line_fact
WHERE invoice_id IN (sqlc.slice('invoice_ids'));

-- name: DeleteSalesFactsByLineIDs :exec
DELETE FROM sales_line_fact WHERE invoice_line_id IN (sqlc.slice('invoice_line_ids'));

-- name: ListInvoicesForFactSweep :many
-- The next page of invoices after the cursor in (created_at, id) order, across all accounts. The leading
-- created_at >= bound keeps it a range read of invoice_created_at_idx (which carries id as its suffix).
SELECT id, created_at
FROM invoice
WHERE created_at >= sqlc.arg('after_created_at')
  AND (created_at > sqlc.arg('after_created_at') OR id > sqlc.arg('after_id'))
ORDER BY created_at, id
LIMIT ?;

-- name: ListInvoiceIDsCreatedSince :many
-- Newest first: when the cap bites, the invoices left out are the oldest in the window, the least
-- likely to be changing, and the reconcile pass still reaches them.
SELECT id FROM invoice WHERE created_at >= ? ORDER BY created_at DESC, id DESC LIMIT ?;

-- name: MarkSalesFactDirty :exec
INSERT INTO sales_fact_dirty (scope_type, scope_id, account_id, marked_at)
VALUES (?, ?, ?, NOW(3))
ON DUPLICATE KEY UPDATE marked_at = VALUES(marked_at);

-- name: ListSalesFactDirty :many
-- The fact scopes only: 'buyer_summary' and 'rollup_day' marks share the table but are drained by their own passes.
SELECT scope_type, scope_id, account_id, marked_at
FROM sales_fact_dirty
WHERE scope_type NOT IN ('buyer_summary', 'rollup_day')
ORDER BY marked_at
LIMIT ?;

-- name: ClearSalesFactDirty :exec
-- Only clears a mark that was not re-marked after it was read.
DELETE FROM sales_fact_dirty WHERE scope_type = ? AND scope_id = ? AND marked_at = ?;

-- name: ListInvoiceIDsBySalesOrders :many
SELECT id FROM invoice WHERE sales_order_id IN (sqlc.slice('sales_order_ids'));

-- name: ListInvoiceIDsBySalesOrderLines :many
SELECT DISTINCT invoice_id FROM invoice_line WHERE sales_order_line_id IN (sqlc.slice('sales_order_line_ids'));

-- name: ListInvoiceIDsByProducts :many
SELECT DISTINCT il.invoice_id
FROM sales_order_line sol
JOIN invoice_line il ON il.sales_order_line_id = sol.id
WHERE sol.product_id IN (sqlc.slice('product_ids'));

-- name: ListSalesFactInvoiceIDsAfter :many
-- A page of the distinct invoices sales_line_fact holds, read from sales_line_fact_invoice_idx alone.
SELECT DISTINCT invoice_id FROM sales_line_fact WHERE invoice_id > ? ORDER BY invoice_id LIMIT ?;

-- name: ListExistingInvoiceIDs :many
SELECT id FROM invoice WHERE id IN (sqlc.slice('invoice_ids'));
