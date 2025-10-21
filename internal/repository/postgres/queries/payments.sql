-- name: CreatePayment :one
INSERT INTO payments (
    id, amount, currency, status, payment_method, customer_id, order_id,
    description, stripe_payment_intent_id, stripe_session_id, metadata
) VALUES (
    sqlc.arg(id), sqlc.arg(amount), sqlc.arg(currency), sqlc.arg(status),
    sqlc.arg(payment_method), sqlc.arg(customer_id), sqlc.arg(order_id),
    sqlc.arg(description), sqlc.narg(stripe_payment_intent_id),
    sqlc.narg(stripe_session_id), sqlc.arg(metadata)
) RETURNING *;

-- name: GetPaymentByID :one
SELECT * FROM payments WHERE id = sqlc.arg(id);

-- name: GetPaymentByOrderID :one
SELECT * FROM payments WHERE order_id = sqlc.arg(order_id);

-- name: GetPaymentsByCustomerID :many
SELECT * FROM payments 
WHERE customer_id = sqlc.arg(customer_id)
ORDER BY created_at DESC
LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset');

-- name: UpdatePayment :one
UPDATE payments 
SET 
    amount = sqlc.arg(amount),
    currency = sqlc.arg(currency),
    status = sqlc.arg(status),
    payment_method = sqlc.arg(payment_method),
    description = sqlc.arg(description),
    stripe_payment_intent_id = sqlc.narg(stripe_payment_intent_id),
    stripe_session_id = sqlc.narg(stripe_session_id),
    metadata = sqlc.arg(metadata),
    updated_at = NOW()
WHERE id = sqlc.arg(id)
RETURNING *;

-- name: UpdatePaymentStatus :one
UPDATE payments 
SET status = sqlc.arg(status), updated_at = NOW()
WHERE id = sqlc.arg(id)
RETURNING *;

-- name: DeletePayment :exec
DELETE FROM payments WHERE id = sqlc.arg(id);

-- name: ListPayments :many
SELECT * FROM payments 
ORDER BY created_at DESC
LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset');

-- name: CountPayments :one
SELECT COUNT(*) FROM payments;

