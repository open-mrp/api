-- name: InsertOrderPaymentIntent :exec
INSERT INTO order_payment_intent (id, payment_intent_id, sales_order_id, created_at, updated_at)
VALUES (sqlc.arg('id'), sqlc.arg('payment_intent_id'), sqlc.arg('sales_order_id'), NOW(3), NOW(3));

-- name: FindOrderPaymentIntentByPaymentIntentID :one
SELECT id, payment_intent_id, sales_order_id
FROM order_payment_intent
WHERE payment_intent_id = sqlc.arg('payment_intent_id')
LIMIT 1;

-- name: DeleteOrderPaymentIntent :exec
DELETE FROM order_payment_intent WHERE id = sqlc.arg('id');
