-- name: ListChildAccountsForward :many
SELECT
    ar.id AS relation_id,
    ar.counterparty_account_id AS account_id,
    a.name AS account_name,
    ar.external_number,
    ab.support_email AS email,
    ar.created_at,
    ar.updated_at
FROM account_relation ar
INNER JOIN account a ON a.id = ar.counterparty_account_id
LEFT JOIN account_branding ab ON ab.owner_account_id = ar.counterparty_account_id
WHERE ar.owner_account_id = sqlc.arg('owner_account_id')
  AND ar.parent_account_relation_id = sqlc.arg('parent_relation_id')
  AND (
    sqlc.narg('search_query') IS NULL
    OR a.name LIKE sqlc.narg('search_query')
  )
  AND (
    sqlc.narg('cursor_created_at') IS NULL
    OR ar.created_at < sqlc.narg('cursor_created_at')
    OR (ar.created_at = sqlc.narg('cursor_created_at') AND ar.id < sqlc.narg('cursor_id'))
  )
ORDER BY ar.created_at DESC, ar.id DESC
LIMIT ?;

-- name: ListChildAccountsBackward :many
SELECT
    ar.id AS relation_id,
    ar.counterparty_account_id AS account_id,
    a.name AS account_name,
    ar.external_number,
    ab.support_email AS email,
    ar.created_at,
    ar.updated_at
FROM account_relation ar
INNER JOIN account a ON a.id = ar.counterparty_account_id
LEFT JOIN account_branding ab ON ab.owner_account_id = ar.counterparty_account_id
WHERE ar.owner_account_id = sqlc.arg('owner_account_id')
  AND ar.parent_account_relation_id = sqlc.arg('parent_relation_id')
  AND (
    sqlc.narg('search_query') IS NULL
    OR a.name LIKE sqlc.narg('search_query')
  )
  AND (
    ar.created_at > sqlc.narg('cursor_created_at')
    OR (ar.created_at = sqlc.narg('cursor_created_at') AND ar.id > sqlc.narg('cursor_id'))
  )
ORDER BY ar.created_at ASC, ar.id ASC
LIMIT ?;

-- name: SetParentAccountRelation :exec
UPDATE account_relation
SET parent_account_relation_id = sqlc.arg('parent_relation_id'),
    updated_at = NOW(3)
WHERE id = sqlc.arg('child_relation_id')
  AND owner_account_id = sqlc.arg('owner_account_id');

-- name: ClearParentAccountRelation :exec
UPDATE account_relation
SET parent_account_relation_id = NULL,
    updated_at = NOW(3)
WHERE id = sqlc.arg('child_relation_id')
  AND owner_account_id = sqlc.arg('owner_account_id')
  AND parent_account_relation_id = sqlc.arg('parent_relation_id');

-- name: GetParentAccountRelationID :one
SELECT parent_account_relation_id
FROM account_relation
WHERE id = ?;

-- name: GetChildAccountDetail :one
SELECT
    ar.id AS relation_id,
    ar.counterparty_account_id AS account_id,
    a.name AS account_name,
    ar.external_number,
    ab.support_email AS email,
    ar.created_at,
    ar.updated_at
FROM account_relation ar
INNER JOIN account a ON a.id = ar.counterparty_account_id
LEFT JOIN account_branding ab ON ab.owner_account_id = ar.counterparty_account_id
WHERE ar.owner_account_id = sqlc.arg('owner_account_id')
  AND ar.counterparty_account_id = sqlc.arg('counterparty_account_id');
