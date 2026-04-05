-- name: InsertTokenPackPurchase :exec
INSERT INTO token_pack_purchase (id, account_id, pack_id, token_count, price_cents, status)
VALUES (?, ?, ?, ?, ?, ?);

-- name: UpdateTokenPackPurchaseStatus :exec
UPDATE token_pack_purchase
SET status = ?
WHERE id = ?;

-- name: UpdateTokenPackPurchasePaymentIntent :exec
UPDATE token_pack_purchase
SET stripe_payment_intent_id = ?
WHERE id = ?;

-- name: UpdateTokenPackPurchaseCheckoutSession :exec
UPDATE token_pack_purchase
SET stripe_checkout_session_id = ?
WHERE id = ?;

-- name: GetCompletedTokensByAccount :one
SELECT CAST(COALESCE(SUM(token_count), 0) AS SIGNED) AS total_tokens
FROM token_pack_purchase
WHERE account_id = ? AND status = 'completed';

-- name: GetTokenPackPurchaseByPaymentIntent :one
SELECT id, account_id, pack_id, token_count, price_cents, stripe_payment_intent_id, stripe_checkout_session_id, status, created_at, updated_at
FROM token_pack_purchase
WHERE stripe_payment_intent_id = ?;
