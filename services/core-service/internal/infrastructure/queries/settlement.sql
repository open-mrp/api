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

-- name: InsertSettlement :exec
INSERT INTO settlement (id, number, note, responsible_user_id, account_id, created_at, updated_at)
VALUES (sqlc.arg('id'), sqlc.arg('number'), sqlc.narg('note'), sqlc.narg('responsible_user_id'), sqlc.arg('account_id'), NOW(3), NOW(3));

-- name: UpdateSettlement :exec
UPDATE settlement SET
    number = COALESCE(sqlc.narg('number'), number),
    note = CASE
        WHEN sqlc.arg('clear_note') = 1 THEN NULL
        WHEN sqlc.narg('note') IS NOT NULL THEN sqlc.narg('note')
        ELSE note
    END,
    responsible_user_id = COALESCE(sqlc.narg('responsible_user_id'), responsible_user_id),
    updated_at = NOW(3)
WHERE id = sqlc.arg('id')
AND account_id = sqlc.arg('account_id');

-- name: DeleteSettlement :exec
DELETE FROM settlement WHERE id = sqlc.arg('id') AND account_id = sqlc.arg('account_id');

-- name: InsertTransactionAllocation :exec
-- account_id and transaction_type_code are copied from the transaction, which never changes them, so
-- the allocation entry list can key on them.
INSERT INTO transaction_allocation (id, transaction_id, amount_id, invoice_id, settlement_id, note, created_at, updated_at, account_id, transaction_type_code)
VALUES (sqlc.arg('id'), sqlc.arg('transaction_id'), sqlc.arg('amount_id'), sqlc.arg('invoice_id'), sqlc.arg('settlement_id'), sqlc.narg('note'), COALESCE(sqlc.narg('created_at'), NOW(3)), NOW(3),
    (SELECT t.account_id FROM `transaction` t WHERE t.id = sqlc.arg('transaction_id')),
    (SELECT t.transaction_type_code FROM `transaction` t WHERE t.id = sqlc.arg('transaction_id')));

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

-- name: DeleteSettlementOwnedTransactions :exec
-- Removes, with their amounts, the transactions a deleted settlement leaves with nothing to show for
-- them: those it recorded itself (new_transactions, any type), and adjustments only it drew on. A
-- transaction another settlement or a standalone allocation still draws on is kept. Run before the
-- settlement's allocations are deleted.
DELETE t, q FROM `transaction` t
JOIN quantity q ON q.id = t.amount_id
WHERE t.account_id = sqlc.arg('account_id')
AND (
    t.created_by_settlement_id = sqlc.arg('settlement_id')
    OR (t.transaction_type_code = 'adjustment' AND t.id IN (
        SELECT ta.transaction_id
        FROM transaction_allocation ta
        WHERE ta.settlement_id = sqlc.arg('settlement_id')
    ))
)
AND NOT EXISTS (
    SELECT 1 FROM transaction_allocation ta2
    WHERE ta2.transaction_id = t.id
    AND (ta2.settlement_id IS NULL OR ta2.settlement_id != sqlc.arg('settlement_id'))
);

-- name: MarkTransactionsCreatedBySettlement :exec
UPDATE `transaction`
SET created_by_settlement_id = sqlc.arg('settlement_id')
WHERE account_id = sqlc.arg('account_id')
AND id IN (sqlc.slice('transaction_ids'));

-- name: UpdateTransactionsFullyAllocated :exec
UPDATE `transaction`
SET is_fully_allocated = sqlc.arg('is_fully_allocated'), updated_at = NOW(3)
WHERE account_id = sqlc.arg('account_id')
AND id IN (sqlc.slice('transaction_ids'));

-- name: GetTransactionAllocationTotals :many
-- For a set of transactions, the amount and the sum of every allocation drawn from it, from any
-- settlement. Whether the transaction is fully allocated is decided in Go
-- (domain.TransactionFullyAllocated), which rounds the remainder the way the dashboard does.
SELECT
    t.id AS transaction_id,
    CAST(q.value AS CHAR) AS amount,
    CAST(COALESCE((
        SELECT SUM(aq.value)
        FROM transaction_allocation ta
        JOIN quantity aq ON aq.id = ta.amount_id
        WHERE ta.transaction_id = t.id
    ), 0) AS CHAR) AS allocated_total
FROM `transaction` t
JOIN quantity q ON q.id = t.amount_id
WHERE t.account_id = sqlc.arg('account_id')
AND t.id IN (sqlc.slice('transaction_ids'));

-- name: UpdateInvoicePaymentStatus :exec
UPDATE invoice
SET is_paid_in_full = sqlc.arg('is_paid_in_full'),
    is_over_paid = sqlc.arg('is_over_paid'),
    paid_in_full_marked_by_id = IF(sqlc.arg('clear_mark'), NULL, paid_in_full_marked_by_id),
    updated_at = NOW(3)
WHERE id = sqlc.arg('id')
AND account_id = sqlc.arg('account_id');

-- name: GetDollarUnitID :one
-- Keyed on the well-known id rather than the abbreviation, which is editable per environment.
SELECT id FROM unit WHERE id = 'dollar' LIMIT 1;
