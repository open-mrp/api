-- name: FindAccountRelationByOwnerAccountIDAndUserID :one
SELECT 
    account_relation.id,
    account_relation.counterparty_account_id,
    account_relation.account_relation_role_code
FROM account_relation
INNER JOIN account_user ON account_relation.counterparty_account_id = account_user.account_id
WHERE account_relation.owner_account_id = ?
  AND account_user.user_id = ?
LIMIT 1;

-- name: FindAccountRelationByOwnerAccountIDAndAPIKeyID :one
SELECT 
    account_relation.id,
    account_relation.counterparty_account_id,
    account_relation.account_relation_role_code
FROM account_relation
INNER JOIN api_key ON account_relation.counterparty_account_id = api_key.owner_account_id
WHERE account_relation.owner_account_id = ?
  AND api_key.id = ?
LIMIT 1;

