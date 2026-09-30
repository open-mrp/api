-- name: InsertTransaction :exec
INSERT INTO transaction (
    id, number, transaction_type_code, stripe_payment_id,
    customer_account_id, account_id, transaction_method_code, note,
    amount_id, adjustment_type_code, responsible_user_id, funds_received_at, created_at, updated_at
) VALUES (
    sqlc.arg('id'), sqlc.arg('number'), sqlc.arg('transaction_type_code'), sqlc.narg('stripe_payment_id'),
    sqlc.arg('customer_account_id'), sqlc.arg('account_id'), sqlc.narg('transaction_method_code'), sqlc.narg('note'),
    sqlc.arg('amount_id'), sqlc.narg('adjustment_type_code'), sqlc.narg('responsible_user_id'), sqlc.narg('funds_received_at'),
    COALESCE(sqlc.narg('created_at'), NOW(3)), NOW(3)
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

-- name: AllocateNextTransactionNumber :execresult
-- Atomically reserves the next transaction number for the account and returns it via LAST_INSERT_ID.
-- See AllocateNextOrderNumber: the row lock on the per-account counter is what stops two concurrent
-- payments from being recorded under the same number.
INSERT INTO sys_property (id, account_id, sys_property_type_code, value, created_at, updated_at)
VALUES (sqlc.arg('id'), sqlc.arg('account_id'), 'transaction_number', LAST_INSERT_ID(1), NOW(3), NOW(3))
ON DUPLICATE KEY UPDATE value = LAST_INSERT_ID(value + 1), updated_at = NOW(3);

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
    created_at = COALESCE(sqlc.narg('created_at'), created_at),
    funds_received_at = CASE
        WHEN sqlc.arg('clear_funds_received_at') = 1 THEN NULL
        WHEN sqlc.narg('funds_received_at') IS NOT NULL THEN sqlc.narg('funds_received_at')
        ELSE funds_received_at
    END,
    updated_at = NOW(3)
WHERE id = sqlc.arg('id')
AND account_id = sqlc.arg('account_id');

-- name: ResolveResponsibleUserID :one
-- Resolves an account_user id or a user id to the account_user, whatever its status: the caller
-- decides whether an inactive user may stay responsible (they may, if nothing changes).
SELECT au.id, (au.status_code = 'active' OR au.status_code IS NULL) AS is_active
FROM account_user au
WHERE au.account_id = sqlc.arg('account_id')
AND (au.id = sqlc.arg('user_or_account_user_id') OR au.user_id = sqlc.arg('user_or_account_user_id'))
ORDER BY is_active DESC
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

-- name: GetDollarUnitIDForTransaction :one
-- Keyed on the well-known id rather than the abbreviation, which is editable per environment.
SELECT id FROM unit WHERE id = 'dollar' LIMIT 1;

-- name: UpdateTransactionFundsReceivedByStripePaymentIDs :exec
UPDATE transaction
SET funds_received_at = sqlc.arg('funds_received_at'), updated_at = NOW(3)
WHERE account_id = sqlc.arg('account_id')
  AND stripe_payment_id IN (sqlc.slice('stripe_payment_ids'));
