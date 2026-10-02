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
