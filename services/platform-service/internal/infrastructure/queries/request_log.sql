-- name: FindRequestLogByID :one
SELECT rl.id, rl.method, rl.host, rl.path, rl.normalized_route, rl.query_json,
       rl.status_code, rl.latency_us, rl.api_version, rl.actor_id,
       rl.actor_type, rl.identity_type, rl.client_ip_string, rl.user_agent,
       rl.referrer, rl.error_code, rl.error_message, rl.occurred_at, rl.created_at,
       rl.idempotency_key_id, rl.request_body_json, rl.response_body_json,
       u.email AS user_email, u.name AS user_name,
       ak.type_id AS api_key_type_id, ak.redacted_value AS api_key_redacted_value,
       ak.name AS api_key_name,
       au.role_id AS user_role_id,
       r_user.name AS user_role_name, r_user.role_type_code AS user_role_type_code,
       r_key.id AS api_key_role_id, r_key.name AS api_key_role_name, r_key.role_type_code AS api_key_role_type_code,
       rl.target_account_id, a.name AS account_name,
       ik.idempotency_key
FROM request_log rl
LEFT JOIN `user` u ON rl.actor_id = u.id AND rl.identity_type = 'user'
LEFT JOIN api_key ak ON rl.actor_id = ak.key_id AND rl.identity_type = 'api_key'
LEFT JOIN account_user au ON au.user_id = rl.actor_id
  AND au.account_id = rl.target_account_id AND rl.identity_type = 'user'
LEFT JOIN role r_user ON au.role_id = r_user.id
LEFT JOIN role r_key ON ak.role_id = r_key.id
LEFT JOIN account a ON rl.target_account_id = a.id
LEFT JOIN idempotency_key ik ON rl.idempotency_key_id = ik.type_id
WHERE rl.id = ? AND rl.target_account_id = ?;

-- name: ListRequestLogsForward :many
SELECT rl.id, rl.method, rl.host, rl.path, rl.normalized_route, rl.query_json,
       rl.status_code, rl.latency_us, rl.api_version, rl.actor_id,
       rl.actor_type, rl.identity_type, rl.client_ip_string, rl.user_agent,
       rl.referrer, rl.error_code, rl.error_message, rl.occurred_at, rl.created_at,
       rl.idempotency_key_id, rl.request_body_json, rl.response_body_json,
       u.email AS user_email, u.name AS user_name,
       ak.type_id AS api_key_type_id, ak.redacted_value AS api_key_redacted_value,
       ak.name AS api_key_name,
       au.role_id AS user_role_id,
       r_user.name AS user_role_name, r_user.role_type_code AS user_role_type_code,
       r_key.id AS api_key_role_id, r_key.name AS api_key_role_name, r_key.role_type_code AS api_key_role_type_code,
       rl.target_account_id, a.name AS account_name,
       ik.idempotency_key
FROM request_log rl
LEFT JOIN `user` u ON rl.actor_id = u.id AND rl.identity_type = 'user'
LEFT JOIN api_key ak ON rl.actor_id = ak.key_id AND rl.identity_type = 'api_key'
LEFT JOIN account_user au ON au.user_id = rl.actor_id
  AND au.account_id = rl.target_account_id AND rl.identity_type = 'user'
LEFT JOIN role r_user ON au.role_id = r_user.id
LEFT JOIN role r_key ON ak.role_id = r_key.id
LEFT JOIN account a ON rl.target_account_id = a.id
LEFT JOIN idempotency_key ik ON rl.idempotency_key_id = ik.type_id
WHERE rl.target_account_id = sqlc.arg('target_account_id')
AND (sqlc.arg('query_filter') = '' OR rl.id = sqlc.arg('query_filter') OR rl.path LIKE sqlc.arg('query_path_filter') OR rl.error_message LIKE sqlc.arg('query_error_msg_filter'))
AND (sqlc.narg('start_date') IS NULL OR rl.occurred_at >= sqlc.narg('start_date'))
AND (sqlc.narg('end_date') IS NULL OR rl.occurred_at <= sqlc.narg('end_date'))
AND (sqlc.arg('method_filter') = '' OR rl.method LIKE sqlc.arg('method_filter'))
AND (sqlc.narg('status_code') IS NULL OR rl.status_code = sqlc.narg('status_code'))
AND (sqlc.arg('error_code_filter') = '' OR rl.error_code LIKE sqlc.arg('error_code_filter'))
AND (sqlc.arg('account_id_filter') = '' OR rl.account_id = sqlc.arg('account_id_filter'))
AND (sqlc.arg('actor_id_filter') = '' OR rl.actor_id = sqlc.arg('actor_id_filter'))
AND (sqlc.arg('actor_type_filter') = '' OR rl.identity_type = sqlc.arg('actor_type_filter'))
AND (sqlc.arg('actor_name_filter') = '' OR u.name LIKE sqlc.arg('actor_name_filter') OR ak.name LIKE sqlc.arg('actor_name_filter'))
AND (sqlc.narg('public_endpoint') IS NULL OR rl.public_endpoint = sqlc.narg('public_endpoint'))
AND (
    sqlc.narg('cursor_occurred_at') IS NULL
    OR rl.occurred_at < sqlc.narg('cursor_occurred_at')
    OR (rl.occurred_at = sqlc.narg('cursor_occurred_at') AND rl.id < sqlc.narg('cursor_id'))
)
ORDER BY rl.occurred_at DESC, rl.id DESC
LIMIT ?;

-- name: ListRequestLogsBackward :many
SELECT rl.id, rl.method, rl.host, rl.path, rl.normalized_route, rl.query_json,
       rl.status_code, rl.latency_us, rl.api_version, rl.actor_id,
       rl.actor_type, rl.identity_type, rl.client_ip_string, rl.user_agent,
       rl.referrer, rl.error_code, rl.error_message, rl.occurred_at, rl.created_at,
       rl.idempotency_key_id, rl.request_body_json, rl.response_body_json,
       u.email AS user_email, u.name AS user_name,
       ak.type_id AS api_key_type_id, ak.redacted_value AS api_key_redacted_value,
       ak.name AS api_key_name,
       au.role_id AS user_role_id,
       r_user.name AS user_role_name, r_user.role_type_code AS user_role_type_code,
       r_key.id AS api_key_role_id, r_key.name AS api_key_role_name, r_key.role_type_code AS api_key_role_type_code,
       rl.target_account_id, a.name AS account_name,
       ik.idempotency_key
FROM request_log rl
LEFT JOIN `user` u ON rl.actor_id = u.id AND rl.identity_type = 'user'
LEFT JOIN api_key ak ON rl.actor_id = ak.key_id AND rl.identity_type = 'api_key'
LEFT JOIN account_user au ON au.user_id = rl.actor_id
  AND au.account_id = rl.target_account_id AND rl.identity_type = 'user'
LEFT JOIN role r_user ON au.role_id = r_user.id
LEFT JOIN role r_key ON ak.role_id = r_key.id
LEFT JOIN account a ON rl.target_account_id = a.id
LEFT JOIN idempotency_key ik ON rl.idempotency_key_id = ik.type_id
WHERE rl.target_account_id = sqlc.arg('target_account_id')
AND (sqlc.arg('query_filter') = '' OR rl.id = sqlc.arg('query_filter') OR rl.path LIKE sqlc.arg('query_path_filter') OR rl.error_message LIKE sqlc.arg('query_error_msg_filter'))
AND (sqlc.narg('start_date') IS NULL OR rl.occurred_at >= sqlc.narg('start_date'))
AND (sqlc.narg('end_date') IS NULL OR rl.occurred_at <= sqlc.narg('end_date'))
AND (sqlc.arg('method_filter') = '' OR rl.method LIKE sqlc.arg('method_filter'))
AND (sqlc.narg('status_code') IS NULL OR rl.status_code = sqlc.narg('status_code'))
AND (sqlc.arg('error_code_filter') = '' OR rl.error_code LIKE sqlc.arg('error_code_filter'))
AND (sqlc.arg('account_id_filter') = '' OR rl.account_id = sqlc.arg('account_id_filter'))
AND (sqlc.arg('actor_id_filter') = '' OR rl.actor_id = sqlc.arg('actor_id_filter'))
AND (sqlc.arg('actor_type_filter') = '' OR rl.identity_type = sqlc.arg('actor_type_filter'))
AND (sqlc.arg('actor_name_filter') = '' OR u.name LIKE sqlc.arg('actor_name_filter') OR ak.name LIKE sqlc.arg('actor_name_filter'))
AND (sqlc.narg('public_endpoint') IS NULL OR rl.public_endpoint = sqlc.narg('public_endpoint'))
AND (
    rl.occurred_at > sqlc.arg('cursor_occurred_at')
    OR (rl.occurred_at = sqlc.arg('cursor_occurred_at') AND rl.id > sqlc.arg('cursor_id'))
)
ORDER BY rl.occurred_at ASC, rl.id ASC
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
        ?
    );