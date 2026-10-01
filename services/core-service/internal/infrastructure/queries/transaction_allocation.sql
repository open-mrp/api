-- name: ListAllocationEntriesForward :many
SELECT
    ta.id,
    t.note,
    ta.created_at,
    q.value AS amount_value,
    qu.abbreviation AS amount_unit_abbreviation,
    t.customer_account_id AS customer_id,
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
JOIN account cust_acct ON cust_acct.id = t.customer_account_id
LEFT JOIN account_relation ar ON ar.counterparty_account_id = t.customer_account_id
    AND ar.owner_account_id = t.account_id
    AND ar.account_relation_role_code = 'customer'
WHERE t.account_id = sqlc.arg('account_id')
-- The dashboard found an entry by its invoice or transaction number, or by its customer.
AND (sqlc.narg('search_query') IS NULL OR (
    inv.number = sqlc.narg('search_query')
    OR t.number = sqlc.narg('search_query')
    OR ar.external_number = sqlc.narg('search_query')
    OR cust_acct.name LIKE CONCAT('%', sqlc.narg('search_like'), '%')
))
AND (sqlc.narg('transaction_type') IS NULL OR t.transaction_type_code = sqlc.narg('transaction_type'))
AND (sqlc.narg('start_date') IS NULL OR ta.created_at >= sqlc.narg('start_date'))
AND (sqlc.narg('end_date') IS NULL OR ta.created_at <= sqlc.narg('end_date'))
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
    t.customer_account_id AS customer_id,
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
JOIN account cust_acct ON cust_acct.id = t.customer_account_id
LEFT JOIN account_relation ar ON ar.counterparty_account_id = t.customer_account_id
    AND ar.owner_account_id = t.account_id
    AND ar.account_relation_role_code = 'customer'
WHERE t.account_id = sqlc.arg('account_id')
-- The dashboard found an entry by its invoice or transaction number, or by its customer.
AND (sqlc.narg('search_query') IS NULL OR (
    inv.number = sqlc.narg('search_query')
    OR t.number = sqlc.narg('search_query')
    OR ar.external_number = sqlc.narg('search_query')
    OR cust_acct.name LIKE CONCAT('%', sqlc.narg('search_like'), '%')
))
AND (sqlc.narg('transaction_type') IS NULL OR t.transaction_type_code = sqlc.narg('transaction_type'))
AND (sqlc.narg('start_date') IS NULL OR ta.created_at >= sqlc.narg('start_date'))
AND (sqlc.narg('end_date') IS NULL OR ta.created_at <= sqlc.narg('end_date'))
AND (
    sqlc.narg('cursor_created_at') IS NULL
    OR (ta.created_at > sqlc.narg('cursor_created_at'))
    OR (ta.created_at = sqlc.narg('cursor_created_at') AND ta.id > sqlc.narg('cursor_id'))
)
ORDER BY ta.created_at ASC, ta.id ASC
LIMIT ?;

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
