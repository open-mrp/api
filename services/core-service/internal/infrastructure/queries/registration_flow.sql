-- name: ListRegistrationFlowsForward :many
SELECT
    rf.id,
    rf.name,
    rf.account_id,
    rf.created_at,
    rf.updated_at
FROM registration_flow rf
WHERE rf.account_id = sqlc.arg('account_id')
AND (
    sqlc.narg('search_query') IS NULL
    OR rf.name LIKE sqlc.narg('search_query')
)
AND (
    sqlc.narg('cursor_created_at') IS NULL
    OR rf.created_at < sqlc.narg('cursor_created_at')
    OR (rf.created_at = sqlc.narg('cursor_created_at') AND rf.id < sqlc.narg('cursor_id'))
)
ORDER BY rf.created_at DESC, rf.id DESC
LIMIT ?;

-- name: ListRegistrationFlowsBackward :many
SELECT
    rf.id,
    rf.name,
    rf.account_id,
    rf.created_at,
    rf.updated_at
FROM registration_flow rf
WHERE rf.account_id = sqlc.arg('account_id')
AND (
    sqlc.narg('search_query') IS NULL
    OR rf.name LIKE sqlc.narg('search_query')
)
AND (
    rf.created_at > sqlc.arg('cursor_created_at')
    OR (rf.created_at = sqlc.arg('cursor_created_at') AND rf.id > sqlc.arg('cursor_id'))
)
ORDER BY rf.created_at ASC, rf.id ASC
LIMIT ?;

-- name: GetRegistrationFlow :one
SELECT
    rf.id,
    rf.name,
    rf.account_id,
    rf.created_at,
    rf.updated_at
FROM registration_flow rf
WHERE rf.id = sqlc.arg('id')
AND rf.account_id = sqlc.arg('account_id');

-- name: InsertRegistrationFlow :exec
INSERT INTO registration_flow (
    id,
    name,
    account_id,
    created_at,
    updated_at
) VALUES (
    sqlc.arg('id'),
    sqlc.arg('name'),
    sqlc.arg('account_id'),
    NOW(3),
    NOW(3)
);

-- name: UpdateRegistrationFlow :execresult
UPDATE registration_flow SET
    name = COALESCE(sqlc.narg('name'), name),
    updated_at = NOW(3)
WHERE id = sqlc.arg('id')
AND account_id = sqlc.arg('account_id');

-- name: DeleteRegistrationFlow :execresult
DELETE FROM registration_flow
WHERE id = sqlc.arg('id')
AND account_id = sqlc.arg('account_id');

-- name: ListRegistrationFlowsByAccountID :many
SELECT
    rf.id,
    rf.name,
    rf.account_id,
    rf.created_at,
    rf.updated_at
FROM registration_flow rf
WHERE rf.account_id = sqlc.arg('account_id');

-- name: ListPaymentTermOptionsByFlowID :many
SELECT
    pt.id,
    pt.name
FROM payment_term pt
JOIN `_payment_terms_registration_flows` j ON j.A = pt.id
WHERE j.B = sqlc.arg('registration_flow_id');

-- name: ListShippingTermOptionsByFlowID :many
SELECT
    st.id,
    st.name
FROM shipping_term st
JOIN `_registration_flows_shipping_terms` j ON j.A = sqlc.arg('registration_flow_id')
WHERE j.B = st.id;

-- name: ListAccountGroupOptionsByFlowID :many
SELECT
    ag.id,
    ag.name
FROM account_group ag
WHERE ag.registration_flow_id = sqlc.arg('registration_flow_id');

-- name: InsertPaymentTermOption :exec
INSERT INTO `_payment_terms_registration_flows` (A, B) VALUES (sqlc.arg('payment_term_id'), sqlc.arg('registration_flow_id'));

-- name: InsertShippingTermOption :exec
INSERT INTO `_registration_flows_shipping_terms` (A, B) VALUES (sqlc.arg('registration_flow_id'), sqlc.arg('shipping_term_id'));

-- name: DeletePaymentTermOptionsByFlowID :exec
DELETE FROM `_payment_terms_registration_flows` WHERE B = sqlc.arg('registration_flow_id');

-- name: DeleteShippingTermOptionsByFlowID :exec
DELETE FROM `_registration_flows_shipping_terms` WHERE A = sqlc.arg('registration_flow_id');

-- name: SetAccountGroupRegistrationFlowID :exec
UPDATE account_group SET registration_flow_id = sqlc.arg('registration_flow_id'), updated_at = NOW(3) WHERE id = sqlc.arg('account_group_id') AND owner_account_id = sqlc.arg('account_id');

-- name: ClearAccountGroupRegistrationFlowID :exec
UPDATE account_group SET registration_flow_id = NULL, updated_at = NOW(3) WHERE registration_flow_id = sqlc.arg('registration_flow_id') AND owner_account_id = sqlc.arg('account_id');
