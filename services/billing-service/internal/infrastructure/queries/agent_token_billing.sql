-- name: UpsertAgentTokenBilling :exec
INSERT INTO agent_token_billing (id, account_id, period_start, period_end, total_input_tokens, total_output_tokens, total_tokens, run_count)
VALUES (?, ?, ?, ?, ?, ?, ?, 1)
ON DUPLICATE KEY UPDATE
  total_input_tokens = total_input_tokens + VALUES(total_input_tokens),
  total_output_tokens = total_output_tokens + VALUES(total_output_tokens),
  total_tokens = total_tokens + VALUES(total_tokens),
  run_count = run_count + 1;

-- name: GetAgentTokenBillingByAccountAndPeriod :one
SELECT id, account_id, period_start, period_end, total_input_tokens, total_output_tokens, total_tokens, tokens_reported_to_stripe, stripe_metered_item_id, run_count, created_at, updated_at
FROM agent_token_billing
WHERE account_id = ? AND period_start = ?;

-- name: UpdateTokensReportedToStripe :exec
UPDATE agent_token_billing
SET tokens_reported_to_stripe = ?, stripe_metered_item_id = ?
WHERE id = ?;

-- name: GetAgentTokenUsageSummary :one
SELECT COALESCE(total_tokens, 0) AS total_tokens
FROM agent_token_billing
WHERE account_id = ? AND period_start = ?;
