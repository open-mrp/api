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

-- name: FindAccountRelationByCounterpartyAccountIDAndAPIKeyID :one
SELECT
    account_relation.id,
    account_relation.counterparty_account_id,
    account_relation.owner_account_id,
    account_relation.account_relation_role_code
FROM account_relation
INNER JOIN api_key ON account_relation.owner_account_id = api_key.owner_account_id
WHERE account_relation.counterparty_account_id = ?
  AND api_key.id = ?
LIMIT 1;

-- name: FindAccountRelationByCounterpartyAccountIDAndUserID :one
SELECT
    account_relation.id,
    account_relation.counterparty_account_id,
    account_relation.owner_account_id,
    account_relation.account_relation_role_code
FROM account_relation
INNER JOIN account_user ON account_relation.owner_account_id = account_user.account_id
WHERE account_relation.counterparty_account_id = ?
  AND account_relation.owner_account_id = ?
  AND account_user.user_id = ?
  AND (account_user.status_code = 'active' OR account_user.status_code IS NULL)
LIMIT 1;

-- name: HasRelationByOwnerAndCounterparty :one
SELECT EXISTS(
    SELECT 1
    FROM account_relation
    WHERE owner_account_id = ?
      AND counterparty_account_id = ?
) AS has_relation;

-- name: CountCounterpartyRelationsExcluding :one
SELECT COUNT(*) AS cnt
FROM account_relation
WHERE counterparty_account_id = ? AND owner_account_id != ?;

-- name: FindAccountRelationByOwnerAndCounterparty :one
SELECT id FROM account_relation
WHERE owner_account_id = ? AND counterparty_account_id = ?;

-- name: InsertAccountRelationNotificationPreference :exec
INSERT INTO account_relation_notification_preference (id, account_relation_id, recipient_account_user_id, notification_type_code, created_at, updated_at)
VALUES (?, ?, ?, ?, NOW(3), NOW(3));

-- name: ListNotificationPreferences :many
SELECT id, notification_type_code
FROM account_relation_notification_preference
WHERE account_relation_id = ? AND recipient_account_user_id = ?;

-- name: DeleteNotificationPreference :exec
DELETE FROM account_relation_notification_preference
WHERE account_relation_id = ? AND recipient_account_user_id = ? AND notification_type_code = ?;

-- name: ListNotificationPreferencesByRelation :many
SELECT recipient_account_user_id, notification_type_code
FROM account_relation_notification_preference
WHERE account_relation_id = ?
ORDER BY recipient_account_user_id, notification_type_code;

-- name: DeleteNotificationPreferencesByRelationAndTypes :exec
DELETE FROM account_relation_notification_preference
WHERE account_relation_id = ?
  AND notification_type_code IN (sqlc.slice('notification_type_codes'));

-- name: FindCustomerAccountsByVendorAndUser :many
SELECT a.id, a.name
FROM account a
INNER JOIN account_user au ON au.account_id = a.id
INNER JOIN account_relation ar ON ar.counterparty_account_id = a.id
WHERE a.onboarding_status_code IN ('active', 'unclaimed')
  AND au.user_id = ?
  AND au.status_code = 'active'
  AND ar.owner_account_id = ?
  AND ar.account_relation_role_code = 'customer';

-- name: FindCustomerByEmail :one
SELECT ar.id as relation_id, ar.owner_account_id, ar.counterparty_account_id,
       ar.account_relation_role_code, ar.alias, u.email, u.name as user_name
FROM user u
JOIN account_user au ON au.user_id = u.id
JOIN account_relation ar ON ar.counterparty_account_id = au.account_id
     AND ar.owner_account_id = ?
WHERE u.email = ?
LIMIT 1;

-- name: FindContactsByEmail :many
SELECT
    au.id AS account_user_id,
    au.user_id AS user_id,
    au.account_id AS account_id,
    au.role_id AS role_id,
    au.department_id AS department_id,
    au.status_code AS status_code,
    au.last_used_at AS last_used_at,
    au.created_at AS created_at,
    au.updated_at AS updated_at,
    u.email AS email,
    COALESCE(
        CASE
            WHEN au.account_id = sqlc.arg(owner_account_id) THEN 'self'
            ELSE ar.account_relation_role_code
        END,
        ''
    ) AS relationship
FROM user u
JOIN account_user au ON au.user_id = u.id
LEFT JOIN account_relation ar
    ON ar.counterparty_account_id = au.account_id
    AND ar.owner_account_id = sqlc.arg(owner_account_id)
    AND ar.account_relation_role_code IN ('customer', 'supplier')
WHERE u.email = sqlc.arg(email)
    AND au.status_code = 'active'
    AND (au.account_id = sqlc.arg(owner_account_id) OR ar.id IS NOT NULL)
ORDER BY relationship, au.account_id;

