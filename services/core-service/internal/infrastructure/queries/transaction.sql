-- name: InsertTransaction :exec
INSERT INTO transaction (
    id, number, transaction_type_code, stripe_payment_id,
    customer_account_id, account_id, transaction_method_code, note,
    amount_id, adjustment_type_code, responsible_user_id, created_at, updated_at
) VALUES (
    sqlc.arg('id'), sqlc.arg('number'), sqlc.arg('transaction_type_code'), sqlc.narg('stripe_payment_id'),
    sqlc.arg('customer_account_id'), sqlc.arg('account_id'), sqlc.narg('transaction_method_code'), sqlc.narg('note'),
    sqlc.arg('amount_id'), sqlc.narg('adjustment_type_code'), sqlc.narg('responsible_user_id'), NOW(3), NOW(3)
);

-- name: InsertTransactionQuantity :exec
INSERT INTO quantity (id, `value`, unit_id, created_at, updated_at)
VALUES (sqlc.arg('id'), sqlc.arg('value'), sqlc.arg('unit_id'), NOW(3), NOW(3));

-- name: FindTransactionByStripePaymentID :one
SELECT id, number, amount_id
FROM transaction
WHERE stripe_payment_id = sqlc.arg('stripe_payment_id')
LIMIT 1;

-- name: UpdateTransactionNote :exec
UPDATE transaction SET note = sqlc.arg('note'), updated_at = NOW(3)
WHERE id = sqlc.arg('id');

-- name: DeleteTransaction :exec
DELETE FROM transaction WHERE id = sqlc.arg('id');

-- name: DeleteTransactionAllocationsByTransactionID :exec
DELETE FROM transaction_allocation WHERE transaction_id = sqlc.arg('transaction_id');

-- name: DeleteTransactionQuantity :exec
DELETE FROM quantity WHERE id = sqlc.arg('id');

-- name: GetNextTransactionNumber :one
SELECT COALESCE(
    (SELECT MAX(CAST(sp.value AS UNSIGNED)) + 1
     FROM sys_property sp
     WHERE sp.account_id = sqlc.arg('account_id')
     AND sp.sys_property_type_code = 'transaction_number'),
    1
) AS next_number;

-- name: UpsertTransactionNumber :exec
INSERT INTO sys_property (id, account_id, sys_property_type_code, value, created_at, updated_at)
VALUES (sqlc.arg('id'), sqlc.arg('account_id'), 'transaction_number', sqlc.arg('value'), NOW(3), NOW(3))
ON DUPLICATE KEY UPDATE value = sqlc.arg('value'), updated_at = NOW(3);

-- name: IsDuplicateTransactionNumber :one
SELECT COUNT(*) AS cnt FROM transaction
WHERE account_id = sqlc.arg('account_id')
AND number = sqlc.arg('number');

-- name: ListTransactionsForward :many
SELECT
    t.id,
    t.number,
    q.id AS amount_id,
    q.value AS amount_value,
    q.unit_id AS amount_unit_id,
    u.abbreviation AS amount_unit_abbreviation,
    t.customer_account_id,
    ba.name AS customer_name,
    ar.external_number AS customer_number,
    ar.account_status_code AS customer_status,
    ar.commission_status_code AS customer_commission_policy,
    ar.created_at AS customer_created_at,
    ar.updated_at AS customer_updated_at,
    tt.id AS transaction_type_id,
    tt.code AS transaction_type_code,
    tt.name AS transaction_type_name,
    tt.is_commission_affected AS transaction_type_is_commission_affected,
    tm.id AS transaction_method_id,
    tm.code AS transaction_method_code,
    tm.name AS transaction_method_name,
    at2.id AS adjustment_type_id,
    at2.code AS adjustment_type_code,
    at2.name AS adjustment_type_name,
    t.is_fully_allocated,
    (SELECT COUNT(*) FROM transaction_allocation ta WHERE ta.transaction_id = t.id) AS allocation_count,
    t.created_at,
    t.updated_at
FROM transaction t
JOIN quantity q ON q.id = t.amount_id
JOIN unit u ON u.id = q.unit_id
JOIN transaction_type tt ON tt.code = t.transaction_type_code
JOIN account ba ON ba.id = t.customer_account_id
JOIN account_relation ar ON ar.owner_account_id = t.account_id AND ar.counterparty_account_id = t.customer_account_id
LEFT JOIN transaction_method tm ON tm.code = t.transaction_method_code
LEFT JOIN adjustment_type at2 ON at2.code = t.adjustment_type_code
WHERE t.account_id = sqlc.arg('account_id')
AND (sqlc.narg('cursor') IS NULL OR t.id < sqlc.narg('cursor'))
AND (sqlc.narg('query') IS NULL OR MATCH(t.number, t.note) AGAINST(sqlc.narg('query') IN BOOLEAN MODE))
AND (sqlc.narg('status') IS NULL OR
    (sqlc.narg('status') = 'allocated' AND t.is_fully_allocated = 1) OR
    (sqlc.narg('status') = 'unallocated' AND t.is_fully_allocated = 0))
AND (
    sqlc.arg('include_type_codes_filter') = false
    OR t.transaction_type_code IN (sqlc.slice('type_codes'))
)
AND (
    sqlc.arg('include_adjustment_type_codes_filter') = false
    OR t.adjustment_type_code IN (sqlc.slice('adjustment_type_codes'))
)
AND (
    sqlc.arg('include_method_codes_filter') = false
    OR t.transaction_method_code IN (sqlc.slice('method_codes'))
)
AND (
    sqlc.arg('include_customer_ids_filter') = false
    OR t.customer_account_id IN (sqlc.slice('customer_ids'))
)
AND (
    sqlc.arg('include_customer_group_ids_filter') = false
    OR ar.account_group_id IN (sqlc.slice('customer_group_ids'))
)
AND (sqlc.narg('start_date') IS NULL OR t.created_at >= sqlc.narg('start_date'))
AND (sqlc.narg('end_date') IS NULL OR t.created_at <= sqlc.narg('end_date'))
ORDER BY t.created_at DESC, t.id DESC
LIMIT ?;

-- name: ListTransactionsBackward :many
SELECT
    t.id,
    t.number,
    q.id AS amount_id,
    q.value AS amount_value,
    q.unit_id AS amount_unit_id,
    u.abbreviation AS amount_unit_abbreviation,
    t.customer_account_id,
    ba.name AS customer_name,
    ar.external_number AS customer_number,
    ar.account_status_code AS customer_status,
    ar.commission_status_code AS customer_commission_policy,
    ar.created_at AS customer_created_at,
    ar.updated_at AS customer_updated_at,
    tt.id AS transaction_type_id,
    tt.code AS transaction_type_code,
    tt.name AS transaction_type_name,
    tt.is_commission_affected AS transaction_type_is_commission_affected,
    tm.id AS transaction_method_id,
    tm.code AS transaction_method_code,
    tm.name AS transaction_method_name,
    at2.id AS adjustment_type_id,
    at2.code AS adjustment_type_code,
    at2.name AS adjustment_type_name,
    t.is_fully_allocated,
    (SELECT COUNT(*) FROM transaction_allocation ta WHERE ta.transaction_id = t.id) AS allocation_count,
    t.created_at,
    t.updated_at
FROM transaction t
JOIN quantity q ON q.id = t.amount_id
JOIN unit u ON u.id = q.unit_id
JOIN transaction_type tt ON tt.code = t.transaction_type_code
JOIN account ba ON ba.id = t.customer_account_id
JOIN account_relation ar ON ar.owner_account_id = t.account_id AND ar.counterparty_account_id = t.customer_account_id
LEFT JOIN transaction_method tm ON tm.code = t.transaction_method_code
LEFT JOIN adjustment_type at2 ON at2.code = t.adjustment_type_code
WHERE t.account_id = sqlc.arg('account_id')
AND t.id > sqlc.arg('cursor')
AND (sqlc.narg('query') IS NULL OR MATCH(t.number, t.note) AGAINST(sqlc.narg('query') IN BOOLEAN MODE))
AND (sqlc.narg('status') IS NULL OR
    (sqlc.narg('status') = 'allocated' AND t.is_fully_allocated = 1) OR
    (sqlc.narg('status') = 'unallocated' AND t.is_fully_allocated = 0))
AND (
    sqlc.arg('include_type_codes_filter') = false
    OR t.transaction_type_code IN (sqlc.slice('type_codes'))
)
AND (
    sqlc.arg('include_adjustment_type_codes_filter') = false
    OR t.adjustment_type_code IN (sqlc.slice('adjustment_type_codes'))
)
AND (
    sqlc.arg('include_method_codes_filter') = false
    OR t.transaction_method_code IN (sqlc.slice('method_codes'))
)
AND (
    sqlc.arg('include_customer_ids_filter') = false
    OR t.customer_account_id IN (sqlc.slice('customer_ids'))
)
AND (
    sqlc.arg('include_customer_group_ids_filter') = false
    OR ar.account_group_id IN (sqlc.slice('customer_group_ids'))
)
AND (sqlc.narg('start_date') IS NULL OR t.created_at >= sqlc.narg('start_date'))
AND (sqlc.narg('end_date') IS NULL OR t.created_at <= sqlc.narg('end_date'))
ORDER BY t.created_at ASC, t.id ASC
LIMIT ?;

-- name: CountTransactions :one
SELECT COUNT(*) AS cnt
FROM transaction t
JOIN account_relation ar ON ar.owner_account_id = t.account_id AND ar.counterparty_account_id = t.customer_account_id
WHERE t.account_id = sqlc.arg('account_id')
AND (sqlc.narg('query') IS NULL OR MATCH(t.number, t.note) AGAINST(sqlc.narg('query') IN BOOLEAN MODE))
AND (sqlc.narg('status') IS NULL OR
    (sqlc.narg('status') = 'allocated' AND t.is_fully_allocated = 1) OR
    (sqlc.narg('status') = 'unallocated' AND t.is_fully_allocated = 0))
AND (
    sqlc.arg('include_type_codes_filter') = false
    OR t.transaction_type_code IN (sqlc.slice('type_codes'))
)
AND (
    sqlc.arg('include_adjustment_type_codes_filter') = false
    OR t.adjustment_type_code IN (sqlc.slice('adjustment_type_codes'))
)
AND (
    sqlc.arg('include_method_codes_filter') = false
    OR t.transaction_method_code IN (sqlc.slice('method_codes'))
)
AND (
    sqlc.arg('include_customer_ids_filter') = false
    OR t.customer_account_id IN (sqlc.slice('customer_ids'))
)
AND (
    sqlc.arg('include_customer_group_ids_filter') = false
    OR ar.account_group_id IN (sqlc.slice('customer_group_ids'))
)
AND (sqlc.narg('start_date') IS NULL OR t.created_at >= sqlc.narg('start_date'))
AND (sqlc.narg('end_date') IS NULL OR t.created_at <= sqlc.narg('end_date'));

-- name: FindTransactionByID :one
SELECT
    t.id,
    t.number,
    q.id AS amount_id,
    q.value AS amount_value,
    q.unit_id AS amount_unit_id,
    u.abbreviation AS amount_unit_abbreviation,
    t.customer_account_id,
    ba.name AS customer_name,
    ar.external_number AS customer_number,
    ar.account_status_code AS customer_status,
    ar.commission_status_code AS customer_commission_policy,
    ar.created_at AS customer_created_at,
    ar.updated_at AS customer_updated_at,
    t.responsible_user_id,
    au.id AS responsible_account_user_id,
    COALESCE(usr.name, '') AS responsible_user_name,
    au.status_code AS responsible_user_status,
    au.created_at AS responsible_user_created_at,
    au.updated_at AS responsible_user_updated_at,
    t.note,
    tt.id AS transaction_type_id,
    tt.code AS transaction_type_code,
    tt.name AS transaction_type_name,
    tt.is_commission_affected AS transaction_type_is_commission_affected,
    tm.id AS transaction_method_id,
    tm.code AS transaction_method_code,
    tm.name AS transaction_method_name,
    at2.id AS adjustment_type_id,
    at2.code AS adjustment_type_code,
    at2.name AS adjustment_type_name,
    t.is_fully_allocated,
    t.stripe_payment_id,
    (SELECT COUNT(*) FROM transaction_allocation ta WHERE ta.transaction_id = t.id) AS allocation_count,
    t.created_at,
    t.updated_at
FROM transaction t
JOIN quantity q ON q.id = t.amount_id
JOIN unit u ON u.id = q.unit_id
JOIN transaction_type tt ON tt.code = t.transaction_type_code
JOIN account ba ON ba.id = t.customer_account_id
JOIN account_relation ar ON ar.owner_account_id = t.account_id AND ar.counterparty_account_id = t.customer_account_id
LEFT JOIN transaction_method tm ON tm.code = t.transaction_method_code
LEFT JOIN adjustment_type at2 ON at2.code = t.adjustment_type_code
-- responsible_user_id may store either an account_user id (rows written by
-- the v2 API, which resolves on write) or a legacy user id; match both,
-- scoped to the transaction's account.
LEFT JOIN account_user au ON au.account_id = t.account_id AND (au.id = t.responsible_user_id OR au.user_id = t.responsible_user_id)
LEFT JOIN user usr ON usr.id = au.user_id
WHERE t.id = sqlc.arg('id')
AND t.account_id = sqlc.arg('account_id');

-- name: GetTransactionAllocations :many
SELECT
    ta.id,
    taq.id AS amount_id,
    taq.value AS amount_value,
    taq.unit_id AS amount_unit_id,
    tau.abbreviation AS amount_unit_abbreviation,
    ta.note,
    ta.transaction_id,
    t.number AS transaction_number,
    t.transaction_type_code AS transaction_type,
    ta.invoice_id,
    COALESCE(i.number, '') AS invoice_number,
    ta.created_at,
    ta.updated_at
FROM transaction_allocation ta
JOIN quantity taq ON taq.id = ta.amount_id
JOIN unit tau ON tau.id = taq.unit_id
JOIN transaction t ON t.id = ta.transaction_id
LEFT JOIN invoice i ON i.id = ta.invoice_id
WHERE ta.transaction_id = sqlc.arg('transaction_id')
ORDER BY ta.created_at DESC;

-- name: UpdateTransaction :exec
UPDATE transaction SET
    number = COALESCE(sqlc.narg('number'), number),
    note = CASE WHEN sqlc.narg('update_note') IS NOT NULL THEN sqlc.narg('note') ELSE note END,
    transaction_method_code = CASE
        WHEN sqlc.arg('clear_transaction_method') = 1 THEN NULL
        WHEN sqlc.narg('transaction_method_code') IS NOT NULL THEN sqlc.narg('transaction_method_code')
        ELSE transaction_method_code
    END,
    adjustment_type_code = CASE
        WHEN sqlc.arg('clear_adjustment_type') = 1 THEN NULL
        WHEN sqlc.narg('adjustment_type_code') IS NOT NULL THEN sqlc.narg('adjustment_type_code')
        ELSE adjustment_type_code
    END,
    responsible_user_id = CASE
        WHEN sqlc.arg('clear_responsible_user') = 1 THEN NULL
        WHEN sqlc.narg('responsible_user_id') IS NOT NULL THEN sqlc.narg('responsible_user_id')
        ELSE responsible_user_id
    END,
    is_fully_allocated = COALESCE(sqlc.narg('is_fully_allocated'), is_fully_allocated),
    updated_at = NOW(3)
WHERE id = sqlc.arg('id')
AND account_id = sqlc.arg('account_id');

-- name: ResolveResponsibleUserID :one
SELECT au.id
FROM account_user au
WHERE au.account_id = sqlc.arg('account_id')
AND (au.id = sqlc.arg('user_or_account_user_id') OR au.user_id = sqlc.arg('user_or_account_user_id'))
AND (au.status_code = 'active' OR au.status_code IS NULL)
LIMIT 1;

-- name: UpdateTransactionQuantity :exec
UPDATE quantity SET
    value = sqlc.arg('value'),
    updated_at = NOW(3)
WHERE id = sqlc.arg('id');

-- name: GetTransactionAmountID :one
SELECT amount_id FROM transaction WHERE id = sqlc.arg('id');

-- name: ExistsTransactionByNumber :one
SELECT COUNT(*) AS cnt FROM transaction
WHERE account_id = sqlc.arg('account_id')
AND number = sqlc.arg('number')
AND (sqlc.narg('exclude_id') IS NULL OR id != sqlc.narg('exclude_id'));

-- name: ListAccountTransactionsForward :many
SELECT
    t.id,
    t.number,
    q.id AS amount_id,
    q.value AS amount_value,
    q.unit_id AS amount_unit_id,
    u.abbreviation AS amount_unit_abbreviation,
    t.customer_account_id,
    ba.name AS customer_name,
    ar.external_number AS customer_number,
    ar.account_status_code AS customer_status,
    ar.commission_status_code AS customer_commission_policy,
    ar.created_at AS customer_created_at,
    ar.updated_at AS customer_updated_at,
    t.responsible_user_id,
    au.id AS responsible_account_user_id,
    COALESCE(usr.name, '') AS responsible_user_name,
    au.status_code AS responsible_user_status,
    au.created_at AS responsible_user_created_at,
    au.updated_at AS responsible_user_updated_at,
    t.note,
    tt.id AS transaction_type_id,
    tt.code AS transaction_type_code,
    tt.name AS transaction_type_name,
    tt.is_commission_affected AS transaction_type_is_commission_affected,
    tm.id AS transaction_method_id,
    tm.code AS transaction_method_code,
    tm.name AS transaction_method_name,
    at2.id AS adjustment_type_id,
    at2.code AS adjustment_type_code,
    at2.name AS adjustment_type_name,
    t.is_fully_allocated,
    t.stripe_payment_id,
    (SELECT COUNT(*) FROM transaction_allocation ta WHERE ta.transaction_id = t.id) AS allocation_count,
    t.created_at,
    t.updated_at
FROM transaction t
JOIN quantity q ON q.id = t.amount_id
JOIN unit u ON u.id = q.unit_id
JOIN transaction_type tt ON tt.code = t.transaction_type_code
JOIN account ba ON ba.id = t.customer_account_id
JOIN account_relation ar ON ar.owner_account_id = t.account_id AND ar.counterparty_account_id = t.customer_account_id
LEFT JOIN transaction_method tm ON tm.code = t.transaction_method_code
LEFT JOIN adjustment_type at2 ON at2.code = t.adjustment_type_code
-- responsible_user_id may store either an account_user id (rows written by
-- the v2 API, which resolves on write) or a legacy user id; match both,
-- scoped to the transaction's account.
LEFT JOIN account_user au ON au.account_id = t.account_id AND (au.id = t.responsible_user_id OR au.user_id = t.responsible_user_id)
LEFT JOIN user usr ON usr.id = au.user_id
WHERE t.account_id = sqlc.arg('account_id')
AND (
    t.customer_account_id = sqlc.arg('customer_account_id')
    OR (sqlc.arg('include_child_accounts') = 1 AND t.customer_account_id IN (
        SELECT ar2.counterparty_account_id FROM account_relation ar2
        WHERE ar2.parent_account_relation_id IN (
            SELECT ar3.id FROM account_relation ar3
            WHERE ar3.counterparty_account_id = sqlc.arg('customer_account_id')
            AND ar3.owner_account_id = t.account_id
        )
    ))
)
AND (sqlc.narg('cursor') IS NULL OR t.id < sqlc.narg('cursor'))
AND (sqlc.narg('query') IS NULL OR MATCH(t.number, t.note) AGAINST(sqlc.narg('query') IN BOOLEAN MODE))
AND (sqlc.narg('status') IS NULL OR
    (sqlc.narg('status') = 'allocated' AND t.is_fully_allocated = 1) OR
    (sqlc.narg('status') = 'unallocated' AND t.is_fully_allocated = 0))
AND (sqlc.narg('type') IS NULL OR t.transaction_type_code = sqlc.narg('type'))
ORDER BY t.created_at DESC, t.id DESC
LIMIT ?;

-- name: ListAccountTransactionsBackward :many
SELECT
    t.id,
    t.number,
    q.id AS amount_id,
    q.value AS amount_value,
    q.unit_id AS amount_unit_id,
    u.abbreviation AS amount_unit_abbreviation,
    t.customer_account_id,
    ba.name AS customer_name,
    ar.external_number AS customer_number,
    ar.account_status_code AS customer_status,
    ar.commission_status_code AS customer_commission_policy,
    ar.created_at AS customer_created_at,
    ar.updated_at AS customer_updated_at,
    t.responsible_user_id,
    au.id AS responsible_account_user_id,
    COALESCE(usr.name, '') AS responsible_user_name,
    au.status_code AS responsible_user_status,
    au.created_at AS responsible_user_created_at,
    au.updated_at AS responsible_user_updated_at,
    t.note,
    tt.id AS transaction_type_id,
    tt.code AS transaction_type_code,
    tt.name AS transaction_type_name,
    tt.is_commission_affected AS transaction_type_is_commission_affected,
    tm.id AS transaction_method_id,
    tm.code AS transaction_method_code,
    tm.name AS transaction_method_name,
    at2.id AS adjustment_type_id,
    at2.code AS adjustment_type_code,
    at2.name AS adjustment_type_name,
    t.is_fully_allocated,
    t.stripe_payment_id,
    (SELECT COUNT(*) FROM transaction_allocation ta WHERE ta.transaction_id = t.id) AS allocation_count,
    t.created_at,
    t.updated_at
FROM transaction t
JOIN quantity q ON q.id = t.amount_id
JOIN unit u ON u.id = q.unit_id
JOIN transaction_type tt ON tt.code = t.transaction_type_code
JOIN account ba ON ba.id = t.customer_account_id
JOIN account_relation ar ON ar.owner_account_id = t.account_id AND ar.counterparty_account_id = t.customer_account_id
LEFT JOIN transaction_method tm ON tm.code = t.transaction_method_code
LEFT JOIN adjustment_type at2 ON at2.code = t.adjustment_type_code
-- responsible_user_id may store either an account_user id (rows written by
-- the v2 API, which resolves on write) or a legacy user id; match both,
-- scoped to the transaction's account.
LEFT JOIN account_user au ON au.account_id = t.account_id AND (au.id = t.responsible_user_id OR au.user_id = t.responsible_user_id)
LEFT JOIN user usr ON usr.id = au.user_id
WHERE t.account_id = sqlc.arg('account_id')
AND (
    t.customer_account_id = sqlc.arg('customer_account_id')
    OR (sqlc.arg('include_child_accounts') = 1 AND t.customer_account_id IN (
        SELECT ar2.counterparty_account_id FROM account_relation ar2
        WHERE ar2.parent_account_relation_id IN (
            SELECT ar3.id FROM account_relation ar3
            WHERE ar3.counterparty_account_id = sqlc.arg('customer_account_id')
            AND ar3.owner_account_id = t.account_id
        )
    ))
)
AND t.id > sqlc.arg('cursor')
AND (sqlc.narg('query') IS NULL OR MATCH(t.number, t.note) AGAINST(sqlc.narg('query') IN BOOLEAN MODE))
AND (sqlc.narg('status') IS NULL OR
    (sqlc.narg('status') = 'allocated' AND t.is_fully_allocated = 1) OR
    (sqlc.narg('status') = 'unallocated' AND t.is_fully_allocated = 0))
AND (sqlc.narg('type') IS NULL OR t.transaction_type_code = sqlc.narg('type'))
ORDER BY t.created_at ASC, t.id ASC
LIMIT ?;

-- name: CountAccountTransactions :one
SELECT COUNT(*) AS cnt
FROM transaction t
WHERE t.account_id = sqlc.arg('account_id')
AND (
    t.customer_account_id = sqlc.arg('customer_account_id')
    OR (sqlc.arg('include_child_accounts') = 1 AND t.customer_account_id IN (
        SELECT ar2.counterparty_account_id FROM account_relation ar2
        WHERE ar2.parent_account_relation_id IN (
            SELECT ar3.id FROM account_relation ar3
            WHERE ar3.counterparty_account_id = sqlc.arg('customer_account_id')
            AND ar3.owner_account_id = t.account_id
        )
    ))
)
AND (sqlc.narg('query') IS NULL OR MATCH(t.number, t.note) AGAINST(sqlc.narg('query') IN BOOLEAN MODE))
AND (sqlc.narg('status') IS NULL OR
    (sqlc.narg('status') = 'allocated' AND t.is_fully_allocated = 1) OR
    (sqlc.narg('status') = 'unallocated' AND t.is_fully_allocated = 0))
AND (sqlc.narg('type') IS NULL OR t.transaction_type_code = sqlc.narg('type'));

-- name: GetDollarUnitIDForTransaction :one
SELECT id FROM unit WHERE abbreviation = '$' LIMIT 1;

-- name: UpdateTransactionFundsReceivedByStripePaymentIDs :exec
UPDATE transaction
SET funds_received_at = sqlc.arg('funds_received_at'), updated_at = NOW(3)
WHERE account_id = sqlc.arg('account_id')
  AND stripe_payment_id IN (sqlc.slice('stripe_payment_ids'));
