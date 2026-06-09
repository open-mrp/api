-- name: ListAllocationEntriesForward :many
SELECT
    ta.id,
    t.note,
    ta.created_at,
    q.value AS amount_value,
    qu.abbreviation AS amount_unit_abbreviation,
    cust_acct.name AS customer_name,
    ar.external_number AS customer_number,
    t.id AS transaction_id,
    t.transaction_type_code AS transaction_type,
    t.transaction_method_code AS transaction_method,
    t.adjustment_type_code AS adjustment_type,
    inv.id AS invoice_id,
    inv.number AS invoice_number
FROM transaction_allocation ta
JOIN quantity q ON q.id = ta.amount_id
JOIN unit qu ON qu.id = q.unit_id
JOIN `transaction` t ON t.id = ta.transaction_id
JOIN invoice inv ON inv.id = ta.invoice_id
JOIN sales_order so ON so.id = inv.sales_order_id
JOIN account cust_acct ON cust_acct.id = t.customer_account_id
LEFT JOIN account_relation ar ON ar.counterparty_account_id = t.customer_account_id
    AND ar.owner_account_id = t.account_id
    AND ar.account_relation_role_code = 'customer'
WHERE t.account_id = sqlc.arg('account_id')
AND (sqlc.narg('search_query') IS NULL OR (
    MATCH(inv.number) AGAINST(sqlc.narg('search_query') IN BOOLEAN MODE)
    OR MATCH(t.number) AGAINST(sqlc.narg('search_query') IN BOOLEAN MODE)
))
AND (sqlc.narg('transaction_type') IS NULL OR t.transaction_type_code = sqlc.narg('transaction_type'))
AND (sqlc.narg('start_date') IS NULL OR ta.created_at >= sqlc.narg('start_date'))
AND (sqlc.narg('end_date') IS NULL OR ta.created_at < sqlc.narg('end_date'))
AND (
    sqlc.narg('cursor_created_at') IS NULL
    OR (ta.created_at < sqlc.narg('cursor_created_at'))
    OR (ta.created_at = sqlc.narg('cursor_created_at') AND ta.id < sqlc.narg('cursor_id'))
)
ORDER BY ta.created_at DESC, ta.id DESC
LIMIT ?;

-- name: ListAllocationEntriesBackward :many
SELECT
    ta.id,
    t.note,
    ta.created_at,
    q.value AS amount_value,
    qu.abbreviation AS amount_unit_abbreviation,
    cust_acct.name AS customer_name,
    ar.external_number AS customer_number,
    t.id AS transaction_id,
    t.transaction_type_code AS transaction_type,
    t.transaction_method_code AS transaction_method,
    t.adjustment_type_code AS adjustment_type,
    inv.id AS invoice_id,
    inv.number AS invoice_number
FROM transaction_allocation ta
JOIN quantity q ON q.id = ta.amount_id
JOIN unit qu ON qu.id = q.unit_id
JOIN `transaction` t ON t.id = ta.transaction_id
JOIN invoice inv ON inv.id = ta.invoice_id
JOIN sales_order so ON so.id = inv.sales_order_id
JOIN account cust_acct ON cust_acct.id = t.customer_account_id
LEFT JOIN account_relation ar ON ar.counterparty_account_id = t.customer_account_id
    AND ar.owner_account_id = t.account_id
    AND ar.account_relation_role_code = 'customer'
WHERE t.account_id = sqlc.arg('account_id')
AND (sqlc.narg('search_query') IS NULL OR (
    MATCH(inv.number) AGAINST(sqlc.narg('search_query') IN BOOLEAN MODE)
    OR MATCH(t.number) AGAINST(sqlc.narg('search_query') IN BOOLEAN MODE)
))
AND (sqlc.narg('transaction_type') IS NULL OR t.transaction_type_code = sqlc.narg('transaction_type'))
AND (sqlc.narg('start_date') IS NULL OR ta.created_at >= sqlc.narg('start_date'))
AND (sqlc.narg('end_date') IS NULL OR ta.created_at < sqlc.narg('end_date'))
AND (
    sqlc.narg('cursor_created_at') IS NULL
    OR (ta.created_at > sqlc.narg('cursor_created_at'))
    OR (ta.created_at = sqlc.narg('cursor_created_at') AND ta.id > sqlc.narg('cursor_id'))
)
ORDER BY ta.created_at ASC, ta.id ASC
LIMIT ?;

-- name: GetTransactionAllocationByID :one
SELECT
    ta.id,
    ta.note,
    ta.created_at,
    ta.updated_at,
    q.id AS amount_id,
    q.value AS amount_value,
    qu.id AS amount_unit_id,
    qu.abbreviation AS amount_unit_abbreviation,
    t.id AS transaction_id,
    t.number AS transaction_number,
    t.transaction_type_code AS transaction_type,
    inv.id AS invoice_id,
    inv.number AS invoice_number
FROM transaction_allocation ta
JOIN quantity q ON q.id = ta.amount_id
JOIN unit qu ON qu.id = q.unit_id
JOIN `transaction` t ON t.id = ta.transaction_id
JOIN invoice inv ON inv.id = ta.invoice_id
WHERE ta.id = sqlc.arg('id')
AND t.account_id = sqlc.arg('account_id');

-- name: UpdateAllocationAmount :exec
UPDATE quantity SET value = sqlc.arg('value'), updated_at = NOW(3) WHERE id = sqlc.arg('id');

-- name: DeleteTransactionAllocation :exec
DELETE ta FROM transaction_allocation ta
JOIN `transaction` t ON t.id = ta.transaction_id
WHERE ta.id = sqlc.arg('id')
AND t.account_id = sqlc.arg('account_id');

-- name: DeleteTransactionAllocationQuantity :exec
DELETE q FROM quantity q
JOIN transaction_allocation ta ON ta.amount_id = q.id
WHERE ta.id = sqlc.arg('allocation_id');

-- name: ListOpenCredits :many
SELECT
    t.id,
    t.number,
    t.note,
    t.stripe_payment_id,
    t.created_at,
    tt.name AS transaction_type,
    q.value AS original_amount,
    t.customer_account_id AS customer_id,
    cust_acct.name AS customer_name,
    ar.external_number AS customer_number,
    tm.name AS transaction_method,
    adjt.name AS adjustment_type,
    COALESCE(au_user.username, '') AS responsible_user_name,
    COALESCE(alloc_sum.allocated_amount, 0) AS allocated_amount
FROM `transaction` t
JOIN quantity q ON q.id = t.amount_id
JOIN account cust_acct ON cust_acct.id = t.customer_account_id
JOIN transaction_type tt ON tt.code = t.transaction_type_code
LEFT JOIN account_relation ar ON ar.counterparty_account_id = t.customer_account_id
    AND ar.owner_account_id = t.account_id
    AND ar.account_relation_role_code = 'customer'
LEFT JOIN transaction_method tm ON tm.code = t.transaction_method_code
LEFT JOIN adjustment_type adjt ON adjt.code = t.adjustment_type_code
LEFT JOIN account_user au ON au.id = t.responsible_user_id
LEFT JOIN `user` au_user ON au_user.id = au.user_id
LEFT JOIN (
    SELECT ta2.transaction_id, SUM(q2.value) AS allocated_amount
    FROM transaction_allocation ta2
    JOIN quantity q2 ON q2.id = ta2.amount_id
    GROUP BY ta2.transaction_id
) alloc_sum ON alloc_sum.transaction_id = t.id
WHERE t.account_id = sqlc.arg('account_id')
AND t.is_fully_allocated = false
AND (sqlc.arg('include_customer_filter') = false OR t.customer_account_id IN (sqlc.slice('customer_ids')))
AND (sqlc.narg('start_date') IS NULL OR t.created_at >= sqlc.narg('start_date'))
AND (sqlc.narg('end_date') IS NULL OR t.created_at < sqlc.narg('end_date'))
AND (
    sqlc.narg('search') IS NULL
    OR t.id LIKE CONCAT('%', sqlc.narg('search'), '%')
    OR t.number LIKE CONCAT('%', sqlc.narg('search'), '%')
    OR cust_acct.name LIKE CONCAT('%', sqlc.narg('search'), '%')
    OR COALESCE(t.note, '') LIKE CONCAT('%', sqlc.narg('search'), '%')
)
AND (
    sqlc.narg('cursor_created_at') IS NULL
    OR t.created_at < sqlc.narg('cursor_created_at')
    OR (t.created_at = sqlc.narg('cursor_created_at') AND t.id < sqlc.narg('cursor_id'))
)
ORDER BY t.created_at DESC, t.id DESC
LIMIT ?;

-- name: GetOpenCreditAllocations :many
SELECT
    ta.transaction_id,
    inv.number AS invoice_number,
    q.value AS amount
FROM transaction_allocation ta
JOIN quantity q ON q.id = ta.amount_id
JOIN invoice inv ON inv.id = ta.invoice_id
WHERE ta.transaction_id IN (sqlc.slice('transaction_ids'))
ORDER BY ta.transaction_id, ta.created_at ASC;
