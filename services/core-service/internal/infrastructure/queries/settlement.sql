-- name: ListSettlementsForward :many
SELECT
    s.id,
    s.number,
    s.created_at,
    s.updated_at,
    COUNT(ta.id) AS allocation_count,
    SUM(CASE WHEN t.transaction_type_code = 'payment' THEN q.value ELSE 0 END) AS total_payments,
    SUM(CASE WHEN t.transaction_type_code = 'rebate' THEN q.value ELSE 0 END) AS total_rebates,
    SUM(CASE WHEN t.transaction_type_code = 'adjustment' THEN q.value ELSE 0 END) AS total_adjustments,
    SUM(CASE WHEN t.transaction_type_code = 'credit_memo' THEN q.value ELSE 0 END) AS total_credits,
    GROUP_CONCAT(DISTINCT inv.number ORDER BY inv.number SEPARATOR ',') AS invoice_numbers,
    GROUP_CONCAT(DISTINCT buyer.name ORDER BY buyer.name SEPARATOR ',') AS customer_names
FROM settlement s
LEFT JOIN transaction_allocation ta ON ta.settlement_id = s.id
LEFT JOIN `transaction` t ON t.id = ta.transaction_id
LEFT JOIN quantity q ON q.id = ta.amount_id
LEFT JOIN invoice inv ON inv.id = ta.invoice_id
LEFT JOIN sales_order so ON so.id = inv.sales_order_id
LEFT JOIN account buyer ON buyer.id = so.buyer_account_id
WHERE s.account_id = sqlc.arg('account_id')
AND (
    sqlc.narg('search_query') IS NULL
    OR MATCH(s.number, s.note) AGAINST (sqlc.narg('search_query') IN BOOLEAN MODE)
)
AND (
    sqlc.arg('include_transaction_filter') = false
    OR ta.transaction_id IN (sqlc.slice('transaction_ids'))
)
AND (
    sqlc.arg('include_invoice_filter') = false
    OR ta.invoice_id IN (sqlc.slice('invoice_ids'))
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
GROUP BY s.id, s.number, s.created_at, s.updated_at
ORDER BY s.created_at DESC, s.id DESC
LIMIT ?;

-- name: ListSettlementsBackward :many
SELECT
    s.id,
    s.number,
    s.created_at,
    s.updated_at,
    COUNT(ta.id) AS allocation_count,
    SUM(CASE WHEN t.transaction_type_code = 'payment' THEN q.value ELSE 0 END) AS total_payments,
    SUM(CASE WHEN t.transaction_type_code = 'rebate' THEN q.value ELSE 0 END) AS total_rebates,
    SUM(CASE WHEN t.transaction_type_code = 'adjustment' THEN q.value ELSE 0 END) AS total_adjustments,
    SUM(CASE WHEN t.transaction_type_code = 'credit_memo' THEN q.value ELSE 0 END) AS total_credits,
    GROUP_CONCAT(DISTINCT inv.number ORDER BY inv.number SEPARATOR ',') AS invoice_numbers,
    GROUP_CONCAT(DISTINCT buyer.name ORDER BY buyer.name SEPARATOR ',') AS customer_names
FROM settlement s
LEFT JOIN transaction_allocation ta ON ta.settlement_id = s.id
LEFT JOIN `transaction` t ON t.id = ta.transaction_id
LEFT JOIN quantity q ON q.id = ta.amount_id
LEFT JOIN invoice inv ON inv.id = ta.invoice_id
LEFT JOIN sales_order so ON so.id = inv.sales_order_id
LEFT JOIN account buyer ON buyer.id = so.buyer_account_id
WHERE s.account_id = sqlc.arg('account_id')
AND (
    sqlc.narg('search_query') IS NULL
    OR MATCH(s.number, s.note) AGAINST (sqlc.narg('search_query') IN BOOLEAN MODE)
)
AND (
    sqlc.arg('include_transaction_filter') = false
    OR ta.transaction_id IN (sqlc.slice('transaction_ids'))
)
AND (
    sqlc.arg('include_invoice_filter') = false
    OR ta.invoice_id IN (sqlc.slice('invoice_ids'))
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
    OR s.created_at > sqlc.narg('cursor_created_at')
    OR (s.created_at = sqlc.narg('cursor_created_at') AND s.id > sqlc.narg('cursor_id'))
)
GROUP BY s.id, s.number, s.created_at, s.updated_at
ORDER BY s.created_at ASC, s.id ASC
LIMIT ?;

-- name: CountSettlements :one
SELECT COUNT(DISTINCT s.id) AS total_count
FROM settlement s
LEFT JOIN transaction_allocation ta ON ta.settlement_id = s.id
WHERE s.account_id = sqlc.arg('account_id')
AND (
    sqlc.narg('search_query') IS NULL
    OR MATCH(s.number, s.note) AGAINST (sqlc.narg('search_query') IN BOOLEAN MODE)
)
AND (
    sqlc.arg('include_transaction_filter') = false
    OR ta.transaction_id IN (sqlc.slice('transaction_ids'))
)
AND (
    sqlc.arg('include_invoice_filter') = false
    OR ta.invoice_id IN (sqlc.slice('invoice_ids'))
)
AND (
    sqlc.narg('start_date') IS NULL
    OR s.created_at >= sqlc.narg('start_date')
)
AND (
    sqlc.narg('end_date') IS NULL
    OR s.created_at <= sqlc.narg('end_date')
);

-- name: GetSettlement :one
SELECT
    s.id,
    s.number,
    s.note,
    s.responsible_user_id,
    au.id AS responsible_user_account_user_id,
    u.name AS responsible_user_name,
    s.created_at,
    s.updated_at
FROM settlement s
-- responsible_user_id may store either an account_user id (rows written by the
-- v2 API, which resolves on write) or a legacy user id; match both, scoped to
-- the settlement's account, so the account_user loader can populate it.
LEFT JOIN account_user au ON au.account_id = s.account_id AND (au.id = s.responsible_user_id OR au.user_id = s.responsible_user_id)
LEFT JOIN `user` u ON u.id = au.user_id
WHERE s.id = sqlc.arg('id')
AND s.account_id = sqlc.arg('account_id');

-- name: GetSettlementAllocations :many
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
WHERE ta.settlement_id = sqlc.arg('settlement_id')
ORDER BY ta.created_at ASC;

-- name: InsertSettlement :exec
INSERT INTO settlement (id, number, note, responsible_user_id, account_id, created_at, updated_at)
VALUES (sqlc.arg('id'), sqlc.arg('number'), sqlc.narg('note'), sqlc.narg('responsible_user_id'), sqlc.arg('account_id'), NOW(3), NOW(3));

-- name: UpdateSettlement :exec
UPDATE settlement SET
    number = COALESCE(sqlc.narg('number'), number),
    note = COALESCE(sqlc.narg('note'), note),
    responsible_user_id = COALESCE(sqlc.narg('responsible_user_id'), responsible_user_id),
    updated_at = NOW(3)
WHERE id = sqlc.arg('id')
AND account_id = sqlc.arg('account_id');

-- name: DeleteSettlement :exec
DELETE FROM settlement WHERE id = sqlc.arg('id') AND account_id = sqlc.arg('account_id');

-- name: InsertTransactionAllocation :exec
INSERT INTO transaction_allocation (id, transaction_id, amount_id, invoice_id, settlement_id, note, created_at, updated_at)
VALUES (sqlc.arg('id'), sqlc.arg('transaction_id'), sqlc.arg('amount_id'), sqlc.arg('invoice_id'), sqlc.arg('settlement_id'), sqlc.narg('note'), NOW(3), NOW(3));

-- name: InsertAllocationQuantity :exec
INSERT INTO quantity (id, value, unit_id, created_at, updated_at)
VALUES (sqlc.arg('id'), sqlc.arg('value'), sqlc.arg('unit_id'), NOW(3), NOW(3));

-- name: CheckSettlementNumberDuplicate :one
SELECT COUNT(*) > 0 AS result
FROM settlement
WHERE account_id = sqlc.arg('account_id')
AND number = sqlc.arg('number')
AND (sqlc.narg('exclude_id') IS NULL OR id != sqlc.narg('exclude_id'));

-- name: AllocateNextSettlementNumber :execresult
-- Atomically reserves the next settlement number for the account and returns it via LAST_INSERT_ID.
-- The single upsert holds a row lock on the per-account counter, so concurrent creates serialize
-- and never collide on the same number (the old read-MAX-then-write pattern raced). Read the reserved
-- number back with the statement result's LastInsertId().
INSERT INTO sys_property (id, account_id, sys_property_type_code, value, created_at, updated_at)
VALUES (sqlc.arg('id'), sqlc.arg('account_id'), 'settlement_number', LAST_INSERT_ID(1), NOW(3), NOW(3))
ON DUPLICATE KEY UPDATE value = LAST_INSERT_ID(value + 1), updated_at = NOW(3);

-- name: GetSettlementAllocationTransactionIDs :many
SELECT DISTINCT ta.transaction_id
FROM transaction_allocation ta
WHERE ta.settlement_id = sqlc.arg('settlement_id');

-- name: GetSettlementAllocationInvoiceIDs :many
SELECT DISTINCT ta.invoice_id
FROM transaction_allocation ta
WHERE ta.settlement_id = sqlc.arg('settlement_id');

-- name: DeleteSettlementAllocations :many
SELECT ta.id, ta.transaction_id, ta.invoice_id, ta.note,
    q.id AS amount_id, q.value AS amount_value,
    qu.id AS amount_unit_id, qu.abbreviation AS amount_unit_abbreviation,
    t.number AS transaction_number, t.transaction_type_code AS transaction_type,
    inv.number AS invoice_number,
    ta.created_at, ta.updated_at
FROM transaction_allocation ta
JOIN quantity q ON q.id = ta.amount_id
JOIN unit qu ON qu.id = q.unit_id
JOIN `transaction` t ON t.id = ta.transaction_id
JOIN invoice inv ON inv.id = ta.invoice_id
WHERE ta.settlement_id = sqlc.arg('settlement_id');

-- name: DeleteTransactionAllocationsBySettlement :exec
DELETE FROM transaction_allocation WHERE settlement_id = sqlc.arg('settlement_id');

-- name: DeleteQuantitiesBySettlementAllocations :exec
DELETE q FROM quantity q
JOIN transaction_allocation ta ON ta.amount_id = q.id
WHERE ta.settlement_id = sqlc.arg('settlement_id');

-- name: DeleteOrphanedAdjustmentTransactions :exec
DELETE t FROM `transaction` t
WHERE t.transaction_type_code = 'adjustment'
AND t.id IN (
    SELECT ta.transaction_id
    FROM transaction_allocation ta
    WHERE ta.settlement_id = sqlc.arg('settlement_id')
)
AND NOT EXISTS (
    SELECT 1 FROM transaction_allocation ta2
    WHERE ta2.transaction_id = t.id
    AND (ta2.settlement_id IS NULL OR ta2.settlement_id != sqlc.arg('settlement_id'))
);

-- name: UpdateTransactionsFullyAllocated :exec
UPDATE `transaction`
SET is_fully_allocated = sqlc.arg('is_fully_allocated'), updated_at = NOW(3)
WHERE id IN (sqlc.slice('transaction_ids'));

-- name: UpdateInvoicePaymentStatus :exec
UPDATE invoice
SET is_paid_in_full = sqlc.arg('is_paid_in_full'),
    is_over_paid = sqlc.arg('is_over_paid'),
    updated_at = NOW(3)
WHERE id = sqlc.arg('id');

-- name: GetDollarUnitID :one
-- Keyed on the well-known id rather than the abbreviation, which is editable per environment.
SELECT id FROM unit WHERE id = 'dollar' LIMIT 1;
