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
-- Visible when the caller's account is either the acting account or the target.
WHERE ae.type_id = sqlc.arg('type_id')
  AND (ae.account_id = sqlc.arg('caller_account_id_actor') OR ae.target_account_id = sqlc.arg('caller_account_id_target'));

-- name: BatchGetResourceCreators :many
-- Returns the `create` audit-event actor for each of the given resource IDs,
-- scoped to the caller's account. Backs the `created_by` include without granting
-- full audit-event read access. actor_type carries the relation
-- (internal/customer/supplier); identity_type the kind (user/api_key).
SELECT ae.resource_id,
       ae.actor_id,
       ae.actor_type,
       ae.identity_type,
       u.name AS user_name,
       u.email AS user_email,
       ak.name AS api_key_name,
       ak.redacted_value AS api_key_redacted_value
FROM audit_event ae
LEFT JOIN `user` u ON ae.actor_id = u.id AND ae.identity_type = 'user'
LEFT JOIN api_key ak ON ae.actor_id = ak.type_id AND ae.identity_type = 'api_key'
WHERE ae.resource_type = sqlc.arg('resource_type')
  AND ae.action = 'create'
  AND ae.resource_id IN (sqlc.slice('resource_ids'))
  AND (ae.account_id = sqlc.arg('caller_account_id_actor') OR ae.target_account_id = sqlc.arg('caller_account_id_target'));

-- name: DeleteExpiredAuditEvents :execresult
DELETE FROM audit_event
WHERE occurred_at < DATE_SUB(NOW(3), INTERVAL 12 MONTH)
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
        root_resource_type,
        root_resource_id,
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
        ?,
        ?,
        ?
    );


-- name: GetResourceCreateChanges :one
-- The field changes recorded when the resource was created (a full snapshot of its audited fields). Lets consumers recover display attributes (e.g. an order number) without a cross-service lookup.
SELECT changes FROM audit_event
WHERE account_id = ?
  AND resource_type = ?
  AND resource_id = ?
  AND action = 'create'
ORDER BY occurred_at ASC
LIMIT 1;

-- name: GetResourceCreateEvent :one
-- The resource's create event: who created it, in which request, and the snapshot of its audited fields at creation. Served by audit_event_resource_idx (resource_type, resource_id, occurred_at).
SELECT actor_id, identity_type, request_id, changes FROM audit_event
WHERE account_id = ?
  AND resource_type = ?
  AND resource_id = ?
  AND action = 'create'
ORDER BY occurred_at ASC
LIMIT 1;

-- name: GetLatestResourceFieldChanges :one
-- The changes of the newest event on the resource (its create snapshot included) that recorded the field, i.e. the event holding the field's current value. A delete's snapshot records every field with a null new value, so deletes are skipped: the value is the one the resource held when it was deleted. Walks the resource's own events newest first on audit_event_account_id_resource_id_occurred_at_type_id_idx and stops at the first match, so the cost is bounded by the resource's event count — forced because audit_event_resource_idx pins the same rows but filesorts the type_id tie-break.
SELECT changes FROM audit_event FORCE INDEX (audit_event_account_id_resource_id_occurred_at_type_id_idx)
WHERE account_id = sqlc.arg('account_id')
  AND resource_id = sqlc.arg('resource_id')
  AND resource_type = sqlc.arg('resource_type')
  AND action <> 'delete'
  AND JSON_CONTAINS(changes, JSON_OBJECT('Field', CAST(sqlc.arg('field') AS CHAR)))
ORDER BY occurred_at DESC, type_id DESC
LIMIT 1;

-- name: GetAccountUserUserID :one
-- The user behind an account membership, scoped to the account. Primary-key lookup.
SELECT user_id FROM account_user
WHERE id = ?
  AND account_id = ?;
