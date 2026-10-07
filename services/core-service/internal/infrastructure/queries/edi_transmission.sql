-- name: IsCustomerEdiEnabled :one
SELECT ar.is_edi_enabled
FROM account_relation ar
WHERE ar.owner_account_id = sqlc.arg('owner_account_id')
AND ar.counterparty_account_id = sqlc.arg('customer_id')
AND ar.account_relation_role_code = 'customer';

-- name: EnqueueOutboundEdiTransmission :exec
-- Records a document owed to a trading partner, due now. The (account, document, subject) key makes
-- a replayed enqueue land on the row already there, as the dashboard's skipDuplicates does.
INSERT INTO edi_transmission (
    id, account_id, direction, document_type, subject_type, subject_id,
    counterparty_account_id, status, attempts, available_at, created_at, updated_at
) VALUES (
    sqlc.arg('id'), sqlc.arg('account_id'), 'outbound', sqlc.arg('document_type'), sqlc.arg('subject_type'), sqlc.arg('subject_id'),
    sqlc.arg('counterparty_account_id'), 'pending', 0, NOW(3), NOW(3), NOW(3)
)
ON DUPLICATE KEY UPDATE id = id;
