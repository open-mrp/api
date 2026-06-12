-- name: FindAuditEventByID :one
SELECT ae.type_id,
       ae.actor_id AS actor_id,
       ae.actor_type,
       ae.identity_type,
       ae.account_id,
       ae.action,
       ae.resource_type,
       ae.resource_id,
       COALESCE(CASE WHEN sqlc.arg('include_changes') THEN ae.changes ELSE NULL END, '') AS changes,
       COALESCE(CASE WHEN sqlc.arg('include_metadata') THEN ae.metadata ELSE NULL END, '') AS metadata,
       ae.service_name,
       ae.request_id,
       ae.idempotency_key_id,
       ae.source_ip,
       ae.occurred_at,
       ae.created_at,
       ae.target_account_id,
       a.name AS account_name,
       a.created_at AS account_created_at,
       a.updated_at AS account_updated_at,
       u.name AS user_name,
       u.email AS user_email,
       ak.name AS api_key_name,
       ak.redacted_value AS api_key_redacted_value,
       ik.idempotency_key
FROM audit_event ae
-- actor_id stores the raw actor key (user_id / api_key.type_id) — exposed
-- directly. Enrichment joins key on it: user by id, api_key by type_id.
LEFT JOIN `user` u ON ae.actor_id = u.id AND ae.identity_type = 'user'
LEFT JOIN api_key ak ON ae.actor_id = ak.type_id AND ae.identity_type = 'api_key'
LEFT JOIN idempotency_key ik ON ae.idempotency_key_id = ik.type_id
-- Resolves the target account (target_account_id) for the `account` sub-resource.
LEFT JOIN account a ON ae.target_account_id = a.id
WHERE ae.type_id = ? AND ae.account_id = ?;

-- name: ListAuditEventsForward :many
SELECT ae.type_id,
       ae.actor_id AS actor_id,
       ae.actor_type,
       ae.identity_type,
       ae.account_id,
       ae.action,
       ae.resource_type,
       ae.resource_id,
       COALESCE(CASE WHEN sqlc.arg('include_changes') THEN ae.changes ELSE NULL END, '') AS changes,
       COALESCE(CASE WHEN sqlc.arg('include_metadata') THEN ae.metadata ELSE NULL END, '') AS metadata,
       ae.service_name,
       ae.request_id,
       ae.idempotency_key_id,
       ae.source_ip,
       ae.occurred_at,
       ae.created_at,
       ae.target_account_id,
       a.name AS account_name,
       a.created_at AS account_created_at,
       a.updated_at AS account_updated_at,
       u.name AS user_name,
       u.email AS user_email,
       ak.name AS api_key_name,
       ak.redacted_value AS api_key_redacted_value,
       ik.idempotency_key
FROM audit_event ae
-- actor_id stores the raw actor key (user_id / api_key.type_id) — exposed
-- directly. Enrichment joins key on it: user by id, api_key by type_id.
LEFT JOIN `user` u ON ae.actor_id = u.id AND ae.identity_type = 'user'
LEFT JOIN api_key ak ON ae.actor_id = ak.type_id AND ae.identity_type = 'api_key'
LEFT JOIN idempotency_key ik ON ae.idempotency_key_id = ik.type_id
-- Resolves the target account (target_account_id) for the `account` sub-resource.
LEFT JOIN account a ON ae.target_account_id = a.id
WHERE ae.account_id = sqlc.arg('target_account_id')
AND (sqlc.arg('include_account_filter') = false OR ae.target_account_id IN (sqlc.slice('account_ids')))
AND (sqlc.arg('include_resource_type_filter') = false OR ae.resource_type IN (sqlc.slice('resource_types')))
AND (sqlc.arg('include_resource_id_filter') = false OR ae.resource_id IN (sqlc.slice('resource_ids')))
AND (sqlc.arg('include_actor_id_filter') = false OR ae.actor_id IN (sqlc.slice('actor_ids')))
AND (sqlc.arg('include_action_filter') = false OR ae.action IN (sqlc.slice('actions')))
AND (sqlc.narg('start_date') IS NULL OR ae.occurred_at >= sqlc.narg('start_date'))
AND (sqlc.narg('end_date') IS NULL OR ae.occurred_at <= sqlc.narg('end_date'))
AND (
    sqlc.narg('search_query') IS NULL
    OR ae.resource_type LIKE sqlc.narg('search_query')
    OR ae.action LIKE sqlc.narg('search_query')
    OR ae.resource_id LIKE sqlc.narg('search_query')
    OR ae.request_id LIKE sqlc.narg('search_query')
)
AND (
    sqlc.narg('cursor_occurred_at') IS NULL
    OR ae.occurred_at < sqlc.narg('cursor_occurred_at')
    OR (ae.occurred_at = sqlc.narg('cursor_occurred_at') AND ae.type_id < sqlc.narg('cursor_id'))
)
ORDER BY ae.occurred_at DESC, ae.type_id DESC
LIMIT ?;

-- name: ListAuditEventsBackward :many
SELECT ae.type_id,
       ae.actor_id AS actor_id,
       ae.actor_type,
       ae.identity_type,
       ae.account_id,
       ae.action,
       ae.resource_type,
       ae.resource_id,
       COALESCE(CASE WHEN sqlc.arg('include_changes') THEN ae.changes ELSE NULL END, '') AS changes,
       COALESCE(CASE WHEN sqlc.arg('include_metadata') THEN ae.metadata ELSE NULL END, '') AS metadata,
       ae.service_name,
       ae.request_id,
       ae.idempotency_key_id,
       ae.source_ip,
       ae.occurred_at,
       ae.created_at,
       ae.target_account_id,
       a.name AS account_name,
       a.created_at AS account_created_at,
       a.updated_at AS account_updated_at,
       u.name AS user_name,
       u.email AS user_email,
       ak.name AS api_key_name,
       ak.redacted_value AS api_key_redacted_value,
       ik.idempotency_key
FROM audit_event ae
-- actor_id stores the raw actor key (user_id / api_key.type_id) — exposed
-- directly. Enrichment joins key on it: user by id, api_key by type_id.
LEFT JOIN `user` u ON ae.actor_id = u.id AND ae.identity_type = 'user'
LEFT JOIN api_key ak ON ae.actor_id = ak.type_id AND ae.identity_type = 'api_key'
LEFT JOIN idempotency_key ik ON ae.idempotency_key_id = ik.type_id
-- Resolves the target account (target_account_id) for the `account` sub-resource.
LEFT JOIN account a ON ae.target_account_id = a.id
WHERE ae.account_id = sqlc.arg('target_account_id')
AND (sqlc.arg('include_account_filter') = false OR ae.target_account_id IN (sqlc.slice('account_ids')))
AND (sqlc.arg('include_resource_type_filter') = false OR ae.resource_type IN (sqlc.slice('resource_types')))
AND (sqlc.arg('include_resource_id_filter') = false OR ae.resource_id IN (sqlc.slice('resource_ids')))
AND (sqlc.arg('include_actor_id_filter') = false OR ae.actor_id IN (sqlc.slice('actor_ids')))
AND (sqlc.arg('include_action_filter') = false OR ae.action IN (sqlc.slice('actions')))
AND (sqlc.narg('start_date') IS NULL OR ae.occurred_at >= sqlc.narg('start_date'))
AND (sqlc.narg('end_date') IS NULL OR ae.occurred_at <= sqlc.narg('end_date'))
AND (
    sqlc.narg('search_query') IS NULL
    OR ae.resource_type LIKE sqlc.narg('search_query')
    OR ae.action LIKE sqlc.narg('search_query')
    OR ae.resource_id LIKE sqlc.narg('search_query')
    OR ae.request_id LIKE sqlc.narg('search_query')
)
AND (
    ae.occurred_at > sqlc.arg('cursor_occurred_at')
    OR (ae.occurred_at = sqlc.arg('cursor_occurred_at') AND ae.type_id > sqlc.arg('cursor_id'))
)
ORDER BY ae.occurred_at ASC, ae.type_id ASC
LIMIT ?;

-- name: DeleteExpiredAuditEvents :execresult
DELETE FROM audit_event
WHERE occurred_at < DATE_SUB(NOW(3), INTERVAL 7 YEAR)
LIMIT ?;

-- name: CreateAuditEvent :exec
INSERT IGNORE INTO audit_event (
        type_id,
        actor_id,
        actor_type,
        identity_type,
        account_id,
        target_account_id,
        action,
        resource_type,
        resource_id,
        changes,
        metadata,
        service_name,
        request_id,
        idempotency_key_id,
        source_ip,
        occurred_at
    )
VALUES (
        ?,
        ?,
        ?,
        ?,
        ?,
        ?,
        ?,
        ?,
        ?,
        ?,
        ?,
        ?,
        ?,
        ?,
        ?,
        ?
    );

