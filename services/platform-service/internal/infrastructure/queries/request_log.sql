-- name: FindRequestLogByID :one
SELECT rl.id, rl.method, rl.host, rl.path, rl.normalized_route,
       COALESCE(CASE WHEN sqlc.arg('include_query_json') THEN rl.query_json ELSE NULL END, '') AS query_json,
       rl.status_code, rl.latency_us, rl.api_version, rl.actor_id AS actor_id,
       rl.actor_type, rl.identity_type, rl.client_ip_string, rl.user_agent,
       rl.referrer, rl.error_code, rl.error_message, rl.occurred_at, rl.created_at,
       rl.idempotency_key_id,
       COALESCE(CASE WHEN sqlc.arg('include_request_body_json') THEN rl.request_body_json ELSE NULL END, '') AS request_body_json,
       COALESCE(CASE WHEN sqlc.arg('include_response_body_json') THEN rl.response_body_json ELSE NULL END, '') AS response_body_json,
       u.email AS user_email, u.name AS user_name,
       ak.type_id AS api_key_type_id, ak.redacted_value AS api_key_redacted_value,
       ak.name AS api_key_name,
       au.role_id AS user_role_id,
       r_user.name AS user_role_name, r_user.role_type_code AS user_role_type_code,
       r_key.id AS api_key_role_id, r_key.name AS api_key_role_name, r_key.role_type_code AS api_key_role_type_code,
       rl.target_account_id, a.name AS account_name,
       a.created_at AS account_created_at, a.updated_at AS account_updated_at,
       ik.idempotency_key
FROM request_log rl
-- actor_id stores the raw actor key (user_id for user actors, api_key.type_id
-- for api_key actors) — the value the API exposes directly. Enrichment joins key
-- on it: user by id, api_key by type_id, and account_user (for the role) by
-- (user_id, target account).
LEFT JOIN `user` u ON rl.actor_id = u.id AND rl.identity_type = 'user'
LEFT JOIN api_key ak ON rl.actor_id = ak.type_id AND rl.identity_type = 'api_key'
LEFT JOIN account_user au ON au.user_id = rl.actor_id
  AND au.account_id = rl.target_account_id AND rl.identity_type = 'user'
LEFT JOIN role r_user ON au.role_id = r_user.id
LEFT JOIN role r_key ON ak.role_id = r_key.id
LEFT JOIN account a ON rl.target_account_id = a.id
LEFT JOIN idempotency_key ik ON rl.idempotency_key_id = ik.type_id
-- Visible when the caller's account is either the acting account or the target.
WHERE rl.id = sqlc.arg('id')
  AND (rl.account_id = sqlc.arg('caller_account_id') OR rl.target_account_id = sqlc.arg('caller_account_id'));

-- Request log list queries are built dynamically in Go — see
-- repository/request_log_list_query.go. Filter predicates are only emitted
-- when the caller supplied a value. The actor-or-target security scope is an OR
-- over account_id / target_account_id, satisfied by index_merge over the
-- per-side (…, occurred_at DESC, id DESC) cursor indexes.

-- name: FindRequestLogBaseByID :one
SELECT rl.id, rl.method, rl.host, rl.path, rl.normalized_route,
       COALESCE(CASE WHEN sqlc.arg('include_query_json') THEN rl.query_json ELSE NULL END, '') AS query_json,
       rl.status_code, rl.latency_us, rl.api_version, rl.actor_id AS actor_id,
       rl.actor_type, rl.identity_type, rl.client_ip_string, rl.user_agent,
       rl.referrer, rl.error_code, rl.error_message, rl.occurred_at, rl.created_at,
       rl.idempotency_key_id,
       COALESCE(CASE WHEN sqlc.arg('include_request_body_json') THEN rl.request_body_json ELSE NULL END, '') AS request_body_json,
       COALESCE(CASE WHEN sqlc.arg('include_response_body_json') THEN rl.response_body_json ELSE NULL END, '') AS response_body_json,
       rl.target_account_id,
       ik.idempotency_key
FROM request_log rl
LEFT JOIN idempotency_key ik ON rl.idempotency_key_id = ik.type_id
-- Visible when the caller's account is either the acting account or the target.
WHERE rl.id = sqlc.arg('id')
  AND (rl.account_id = sqlc.arg('caller_account_id') OR rl.target_account_id = sqlc.arg('caller_account_id'));

-- name: DeleteExpiredRequestLogs :execresult
DELETE FROM request_log
WHERE occurred_at < DATE_SUB(NOW(3), INTERVAL 7 YEAR)
LIMIT ?;

-- name: CreateRequestLog :exec
INSERT INTO request_log (
        id,
        method,
        host,
        path,
        normalized_route,
        query_json,
        status_code,
        latency_us,
        account_id,
        target_account_id,
        client_ip,
        client_ip_string,
        user_agent,
        referrer,
        error_code,
        error_message,
        occurred_at,
        created_at,
        idempotency_key_id,
        actor_id,
        actor_type,
        internal_error_message,
        stack_trace,
        identity_type,
        api_version,
        trace_id,
        public_endpoint,
        hidden,
        request_body_json,
        response_body_json
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