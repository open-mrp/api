-- name: ListPricingPlansForward :many
SELECT id, created_at, type_id, name, plan_type_code, price_per_seat, price_per_month,
       seat_minimum, display_features, display_order, is_highlighted,
       button_text, includes_previous_plan
FROM account_plan
WHERE (expires_at IS NULL OR expires_at > NOW(3))
  AND effective_at <= NOW(3)
  AND plan_type_code != 'enterprise'
  AND (
    sqlc.narg('cursor_created_at') IS NULL
    OR account_plan.created_at < sqlc.narg('cursor_created_at')
    OR (account_plan.created_at = sqlc.narg('cursor_created_at') AND account_plan.id < sqlc.narg('cursor_id'))
  )
ORDER BY account_plan.created_at DESC, account_plan.id DESC
LIMIT ?;

-- name: ListPricingPlansBackward :many
SELECT id, created_at, type_id, name, plan_type_code, price_per_seat, price_per_month,
       seat_minimum, display_features, display_order, is_highlighted,
       button_text, includes_previous_plan
FROM account_plan
WHERE (expires_at IS NULL OR expires_at > NOW(3))
  AND effective_at <= NOW(3)
  AND plan_type_code != 'enterprise'
  AND (
    account_plan.created_at > sqlc.arg('cursor_created_at')
    OR (account_plan.created_at = sqlc.arg('cursor_created_at') AND account_plan.id > sqlc.arg('cursor_id'))
  )
ORDER BY account_plan.created_at ASC, account_plan.id ASC
LIMIT ?;

-- name: GetPlanByCode :one
SELECT id, created_at, type_id, name, plan_type_code, price_per_seat, price_per_month,
       seat_minimum, display_features, display_order, is_highlighted,
       button_text, includes_previous_plan
FROM account_plan
WHERE plan_type_code = ?
  AND (expires_at IS NULL OR expires_at > NOW(3))
  AND effective_at <= NOW(3)
LIMIT 1;

-- name: GetPlanByTypeID :one
SELECT id, created_at, type_id, name, plan_type_code, price_per_seat, price_per_month,
       seat_minimum, display_features, display_order, is_highlighted,
       button_text, includes_previous_plan
FROM account_plan
WHERE type_id = ?
  AND (expires_at IS NULL OR expires_at > NOW(3))
  AND effective_at <= NOW(3)
LIMIT 1;

-- name: GetPlanLimitsByTypeID :many
SELECT `key`, value
FROM account_plan_limit
WHERE account_plan_id = ?
ORDER BY `key` ASC;
