-- name: ListPricingPlansForward :many
SELECT id, created_at, type_id, name, plan_type_code, price_per_seat, price_per_month,
       seat_minimum, display_features, display_order, is_highlighted,
       button_text, includes_previous_plan, stripe_pricing_plan_id
FROM account_plan
WHERE (expires_at IS NULL OR expires_at > NOW(3))
  AND effective_at <= NOW(3)
  AND is_publicly_visible = 1
  AND (
    sqlc.narg('cursor_created_at') IS NULL
    OR account_plan.created_at < sqlc.narg('cursor_created_at')
    OR (account_plan.created_at = sqlc.narg('cursor_created_at') AND account_plan.id < sqlc.narg('cursor_id'))
  )
  AND (
    sqlc.narg('search_query') IS NULL
    OR account_plan.name LIKE sqlc.narg('search_query')
  )
ORDER BY account_plan.created_at DESC, account_plan.id DESC
LIMIT ?;

-- name: ListPricingPlansBackward :many
SELECT id, created_at, type_id, name, plan_type_code, price_per_seat, price_per_month,
       seat_minimum, display_features, display_order, is_highlighted,
       button_text, includes_previous_plan, stripe_pricing_plan_id
FROM account_plan
WHERE (expires_at IS NULL OR expires_at > NOW(3))
  AND effective_at <= NOW(3)
  AND is_publicly_visible = 1
  AND (
    account_plan.created_at > sqlc.arg('cursor_created_at')
    OR (account_plan.created_at = sqlc.arg('cursor_created_at') AND account_plan.id > sqlc.arg('cursor_id'))
  )
  AND (
    sqlc.narg('search_query') IS NULL
    OR account_plan.name LIKE sqlc.narg('search_query')
  )
ORDER BY account_plan.created_at ASC, account_plan.id ASC
LIMIT ?;

-- name: GetPlanByCode :one
SELECT id, created_at, type_id, name, plan_type_code, price_per_seat, price_per_month,
       seat_minimum, display_features, display_order, is_highlighted,
       button_text, includes_previous_plan, stripe_pricing_plan_id
FROM account_plan
WHERE plan_type_code = ?
  AND (expires_at IS NULL OR expires_at > NOW(3))
  AND effective_at <= NOW(3)
LIMIT 1;

-- name: GetPlanByTypeID :one
SELECT id, created_at, type_id, name, plan_type_code, price_per_seat, price_per_month,
       seat_minimum, display_features, display_order, is_highlighted,
       button_text, includes_previous_plan, stripe_pricing_plan_id
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
